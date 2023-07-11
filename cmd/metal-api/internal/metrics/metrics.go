package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

var (
	counter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "metal",
			Subsystem: "api",
			Name:      "requests_total",
			Help:      "A counter for requests to the whole metal api.",
		},
		[]string{"code", "method"},
	)

	duration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "metal",
			Subsystem: "api",
			Name:      "request_duration_seconds",
			Help:      "A histogram of latencies for requests.",
			Buckets:   []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"route", "method"},
	)
	grpcDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "metal",
			Subsystem: "api",
			Name:      "grpc_request_duration_seconds",
			Help:      "A histogram of latencies for requests.",
			Buckets:   []float64{.25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method"},
	)
)

func init() {
	prometheus.MustRegister(counter, duration, grpcDuration)
}

func RestfulMetrics(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	n := time.Now()
	chain.ProcessFilter(req, resp)
	counter.WithLabelValues(fmt.Sprintf("%d", resp.StatusCode()), req.Request.Method).Inc()
	duration.WithLabelValues(req.SelectedRoutePath(), req.Request.Method).Observe(time.Since(n).Seconds())
}

func GrpcMetrics(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	n := time.Now()
	resp, err = handler(ctx, req)
	grpcDuration.WithLabelValues(info.FullMethod).Observe(time.Since(n).Seconds())
	return resp, err
}
