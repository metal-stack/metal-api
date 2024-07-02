package ipam

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	goipam "github.com/metal-stack/go-ipam"
	"github.com/metal-stack/go-ipam/api/v1/apiv1connect"
	"github.com/metal-stack/go-ipam/pkg/service"
)

func InitTestIpam(t *testing.T) IPAMer {

	ctx := context.Background()
	mux := http.NewServeMux()
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))

	mux.Handle(apiv1connect.NewIpamServiceHandler(
		service.New(log, goipam.New(ctx)),
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
