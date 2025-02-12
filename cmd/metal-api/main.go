package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/Masterminds/semver/v3"
	"github.com/avast/retry-go/v4"
	v1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/service/s3client"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metrics"
	"github.com/metal-stack/metal-lib/auditing"
	"github.com/metal-stack/metal-lib/rest"

	nsq2 "github.com/nsqio/go-nsq"

	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/metal-lib/jwt/sec"
	"github.com/metal-stack/metal-lib/pkg/pointer"

	"connectrpc.com/connect"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	compress "github.com/klauspost/connect-compress/v2"
	apiv1 "github.com/metal-stack/go-ipam/api/v1"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/masterdata-api/pkg/auth"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	_ "github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore/migrations"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/eventbus"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/headscale"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/service"
	bus "github.com/metal-stack/metal-lib/bus"
	httperrors "github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/security"
	"github.com/metal-stack/v"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type dsConnectOpt int

const (
	cfgFileType = "yaml"
	moduleName  = "metal-api"

	// DataStoreConnectTableInit connects to the data store and then runs data store initialization
	DataStoreConnectTableInit dsConnectOpt = 0
	// DataStoreConnectNoDemotion connects to the data store without demoting to runtime user in the end
	DataStoreConnectNoDemotion dsConnectOpt = 1
)

var (
	logger *slog.Logger

	ds                 *datastore.RethinkStore
	ipamer             ipam.IPAMer
	publisherTLSConfig *bus.TLSConfig
	nsqer              *eventbus.NSQClient
	mdc                mdm.Client
	headscaleClient    *headscale.HeadscaleClient
)

var rootCmd = &cobra.Command{
	Use:           moduleName,
	Short:         "an api to offer pure metal",
	Version:       v.V.String(),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging()
		initMetrics()

		var opts []dsConnectOpt
		if viper.GetBool("init-data-store") {
			opts = append(opts, DataStoreConnectTableInit)
		}
		err := connectDataStore(opts...)
		if err != nil {
			return err
		}

		initEventBus()
		initIpam()
		initMasterData()
		initSignalHandlers()
		err = initHeadscale()
		if err != nil {
			return err
		}
		return run()
	},
}

var migrateDatabase = &cobra.Command{
	Use:     "migrate",
	Short:   "migrates the database to the latest version",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging()

		err := connectDataStore(DataStoreConnectNoDemotion)
		if err != nil {
			return err
		}
		var targetVersion *int
		specificVersion := viper.GetInt("target-version")
		if specificVersion != -1 {
			targetVersion = &specificVersion
		}
		return ds.Migrate(targetVersion, viper.GetBool("dry-run"))
	},
}

var dumpSwagger = &cobra.Command{
	Use:     "dump-swagger",
	Short:   "dump the current swagger configuration",
	Version: v.V.String(),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()
		dumpSwaggerJSON()
	},
}

var initDatabase = &cobra.Command{
	Use:     "initdb",
	Short:   "initializes the database with all tables and indices",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging()

		return connectDataStore(DataStoreConnectTableInit, DataStoreConnectNoDemotion)
	},
}

var resurrectMachines = &cobra.Command{
	Use:     "resurrect-machines",
	Short:   "resurrect dead machines",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging()

		return resurrectDeadMachines()
	},
}

var machineLiveliness = &cobra.Command{
	Use:     "machine-liveliness",
	Short:   "evaluates whether machines are still alive or if they have died",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging()

		return evaluateLiveliness()
	},
}
var machineConnectedToVPN = &cobra.Command{
	Use:     "machines-vpn-connected",
	Short:   "evaluates whether machines connected to vpn",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging()
		err := initHeadscale()
		if err != nil {
			return err
		}
		return evaluateVPNConnected()
	},
}

