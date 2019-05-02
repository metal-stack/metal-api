package service

import (
	"github.com/emicklei/go-restful"
	"io"
	"os"
)

const generatedApiDocPath = "./generate/redoc.html"

// NewApiDoc returns a webservice for apidoc endpoint.
func NewApiDoc() *restful.WebService {

	return webService()
}

func webService() *restful.WebService {

	ws := new(restful.WebService)
	ws.Route(ws.GET("/apidocs.html").
		Produces("text/html").
		To(apiDoc))

	return ws
}

// returns self contained apidoc html document
func apiDoc(request *restful.Request, response *restful.Response) {

	file, err := os.Open(generatedApiDocPath)
	if err != nil {
		_ = response.WriteErrorString(500, "Documentation is not available")
	}

	_, err = io.Copy(response.ResponseWriter, file)
	if err != nil {
		_ = response.WriteErrorString(500, "error while writing Documentation response")
	}
}
