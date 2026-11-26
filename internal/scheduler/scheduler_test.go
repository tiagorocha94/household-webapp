package scheduler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type clock struct {
	mu  sync.Mutex
	now time.Time
}

// harness runs a scheduler on a clock and ticks that the test controls.
type harness struct {
	clock *clock
	ticks chan time.Time
	runs  chan string
	done  chan struct{}
	stop  context.CancelFunc
}

func (c *clock) get() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *clock) set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}

func start(t *testing.T, first time.Time, jobs func(runs chan<- string) []Job) *harness {
	t.Helper()
	h := &harness{
		clock: &clock{now: first},
		ticks: make(chan time.Time),
		runs:  make(chan string, 16),
		done:  make(chan struct{}),
	}
	s := New(slog.New(slog.NewTextHandler(io.Discard, nil)), jobs(h.runs)...)
	s.now = h.clock.get
	s.ticks = func() (<-chan time.Time, func()) { return h.ticks, func() {} }
	ctx, cancel := context.WithCancel(context.Background())
	h.stop = cancel
	go func() {
		defer close(h.done)
		s.Run(ctx)
	}()
	t.Cleanup(func() {
		cancel()
		<-h.done
	})
	// The channel is not buffered, so a tick is only taken once the scheduler has
	// noted the day it started on and run its start jobs.
	h.tick(t, first)
	return h
}

// tick moves the clock and waits for the scheduler to have looked at it.
func (h *harness) tick(t *testing.T, now time.Time) {
	t.Helper()
	h.clock.set(now)
	select {
	case h.ticks <- now:
	case <-time.After(2 * time.Second):
		t.Fatal("the scheduler did not take a tick")
	}
}

func (h *harness) expect(t *testing.T, want ...string) {
	t.Helper()
	for _, w := range want {
		select {
		case got := <-h.runs:
			if got != w {
				t.Fatalf("ran %q, want %q", got, w)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("%q did not run", w)
		}
	}
}

func (h *harness) expectNone(t *testing.T) {
	t.Helper()
	// A tick is only taken once the previous one has been handled, so one more
	// tick proves the scheduler is idle before the channel is read.
	h.ticks <- h.clock.get()
	select {
	case got := <-h.runs:
		t.Fatalf("%q ran, want nothing", got)
	default:
	}
}

func recorder(name string, atStart bool) func(chan<- string) Job {
	return every(name, 0, atStart)
}

func every(name string, interval time.Duration, atStart bool) func(chan<- string) Job {
	return func(runs chan<- string) Job {
		return Job{Name: name, Every: interval, AtStart: atStart, Run: func(context.Context, time.Time) error {
			runs <- name
			return nil
		}}
	}
}

func TestRunsOnlyTheJobsThatAskForItAtStart(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)
	h := start(t, day, func(runs chan<- string) []Job {
		return []Job{recorder("backup", true)(runs), recorder("allocations", false)(runs)}
	})
	h.expect(t, "backup")
	h.expectNone(t)
}

func TestRunsEveryJobWhenTheDateChanges(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)
	h := start(t, day, func(runs chan<- string) []Job {
		return []Job{recorder("allocations", false)(runs), recorder("backup", true)(runs)}
	})
	h.expect(t, "backup")

	h.tick(t, day.Add(5*time.Hour)) // same day
	h.expectNone(t)

	h.tick(t, time.Date(2026, 12, 1, 0, 1, 0, 0, time.UTC))
	h.expect(t, "allocations", "backup")

	h.tick(t, time.Date(2026, 12, 1, 23, 59, 0, 0, time.UTC)) // still that day
	h.expectNone(t)

	h.tick(t, time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC))
	h.expect(t, "allocations", "backup")
}

func TestAJobWithAnIntervalRunsWhenThatMuchTimeHasPassedWhateverTheDate(t *testing.T) {
	t.Parallel()
	start0 := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	h := start(t, start0, func(runs chan<- string) []Job {
		return []Job{every("prune", 15*time.Minute, false)(runs), recorder("daily", false)(runs)}
	})

	h.tick(t, start0.Add(14*time.Minute))
	h.expectNone(t)

	h.tick(t, start0.Add(15*time.Minute))
	h.expect(t, "prune")

	h.tick(t, start0.Add(29*time.Minute)) // 14 minutes after its last run
	h.expectNone(t)

	h.tick(t, start0.Add(30*time.Minute))
	h.expect(t, "prune")

	// Midnight: the daily job runs, and the prune job has its own turn too.
	h.tick(t, time.Date(2026, 11, 27, 0, 1, 0, 0, time.UTC))
	h.expect(t, "prune", "daily")
}

func TestAnIntervalJobThatRunsAtStartCountsFromThen(t *testing.T) {
	t.Parallel()
	start0 := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	h := start(t, start0, func(runs chan<- string) []Job {
		return []Job{every("prune", time.Hour, true)(runs)}
	})
	h.expect(t, "prune")
	h.tick(t, start0.Add(59*time.Minute))
	h.expectNone(t)
	h.tick(t, start0.Add(time.Hour))
	h.expect(t, "prune")
}

func TestRunsOnceWhenSeveralDaysWerePassedOver(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	h := start(t, day, func(runs chan<- string) []Job {
		return []Job{recorder("allocations", false)(runs)}
	})
	// A laptop that slept for three days wakes up.
	h.tick(t, time.Date(2026, 11, 29, 8, 0, 0, 0, time.UTC))
	h.expect(t, "allocations")
	h.expectNone(t)
}

func TestAFailingJobDoesNotStopTheOthersOrTheNextDay(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)
	h := start(t, day, func(runs chan<- string) []Job {
		return []Job{
			{Name: "broken", Run: func(context.Context, time.Time) error {
				runs <- "broken"
				return errors.New("disk full")
			}},
			{Name: "panics", Run: func(context.Context, time.Time) error {
				runs <- "panics"
				panic("boom")
			}},
			recorder("backup", false)(runs),
		}
	})
	h.tick(t, time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC))
	h.expect(t, "broken", "panics", "backup")
	h.tick(t, time.Date(2026, 12, 2, 0, 0, 0, 0, time.UTC))
	h.expect(t, "broken", "panics", "backup")
}

func TestJobsGetTheTimeOfTheRun(t *testing.T) {
	t.Parallel()
	day := time.Date(2026, 11, 30, 10, 0, 0, 0, time.UTC)
	got := make(chan time.Time, 1)
	h := start(t, day, func(chan<- string) []Job {
		return []Job{{Name: "clock", Run: func(_ context.Context, now time.Time) error {
			got <- now
			return nil
		}}}
	})
	next := time.Date(2026, 12, 1, 0, 0, 5, 0, time.UTC)
	h.tick(t, next)
	select {
	case at := <-got:
		if !at.Equal(next) {
			t.Errorf("job saw %v, want %v", at, next)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the job did not run")
	}
}

func TestWithNoJobsRunReturnsAtOnce(t *testing.T) {
	t.Parallel()
	New(slog.New(slog.NewTextHandler(io.Discard, nil))).Run(context.Background())
}

func TestStopsWithItsContext(t *testing.T) {
	t.Parallel()
	h := start(t, time.Now(), func(runs chan<- string) []Job { return []Job{recorder("x", false)(runs)} })
	h.stop()
	select {
	case <-h.done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after its context was cancelled")
	}
}