var deleteOrphanImagesCmd = &cobra.Command{
	Use:     "delete-orphan-images",
	Short:   "delete orphan images",
	Long:    "removes images which are expired and not used by any allocated machine, still one image per operating system is preserved",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		initLogging()
		err := connectDataStore()
		if err != nil {
			return err
		}
		initEventBus()

		_, err = ds.DeleteOrphanImages(nil, nil)
		return err
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("failed executing root command: %s", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(
		dumpSwagger,
		initDatabase,
		migrateDatabase,
		resurrectMachines,
		machineLiveliness,
		deleteOrphanImagesCmd,
		machineConnectedToVPN,
	)

	rootCmd.Flags().StringP("config", "c", "", "alternative path to config file")
	rootCmd.Flags().String("log-level", "info", "the log level of the application")

	rootCmd.Flags().StringP("bind-addr", "", "127.0.0.1", "the bind addr of the api server")
	rootCmd.Flags().IntP("port", "", 8080, "the port to serve on")
	rootCmd.Flags().IntP("grpc-port", "", 50051, "the port to serve gRPC on")
	rootCmd.Flags().Bool("init-data-store", true, "initializes the data store on start (can be switched off when running the init command before starting instances)")
	rootCmd.Flags().UintP("password-reason-minlength", "", 0, "if machine console password is requested this defines if and how long the given reason must be")

	rootCmd.Flags().StringP("base-path", "", "/", "the base path of the api server")

	rootCmd.Flags().StringP("s3-address", "", "", "the address of the s3 server that provides firmwares")
	rootCmd.Flags().StringP("s3-key", "", "", "the key of the s3 server that provides firmwares")
	rootCmd.Flags().StringP("s3-secret", "", "", "the secret of the s3 server that provides firmwares")
	rootCmd.Flags().StringP("s3-firmware-bucket", "", "", "the bucket that contains the firmwares")

	rootCmd.PersistentFlags().StringP("db", "", "rethinkdb", "the database adapter to use")
	rootCmd.PersistentFlags().StringP("db-name", "", "metalapi", "the database name to use")
	rootCmd.PersistentFlags().StringP("db-addr", "", "", "the database address string to use")
	rootCmd.PersistentFlags().StringP("db-user", "", "", "the database user to use")
	rootCmd.PersistentFlags().StringP("db-password", "", "", "the database password to use")

	rootCmd.Flags().String("ipam-grpc-server-endpoint", "http://ipam:9090", "the ipam grpc server endpoint")

	rootCmd.Flags().StringP("metrics-server-bind-addr", "", ":2112", "the bind addr of the metrics server")

	rootCmd.Flags().StringP("nsqd-tcp-addr", "", "", "the TCP address of the nsqd")
	rootCmd.Flags().StringP("nsqd-http-endpoint", "", "nsqd:4151", "the address of the nsqd http endpoint")
	rootCmd.Flags().StringP("nsqd-ca-cert-file", "", "", "the CA certificate file to verify nsqd certificate")
	rootCmd.Flags().StringP("nsqd-client-cert-file", "", "", "the client certificate file to access nsqd")
	rootCmd.Flags().StringP("nsqd-write-timeout", "", "10s", "the write timeout for nsqd")
	rootCmd.Flags().StringP("nsqlookupd-addr", "", "", "the http addresses of the nsqlookupd as a commalist")

	rootCmd.Flags().StringP("grpc-tls-enabled", "", "false", "indicates whether gRPC TLS is enabled")
	rootCmd.Flags().StringP("grpc-ca-cert-file", "", "", "the CA certificate file to verify gRPC certificate")
	rootCmd.Flags().StringP("grpc-server-cert-file", "", "", "the gRPC server certificate file")
	rootCmd.Flags().StringP("grpc-server-key-file", "", "", "the gRPC server key file")

	rootCmd.Flags().StringP("bmc-superuser-pwd-file", "", "", "the path to the BMC superuser password file")

	rootCmd.Flags().StringP("hmac-view-key", "", "must-be-changed", "the preshared key for hmac security for a viewing user")
	rootCmd.Flags().StringP("hmac-view-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("hmac-edit-key", "", "must-be-changed", "the preshared key for hmac security for a editing user")
	rootCmd.Flags().StringP("hmac-edit-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("hmac-admin-key", "", "must-be-changed", "the preshared key for hmac security for a admin user")
	rootCmd.Flags().StringP("hmac-admin-lifetime", "", "90s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("provider-tenant", "", "", "the tenant of the maas-provider who operates the whole thing")
	rootCmd.Flags().StringP("issuercache-interval", "", "30m", "issuercache invalidation interval, e.g. 60s, 30m, 2h45m - default 30m")

	rootCmd.Flags().StringP("masterdata-hmac", "", "must-be-changed", "the preshared key for hmac security to talk to the masterdata-api")
	rootCmd.Flags().StringP("masterdata-hostname", "", "", "the hostname of the masterdata-api")
	rootCmd.Flags().IntP("masterdata-port", "", 8443, "the port of the masterdata-api")
	rootCmd.Flags().StringP("masterdata-capath", "", "", "the tls ca certificate to talk to the masterdata-api")
	rootCmd.Flags().StringP("masterdata-certpath", "", "", "the tls certificate to talk to the masterdata-api")
	rootCmd.Flags().StringP("masterdata-certkeypath", "", "", "the tls certificate key to talk to the masterdata-api")

	rootCmd.Flags().Bool("auditing-enabled", false, "enable auditing")
	rootCmd.Flags().String("auditing-search-backend", "", "the auditing backend used as a source for search in the audit service. if explicitly specified the first one configured is picked given the following order of precedence: timescaledb,meilisearch")

	rootCmd.Flags().String("auditing-meili-url", "http://localhost:7700", "url of the auditing service")
	rootCmd.Flags().String("auditing-meili-api-key", "secret", "api key for the auditing service")
	rootCmd.Flags().String("auditing-meili-index-prefix", "auditing", "auditing index prefix")
	rootCmd.Flags().String("auditing-meili-index-interval", "@daily", "auditing index creation interval, can be one of @hourly|@daily|@monthly")
	rootCmd.Flags().Int64("auditing-meili-keep", 14, "the amount of indexes to keep until cleanup")

	rootCmd.Flags().String("auditing-timescaledb-host", "", "host of the auditing service")
	rootCmd.Flags().String("auditing-timescaledb-port", "", "port of the auditing service")
	rootCmd.Flags().String("auditing-timescaledb-db", "", "database name of the auditing service")
	rootCmd.Flags().String("auditing-timescaledb-user", "", "user for the auditing service")
	rootCmd.Flags().String("auditing-timescaledb-password", "", "password for the auditing service")
	rootCmd.Flags().String("auditing-timescaledb-retention", "", "the time until audit traces are cleaned up")

	rootCmd.Flags().String("headscale-addr", "", "address of headscale server")
	rootCmd.Flags().String("headscale-cp-addr", "", "address of headscale control plane")
	rootCmd.Flags().String("headscale-api-key", "", "initial api key to connect to headscale server")

	rootCmd.Flags().StringP("minimum-client-version", "", "v0.0.1", "the minimum metalctl version required to talk to this version of metal-api")
	rootCmd.Flags().String("release-version", "", "the metal-stack release version")

	must(viper.BindPFlags(rootCmd.Flags()))
	must(viper.BindPFlags(rootCmd.PersistentFlags()))

	migrateDatabase.Flags().Int("target-version", -1, "the target version of the migration, when set to -1 will migrate to latest version")
	migrateDatabase.Flags().Bool("dry-run", false, "only shows which migrations would run, but does not execute them")

	must(viper.BindPFlags(migrateDatabase.Flags()))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}

func initConfig() {
	viper.SetEnvPrefix("METAL_API")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetConfigType(cfgFileType)

	if viper.IsSet("config") {
		viper.SetConfigFile(viper.GetString("config"))
		if err := viper.ReadInConfig(); err != nil {
			panic(fmt.Errorf("config file path set explicitly, but unreadable: %w", err))
		}
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("/etc/" + moduleName)
		viper.AddConfigPath("$HOME/." + moduleName)
		viper.AddConfigPath(".")
		if err := viper.ReadInConfig(); err != nil {
			usedCfg := viper.ConfigFileUsed()
			if usedCfg != "" {
				panic(fmt.Errorf("config file found at %q, but unreadable: %w", usedCfg, err))
			}
		}
	}
}

func initLogging() {
	level := slog.LevelInfo
	if viper.IsSet("log-level") {
		var (
			lvlvar slog.LevelVar
		)
		err := lvlvar.UnmarshalText([]byte(viper.GetString("log-level")))
		if err != nil {
			panic(fmt.Errorf("can't initialize logger: %w", err))
		}
		level = lvlvar.Level()
	}

	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func initMetrics() {
	logger.Info("starting metrics endpoint")
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	// see: https://dev.to/davidsbond/golang-debugging-memory-leaks-using-pprof-5di8
	// inspect via
	// go tool pprof -http :8080 localhost:2112/pprof/heap
	// go tool pprof -http :8080 localhost:2112/pprof/goroutine
	metricsServer.Handle("/pprof/heap", httppprof.Handler("heap"))
	metricsServer.Handle("/pprof/goroutine", httppprof.Handler("goroutine"))

	go func() {
		server := &http.Server{
			Addr:              viper.GetString("metrics-server-bind-addr"),
			Handler:           metricsServer,
			ReadHeaderTimeout: time.Minute,
		}

		err := server.ListenAndServe()
		if err != nil {
			logger.Error("failed to start metrics endpoint, exiting...", "error", err)
			os.Exit(1)
		}
		logger.Error("metrics server has stopped unexpectedly without an error")
	}()
}

func initSignalHandlers() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		logger.Error("Received keyboard interrupt, shutting down...")
		if ds != nil {
			logger.Info("Closing connection to datastore")
			err := ds.Close()
			if err != nil {
				logger.Info("Unable to properly shutdown datastore", "error", err)
				os.Exit(1)
			}
		}
		if headscaleClient != nil {
			logger.Info("Closing connection to Headscale")
			if err := headscaleClient.Close(); err != nil {
				logger.Info("Failed to close connection to Headscale", "error", err)
				os.Exit(1)
			}
		}

		os.Exit(0)
	}()
}

func initEventBus() {
	writeTimeout, err := time.ParseDuration(viper.GetString("nsqd-write-timeout"))
	if err != nil {
		writeTimeout = 0
	}
	caCertFile := viper.GetString("nsqd-ca-cert-file")
	clientCertFile := viper.GetString("nsqd-client-cert-file")
	if caCertFile != "" && clientCertFile != "" {
		publisherTLSConfig = &bus.TLSConfig{
			CACertFile:     caCertFile,
			ClientCertFile: clientCertFile,
		}
	}
	publisherCfg := &bus.PublisherConfig{
		TCPAddress:   viper.GetString("nsqd-tcp-addr"),
		HTTPEndpoint: viper.GetString("nsqd-http-endpoint"),
		TLS:          publisherTLSConfig,
		NSQ:          nsq2.NewConfig(),
	}
	publisherCfg.NSQ.WriteTimeout = writeTimeout

	partitions := waitForPartitions()

	nsq := eventbus.NewNSQ(publisherCfg, logger.WithGroup("nsq-eventbus"), bus.NewPublisher) // FIXME
	nsq.WaitForPublisher()
	nsq.WaitForTopicsCreated(partitions, metal.Topics)
	if err := nsq.CreateEndpoints(viper.GetString("nsqlookupd-addr")); err != nil {
		panic(err)
	}
	nsqer = &nsq
}

func waitForPartitions() metal.Partitions {
	var partitions metal.Partitions
	var err error
	for {
		partitions, err = ds.ListPartitions()
		if err != nil {
			logger.Error("cannot list partitions", "error", err)
			time.Sleep(3 * time.Second)
			continue
		}
		break
	}
	return partitions
}

func connectDataStore(opts ...dsConnectOpt) error {
	dbAdapter := viper.GetString("db")
	if dbAdapter == "rethinkdb" {
		ds = datastore.New(
			logger.WithGroup("datastore"),
			viper.GetString("db-addr"),
			viper.GetString("db-name"),
			viper.GetString("db-user"),
			viper.GetString("db-password"),
		)
	} else {
		return fmt.Errorf("database not supported: %v", dbAdapter)
	}

	initTables := false
	demote := true

	for _, opt := range opts {
		switch opt {
		case DataStoreConnectNoDemotion:
			demote = false
		case DataStoreConnectTableInit:
			initTables = true
		default:
			return errors.New("unsupported datastore connect option")
		}
	}

	err := ds.Connect()
	if err != nil {
		return fmt.Errorf("cannot connect to data store: %w", err)
	}

	if initTables {
		err := ds.Initialize()
		if err != nil {
			return fmt.Errorf("error initializing data store tables: %w", err)
		}
	}

	if demote {
		err = ds.Demote()
		if err != nil {
			return fmt.Errorf("error demoting to data store runtime user: %w", err)
		}
	}

	return nil
}

func initMasterData() {
	hmacKey := viper.GetString("masterdata-hmac")
	if hmacKey == "" {
		hmacKey = auth.HmacDefaultKey
	}

	ca := viper.GetString("masterdata-capath")
	if ca == "" {
		log.Fatal("no masterdata-api capath given")
	}

	certpath := viper.GetString("masterdata-certpath")
	if certpath == "" {
		log.Fatal("no masterdata-api certpath given")
	}

	certkeypath := viper.GetString("masterdata-certkeypath")
	if certkeypath == "" {
		log.Fatal("no masterdata-api certkeypath given")
	}

	hostname := viper.GetString("masterdata-hostname")
	if hostname == "" {
		log.Fatal("no masterdata-hostname given")
	}

	port := viper.GetInt("masterdata-port")
	if port == 0 {
		log.Fatal("no masterdata-port given")
	}

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	var err error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		mdc, err = mdm.NewClient(ctx, hostname, port, certpath, certkeypath, ca, hmacKey, false, log)
		if err == nil {
			cancel()
			break
		}
		logger.Error("unable to initialize masterdata-api client, retrying...", "error", err)
		time.Sleep(3 * time.Second)
	}

	logger.Info("masterdata client initialized")
}

func initIpam() {
	ipamgrpcendpoint := viper.GetString("ipam-grpc-server-endpoint")

	ipamService := apiv1connect.NewIpamServiceClient(
		http.DefaultClient,
		ipamgrpcendpoint,
		connect.WithGRPC(),
		compress.WithAll(compress.LevelBalanced),
	)

	ipamer = ipam.New(ipamService)

	err := retry.Do(func() error {
		version, err := ipamService.Version(context.Background(), connect.NewRequest(&apiv1.VersionRequest{}))
		if err != nil {
			return err
		}
		logger.Info("connected to ipam service", "version", version.Msg)
		return nil
	})

	if err != nil {
		logger.Error("unable to connect to ipam service", "error", err)
		os.Exit(1)
	}

	logger.Info("ipam initialized")
}

func initAuth(lg *slog.Logger) security.UserGetter {
	var auths []security.CredsOpt

	providerTenant := viper.GetString("provider-tenant")

	grpr, err := grp.NewGrpr(grp.Config{ProviderTenant: providerTenant})
	if err != nil {
		log.Fatalf("error creating grpr: %s", err)
	}
	plugin := sec.NewPlugin(grpr)

	issuerCacheInterval, err := time.ParseDuration(viper.GetString("issuercache-interval"))
	if err != nil {
		log.Fatalf("error parsing issuercache-interval: %s", err)
	}

	// create multi issuer cache that holds all trusted issuers from masterdata, in this case: only provider tenant
	// FIXME create a slog.Logger instance with the same log level as configured for zap and pass this logger instance
	issuerCache, err := security.NewMultiIssuerCache(nil, func() ([]*security.IssuerConfig, error) {
		logger.Info("loading tenants for issuercache", "providerTenant", providerTenant)

		// get provider tenant from masterdata
		ts, err := mdc.Tenant().Find(context.Background(), &v1.TenantFindRequest{
			Id: wrapperspb.String(providerTenant),
		})
		if err != nil {
			return nil, err
		}

		if len(ts.Tenants) != 1 {
			return nil, fmt.Errorf("no masterdata for tenant %s found", providerTenant)
		}

		t := ts.Tenants[0]
		if t.IamConfig != nil {
			directory := ""
			if t.IamConfig.IdmConfig != nil {
				directory = t.IamConfig.IdmConfig.IdmType
			}
			tenantID := t.Meta.Id
			return []*security.IssuerConfig{
				{
					Annotations: map[string]string{
						sec.OidcDirectory: directory,
					},
					Tenant:   tenantID,
					Issuer:   t.IamConfig.IssuerConfig.Url,
					ClientID: t.IamConfig.IssuerConfig.ClientId,
				},
			}, nil
		}
		return []*security.IssuerConfig{}, nil
	}, func(ic *security.IssuerConfig) (security.UserGetter, error) {
		return security.NewGenericOIDC(ic, security.GenericUserExtractor(plugin.GenericOIDCExtractUserProcessGroups))
	}, security.IssuerReloadInterval(issuerCacheInterval))

	if err != nil || issuerCache == nil {
		log.Fatalf("error creating dynamic oidc resolver: %s", err)
	}
	logger.Info("dynamic oidc resolver successfully initialized")

	var ugsOpts []security.UserGetterProxyOption
	dexClientID := viper.GetString("dex-clientid")
	dexAddr := viper.GetString("dex-addr")
	if dexAddr != "" {
		dx, err := security.NewDex(dexAddr)
		if err != nil {
			log.Fatalf("dex not reachable: %s", err)
		}
		if dx != nil {
			// use custom user extractor and group processor
			dx.With(security.UserExtractor(plugin.ExtractUserProcessGroups))
			ugsOpts = append(ugsOpts, security.UserGetterProxyMapping(dexAddr, dexClientID, dx))
			logger.Info("dex successfully configured")
		} else {
			log.Fatal("dex is configured, but not initialized")
		}
	}

	// UserGetterProxy with dynamic oidc as default and legacy dex as explicit mapping
	ugp := security.NewUserGetterProxy(issuerCache, ugsOpts...)

	// add UserGetterProxy as CredsOpt
	auths = append(auths, security.WithDex(ugp))

	defaultUsers := service.NewUserDirectory(providerTenant)
	for _, u := range defaultUsers.UserNames() {
		lfkey := fmt.Sprintf("hmac-%s-lifetime", u)
		mackey := viper.GetString(fmt.Sprintf("hmac-%s-key", u))
		lf, err := time.ParseDuration(viper.GetString(lfkey))
		if err != nil {
			lg.Warn("illegal value for hmac lifetime, use 30secs as default", "error", err, "val", lfkey)
			lf = 30 * time.Second
		}

		user := defaultUsers.Get(u)

		auths = append(auths, security.WithHMAC(security.NewHMACAuth(
			user.Name,
			[]byte(mackey),
			security.WithLifetime(lf),
			security.WithUser(user))))
	}

	return security.NewCreds(auths...)
}

func initRestServices(searchAuditBackend auditing.Auditing, allAuditBackends []auditing.Auditing, withauth bool, ipmiSuperUser metal.MachineIPMISuperUser) *restfulspec.Config {
	service.BasePath = viper.GetString("base-path")
	if !strings.HasPrefix(service.BasePath, "/") || !strings.HasSuffix(service.BasePath, "/") {
		log.Fatal("base path must start and end with a slash")
	}

	var p bus.Publisher
	ep := bus.DirectEndpoints()
	if nsqer != nil {
		p = nsqer.Publisher
		ep = nsqer.Endpoints
	}
	ipService, err := service.NewIP(logger.WithGroup("ip-service"), ds, ep, ipamer, mdc)
	if err != nil {
		log.Fatal(err)
	}

	var s3Client *s3client.Client
	s3Address := viper.GetString("s3-address")
	if s3Address != "" {
		s3Key := viper.GetString("s3-key")
		s3Secret := viper.GetString("s3-secret")
		s3FirmwareBucket := viper.GetString("s3-firmware-bucket")
		s3Client, err = s3client.New(s3Address, s3Key, s3Secret, s3FirmwareBucket)
		if err != nil {
			log.Fatal(err)
		}
		logger.Info("connected to s3 server that provides firmwares", "address", s3Address)
	} else {
		logger.Debug("s3 server that provides firmware is disabled")
	}
	firmwareService, err := service.NewFirmware(logger.WithGroup("firmware-service"), ds, s3Client)
	if err != nil {
		log.Fatal(err)
	}
	var userGetter security.UserGetter
	if withauth {
		userGetter = initAuth(logger)
	}
	reasonMinLength := viper.GetUint("password-reason-minlength")

	machineService, err := service.NewMachine(logger.WithGroup("machine-service"), ds, p, ep, ipamer, mdc, s3Client, userGetter, reasonMinLength, headscaleClient, ipmiSuperUser)
	if err != nil {
		log.Fatal(err)
	}

	firewallService, err := service.NewFirewall(logger.WithGroup("firewall-service"), ds, p, ipamer, ep, mdc, userGetter, headscaleClient)
	if err != nil {
		log.Fatal(err)
	}

	healthService, err := rest.NewHealth(logger, service.BasePath, ds, ipamer)
	if err != nil {
		log.Fatal(err)
	}

	minClientVersion, err := semver.NewVersion(viper.GetString("minimum-client-version"))
	if err != nil {
		log.Fatalf("given minimum client version is not semver parsable: %s", err)
	}

	var releaseVersion *string
	if viper.IsSet("release-version") {
		releaseVersion = pointer.Pointer(viper.GetString("release-version"))
	}

	restful.DefaultContainer.Add(service.NewAudit(logger.WithGroup("audit-service"), searchAuditBackend))
	restful.DefaultContainer.Add(service.NewPartition(logger.WithGroup("partition-service"), ds, nsqer))
	restful.DefaultContainer.Add(service.NewImage(logger.WithGroup("image-service"), ds))
	restful.DefaultContainer.Add(service.NewSize(logger.WithGroup("size-service"), ds, mdc))
	restful.DefaultContainer.Add(service.NewSizeImageConstraint(logger.WithGroup("size-image-constraint-service"), ds))
	restful.DefaultContainer.Add(service.NewNetwork(logger.WithGroup("network-service"), ds, ipamer, mdc))
	restful.DefaultContainer.Add(ipService)
	restful.DefaultContainer.Add(firmwareService)
	restful.DefaultContainer.Add(machineService)
	restful.DefaultContainer.Add(service.NewProject(logger.WithGroup("project-service"), ds, mdc))
	restful.DefaultContainer.Add(service.NewTenant(logger.WithGroup("tenant-service"), mdc))
	restful.DefaultContainer.Add(service.NewUser(logger.WithGroup("user-service"), userGetter))
	restful.DefaultContainer.Add(firewallService)
	restful.DefaultContainer.Add(service.NewFilesystemLayout(logger.WithGroup("filesystem-layout-service"), ds))
	restful.DefaultContainer.Add(service.NewSwitch(logger.WithGroup("switch-service"), ds))
	restful.DefaultContainer.Add(healthService)
	restful.DefaultContainer.Add(service.NewVPN(logger.WithGroup("vpn-service"), headscaleClient))
	restful.DefaultContainer.Add(rest.NewVersion(moduleName, &rest.VersionOpts{
		BasePath:         service.BasePath,
		MinClientVersion: minClientVersion.Original(),
		ReleaseVersion:   releaseVersion,
	}))
	restful.DefaultContainer.Filter(rest.RequestLoggerFilter(logger)) // FIXME
	restful.DefaultContainer.Filter(metrics.RestfulMetrics)

	if withauth {
		restful.DefaultContainer.Filter(rest.UserAuth(userGetter, logger)) // FIXME
		providerTenant := viper.GetString("provider-tenant")
		excludedPathSuffixes := []string{"liveliness", "health", "version", "apidocs.json"}
		ensurer := service.NewTenantEnsurer(logger.WithGroup("tenant-ensurer-filter"), []string{providerTenant}, excludedPathSuffixes)
		restful.DefaultContainer.Filter(ensurer.EnsureAllowedTenantFilter)
	}

	for _, backend := range allAuditBackends {
		httpFilter, err := auditing.HttpFilter(backend, logger.WithGroup("audit-middleware"))
		if err != nil {
			log.Fatalf("unable to create http filter for auditing: %s", err)
		}
		restful.DefaultContainer.Filter(httpFilter) // FIXME
	}

	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(), // you control what services are visible
		APIPath:                       service.BasePath + "apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))
	return &config
}

func initHeadscale() error {
	var err error

	if !viper.IsSet("headscale-addr") {
		logger.Info("headscale disabled")
		return nil
	}

	headscaleClient, err = headscale.NewHeadscaleClient(
		viper.GetString("headscale-addr"),
		viper.GetString("headscale-cp-addr"),
		viper.GetString("headscale-api-key"),
		logger.WithGroup("headscale"),
	)
	if err != nil || headscaleClient == nil {
		return fmt.Errorf("failed to init headscale client %w", err)
	}

	logger.Info("headscale initialized")
	return nil
}

func dumpSwaggerJSON() {
	// This is required to make dump work
	ipamer = ipam.New(nil)
	cfg := initRestServices(nil, nil, false, metal.DisabledIPMISuperUser())
	actual := restfulspec.BuildSwagger(*cfg)

	// declare custom type for default errors, see:
	// https://github.com/go-swagger/go-swagger/blob/master/docs/use/models/schemas.md#using-custom-types
	// amongst other things, this has the advantage that the Error() function for printing of the original
	// type is preserved.
	//
	// unfortunately, gorestful does not support injecting the type, therefore we need to forcefully
	// add the definition into the spec definition
	customGoType := map[string]interface{}{
		"x-go-type": map[string]interface{}{
			"type": "HTTPErrorResponse",
			"import": map[string]interface{}{
				"package": "github.com/metal-stack/metal-lib/httperrors",
			},
		},
	}
	httpErrDef := actual.Definitions["httperrors.HTTPErrorResponse"]
	httpErrDef.ExtraProps = customGoType
	actual.Definitions["httperrors.HTTPErrorResponse"] = httpErrDef

	js, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", js)
}

func resurrectDeadMachines() error {
	err := connectDataStore()
	if err != nil {
		return err
	}
	initEventBus()
	initIpam()

	var p bus.Publisher
	ep := bus.DirectEndpoints()
	if nsqer != nil {
		p = nsqer.Publisher
		ep = nsqer.Endpoints
	}
	err = service.ResurrectMachines(context.Background(), ds, p, ep, ipamer, headscaleClient, logger)
	if err != nil {
		return fmt.Errorf("unable to resurrect machines: %w", err)
	}

	return nil
}

func evaluateLiveliness() error {
	err := connectDataStore()
	if err != nil {
		return err
	}

	err = service.MachineLiveliness(ds, logger)
	if err != nil {
		return fmt.Errorf("unable to evaluate machine liveliness: %w", err)
	}

	return nil
}

func evaluateVPNConnected() error {
	err := connectDataStore()
	if err != nil {
		return err
	}

	return service.EvaluateVPNConnected(logger, ds, headscaleClient)
}

// might return (nil, nil) if auditing is disabled!
func createAuditingClient(log *slog.Logger) (searchBackend auditing.Auditing, backends []auditing.Auditing, err error) {
	isEnabled := viper.GetBool("auditing-enabled")
	if !isEnabled {
		log.Warn("auditing is disabled, can be enabled by setting --auditing-enabled=true")
		return nil, nil, nil
	}

	c := auditing.Config{
		Component: "metal-api",
		Log:       log,
	}

	if viper.IsSet("auditing-timescaledb-host") {
		backend, err := auditing.NewTimescaleDB(c, auditing.TimescaleDbConfig{
			Host:      viper.GetString("auditing-timescaledb-host"),
			Port:      viper.GetString("auditing-timescaledb-port"),
			DB:        viper.GetString("auditing-timescaledb-db"),
			User:      viper.GetString("auditing-timescaledb-user"),
			Password:  viper.GetString("auditing-timescaledb-password"),
			Retention: viper.GetString("auditing-timescaledb-retention"),
		})

		if err != nil {
			return nil, nil, err
		}

		backends = append(backends, backend)

		if viper.GetString("auditing-search-backend") == "timescaledb" {
			searchBackend = backend
		}
	}

	if viper.IsSet("auditing-meili-api-key") {
		backend, err := auditing.NewMeilisearch(c, auditing.MeilisearchConfig{
			URL:              viper.GetString("auditing-meili-url"),
			APIKey:           viper.GetString("auditing-meili-api-key"),
			IndexPrefix:      viper.GetString("auditing-meili-index-prefix"),
			RotationInterval: auditing.Interval(viper.GetString("auditing-meili-index-interval")),
			Keep:             viper.GetInt64("auditing-meili-keep"),
		})

		if err != nil {
			return nil, nil, err
		}

		backends = append(backends, backend)

		if viper.GetString("auditing-search-backend") == "meilisearch" {
			searchBackend = backend
		}
	}

	if searchBackend == nil {
		searchBackend = pointer.FirstOrZero(backends)
	}

	return searchBackend, backends, nil
}

func run() error {
	ipmiSuperUser := metal.NewIPMISuperUser(logger, viper.GetString("bmc-superuser-pwd-file"))

	auditSearchBackend, allAuditBackends, err := createAuditingClient(logger)
	if err != nil {
		log.Fatalf("cannot create auditing client:%s ", err)
	}
	initRestServices(auditSearchBackend, allAuditBackends, true, ipmiSuperUser)

	// enable OPTIONS-request so clients can query CORS information
	restful.DefaultContainer.Filter(restful.DefaultContainer.OPTIONSFilter)

	// enable CORS for the UI to work.
	// if we will add support for api-tokens as headers, we had to add them
	// here to. note: the token's should not contain the product (aka. metal)
	// because customers should have ONE token for many products.
	// ExposeHeaders:  []string{"X-TOKEN"},
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept", "Authorization"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer,
	}
	restful.DefaultContainer.Filter(cors.Filter)

	// catch all other errors
	restful.DefaultContainer.Add(new(restful.WebService).Path("/"))
	restful.DefaultContainer.ServiceErrorHandler(func(serviceErr restful.ServiceError, request *restful.Request, response *restful.Response) {
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(serviceErr.Code)
		err := response.WriteAsJson(httperrors.NewHTTPError(serviceErr.Code, errors.New(serviceErr.Message)))
		if err != nil {
			logger.Error("Failed to send response", "error", err)
			return
		}
	})

	var p bus.Publisher
	if nsqer != nil {
		p = nsqer.Publisher
	}

	c, err := bus.NewConsumer(logger, publisherTLSConfig, viper.GetString("nsqlookupd-addr"))
	if err != nil {
		log.Fatalf("cannot connect to NSQ: %s", err)
	}

	addr := fmt.Sprintf(":%d", viper.GetInt("grpc-port"))
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("cannot create grpc server listener on addr %s: %s", addr, err)
		return err
	}

	go func() {
		err = grpc.Run(&grpc.ServerConfig{
			Context:                  context.Background(),
			Publisher:                p,
			Consumer:                 c,
			Store:                    ds,
			Logger:                   logger,
			Listener:                 listener,
			TlsEnabled:               viper.GetBool("grpc-tls-enabled"),
			CaCertFile:               viper.GetString("grpc-ca-cert-file"),
			ServerCertFile:           viper.GetString("grpc-server-cert-file"),
			ServerKeyFile:            viper.GetString("grpc-server-key-file"),
			BMCSuperUserPasswordFile: viper.GetString("bmc-superuser-pwd-file"),
			Auditing:                 allAuditBackends,
			IPMISuperUser:            ipmiSuperUser,
		})
		if err != nil {
			log.Fatalf("error running grpc server:%s", err)
		}
	}()

	addr = fmt.Sprintf("%s:%d", viper.GetString("bind-addr"), viper.GetInt("port"))
	server := &http.Server{
		Addr:              addr,
		Handler:           restful.DefaultContainer,
		ReadHeaderTimeout: time.Minute,
	}

	logger.Info("start metal api", "version", v.V.String(), "address", addr, "base-path", service.BasePath)

	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("failed to start metal api: %w", err)
	}

	return nil
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       moduleName,
			Description: "API to manage and control plane resources like machines, switches, operating system images, machine sizes, networks, IP addresses and more",
			Contact: &spec.ContactInfo{
				ContactInfoProps: spec.ContactInfoProps{
					Name: "metal-stack",
					URL:  "https://metal-stack.io",
				},
			},
			License: &spec.License{
				LicenseProps: spec.LicenseProps{
					Name: "AGPL-3.0",
					URL:  "https://www.gnu.org/licenses/agpl-3.0.de.html",
				},
			},
		},
	}
	swo.Tags = []spec.Tag{
		{TagProps: spec.TagProps{
			Name:        "audit",
			Description: "Managing audit entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "image",
			Description: "Managing image entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "network",
			Description: "Managing network entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "ip",
			Description: "Managing ip entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "size",
			Description: "Managing size entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "machine",
			Description: "Managing machine entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "partition",
			Description: "Managing partition entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "project",
			Description: "Managing project entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "switch",
			Description: "Managing switch entities",
		}},
		{TagProps: spec.TagProps{
			Name:        "user",
			Description: "Managing user entities",
		}},
	}

	hmacspec := spec.APIKeyAuth("Authorization", "header")
	hmacspec.Description = "Generate a 'Authorization: Metal xxxx' header where 'xxxx' is a HMAC generated by the request-date, the request-method and the body"
	jwtspec := spec.APIKeyAuth("Authorization", "header")
	jwtspec.Description = "Add a 'Authorization: Bearer xxxx' header to the request where 'xxxx' is the OIDC token retrieved from the identity provider's authentication endpoint"

	swo.SecurityDefinitions = spec.SecurityDefinitions{
		"HMAC": hmacspec,
		"jwt":  jwtspec,
	}
	swo.BasePath = viper.GetString("base-path")
	swo.Security = []map[string][]string{
		{"HMAC": []string{}},
		{"jwt": []string{}},
	}

	// Maybe this leads to an issue, investigating...:
	// swo.Schemes = []string{"http", "https"}
}
