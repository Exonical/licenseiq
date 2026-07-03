package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type Job interface {
	Name() string
	Interval() time.Duration
	Run(context.Context) error
}

type Scheduler struct {
	jobs    []Job
	timeout time.Duration
	logger  *zap.Logger
	tracer  trace.Tracer
	meter   metric.Meter
	mu      sync.Mutex
	wg      sync.WaitGroup
	started bool
}

func NewScheduler(logger *zap.Logger, timeout time.Duration) *Scheduler {
	if logger == nil {
		logger = zap.NewNop()
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &Scheduler{timeout: timeout, logger: logger, tracer: otel.Tracer("licenseiq/internal/worker"), meter: otel.Meter("licenseiq/internal/worker")}
}

func (s *Scheduler) Register(job Job) {
	if s == nil || job == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs = append(s.jobs, job)
}

func (s *Scheduler) Start(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	s.started = true
	jobs := append([]Job(nil), s.jobs...)
	s.mu.Unlock()
	s.wg.Add(len(jobs))
	for _, job := range jobs {
		job := job
		go s.loop(ctx, job)
	}
	<-ctx.Done()
	s.wg.Wait()
	return ctx.Err()
}

func (s *Scheduler) loop(ctx context.Context, job Job) {
	defer s.wg.Done()
	if job == nil {
		return
	}
	interval := job.Interval()
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	s.runOnce(ctx, job)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runOnce(ctx, job)
		}
	}
}

func (s *Scheduler) runOnce(ctx context.Context, job Job) {
	if job == nil {
		return
	}
	start := time.Now()
	ctxRun, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	spanCtx, span := s.tracer.Start(ctxRun, "worker.job.run")
	defer span.End()
	if c, err := s.meter.Int64Counter("licenseiq_worker_job_runs"); err == nil {
		defer c.Add(spanCtx, 1)
	}
	if h, err := s.meter.Float64Histogram("licenseiq_worker_job_duration_seconds"); err == nil {
		defer func() { h.Record(spanCtx, time.Since(start).Seconds()) }()
	}
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("worker job panicked", zap.String("job", job.Name()), zap.Any("panic", r))
			if c, err := s.meter.Int64Counter("licenseiq_worker_job_failures"); err == nil {
				c.Add(spanCtx, 1)
			}
		}
	}()
	if err := job.Run(spanCtx); err != nil {
		s.logger.Warn("worker job failed", zap.String("job", job.Name()), zap.Error(err))
		span.RecordError(err)
		if c, err := s.meter.Int64Counter("licenseiq_worker_job_failures"); err == nil {
			c.Add(spanCtx, 1)
		}
	}
}

func (s *Scheduler) String() string { return fmt.Sprintf("Scheduler(jobs=%d)", len(s.jobs)) }
