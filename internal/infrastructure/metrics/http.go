package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type HTTPMetrics struct {
	requestsTotal    *prometheus.CounterVec
	requestDuration  *prometheus.HistogramVec
	requestsInFlight prometheus.Gauge
}

func NewHTTPMetrics(register prometheus.Registerer) *HTTPMetrics {
	const (
		namespace = "orderflow"
		subsystem = "http"
	)
	m := HTTPMetrics{
		requestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "requests_total",
				Help:      "Total number of completed HTTP requests.",
			},
			[]string{"method", "route", "status"},
		),
		requestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "request_duration_seconds",
				Help:      "HTTP request duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "route"},
		),
		requestsInFlight: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "orderflow",
				Subsystem: "http",
				Name:      "requests_in_flight",
				Help:      "Current number of HTTP requests being processed.",
			},
		),
	}

	register.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.requestsInFlight,
	)

	return &m
}

func (m *HTTPMetrics) RequestStarted() {
	m.requestsInFlight.Inc()
}

func (m *HTTPMetrics) RequestFinished(method string, route string, status int, duration time.Duration) {
	m.requestsInFlight.Dec()

	m.requestsTotal.WithLabelValues(
		method, route, strconv.Itoa(status),
	).Inc()

	m.requestDuration.WithLabelValues(
		method, route,
	).Observe(duration.Seconds())
}
