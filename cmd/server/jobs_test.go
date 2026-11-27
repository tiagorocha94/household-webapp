package main

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	authstore "github.com/tiagorocha94/household-webapp/internal/auth/store"

	"github.com/tiagorocha94/household-webapp/internal/config"
	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestBackgroundJobsFollowTheModulesThatRun(t *testing.T) {
	t.Parallel()

	db, err := database.Open(filepath.Join(t.TempDir(), "jobs-test.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	auth := authapi.New(authservice.New(authstore.New(db)), testutil.Logger(), authapi.Options{})

	names := func(set modules.Set, backups config.BackupConfig) []string {
		var out []string
		for _, job := range backgroundJobs(db, t.TempDir(), set, auth, backups, testutil.Logger()) {
			// Only the backup asks to run at start: the other jobs do work that
			// start-up has just done.
			if job.AtStart != (job.Name == "database backup") {
				t.Errorf("job %q: AtStart = %v", job.Name, job.AtStart)
			}
			out = append(out, job.Name)
		}
		return out
	}
	off := config.BackupConfig{}
	on := config.BackupConfig{Dir: t.TempDir(), Keep: 3}

	if got, want := names(modules.Set{}, off), []string{"login limits", "goal allocations"}; !slices.Equal(got, want) {
		t.Errorf("jobs with every module = %v, want %v", got, want)
	}
	noGoals, err := modules.NewSet([]string{modules.Cashflow, modules.Savings})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := names(noGoals, off), []string{"login limits"}; !slices.Equal(got, want) {
		t.Errorf("jobs without goals = %v, want %v", got, want)
	}
	// The backup comes last, so it holds what the other jobs changed, and
	// follows the config, not the modules.
	if got, want := names(modules.Set{}, on), []string{"login limits", "goal allocations", "database backup"}; !slices.Equal(got, want) {
		t.Errorf("jobs with backups on = %v, want %v", got, want)
	}
	if got, want := names(noGoals, on), []string{"login limits", "database backup"}; !slices.Equal(got, want) {
		t.Errorf("jobs without goals, backups on = %v, want %v", got, want)
	}
}

// The job copies the database that is running, whole, under the day's name.
func TestDatabaseBackupJobCopiesTheRunningDatabase(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	dir := filepath.Join(t.TempDir(), "backups")
	job := databaseBackupJob(db, config.BackupConfig{Dir: dir, Keep: 2}, testutil.Logger())
	if !job.AtStart || job.Every != 0 {
		t.Fatalf("schedule = AtStart %v, Every %v; want at start, then daily", job.AtStart, job.Every)
	}

	if err := job.Run(context.Background(), time.Date(2026, 11, 27, 3, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	copied, err := database.Open(filepath.Join(dir, "household-2026-11-27.db"))
	if err != nil {
		t.Fatalf("open backup: %v", err)
	}
	t.Cleanup(func() { copied.Close() })

	count := func(d *sql.DB) int {
		var n int
		if err := d.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil {
			t.Fatalf("count migrations: %v", err)
		}
		return n
	}
	if got, want := count(copied), count(db); got == 0 || got != want {
		t.Fatalf("backup has %d migrations, database has %d", got, want)
	}
}

// A failing backup is reported to the scheduler, which logs it and tries again.
func TestDatabaseBackupJobReportsAFailure(t *testing.T) {
	t.Parallel()

	db := testutil.NewDB(t)
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	job := databaseBackupJob(db, config.BackupConfig{Dir: filepath.Join(blocker, "backups"), Keep: 2}, testutil.Logger())
	if err := job.Run(context.Background(), time.Now()); err == nil {
		t.Fatal("Run() succeeded although the directory cannot be created")
	}
}

// Running the job on data that start-up has just synced changes nothing, and
// leaves the goals files marked as up to date.
func TestGoalAllocationsJobIsRepeatableAndKeepsTheFilesInStep(t *testing.T) {
	t.Parallel()

	db, err := database.Open(filepath.Join(t.TempDir(), "jobs-test.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	dir := t.TempDir()
	copyDir(t, filepath.Join("..", "..", "demo"), dir)
	if err := prepareData(db, dir, modules.Set{}, testutil.Logger()); err != nil {
		t.Fatalf("prepareData() error = %v", err)
	}
	before := readDir(t, dir)

	job := goalAllocationsJob(db, dir, testutil.Logger())
	for i := 0; i < 2; i++ {
		if err := job.Run(context.Background(), time.Now()); err != nil {
			t.Fatalf("run %d error = %v", i+1, err)
		}
	}

	after := readDir(t, dir)
	for name, want := range before {
		if after[name] != want {
			t.Errorf("%s changed after the daily job", name)
		}
	}
	behind, err := csvstate.Behind(db, modules.Goals)
	if err != nil {
		t.Fatalf("Behind() error = %v", err)
	}
	if behind {
		t.Error("the goals files are behind the database after the job")
	}
}
