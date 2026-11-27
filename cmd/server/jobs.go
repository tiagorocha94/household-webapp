package main

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
	"github.com/tiagorocha94/household-webapp/internal/backup"
	"github.com/tiagorocha94/household-webapp/internal/config"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/goals"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/scheduler"
	"github.com/tiagorocha94/household-webapp/internal/seed"
)

// backgroundJobs lists the work the running server does by itself, each job on
// its own schedule. The jobs that tidy up or sync need not run at start, since
// start-up has just done that work; the backup does, so a server that is
// started every day still makes its daily copy. The backup comes last, so it
// holds what the jobs before it changed.
func backgroundJobs(db *sql.DB, seedDir string, enabled modules.Set, auth *authapi.Handler, backups config.BackupConfig, logger *slog.Logger) []scheduler.Job {
	jobs := []scheduler.Job{loginLimitsJob(auth)}
	if enabled.Has(modules.Goals) {
		jobs = append(jobs, goalAllocationsJob(db, seedDir, logger))
	}
	if backups.Enabled() {
		jobs = append(jobs, databaseBackupJob(db, backups, logger))
	}
	return jobs
}

// loginLimitsJob forgets the failed logins whose window has ended, so they do
// not stay in memory after a burst of attempts. It runs as often as a window
// lasts.
func loginLimitsJob(auth *authapi.Handler) scheduler.Job {
	return scheduler.Job{
		Name:  "login limits",
		Every: authapi.LimitPruneInterval,
		Run: func(context.Context, time.Time) error {
			auth.PruneLimits()
			return nil
		},
	}
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

// databaseBackupJob copies the database into the backup directory once a day,
// and when the server starts. The copy is the real database, made while the
// server runs, so it holds what phones have written since the CSV files were
// last read. A failure is logged and the next run tries again.
func databaseBackupJob(db *sql.DB, cfg config.BackupConfig, logger *slog.Logger) scheduler.Job {
	return scheduler.Job{
		Name:    "database backup",
		AtStart: true,
		Run: func(ctx context.Context, now time.Time) error {
			res, err := backup.Run(ctx, db, cfg.Dir, cfg.Keep, now)
			if err != nil {
				return err
			}
			logger.Info("database backed up", "file", res.File, "removed", res.Removed)
			return nil
		},
	}
}
