package ipam

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
)

func InitTestIpam(t *testing.T) IPAMer {

	ctx := context.Background()
	mux := http.NewServeMux()
	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(slog.Default(), goipam.New(ctx)),
	))
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()

	ipamclient := apiv1connect.NewIpamServiceClient(
		server.Client(),
		server.URL,
	)

	ipamer := New(ipamclient)
	return ipamer
}
