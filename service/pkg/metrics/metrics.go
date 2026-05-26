package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	// HTTPRequestTotal counts all HTTP requests by method, path, and status code.
	HTTPRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)

	// HTTPRequestDuration records HTTP request latency in seconds.
	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	// PaymentsCreatedTotal counts payments created by channel and status.
	PaymentsCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payments_created_total",
			Help: "Total number of payments created.",
		},
		[]string{"channel", "status"},
	)

	// RefundsCreatedTotal counts refunds created by channel and status.
	RefundsCreatedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "refunds_created_total",
			Help: "Total number of refunds created.",
		},
		[]string{"channel", "status"},
	)

	// ChannelAPIRequestTotal counts channel API calls by channel, operation, and status.
	ChannelAPIRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "channel_api_requests_total",
			Help: "Total number of channel API calls.",
		},
		[]string{"channel", "operation", "status"},
	)

	// ChannelAPIRequestDuration records channel API call latency in seconds.
	ChannelAPIRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "channel_api_duration_seconds",
			Help:    "Channel API call latency in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"channel", "operation"},
	)
)

func Init() {
	prometheus.MustRegister(
		HTTPRequestTotal,
		HTTPRequestDuration,
		PaymentsCreatedTotal,
		RefundsCreatedTotal,
		ChannelAPIRequestTotal,
		ChannelAPIRequestDuration,
	)
}
