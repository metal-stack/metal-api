package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/netbox"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/service"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/metal/metal-api/health"
	"git.f-i-ts.de/cloud-native/metal/metal-api/metal"
	"git.f-i-ts.de/cloud-native/metallib/bus"
	"git.f-i-ts.de/cloud-native/metallib/version"
	"git.f-i-ts.de/cloud-native/metallib/zapup"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cfgFileType = "yaml"
)

var (
	cfgFile  string
	ds       *datastore.RethinkStore
	producer bus.Publisher
	nbproxy  *netbox.APIProxy
	logger   log15.Logger
	debug    = false
)

var rootCmd = &cobra.Command{
	Use:     "metal-api",
	Short:   "an api to offer pure metal",
	Version: version.V.String(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging()
		initDataStore()
		initEventBus()
		initNetboxProxy()
		initSignalHandlers()
	},
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log15.Error("failed executing root command", "error", err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "", "alternative path to config file")
	rootCmd.Flags().StringP("log-level", "", "info", "the application log level")
	rootCmd.Flags().StringP("log-formatter", "", "text", "the application log fromatter (text or json)")

	rootCmd.Flags().StringP("bind-addr", "", "127.0.0.1", "the bind addr of the api server")
	rootCmd.Flags().IntP("port", "", 8080, "the port to serve on")

	rootCmd.Flags().StringP("db", "", "rethinkdb", "the database adapter to use")
	rootCmd.Flags().StringP("db-name", "", "metalapi", "the database name to use")
	rootCmd.Flags().StringP("db-addr", "", "", "the database address string to use")
	rootCmd.Flags().StringP("db-user", "", "", "the database user to use")
	rootCmd.Flags().StringP("db-password", "", "", "the database password to use")

	rootCmd.Flags().StringP("nsqd-addr", "", "nsqd:4150", "the address of the nsqd")
	rootCmd.Flags().StringP("nsqd-http-addr", "", "nsqd:4151", "the address of the nsqd rest endpoint")
	rootCmd.Flags().StringP("nsqlookupd-addr", "", "nsqlookupd:4160", "the address of the nsqlookupd as a commalist")

	rootCmd.Flags().StringP("netbox-addr", "", "localhost:8001", "the address of netbox proxy")
	rootCmd.Flags().StringP("netbox-api-token", "", "", "the api token to access the netbox proxy")

	viper.BindPFlags(rootCmd.Flags())
}

func initConfig() {
	viper.SetEnvPrefix("METAL_API")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	viper.SetConfigType(cfgFileType)

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			log15.Error("Config file path set explicitly, but unreadable", "error", err)
		}
	} else {
		viper.SetConfigName("config")
		viper.AddConfigPath("/etc/metal-api")
		viper.AddConfigPath("$HOME/.metal-api")
		viper.AddConfigPath(".")
		if err := viper.ReadInConfig(); err != nil {
			usedCfg := viper.ConfigFileUsed()
			if usedCfg != "" {
				log15.Error("Config file unreadable", "config-file", usedCfg, "error", err)
			}
		}
	}

	usedCfg := viper.ConfigFileUsed()
	if usedCfg != "" {
		log15.Info("Read config file", "config-file", usedCfg)
	}
}

func initLogging() {
	var formatHandler log15.Handler
	if viper.GetString("log-formatter") == "json" {
		formatHandler = log15.StreamHandler(os.Stdout, log15.JsonFormat())
	} else if viper.GetString("log-formatter") == "text" {
		formatHandler = log15.StdoutHandler
	} else {
		log15.Error("Unsupported log formatter", "log-formatter", viper.GetString("log-formatter"))
		os.Exit(1)
	}
	level, err := log15.LvlFromString(viper.GetString("log-level"))
	if err != nil {
		log15.Error("Unparsable log level", "log-level", viper.GetString("log-level"))
		os.Exit(1)
	}

	if level == log15.LvlDebug {
		debug = true
	}

	handler := log15.CallerFileHandler(formatHandler)
	handler = log15.LvlFilterHandler(level, handler)

	log15.Root().SetHandler(handler)
	logger = log15.New("app", "metal-api")
}

func initSignalHandlers() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-c
		log15.Error("Received keyboard interrupt, shutting down...")
		if ds != nil {
			log15.Info("Closing connection to datastore")
			err := ds.Close()
			if err != nil {
				log15.Info("Unable to properly shutdown datastore", "error", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}()
}

func initNetboxProxy() {
	nbproxy = netbox.New()
}

func initEventBus() {
	nsqd := viper.GetString("nsqd-addr")
	httpnsqd := viper.GetString("nsqd-http-addr")
	p, err := bus.NewPublisher(zapup.MustRootLogger(), nsqd, httpnsqd)
	if err != nil {
		panic(err)
	}
	log15.Info("nsq connected", "nsqd", nsqd)
	if err := p.CreateTopic(string(metal.TopicDevice)); err != nil {
		panic(err)
	}
	producer = p
}

func initDataStore() {
	dbAdapter := viper.GetString("db")
	if dbAdapter == "rethinkdb" {
		ds = datastore.New(
			logger,
			viper.GetString("db-addr"),
			viper.GetString("db-name"),
			viper.GetString("db-user"),
			viper.GetString("db-password"),
		)
	} else {
		log15.Error("database not supported", "db", dbAdapter)
	}
	ds.Connect()
}

func run() {
	restful.DefaultContainer.Add(service.NewSite(logger, ds))
	restful.DefaultContainer.Add(service.NewImage(logger, ds))
	restful.DefaultContainer.Add(service.NewSize(logger, ds))
	restful.DefaultContainer.Add(service.NewDevice(logger, ds, producer, nbproxy))
	restful.DefaultContainer.Add(health.New(logger, func() error { return nil }))
	restful.DefaultContainer.Filter(utils.RestfulLogger(logger, debug))

	config := restfulspec.Config{
		WebServices:                   restful.RegisteredWebServices(), // you control what services are visible
		APIPath:                       "/apidocs.json",
		PostBuildSwaggerObjectHandler: enrichSwaggerObject}
	restful.DefaultContainer.Add(restfulspec.NewOpenAPIService(config))

	// enable CORS for the UI to work.
	cors := restful.CrossOriginResourceSharing{
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      restful.DefaultContainer}
	restful.DefaultContainer.Filter(cors.Filter)

	addr := fmt.Sprintf("%s:%d", viper.GetString("bind-addr"), viper.GetInt("port"))
	log15.Info("start metal api", "version", version.V.String())
	http.ListenAndServe(addr, nil)
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "metal-api",
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
		spec.Tag{TagProps: spec.TagProps{
			Name:        "facility",
			Description: "Managing facility entities"}},
		spec.Tag{TagProps: spec.TagProps{
			Name:        "image",
			Description: "Managing image entities"}},
		spec.Tag{TagProps: spec.TagProps{
			Name:        "size",
			Description: "Managing size entities"}},
		spec.Tag{TagProps: spec.TagProps{
			Name:        "device",
			Description: "Managing devices"},
		},
	}
}
