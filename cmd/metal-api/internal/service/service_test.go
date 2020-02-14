package service

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"io/ioutil"

	"bytes"

	"github.com/emicklei/go-restful"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
)

var testUserDirectory = NewUserDirectory("")

func injectViewer(container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(testUserDirectory.viewer, container, rq)
}

func injectEditor(container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(testUserDirectory.edit, container, rq)
}
func injectAdmin(container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(testUserDirectory.admin, container, rq)
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

func TestTenantEnsurer(t *testing.T) {
	e := NewTenantEnsurer([]string{"pvdr", "Pv", "pv-DR"}, nil)
	require.True(t, e.allowed("pvdr"))
	require.True(t, e.allowed("Pv"))
	require.True(t, e.allowed("pv"))
	require.True(t, e.allowed("pv-DR"))
	require.True(t, e.allowed("PV-DR"))
	require.True(t, e.allowed("PV-dr"))
	require.False(t, e.allowed(""))
	require.False(t, e.allowed("abc"))
}

func foo(req *restful.Request, resp *restful.Response) {
	_, _ = io.WriteString(resp.ResponseWriter, "foo")
}

func TestAllowedPathSuffixes(t *testing.T) {
	e := NewTenantEnsurer([]string{"a", "b", "c"}, []string{"/health", "/liveliness"})
	ws := new(restful.WebService).Path("")
	ws.Filter(e.EnsureAllowedTenantFilter)
	health := ws.GET("/health").To(foo)
	liveliness := ws.GET("/liveliness").To(foo)
	machine := ws.GET("/machine").To(foo)
	ws.Route(health)
	ws.Route(liveliness)
	ws.Route(machine)
	restful.DefaultContainer.Add(ws)

	// health must be allowed without tenant check
	httpRequest, _ := http.NewRequest("GET", "http://localhost/health", nil)
	httpRequest.Header.Set("Accept", "*/*")
	httpWriter := httptest.NewRecorder()

	restful.DefaultContainer.Dispatch(httpWriter, httpRequest)

	require.Equal(t, http.StatusOK, httpWriter.Code)

	// liveliness must be allowed without tenant check
	httpRequest, _ = http.NewRequest("GET", "http://localhost/liveliness", nil)
	httpRequest.Header.Set("Accept", "*/*")
	httpWriter = httptest.NewRecorder()

	restful.DefaultContainer.Dispatch(httpWriter, httpRequest)

	require.Equal(t, http.StatusOK, httpWriter.Code)

	// machine must not be allowed without tenant check
	httpRequest, _ = http.NewRequest("GET", "http://localhost/machine", nil)
	httpRequest.Header.Set("Accept", "*/*")
	httpWriter = httptest.NewRecorder()

	restful.DefaultContainer.Dispatch(httpWriter, httpRequest)

	require.Equal(t, http.StatusForbidden, httpWriter.Code)
}
