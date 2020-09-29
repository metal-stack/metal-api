package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/grpc"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metrics"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	httppprof "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	nsq2 "github.com/nsqio/go-nsq"
	"github.com/pkg/errors"

	"github.com/metal-stack/metal-lib/jwt/grp"
	"github.com/metal-stack/metal-lib/jwt/sec"

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

const (
	cfgFileType             = "yaml"
	moduleName              = "metal-api"
	generatedHTMLAPIDocPath = "./generate/"
)

var (
	cfgFile            string
	ds                 *datastore.RethinkStore
	ipamer             *ipam.Ipam
	publisherTLSConfig *bus.TLSConfig
	nsqer              *eventbus.NSQClient
	logger             = zapup.MustRootLogger().Sugar()
	debug              = false
	mdc                mdm.Client
	grpcServer         *grpc.Server
)

var rootCmd = &cobra.Command{
	Use:     moduleName,
	Short:   "an api to offer pure metal",
	Version: v.V.String(),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()
		initMetrics()
		initDataStore()
		initEventBus()
		initIpam()
		initMasterData()
		initSignalHandlers()
		initGrpcServer()
		run()
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
	Run: func(cmd *cobra.Command, args []string) {
		initDataStore()
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
		initDataStore()
		return deleteOrphanImages()
	},
}

