package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	v1 "github.com/metal-stack/masterdata-api/api/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/service/s3client"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/go-logr/zapr"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metrics"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	nsq2 "github.com/nsqio/go-nsq"

	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/metal-lib/jwt/sec"

	_ "github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore/migrations"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/eventbus"

	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/masterdata-api/pkg/auth"
	mdm "github.com/metal-stack/masterdata-api/pkg/client"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/ipam"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/service"
	bus "github.com/metal-stack/metal-lib/bus"
	httperrors "github.com/metal-stack/metal-lib/httperrors"
	zapup "github.com/metal-stack/metal-lib/zapup"
	"github.com/metal-stack/security"
	"github.com/metal-stack/v"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
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
	cfgFile            string
	ds                 *datastore.RethinkStore
	eventds            *datastore.RethinkStore
	ipamer             *ipam.Ipam
	publisherTLSConfig *bus.TLSConfig
	nsqer              *eventbus.NSQClient
	logger             = zapup.MustRootLogger().Sugar()
	debug              = false
	mdc                mdm.Client
	grpcServer         *grpc.Server
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
		initGrpcServer()
		return run()
	},
}

var migrateDatabase = &cobra.Command{
	Use:     "migrate",
	Short:   "migrates the database to the latest version",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
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
		dumpSwaggerJSON()
	},
}

var initDatabase = &cobra.Command{
	Use:     "initdb",
	Short:   "initializes the database with all tables and indices",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		return connectDataStore(DataStoreConnectTableInit, DataStoreConnectNoDemotion)
	},
}

var resurrectMachines = &cobra.Command{
	Use:     "resurrect-machines",
	Short:   "resurrect dead machines",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		return resurrectDeadMachines()
	},
}

var machineLiveliness = &cobra.Command{
	Use:     "machine-liveliness",
	Short:   "evaluates whether machines are still alive or if they have died",
	Version: v.V.String(),
	RunE: func(cmd *cobra.Command, args []string) error {
		return evaluateLiveliness()
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
		logger.Fatalw("failed executing root command", "error", err)
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
	)

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "alternative path to config file")

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

	rootCmd.Flags().Bool("event-db-enabled", false, "store events in a separate database")
	rootCmd.PersistentFlags().StringP("event-db-name", "", "metalapi", "the event database name to use")
	rootCmd.PersistentFlags().StringP("event-db-addr", "", "", "the event database address string to use")
	rootCmd.PersistentFlags().StringP("event-db-user", "", "", "the event database user to use")
	rootCmd.PersistentFlags().StringP("event-db-password", "", "", "the event database password to use")

	rootCmd.Flags().StringP("ipam-db", "", "postgres", "the database adapter to use")
	rootCmd.Flags().StringP("ipam-db-name", "", "metal-ipam", "the database name to use")
	rootCmd.Flags().StringP("ipam-db-addr", "", "", "the database address string to use")
	rootCmd.Flags().StringP("ipam-db-port", "", "5432", "the database port string to use")
	rootCmd.Flags().StringP("ipam-db-user", "", "", "the database user to use")
	rootCmd.Flags().StringP("ipam-db-password", "", "", "the database password to use")

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

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		logger.Error("unable to construct root command:%w", err)
	}

	err = viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		logger.Error("unable to construct root command:%w", err)
	}

	migrateDatabase.Flags().Int("target-version", -1, "the target version of the migration, when set to -1 will migrate to latest version")
	migrateDatabase.Flags().Bool("dry-run", false, "only shows which migrations would run, but does not execute them")
	err = viper.BindPFlags(migrateDatabase.Flags())
	if err != nil {
		logger.Error("unable to construct migrate command:%w", err)
	}
}

func initConfig() {
	viper.SetEnvPrefix("METAL_API")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetConfigType(cfgFileType)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			logger.Error("Config file path set explicitly, but unreadable", "error", err)
		}
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("/etc/" + moduleName)
		viper.AddConfigPath("$HOME/." + moduleName)
		viper.AddConfigPath(".")
		if err := viper.ReadInConfig(); err != nil {
			usedCfg := viper.ConfigFileUsed()
			if usedCfg != "" {
				logger.Error("Config file unreadable", "config-file", usedCfg, "error", err)
			}
		}
	}

	usedCfg := viper.ConfigFileUsed()
	if usedCfg != "" {
		logger.Info("Read config file", "config-file", usedCfg)
	}
}

