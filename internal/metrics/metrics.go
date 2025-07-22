package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ErrorCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bank_api_errors_total",
		Help: "Total number of errors by type",
	}, []string{"type"})

	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bank_api_request_duration_seconds",
		Help:    "Duration of HTTP requests in seconds",
		Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10},
	}, []string{"method", "path", "status"})

	RequestCounter = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bank_api_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})
)

func Init() {
	// Metrics are automatically registered via promauto
}
