package service

import (
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"

	"io/ioutil"

	"bytes"

	"git.f-i-ts.de/cloud-native/metallib/rest"
	"github.com/emicklei/go-restful"
	"github.com/metal-pod/security"
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

	e := NewTenantEnsurer([]string{"pvdr", "Pv", "pv-DR"})
	require.True(t, e.allowed("pvdr"))
	require.True(t, e.allowed("Pv"))
	require.True(t, e.allowed("pv"))
	require.True(t, e.allowed("pv-DR"))
	require.True(t, e.allowed("PV-DR"))
	require.True(t, e.allowed("PV-dr"))
	require.False(t, e.allowed(""))
	require.False(t, e.allowed("abc"))
}
