package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/eventbus"

	"git.f-i-ts.de/cloud-native/metallib/httperrors"
	"github.com/metal-pod/v"
	"go.uber.org/zap"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/ipam"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/rest"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	goipam "github.com/metal-pod/go-ipam"
	"github.com/metal-pod/security"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cfgFileType             = "yaml"
	moduleName              = "metal-api"
	generatedHTMLAPIDocPath = "./generate/"
)

var (
	cfgFile string
	ds      *datastore.RethinkStore
	ipamer  *ipam.Ipam
	nsqer   *eventbus.NSQClient
	logger  = zapup.MustRootLogger().Sugar()
	debug   = false
)

var rootCmd = &cobra.Command{
	Use:     moduleName,
	Short:   "an api to offer pure metal",
	Version: v.V.String(),
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()
		initDataStore()
		initEventBus()
		initIpam()
		initSignalHandlers()
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
		initializeDatabase()
	},
}

func main() {
	rootCmd.AddCommand(dumpSwagger, initDatabase)
	if err := rootCmd.Execute(); err != nil {
		logger.Error("failed executing root command", "error", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "alternative path to config file")

	rootCmd.Flags().StringP("bind-addr", "", "127.0.0.1", "the bind addr of the api server")
	rootCmd.Flags().IntP("port", "", 8080, "the port to serve on")

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

	rootCmd.Flags().StringP("nsqd-addr", "", "nsqd:4150", "the address of the nsqd")
	rootCmd.Flags().StringP("nsqd-http-addr", "", "nsqd:4151", "the address of the nsqd rest endpoint")
	rootCmd.Flags().StringP("nsqlookupd-addr", "", "nsqlookupd:4160", "the address of the nsqlookupd as a commalist")

	rootCmd.Flags().StringP("hmac-view-key", "", "must-be-changed", "the preshared key for hmac security for a viewing user")
	rootCmd.Flags().StringP("hmac-view-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("hmac-edit-key", "", "must-be-changed", "the preshared key for hmac security for a editing user")
	rootCmd.Flags().StringP("hmac-edit-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

	rootCmd.Flags().StringP("hmac-admin-key", "", "must-be-changed", "the preshared key for hmac security for a admin user")
	rootCmd.Flags().StringP("hmac-admin-lifetime", "", "30s", "the timestamp in the header for the HMAC must not be older than this value. a value of 0 means no limit")

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
	nsqdAddr := viper.GetString("nsqd-addr")
	nsqdRESTEndpoint := viper.GetString("nsqd-http-addr")
	partitions := waitForPartitions()

	nsq := eventbus.NewNSQ(nsqdAddr, nsqdRESTEndpoint, zapup.MustRootLogger(), bus.NewPublisher)
	nsq.WaitForPublisher()
	nsq.WaitForTopicsCreated(partitions, metal.Topics)
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
		logger.Errorw("cannot connect to db in root command metal-api/internal/main.initDatastore()", "error", err)
		panic(err)
	}

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
			"disable")
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
}

func initAuth(lg *zap.SugaredLogger) security.UserGetter {
	dx, err := security.NewDex(viper.GetString("dex-addr"))
	if err != nil {
		logger.Warnw("dex not reachable", "error", err)
	}
	auths := []security.CredsOpt{security.WithDex(dx)}
	for r, u := range service.MetalUsers {
		lfkey := fmt.Sprintf("hmac-%s-lifetime", r)
		mackey := viper.GetString(fmt.Sprintf("hmac-%s-key", r))
		lf, err := time.ParseDuration(viper.GetString(lfkey))
		if err != nil {
			lg.Warnw("illgal value for hmac lifetime, use 30secs as default", "error", err, "val", lfkey)
			lf = 30 * time.Second
		}
		lg.Infow("add hmac user", "name", r, "lifetime", lf, "mac", mackey)
		auths = append(auths, security.WithHMAC(security.NewHMACAuth(
			u.Name,
			[]byte(mackey),
			security.WithLifetime(lf),
			security.WithUser(service.MetalUsers[r]))))
	}
	return security.NewCreds(auths...)
}

func initRestServices(withauth bool) *restfulspec.Config {
	service.BasePath = viper.GetString("base-path")
	if !strings.HasPrefix(service.BasePath, "/") || !strings.HasSuffix(service.BasePath, "/") {
		logger.Fatalf("base path must start and end with a slash")
	}

	lg := logger.Desugar()
	restful.DefaultContainer.Add(service.NewPartition(ds, nsqer))
	restful.DefaultContainer.Add(service.NewImage(ds))
	restful.DefaultContainer.Add(service.NewSize(ds))
	restful.DefaultContainer.Add(service.NewNetwork(ds, ipamer))
	restful.DefaultContainer.Add(service.NewIP(ds, ipamer))
	var p bus.Publisher
	if nsqer != nil {
		p = nsqer.Publisher
	}
	restful.DefaultContainer.Add(service.NewMachine(ds, p, ipamer))
	restful.DefaultContainer.Add(service.NewFirewall(ds, ipamer))
	restful.DefaultContainer.Add(service.NewSwitch(ds))
	restful.DefaultContainer.Add(service.NewProject(ds))
	restful.DefaultContainer.Add(rest.NewHealth(lg, service.BasePath, ds.Health))
	restful.DefaultContainer.Add(rest.NewVersion(moduleName, service.BasePath))
	restful.DefaultContainer.Filter(rest.RequestLogger(debug, lg))
	if withauth {
		restful.DefaultContainer.Filter(rest.UserAuth(initAuth(lg.Sugar())))
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

func initializeDatabase() {
	initDataStore()
	logger.Info("Database initialized")
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
		response.WriteAsJson(httperrors.NewHTTPError(serviceErr.Code, fmt.Errorf(serviceErr.Message)))
	})

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
			Description: "Resource for managing pure metal",
			Contact: &spec.ContactInfo{
				Name:  "Devops Team",
				Email: "devops@f-i-ts.de",
				URL:   "http://www.f-i-ts.de",
			},
			License: &spec.License{
				Name: "MIT",
				URL:  "http://mit.org",
			},
			Version: "1.0.0",
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
	swo.SecurityDefinitions = map[string]*spec.SecurityScheme{
		"HMAC": hmacspec,
		"jwt":  jwtspec,
	}
	swo.BasePath = viper.GetString("base-path")
	swo.Security = []map[string][]string{
		{"Authorization": []string{"HMAC", "jwt"}},
	}

	// Maybe this leads to an issue, investigating...:
	// swo.Schemes = []string{"http", "https"}
}
