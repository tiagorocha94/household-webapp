// Package scheduler runs the server's background jobs: work that has to happen
// again and again without anyone asking, such as creating the next month's goal
// allocations or forgetting old login failures. Each job has its own schedule,
// once a day or every so often, and the scheduler only looks at the clock once a
// minute to see which jobs are due. A failing or panicking job is logged and
// the server carries on; the job is tried again when it is next due.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// A Job is one piece of recurring work. Run must be safe to repeat: it is called
// on its schedule and again after a restart.
type Job struct {
	Name string
	// Every is how often the job runs. Zero means once a day, when the local
	// date changes, which is not the same as every 24 hours: a job that runs
	// daily runs on the first look after midnight.
	Every time.Duration
	// AtStart runs the job once when the scheduler starts, for work that start-up
	// itself has not just done. The job's schedule counts from that moment, or
	// from the start when it does not run then.
	AtStart bool
	Run     func(ctx context.Context, now time.Time) error
}

// A Scheduler runs a list of jobs, in order.
type Scheduler struct {
	jobs   []Job
	logger *slog.Logger

	now func() time.Time
	// ticks tells it to look at the date again; every minute in production.
	ticks func() (<-chan time.Time, func())
}

// New returns a Scheduler for the given jobs.
func New(logger *slog.Logger, jobs ...Job) *Scheduler {
	return &Scheduler{
		jobs:   jobs,
		logger: logger,
		now:    time.Now,
		ticks: func() (<-chan time.Time, func()) {
			t := time.NewTicker(time.Minute)
			return t.C, t.Stop
		},
	}
}

// due reports whether a job that last ran at last is due at now.
func (j Job) due(last, now time.Time) bool {
	if j.Every > 0 {
		return now.Sub(last) >= j.Every
	}
	return !sameDay(last, now)
}

// Run blocks until ctx is done. It runs the jobs that ask for it at once, then,
// at every look at the clock, each job that is due. A job that is due runs once,
// however long it has been (a laptop that slept for days), and a failed run
// counts as a run: the job waits for its next turn rather than retrying every
// minute.
func (s *Scheduler) Run(ctx context.Context) {
	if len(s.jobs) == 0 {
		return
	}
	ticks, stop := s.ticks()
	defer stop()

	last := make([]time.Time, len(s.jobs))
	started := s.now()
	for i, job := range s.jobs {
		last[i] = started
		if job.AtStart {
			s.runJob(ctx, job, started)
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			now := s.now()
			for i, job := range s.jobs {
				if ctx.Err() != nil {
					return
				}
				if !job.due(last[i], now) {
					continue
				}
				last[i] = now
				s.runJob(ctx, job, now)
			}
		}
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job, now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("job panicked", "job", job.Name, "panic", fmt.Sprint(r))
		}
	}()
	began := time.Now()
	if err := job.Run(ctx, now); err != nil {
		s.logger.Error("job failed", "job", job.Name, "error", err)
		return
	}
	s.logger.Info("job done", "job", job.Name, "took", time.Since(began).Round(time.Millisecond).String())
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()
	return ay == by && am == bm && ad == bd
}
