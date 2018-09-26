package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore/hashmapstore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/service"
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/utils"
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
)

var rootCmd = &cobra.Command{
	Use:     "metal-api",
	Short:   "an api to offer pure metal",
	Version: getVersionString(),
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		initLogging()
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
	rootCmd.Flags().BoolP("with-mock-data", "", false, "creates some mock data on startup")

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
			log15.Error("Config file path set explicitly, but unreadble", "error", err)
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

	handler := log15.LvlFilterHandler(level, formatHandler)

	log15.Root().SetHandler(handler)
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

func run() {
	log := log15.New("app", "metal-api")

	// as long as we have not database
	datastore := hashmapstore.NewHashmapStore()
	if viper.GetBool("with-mock-data") {
		datastore.AddMockData()
		log15.Info("Initialized mock data")
	}

	restful.DefaultContainer.Add(service.NewFacility(datastore))
	restful.DefaultContainer.Add(service.NewImage(datastore))
	restful.DefaultContainer.Add(service.NewSize(datastore))
	restful.DefaultContainer.Add(service.NewDevice(datastore))

	restful.DefaultContainer.Filter(utils.RestfulLogger(log))

	config := restfulspec.Config{
		WebServices: restful.RegisteredWebServices(), // you control what services are visible
		APIPath:     "/apidocs.json",
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
	log.Info("start metal api", "revision", revision, "builddate", builddate, "address", addr)
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