func initLogging() {
	debug = logger.Desugar().Core().Enabled(zap.DebugLevel)
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
		err := http.ListenAndServe(viper.GetString("metrics-server-bind-addr"), metricsServer)
		if err != nil {
			logger.Errorw("failed to start metrics endpoint, exiting...", "error", err)
			os.Exit(1)
		}
		logger.Errorw("metrics server has stopped unexpectedly without an error")
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
			os.Exit(0)
		}
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

	nsq := eventbus.NewNSQ(publisherCfg, zapup.MustRootLogger(), bus.NewPublisher)
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
			logger.Errorw("cannot list partitions", "error", err)
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
			logger.Desugar(),
			viper.GetString("db-addr"),
			viper.GetString("db-name"),
			viper.GetString("db-user"),
			viper.GetString("db-password"),
		)
		if viper.GetBool("event-db-enabled") {
			eventds = datastore.New(
				logger.Desugar(),
				viper.GetString("event-db-addr"),
				viper.GetString("event-db-name"),
				viper.GetString("event-db-user"),
				viper.GetString("event-db-password"),
			)
		}
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

	if !viper.GetBool("event-db-enabled") {
		eventds = ds
		return nil
	}

	err = eventds.Connect()
	if err != nil {
		return fmt.Errorf("cannot connect to event data store: %w", err)
	}

	if initTables {
		err := eventds.Initialize()
		if err != nil {
			return fmt.Errorf("error initializing event data store tables: %w", err)
		}
	}

	if demote {
		err = eventds.Demote()
		if err != nil {
			return fmt.Errorf("error demoting to event data store runtime user: %w", err)
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
		logger.Fatal("no masterdata-api capath given")
	}

	certpath := viper.GetString("masterdata-certpath")
	if certpath == "" {
		logger.Fatal("no masterdata-api certpath given")
	}

	certkeypath := viper.GetString("masterdata-certkeypath")
	if certkeypath == "" {
		logger.Fatal("no masterdata-api certkeypath given")
	}

	hostname := viper.GetString("masterdata-hostname")
	if hostname == "" {
		logger.Fatal("no masterdata-hostname given")
	}

	port := viper.GetInt("masterdata-port")
	if port == 0 {
		logger.Fatal("no masterdata-port given")
	}

	var err error
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		mdc, err = mdm.NewClient(ctx, hostname, port, certpath, certkeypath, ca, hmacKey, logger.Desugar())
		if err == nil {
			cancel()
			break
		}
		logger.Errorw("unable to initialize masterdata-api client, retrying...", "error", err)
		time.Sleep(3 * time.Second)
	}

	logger.Info("masterdata client initialized")
}

func initIpam() {
	dbAdapter := viper.GetString("ipam-db")
	switch dbAdapter {
	case "postgres":
		pgStorage, err := goipam.NewPostgresStorage(
			viper.GetString("ipam-db-addr"),
			viper.GetString("ipam-db-port"),
			viper.GetString("ipam-db-user"),
			viper.GetString("ipam-db-password"),
			viper.GetString("ipam-db-name"),
			goipam.SSLModeDisable)
		if err != nil {
			logger.Errorw("cannot connect to db in root command metal-api/internal/main.initIpam()", "error", err)
			time.Sleep(3 * time.Second)
			initIpam()
			return
		}
		ipamInstance := goipam.NewWithStorage(pgStorage)
		ipamer = ipam.New(ipamInstance)
	case "memory":
		ipamInstance := goipam.New()
		ipamer = ipam.New(ipamInstance)
	default:
		logger.Errorw("database not supported", "db", dbAdapter)
	}
	logger.Info("ipam initialized")
}