func main() {
	rootCmd.AddCommand(dumpSwagger, initDatabase, resurrectMachines, machineLiveliness, deleteOrphanImagesCmd)
	if err := rootCmd.Execute(); err != nil {
		logger.Error("failed executing root command", "error", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "alternative path to config file")

	rootCmd.Flags().StringP("bind-addr", "", "127.0.0.1", "the bind addr of the api server")
	rootCmd.Flags().IntP("port", "", 8080, "the port to serve on")
	rootCmd.Flags().IntP("grpc-port", "", 50051, "the port to serve gRPC on")

	rootCmd.Flags().StringP("base-path", "", "/", "the base path of the api server")

	rootCmd.Flags().StringP("db", "", "rethinkdb", "the database adapter to use")
	rootCmd.Flags().StringP("db-name", "", "metalapi", "the database name to use")
	rootCmd.Flags().StringP("db-addr", "", "", "the database address string to use")
	rootCmd.Flags().StringP("db-user", "", "", "the database user to use")
	rootCmd.Flags().StringP("db-password", "", "", "the database password to use")

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

	rootCmd.Flags().StringP("hmac-view-key", "", "must-be-changed", "the preshared key for hmac security for a viewing user")
	rootCmd.Flags().StringP("hmac-view-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("hmac-edit-key", "", "must-be-changed", "the preshared key for hmac security for a editing user")
	rootCmd.Flags().StringP("hmac-edit-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("hmac-admin-key", "", "must-be-changed", "the preshared key for hmac security for a admin user")
	rootCmd.Flags().StringP("hmac-admin-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("provider-tenant", "", "", "the tenant of the maas-provider who operates the whole thing")

	rootCmd.Flags().StringP("masterdata-hmac", "", "must-be-changed", "the preshared key for hmac security to talk to the masterdata-api")
	rootCmd.Flags().StringP("masterdata-hostname", "", "", "the hostname of the masterdata-api")
	rootCmd.Flags().IntP("masterdata-port", "", 8443, "the port of the masterdata-api")
	rootCmd.Flags().StringP("masterdata-capath", "", "", "the tls ca certificate to talk to the masterdata-api")
	rootCmd.Flags().StringP("masterdata-certpath", "", "", "the tls certificate to talk to the masterdata-api")
	rootCmd.Flags().StringP("masterdata-certkeypath", "", "", "the tls certificate key to talk to the masterdata-api")

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		logger.Error("unable to construct root command:%v", err)
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

func initDataStore() {
	dbAdapter := viper.GetString("db")
	if dbAdapter == "rethinkdb" {
		ds = datastore.New(
			logger.Desugar(),
			viper.GetString("db-addr"),
			viper.GetString("db-name"),
			viper.GetString("db-user"),
			viper.GetString("db-password"),
		)
	} else {
		logger.Error("database not supported", "db", dbAdapter)
	}

	err := ds.Connect()

	if err != nil {
		logger.Errorw("cannot connect to data store", "error", err)
		panic(err)
	}
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
		defer cancel()
		mdc, err = mdm.NewClient(ctx, hostname, port, certpath, certkeypath, ca, hmacKey, logger.Desugar())
		if err == nil {
			break
		}
		logger.Errorw("unable to initialize masterdata-api client, retrying...", "error", err)
		time.Sleep(3 * time.Second)
	}

	logger.Info("masterdata client initialized")
}

func initIpam() {
	dbAdapter := viper.GetString("ipam-db")
	if dbAdapter == "postgres" {
	tryAgain:
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
			goto tryAgain
		}
		ipamInstance := goipam.NewWithStorage(pgStorage)
		ipamer = ipam.New(ipamInstance)
	} else if dbAdapter == "memory" {
		ipamInstance := goipam.New()
		ipamer = ipam.New(ipamInstance)
	} else {
		logger.Errorw("database not supported", "db", dbAdapter)
	}
	logger.Info("ipam initialized")
}

func initAuth(lg *zap.SugaredLogger) security.UserGetter {
	var auths []security.CredsOpt

	providerTenant := viper.GetString("provider-tenant")

	dexAddr := viper.GetString("dex-addr")
	if dexAddr != "" {
		dx, err := security.NewDex(dexAddr)
		if err != nil {
			logger.Fatalw("dex not reachable", "error", err)
		}
		if dx != nil {
			// use custom user extractor and group processor
			plugin := sec.NewPlugin(grp.MustNewGrpr(grp.Config{ProviderTenant: providerTenant}))
			dx.With(security.UserExtractor(plugin.ExtractUserProcessGroups))
			auths = append(auths, security.WithDex(dx))
			logger.Info("dex successfully configured")
		} else {
			logger.Fatalw("dex is configured, but not initialized")
		}
	}

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
		Publisher:             p,
		Datasource:            ds,
		Logger:                logger,
		NsqTlsConfig:          publisherTLSConfig,
		NsqlookupdHttpAddress: viper.GetString("nsqlookupd-addr"),
		GrpcPort:              viper.GetInt("grpc-port"),
		TlsEnabled:            viper.GetBool("grpc-tls-enabled"),
		CaCertFile:            viper.GetString("grpc-ca-cert-file"),
		ServerCertFile:        viper.GetString("grpc-server-cert-file"),
		ServerKeyFile:         viper.GetString("grpc-server-key-file"),
	})
	if err != nil {
		logger.Fatalw("cannot connect to NSQ", "error", err)
	}
}

func initRestServices(withauth bool) *restfulspec.Config {
	service.BasePath = viper.GetString("base-path")
	if !strings.HasPrefix(service.BasePath, "/") || !strings.HasSuffix(service.BasePath, "/") {
		logger.Fatalf("base path must start and end with a slash")
	}

	lg := logger.Desugar()
	var p bus.Publisher
	ep := bus.DirectEndpoints()
	if nsqer != nil {
		p = nsqer.Publisher
		ep = nsqer.Endpoints
	}
	ipservice, err := service.NewIP(ds, ep, ipamer, mdc)
	if err != nil {
		logger.Fatal(err)
	}
	mservice, err := service.NewMachine(ds, p, ep, ipamer, mdc, grpcServer)
	if err != nil {
		logger.Fatal(err)
	}
	fservice, err := service.NewFirewall(ds, ipamer, ep, mdc, grpcServer)
	if err != nil {
		logger.Fatal(err)
	}

	restful.DefaultContainer.Add(service.NewPartition(ds, nsqer))
	restful.DefaultContainer.Add(service.NewImage(ds))
	restful.DefaultContainer.Add(service.NewSize(ds))
	restful.DefaultContainer.Add(service.NewNetwork(ds, ipamer, mdc))
	restful.DefaultContainer.Add(ipservice)
	restful.DefaultContainer.Add(mservice)
	restful.DefaultContainer.Add(service.NewProject(ds, mdc))
	restful.DefaultContainer.Add(fservice)
	restful.DefaultContainer.Add(service.NewSwitch(ds))
	restful.DefaultContainer.Add(rest.NewHealth(lg, service.BasePath, ds.Health))
	restful.DefaultContainer.Add(rest.NewVersion(moduleName, service.BasePath))
	restful.DefaultContainer.Filter(rest.RequestLogger(debug, lg))
	restful.DefaultContainer.Filter(metrics.RestfulMetrics)

	if withauth {
		restful.DefaultContainer.Filter(rest.UserAuth(initAuth(lg.Sugar())))
		providerTenant := viper.GetString("provider-tenant")
		excludedPathSuffixes := []string{"liveliness", "health", "version", "apidocs.json"}
		ensurer := service.NewTenantEnsurer([]string{providerTenant}, excludedPathSuffixes)
		restful.DefaultContainer.Filter(ensurer.EnsureAllowedTenantFilter)
	}

	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(), // you control what services are visible
		APIPath:                       service.BasePath + "apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))
	return &config
}

func dumpSwaggerJSON() {
	cfg := initRestServices(false)
	actual := restfulspec.BuildSwagger(*cfg)
	js, err := json.MarshalIndent(actual, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s\n", js)
}

func resurrectDeadMachines() error {
	initDataStore()
	initEventBus()
	initIpam()

	var p bus.Publisher
	ep := bus.DirectEndpoints()
	if nsqer != nil {
		p = nsqer.Publisher
		ep = nsqer.Endpoints
	}
	err := service.ResurrectMachines(ds, p, ep, ipamer, logger)
	if err != nil {
		return errors.Wrap(err, "unable to resurrect machines")
	}

	return nil
}

func evaluateLiveliness() error {
	initDataStore()

	err := service.MachineLiveliness(ds, logger)
	if err != nil {
		return errors.Wrap(err, "unable to evaluate machine liveliness")
	}

	return nil
}

func deleteOrphanImages() error {
	initDataStore()
	initEventBus()
	_, err := ds.DeleteOrphanImages(nil, nil)
	return err
}

func run() {
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
		Container:      restful.DefaultContainer}
	restful.DefaultContainer.Filter(cors.Filter)

	// expose generated apidoc
	http.Handle(service.BasePath+"apidocs/", http.StripPrefix(service.BasePath+"apidocs/", http.FileServer(http.Dir(generatedHTMLAPIDocPath))))

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
	if err != nil {
		logger.Errorw("failed to start metal api", "error", err)
	}
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
			Description: "Managing image entities"}},
		{TagProps: spec.TagProps{
			Name:        "network",
			Description: "Managing network entities"}},
		{TagProps: spec.TagProps{
			Name:        "ip",
			Description: "Managing ip entities"}},
		{TagProps: spec.TagProps{
			Name:        "size",
			Description: "Managing size entities"}},
		{TagProps: spec.TagProps{
			Name:        "machine",
			Description: "Managing machine entities"}},
		{TagProps: spec.TagProps{
			Name:        "partition",
			Description: "Managing partition entities"}},
		{TagProps: spec.TagProps{
			Name:        "project",
			Description: "Managing project entities"}},
		{TagProps: spec.TagProps{
			Name:        "switch",
			Description: "Managing switch entities"}},
	}
	jwtspec := spec.APIKeyAuth("Authorization", "header")
	jwtspec.Description = "Add a 'Authorization: Bearer ....' header to the request"

	hmacspec := spec.APIKeyAuth("Authorization", "header")
	hmacspec.Description = "Generate a 'Authorization: Metal xxxx' header where 'xxxx' is a HMAC generated by the Request-Date, the Request-Method and the Body"
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
