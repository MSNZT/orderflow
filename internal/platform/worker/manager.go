package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Job interface {
	Run(ctx context.Context) error
}

type JobFunc func(ctx context.Context) error

func (f JobFunc) Run(ctx context.Context) error {
	return f(ctx)
}

type JobConfig struct {
	Interval   time.Duration
	RunOnStart bool
}

type MetricsRecorder interface {
	JobStarted(name string)

	JobFinished(
		name string,
		duration time.Duration,
		err error,
	)
}

type Manager struct {
	log             *slog.Logger
	jobs            map[string]Job
	jobConfigs      map[string]JobConfig
	cancels         map[string]context.CancelCauseFunc
	metricsRecorder MetricsRecorder
	wg              sync.WaitGroup
}

func New(log *slog.Logger, metricsRecorder MetricsRecorder) *Manager {
	return &Manager{
		log:             log,
		jobs:            make(map[string]Job),
		jobConfigs:      make(map[string]JobConfig),
		cancels:         make(map[string]context.CancelCauseFunc),
		metricsRecorder: metricsRecorder,
		wg:              sync.WaitGroup{},
	}
}

func (m *Manager) RegisterPeriodic(
	name string,
	interval time.Duration,
	runOnStart bool,
	job Job) {

	m.jobs[name] = job
	m.jobConfigs[name] = JobConfig{
		Interval:   interval,
		RunOnStart: runOnStart,
	}
}

func (m *Manager) StartAll(ctx context.Context) {
	for name, job := range m.jobs {
		cfg := m.jobConfigs[name]

		workerCtx, cancel := context.WithCancelCause(ctx)
		m.cancels[name] = cancel

		m.wg.Add(1)
		go m.runPeriodic(workerCtx, name, job, cfg)
	}
}

func (m *Manager) Wait() {
	m.wg.Wait()
	m.log.Info("all workers stopped")
}

func (m *Manager) runPeriodic(ctx context.Context, name string, job Job, cfg JobConfig) {
	defer m.wg.Done()

	if cfg.RunOnStart {
		m.executeJob(ctx, name, job)
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			reason := context.Cause(ctx)
			m.log.Info("job stopped",
				"name", name,
				"reason", reason,
			)
			return
		case <-ticker.C:
			m.executeJob(ctx, name, job)
		}
	}
}

func (m *Manager) executeJob(ctx context.Context, name string, job Job) {
	startedAt := time.Now()
	m.log.Info("executing job", "name", name)
	m.metricsRecorder.JobStarted(name)

	err := job.Run(ctx)
	duration := time.Since(startedAt)

	if err != nil {
		m.log.Error("job failed",
			"name", name,
			"error", err,
			"duration", duration,
		)
		m.metricsRecorder.JobFinished(name, duration, err)
		return
	}

	m.log.Info("job completed",
		"name", name,
		"duration", duration,
	)
	m.metricsRecorder.JobFinished(name, duration, nil)
}
