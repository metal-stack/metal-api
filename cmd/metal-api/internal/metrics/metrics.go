package metrics

import (
	"fmt"

	"time"

	"github.com/emicklei/go-restful/v3"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	machineLiveliness = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "metal",
			Subsystem: "machine",
			Name:      "liveliness_total",
			Help:      "The liveliness of the machines which are available in the system",
		},
		[]string{"partition", "status"})

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
)

func init() {
	prometheus.MustRegister(machineLiveliness, counter, duration)
}

// PartitionLiveliness is a data container for the liveliness of different partitions.
type PartitionLiveliness map[string]struct {
	Alive   int
	Dead    int
	Unknown int
}

// ProvideLiveliness provides the given values as gauges so a scraper can collect them.
func ProvideLiveliness(lvness PartitionLiveliness) {
	for p, l := range lvness {
		machineLiveliness.WithLabelValues(p, "alive").Set(float64(l.Alive))
		machineLiveliness.WithLabelValues(p, "dead").Set(float64(l.Dead))
		machineLiveliness.WithLabelValues(p, "unknown").Set(float64(l.Unknown))
	}
}

func RestfulMetrics(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	n := time.Now()
	chain.ProcessFilter(req, resp)
	counter.WithLabelValues(fmt.Sprintf("%d", resp.StatusCode()), req.Request.Method).Inc()
	duration.WithLabelValues(req.SelectedRoutePath(), req.Request.Method).Observe(time.Since(n).Seconds())
}
