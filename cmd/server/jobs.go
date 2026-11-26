package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/scheduler"
	"github.com/tiagorocha94/household-webapp/internal/seed"
)

// dailyJobs lists the work the running server does once a day, for the modules
// that run. Start-up already does it once, so none of them asks to run at start.
func dailyJobs(db *sql.DB, seedDir string, enabled modules.Set, logger *slog.Logger) []scheduler.Job {
	var jobs []scheduler.Job
	if enabled.Has(modules.Goals) {
		jobs = append(jobs, goalAllocationsJob(db, seedDir, logger))
	}
	return jobs
}

// goalAllocationsJob creates the goal allocations that have become due, so a new
// month needs no restart. It runs the goals module's sync on the live database,
// then rewrites the goals files through the same locked mirror devices' writes
// use, so an older snapshot can never replace a newer file. The sync is safe to
// repeat: it only adds what is missing.
func goalAllocationsJob(db *sql.DB, seedDir string, logger *slog.Logger) scheduler.Job {
	runner := seed.NewRunner(logger, goals.SeedModule{})
	mirror := csvstate.NewMirror(db, seedDir, goals.SeedModule{})
	return scheduler.Job{
		Name: "goal allocations",
		Run: func(context.Context, time.Time) error {
			if err := runner.Sync(db); err != nil {
				return fmt.Errorf("sync goals: %w", err)
			}
			if err := csvstate.Raise(db, modules.Goals); err != nil {
				return err
			}
			return mirror.Flush()
		},
	}
}
