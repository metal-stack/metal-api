package service

import (
	"github.com/emicklei/go-restful"
	"io"
	"net/http"
	"os"
)

type apiDocResource struct {
	apiDocPath string
}

// NewApiDoc returns a webservice for apidoc endpoint.
func NewApiDoc(generatedHtmlApiDocPath string) *restful.WebService {

	ar := apiDocResource{
		apiDocPath: generatedHtmlApiDocPath,
	}

	return ar.webService()
}

func (ar *apiDocResource) webService() *restful.WebService {

	ws := new(restful.WebService)
	ws.Route(ws.GET("/apidocs.html").
		Produces("text/html").
		To(ar.apiDoc))

	return ws
}

// returns self contained apidoc html document
func (ar *apiDocResource) apiDoc(request *restful.Request, response *restful.Response) {

	file, err := os.Open(ar.apiDocPath)
	if err != nil {
		_ = response.WriteErrorString(http.StatusNotFound, "Documentation is not available")
	}

	_, err = io.Copy(response.ResponseWriter, file)
	if err != nil {
		_ = response.WriteErrorString(http.StatusInternalServerError, "error while writing Documentation response")
	}
}
