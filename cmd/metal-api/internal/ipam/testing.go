package ipam

import (
	"net/http"
	"net/http/httptest"
	"testing"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
	"go.uber.org/zap/zaptest"
)

func InitTestIpam(t *testing.T) IPAMer {
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(zaptest.NewLogger(t).Sugar(), goipam.New()),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	ipamclient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
	)

	ipamer := New(ipamclient)
	return ipamer
}
