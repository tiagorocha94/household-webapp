package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/tiagorocha94/household-webapp/internal/csvstate"
	"github.com/tiagorocha94/household-webapp/internal/database"
	"github.com/tiagorocha94/household-webapp/internal/modules"
	"github.com/tiagorocha94/household-webapp/internal/testutil"
)

func TestDailyJobsFollowTheModulesThatRun(t *testing.T) {
	t.Parallel()

	db, err := database.Open(filepath.Join(t.TempDir(), "jobs-test.db"))
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })

	all := dailyJobs(db, t.TempDir(), modules.Set{}, testutil.Logger())
	if len(all) != 1 || all[0].Name != "goal allocations" || all[0].AtStart {
		t.Errorf("jobs with every module = %+v, want the goal allocations, not at start", all)
	}
	noGoals, err := modules.NewSet([]string{modules.Cashflow, modules.Savings})
	if err != nil {
		t.Fatal(err)
	}
	if got := dailyJobs(db, t.TempDir(), noGoals, testutil.Logger()); len(got) != 0 {
		t.Errorf("jobs without goals = %+v, want none", got)
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
