package service

import (
	"net/http"
	"time"

	"io/ioutil"

	"bytes"

	"git.f-i-ts.de/cloud-native/metallib/rest"
	restful "github.com/emicklei/go-restful"
	"github.com/metal-pod/security"
)

func injectViewer(container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(Viewer, container, rq)
}

func injectEditor(container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(Editor, container, rq)
}
func injectAdmin(container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(Admin, container, rq)
}

func injectUser(u security.User, container *restful.Container, rq *http.Request) *restful.Container {
	hma := security.NewHMACAuth(u.Name, []byte{1, 2, 3}, security.WithUser(u))
	usergetter := security.NewCreds(security.WithHMAC(hma))
	container.Filter(rest.UserAuth(usergetter))
	var body []byte
	if rq.Body != nil {
		data, _ := ioutil.ReadAll(rq.Body)
		body = data
		rq.Body.Close()
		rq.Body = ioutil.NopCloser(bytes.NewReader(data))
	}
	hma.AddAuth(rq, time.Now(), body)
	return container
}
