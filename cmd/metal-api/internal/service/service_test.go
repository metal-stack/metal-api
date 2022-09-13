package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/metal-stack/metal-lib/rest"
	"github.com/metal-stack/security"
)

var testUserDirectory = NewUserDirectory("")

func injectViewer(log *zap.SugaredLogger, container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(log, testUserDirectory.viewer, container, rq)
}

func injectEditor(log *zap.SugaredLogger, container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(log, testUserDirectory.edit, container, rq)
}

func injectAdmin(log *zap.SugaredLogger, container *restful.Container, rq *http.Request) *restful.Container {
	return injectUser(log, testUserDirectory.admin, container, rq)
}

func injectUser(log *zap.SugaredLogger, u security.User, container *restful.Container, rq *http.Request) *restful.Container {
	hma := security.NewHMACAuth(u.Name, []byte{1, 2, 3}, security.WithUser(u))
	usergetter := security.NewCreds(security.WithHMAC(hma))
	container.Filter(rest.UserAuth(usergetter, log))
	var body []byte
	if rq.Body != nil {
		data, _ := io.ReadAll(rq.Body)
		body = data
		rq.Body.Close()
		rq.Body = io.NopCloser(bytes.NewReader(data))
	}
	hma.AddAuth(rq, time.Now(), body)
	return container
}

func TestTenantEnsurer(t *testing.T) {
	e := NewTenantEnsurer(zaptest.NewLogger(t).Sugar(), []string{"pvdr", "Pv", "pv-DR"}, nil)
	require.True(t, e.allowed("pvdr"))
	require.True(t, e.allowed("Pv"))
	require.True(t, e.allowed("pv"))
	require.True(t, e.allowed("pv-DR"))
	require.True(t, e.allowed("PV-DR"))
	require.True(t, e.allowed("PV-dr"))
	require.False(t, e.allowed(""))
	require.False(t, e.allowed("abc"))
}

func TestAllowedPathSuffixes(t *testing.T) {
	foo := func(req *restful.Request, resp *restful.Response) {
		_ = resp.WriteHeaderAndEntity(http.StatusOK, nil)
	}

	e := NewTenantEnsurer(zaptest.NewLogger(t).Sugar(), []string{"a", "b", "c"}, []string{"health", "liveliness"})
	ws := new(restful.WebService).Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON)
	ws.Filter(e.EnsureAllowedTenantFilter)
	health := ws.GET("health").To(foo).Returns(http.StatusOK, "OK", nil).DefaultReturns("Error", httperrors.HTTPErrorResponse{})
	liveliness := ws.GET("liveliness").To(foo).Returns(http.StatusOK, "OK", nil).DefaultReturns("Error", httperrors.HTTPErrorResponse{})
	machine := ws.GET("machine").To(foo).Returns(http.StatusOK, "OK", nil).DefaultReturns("Error", httperrors.HTTPErrorResponse{})
	ws.Route(health)
	ws.Route(liveliness)
	ws.Route(machine)
	restful.DefaultContainer.Add(ws)

	// health must be allowed without tenant check
	httpRequest, _ := http.NewRequestWithContext(context.TODO(), "GET", "http://localhost/health", nil)
	httpRequest.Header.Set("Accept", "application/json")
	httpWriter := httptest.NewRecorder()

	restful.DefaultContainer.Dispatch(httpWriter, httpRequest)

	require.Equal(t, http.StatusOK, httpWriter.Code)

	// liveliness must be allowed without tenant check
	httpRequest, _ = http.NewRequestWithContext(context.TODO(), "GET", "http://localhost/liveliness", nil)
	httpRequest.Header.Set("Accept", "application/json")
	httpWriter = httptest.NewRecorder()

	restful.DefaultContainer.Dispatch(httpWriter, httpRequest)

	require.Equal(t, http.StatusOK, httpWriter.Code)

	// machine must not be allowed without tenant check
	httpRequest, _ = http.NewRequestWithContext(context.TODO(), "GET", "http://localhost/machine", nil)
	httpRequest.Header.Set("Accept", "application/json")
	httpWriter = httptest.NewRecorder()

	restful.DefaultContainer.Dispatch(httpWriter, httpRequest)

	require.Equal(t, http.StatusForbidden, httpWriter.Code)
}
