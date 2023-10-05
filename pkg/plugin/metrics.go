package plugin

import "github.com/prometheus/client_golang/prometheus"

func init() {
	registerPrometheusMetrics()
}

func registerPrometheusMetrics() {
	prometheus.MustRegister(kmsOperationCounter)
	prometheus.MustRegister(kmsLatencyMetric)
	prometheus.MustRegister(kmsOperationCounterV2)
	prometheus.MustRegister(kmsLatencyMetricV2)
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
		},
	)

	kmsOperationCounterV2 = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "aws_encryption_provider_kmsv2_operations_total",
			Help: "total aws encryption provider kms v2 operations",
		},
		[]string{
			"key_arn",
			"status",
			"operation",
		},
	)

	kmsLatencyMetricV2 = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "aws_encryption_provider_kmsv2_operation_latency_ms",
			Help:    "Response latency in milliseconds for aws encryption provider kms v2 operation ",
			Buckets: prometheus.ExponentialBuckets(2, 2, 14),
		},
		[]string{
			"key_arn",
			"status",
			"operation",
		},
	)
)
