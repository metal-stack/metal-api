package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore/rethinkstore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/service"
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/health"
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
	version   = "devel"
	revision  string
	gitsha1   string
	builddate string
	cfgFile   string
	ds        datastore.Datastore
	logger    log15.Logger
	debug     = false
)

var rootCmd = &cobra.Command{
	Use:     "metal-api",
	Short:   "an api to offer pure metal",
	Version: getVersionString(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging()
		initDataStore()
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

func getVersionString() string {
	var versionString = version
	if gitsha1 != "" {
		versionString += " (" + gitsha1 + ")"
	}
	if revision != "" {
		versionString += ", " + revision
	}
	if builddate != "" {
		versionString += ", " + builddate
	}
	return versionString
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

func initDataStore() {
	dbAdapter := viper.GetString("db")
	if dbAdapter == "rethinkdb" {
		ds = rethinkstore.New(
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
	restful.DefaultContainer.Add(service.NewFacility(logger, ds))
	restful.DefaultContainer.Add(service.NewImage(logger, ds))
	restful.DefaultContainer.Add(service.NewSize(logger, ds))
	restful.DefaultContainer.Add(service.NewDevice(logger, ds))
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
	log15.Info("start metal api", "revision", revision, "builddate", builddate, "address", addr)
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
