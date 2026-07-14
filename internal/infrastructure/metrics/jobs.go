package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	jobResultSuccess = "success"
	jobResultError   = "error"
)

type JobsMetrics struct {
	runsTotal *prometheus.CounterVec
	duration  *prometheus.HistogramVec
	running   *prometheus.GaugeVec
}

func NewJobsMetrics(
	registerer prometheus.Registerer,
) *JobsMetrics {
	const (
		namespace = "orderflow"
		subsystem = "jobs"
	)
	jobsMetrics := &JobsMetrics{
		runsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "runs_total",
				Help:      "Total number of background job executions.",
			},
			[]string{
				"name",
				"result",
			},
		),

		duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "duration_seconds",
				Help:      "Background job execution duration in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{
				"name",
			},
		),

		running: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: subsystem,
				Name:      "running",
				Help:      "Current number of running background jobs.",
			},
			[]string{
				"name",
			},
		),
	}

	registerer.MustRegister(
		jobsMetrics.runsTotal,
		jobsMetrics.duration,
		jobsMetrics.running,
	)

	return jobsMetrics
}

func (m *JobsMetrics) JobStarted(name string) {
	m.running.
		WithLabelValues(name).
		Inc()
}

func (m *JobsMetrics) JobFinished(
	name string,
	duration time.Duration,
	err error,
) {
	m.running.
		WithLabelValues(name).
		Dec()

	result := jobResultSuccess
	if err != nil {
		result = jobResultError
	}

	m.runsTotal.
		WithLabelValues(name, result).
		Inc()

	m.duration.
		WithLabelValues(name).
		Observe(duration.Seconds())
}
