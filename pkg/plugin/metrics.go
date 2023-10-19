package plugin

import "github.com/prometheus/client_golang/prometheus"

func init() {
	registerPrometheusMetrics()
}

func registerPrometheusMetrics() {
	prometheus.MustRegister(kmsOperationCounter)
	prometheus.MustRegister(kmsLatencyMetric)
}

var (
	kmsOperationCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aws_encryption_provider_kms_operations_total",
			Help: "total aws encryption provider kms operations",
		},
		[]string{
			"key_arn",
			"status",
			"operation",
			"version",
		},
	)

	kmsLatencyMetric = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aws_encryption_provider_kms_operation_latency_ms",
			Help:    "Response latency in milliseconds for aws encryption provider kms operation ",
			Buckets: prometheus.ExponentialBuckets(2, 2, 14),
		},
		[]string{
			"key_arn",
			"status",
			"operation",
			"version",
		},
	)
)
