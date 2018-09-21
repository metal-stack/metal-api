package main

// Note: this file is copied from https://github.com/emicklei/go-restful-openapi/blob/master/examples/user-resource.go

import (
	"flag"
	"net/http"

	"git.f-i-ts.de/ize0h88/maas-service/cmd/maas-api/interal/service"
	restful "github.com/emicklei/go-restful"
	restfulspec "github.com/emicklei/go-restful-openapi"
	"github.com/go-openapi/spec"
	"github.com/inconshreveable/log15"
)

var (
	revision  string
	builddate string
)

func main() {
	flag.Parse()
	restful.DefaultContainer.Add(service.NewFacility())

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

	log15.Info("start maas api", "revision", revision, "builddate", builddate, "address", flag.Arg(0))
	http.ListenAndServe(flag.Arg(0), nil)
}

func enrichSwaggerObject(swo *spec.Swagger) {
	swo.Info = &spec.Info{
		InfoProps: spec.InfoProps{
			Title:       "MAAS Service",
			Description: "Resource for managing metal as a service",
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
	swo.Tags = []spec.Tag{spec.Tag{TagProps: spec.TagProps{
		Name:        "facility",
		Description: "Managing facilities"}}}
}