func initAuth(lg *zap.SugaredLogger) security.UserGetter {
	var auths []security.CredsOpt

	providerTenant := viper.GetString("provider-tenant")

	grpr, err := grp.NewGrpr(grp.Config{ProviderTenant: providerTenant})
	if err != nil {
		logger.Fatalw("error creating grpr", "error", err)
	}
	plugin := sec.NewPlugin(grpr)

	issuerCacheInterval, err := time.ParseDuration(viper.GetString("issuercache-interval"))
	if err != nil {
		logger.Fatalw("error parsing issuercache-interval", "error", err)
	}

	// create multi issuer cache that holds all trusted issuers from masterdata, in this case: only provider tenant
	issuerCache, err := security.NewMultiIssuerCache(func() ([]*security.IssuerConfig, error) {
		logger.Infow("loading tenants for issuercache", "providerTenant", providerTenant)

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
	}, security.IssuerReloadInterval(issuerCacheInterval), security.Logger(zapr.NewLogger(logger.Desugar())))

	if err != nil || issuerCache == nil {
		logger.Fatalw("error creating dynamic oidc resolver", "error", err)
	}
	logger.Info("dynamic oidc resolver successfully initialized")

	var ugsOpts []security.UserGetterProxyOption
	dexClientID := viper.GetString("dex-clientid")
	dexAddr := viper.GetString("dex-addr")
	if dexAddr != "" {
		dx, err := security.NewDex(dexAddr)
		if err != nil {
			logger.Fatalw("dex not reachable", "error", err)
		}
		if dx != nil {
			// use custom user extractor and group processor
			dx.With(security.UserExtractor(plugin.ExtractUserProcessGroups))
			ugsOpts = append(ugsOpts, security.UserGetterProxyMapping(dexAddr, dexClientID, dx))
			logger.Info("dex successfully configured")
		} else {
			logger.Fatal("dex is configured, but not initialized")
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
			lg.Warnw("illegal value for hmac lifetime, use 30secs as default", "error", err, "val", lfkey)
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

func initGrpcServer() {
	var p bus.Publisher
	if nsqer != nil {
		p = nsqer.Publisher
	}
	var err error
	grpcServer, err = grpc.NewServer(&grpc.ServerConfig{
		Publisher:                p,
		Datasource:               ds,
		Logger:                   logger,
		NsqTlsConfig:             publisherTLSConfig,
		NsqlookupdHttpAddress:    viper.GetString("nsqlookupd-addr"),
		GrpcPort:                 viper.GetInt("grpc-port"),
		TlsEnabled:               viper.GetBool("grpc-tls-enabled"),
		CaCertFile:               viper.GetString("grpc-ca-cert-file"),
		ServerCertFile:           viper.GetString("grpc-server-cert-file"),
		ServerKeyFile:            viper.GetString("grpc-server-key-file"),
		BMCSuperUserPasswordFile: viper.GetString("bmc-superuser-pwd-file"),
	})
	if err != nil {
		logger.Fatalw("cannot connect to NSQ", "error", err)
	}
}

func initRestServices(withauth bool) *restfulspec.Config {
	service.BasePath = viper.GetString("base-path")
	if !strings.HasPrefix(service.BasePath, "/") || !strings.HasSuffix(service.BasePath, "/") {
		logger.Fatal("base path must start and end with a slash")
	}

	lg := logger.Desugar()
	var p bus.Publisher
	ep := bus.DirectEndpoints()
	if nsqer != nil {
		p = nsqer.Publisher
		ep = nsqer.Endpoints
	}
	ipService, err := service.NewIP(ds, ep, ipamer, mdc)
	if err != nil {
		logger.Fatal(err)
	}

	var s3Client *s3client.Client
	s3Address := viper.GetString("s3-address")
	if s3Address != "" {
		s3Key := viper.GetString("s3-key")
		s3Secret := viper.GetString("s3-secret")
		s3FirmwareBucket := viper.GetString("s3-firmware-bucket")
		s3Client, err = s3client.New(s3Address, s3Key, s3Secret, s3FirmwareBucket)
		if err != nil {
			logger.Fatal(err)
		}
		logger.Infow("connected to s3 server that provides firmwares", "address", s3Address)
	} else {
		logger.Info("s3 server that provides firmware is disabled")
	}
	firmwareService, err := service.NewFirmware(ds, s3Client)
	if err != nil {
		logger.Fatal(err)
	}
	var userGetter security.UserGetter
	if withauth {
		userGetter = initAuth(lg.Sugar())
	}
	reasonMinLength := viper.GetUint("password-reason-minlength")

	machineService, err := service.NewMachine(ds, p, ep, ipamer, mdc, grpcServer, s3Client, userGetter, reasonMinLength)
	if err != nil {
		logger.Fatal(err)
	}
	firewallService, err := service.NewFirewall(ds, ipamer, ep, mdc, grpcServer, userGetter)
	if err != nil {
		logger.Fatal(err)
	}

	healthService, err := rest.NewHealth(lg, service.BasePath, ds)
	if err != nil {
		logger.Fatal(err)
	}
	eventService, err := service.NewEvent(eventds, mdc, userGetter)
	if err != nil {
		logger.Fatal(err)
	}

	restful.DefaultContainer.Add(service.NewPartition(ds, nsqer))
	restful.DefaultContainer.Add(service.NewImage(ds))
	restful.DefaultContainer.Add(service.NewSize(ds))
	restful.DefaultContainer.Add(service.NewNetwork(ds, ipamer, mdc))
	restful.DefaultContainer.Add(ipService)
	restful.DefaultContainer.Add(firmwareService)
	restful.DefaultContainer.Add(machineService)
	restful.DefaultContainer.Add(service.NewProject(ds, mdc))
	restful.DefaultContainer.Add(service.NewTenant(mdc))
	restful.DefaultContainer.Add(service.NewUser(userGetter))
	restful.DefaultContainer.Add(firewallService)
	restful.DefaultContainer.Add(service.NewFilesystemLayout(ds))
	restful.DefaultContainer.Add(service.NewSwitch(ds))
	restful.DefaultContainer.Add(healthService)
	restful.DefaultContainer.Add(eventService)
	restful.DefaultContainer.Add(rest.NewVersion(moduleName, service.BasePath))
	restful.DefaultContainer.Filter(rest.RequestLogger(debug, lg))
	restful.DefaultContainer.Filter(metrics.RestfulMetrics)

	if withauth {
		restful.DefaultContainer.Filter(rest.UserAuth(userGetter))
		providerTenant := viper.GetString("provider-tenant")
		excludedPathSuffixes := []string{"liveliness", "health", "version", "apidocs.json"}
		ensurer := service.NewTenantEnsurer([]string{providerTenant}, excludedPathSuffixes)
		restful.DefaultContainer.Filter(ensurer.EnsureAllowedTenantFilter)
	}

	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(), // you control what services are visible
		APIPath:                       service.BasePath + "apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject,
	}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))
	return &config
}

