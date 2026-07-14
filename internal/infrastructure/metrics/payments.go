package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	paymentOperationCreate  = "create"
	paymentOperationCapture = "capture"
	paymentOperationCancel  = "cancel"
	paymentOperationSuccess = "success"
	paymentOperationError   = "error"
)

type PaymentMetrics struct {
	operationsTotal   *prometheus.CounterVec
	operationDuration *prometheus.HistogramVec
}

func NewPaymentMetrics(
	registerer prometheus.Registerer,
) *PaymentMetrics {
	const (
		namespace = "orderflow"
		subsystem = "payments"
	)
	paymentMetrics := &PaymentMetrics{
		operationsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "operations_total",
				Help:      "Total number of completed payment operations.",
			},
			[]string{
				"operation",
				"result",
			},
		),

		operationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "operation_duration_seconds",
				Help:      "Payment operation duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{
				"operation",
			},
		),
	}

	registerer.MustRegister(
		paymentMetrics.operationsTotal,
		paymentMetrics.operationDuration,
	)

	return paymentMetrics
}

func (m *PaymentMetrics) CreateFinished(duration time.Duration, err error) {
	m.operationFinished(paymentOperationCreate, duration, err)
}

func (m *PaymentMetrics) CaptureFinished(duration time.Duration, err error) {
	m.operationFinished(paymentOperationCapture, duration, err)
}

func (m *PaymentMetrics) CancelFinished(duration time.Duration, err error) {
	m.operationFinished(paymentOperationCancel, duration, err)
}

func (m *PaymentMetrics) operationFinished(operation string, duration time.Duration, err error) {
	result := paymentOperationSuccess
	if err != nil {
		result = paymentOperationError
	}

	m.operationsTotal.
		WithLabelValues(operation, result).
		Inc()

	m.operationDuration.
		WithLabelValues(operation).
		Observe(duration.Seconds())
}
