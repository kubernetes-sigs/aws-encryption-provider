package plugin

import "github.com/prometheus/client_golang/prometheus"

func init() {
	registerPrometheusMetrics()
}

func registerPrometheusMetrics() {
	prometheus.MustRegister(kmsOperationCounter)
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
)
