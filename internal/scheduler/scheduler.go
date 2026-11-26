// Package scheduler runs the server's daily jobs: work that has to happen once
// a day without anyone asking, such as creating the next month's goal
// allocations. Jobs run when the server has just started (those that ask for it)
// and each time the local date changes while it runs. A failing or panicking job
// is logged and the server carries on; the job runs again the next day.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// A Job is one piece of daily work. Run must be safe to repeat: it is called
// once a day and again after a restart.
type Job struct {
	Name string
	// AtStart runs the job once when the scheduler starts, for work that start-up
	// itself has not just done.
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

// Run blocks until ctx is done. It runs the jobs that ask for it at once, then
// all of them whenever the local date is not the one it last ran on.
func (s *Scheduler) Run(ctx context.Context) {
	if len(s.jobs) == 0 {
		return
	}
	ticks, stop := s.ticks()
	defer stop()

	last := s.now()
	s.runJobs(ctx, last, true)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticks:
			now := s.now()
			if sameDay(last, now) {
				continue
			}
			last = now
			s.runJobs(ctx, now, false)
		}
	}
}

func (s *Scheduler) runJobs(ctx context.Context, now time.Time, start bool) {
	for _, job := range s.jobs {
		if start && !job.AtStart {
			continue
		}
		if ctx.Err() != nil {
			return
		}
		s.runJob(ctx, job, now)
	}
}

func (s *Scheduler) runJob(ctx context.Context, job Job, now time.Time) {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("daily job panicked", "job", job.Name, "panic", fmt.Sprint(r))
		}
	}()
	began := time.Now()
	if err := job.Run(ctx, now); err != nil {
		s.logger.Error("daily job failed", "job", job.Name, "error", err)
		return
	}
	s.logger.Info("daily job done", "job", job.Name, "took", time.Since(began).Round(time.Millisecond).String())
}

func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.In(a.Location()).Date()
	return ay == by && am == bm && ad == bd
}
