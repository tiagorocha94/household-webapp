package main

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	authapi "github.com/tiagorocha94/household-webapp/internal/auth/api"
	authservice "github.com/tiagorocha94/household-webapp/internal/auth/service"
	authstore "github.com/tiagorocha94/household-webapp/internal/auth/store"

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

	names := func(set modules.Set) []string {
		var out []string
		for _, job := range backgroundJobs(db, t.TempDir(), set, auth, testutil.Logger()) {
			if job.AtStart {
				t.Errorf("job %q runs at start, but start-up already did the work", job.Name)
			}
			out = append(out, job.Name)
		}
		return out
	}

	if got, want := names(modules.Set{}), []string{"login limits", "goal allocations"}; !slices.Equal(got, want) {
		t.Errorf("jobs with every module = %v, want %v", got, want)
	}
	noGoals, err := modules.NewSet([]string{modules.Cashflow, modules.Savings})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := names(noGoals), []string{"login limits"}; !slices.Equal(got, want) {
		t.Errorf("jobs without goals = %v, want %v", got, want)
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