func dumpSwaggerJSON() {
	cfg := initRestServices(false)
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
	err = service.ResurrectMachines(ds, p, ep, ipamer, logger)
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

func run() error {
	initRestServices(true)

	// enable OPTIONS-request so clients can query CORS information
	restful.DefaultContainer.Filter(restful.DefaultContainer.OPTIONSFilter)

	// enable CORS for the UI to work.
	// if we will add support for api-tokens as headers, we had to add them
	// here to. note: the token's should not contain the product (aka. metal)
	// because customers should have ONE token for many products.
	// ExposeHeaders:  []string{"X-FITS-TOKEN"},
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
		err := response.WriteAsJson(httperrors.NewHTTPError(serviceErr.Code, fmt.Errorf(serviceErr.Message)))
		if err != nil {
			logger.Error("Failed to send response", zap.Error(err))
			return
		}
	})

	go func() {
		err := grpcServer.Serve()
		if err != nil {
			logger.Fatalw("failed to serve gRPC", "error", err)
		}
	}()

	addr := fmt.Sprintf("%s:%d", viper.GetString("bind-addr"), viper.GetInt("port"))
	logger.Infow("start metal api", "version", v.V.String(), "address", addr, "base-path", service.BasePath)
	err := http.ListenAndServe(addr, nil)
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
