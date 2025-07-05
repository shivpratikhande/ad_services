package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	ClicksReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ad_clicks_received_total",
			Help: "Total number of click events received",
		},
		[]string{"ad_id"},
	)

	ClicksProcessed = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "ad_clicks_processed_total",
			Help: "Total number of click events processed",
		},
	)

	ResponseTime = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status_code"},
	)

	QueueSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "click_queue_size",
			Help: "Current size of the click processing queue",
		},
	)
)

func init() {
	prometheus.MustRegister(ClicksReceived)
	prometheus.MustRegister(ClicksProcessed)
	prometheus.MustRegister(ResponseTime)
	prometheus.MustRegister(QueueSize)
}
