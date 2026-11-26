package service

import (
	"testing"
	"time"
)

func newTestLimiter(max int, window time.Duration) (*LoginLimiter, *time.Time) {
	now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	l := NewLoginLimiter(max, window)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestLoginLimiterBlocksAfterMaxFailuresUntilTheWindowEnds(t *testing.T) {
	t.Parallel()

	l, now := newTestLimiter(3, time.Minute)
	for i := range 3 {
		if ok, _ := l.Allow("user:a"); !ok {
			t.Fatalf("attempt %d blocked, want allowed", i+1)
		}
		l.Fail("user:a")
	}

	ok, retry := l.Allow("user:a")
	if ok || retry != time.Minute {
		t.Fatalf("Allow() = %v, %v, want blocked for 1m0s", ok, retry)
	}
	*now = now.Add(20 * time.Second)
	if _, retry := l.Allow("user:a"); retry != 40*time.Second {
		t.Errorf("retry after 20s = %v, want 40s", retry)
	}
	*now = now.Add(40 * time.Second)
	if ok, _ := l.Allow("user:a"); !ok {
		t.Error("still blocked once the window ended")
	}
}

func TestLoginLimiterKeysAreIndependentAndAnyBlockedKeyBlocks(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimiter(2, time.Minute)
	l.Fail("ip:1", "user:a")
	l.Fail("ip:1", "user:b")

	if ok, _ := l.Allow("ip:2", "user:a"); !ok {
		t.Error("user:a with a fresh address blocked after one failure")
	}
	if ok, _ := l.Allow("ip:1", "user:c"); ok {
		t.Error("ip:1 allowed after two failures, want blocked whatever the username")
	}
}

func TestLoginLimiterResetClearsOnlyTheGivenKeys(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimiter(1, time.Minute)
	l.Fail("ip:1", "user:a")
	l.Reset("user:a")

	if ok, _ := l.Allow("user:a"); !ok {
		t.Error("user:a blocked after Reset")
	}
	if ok, _ := l.Allow("ip:1"); ok {
		t.Error("ip:1 allowed, want it still blocked: Reset must not clear other keys")
	}
}

func TestLoginLimiterDropsEndedWindows(t *testing.T) {
	t.Parallel()

	l, now := newTestLimiter(1, time.Minute)
	l.Fail("user:old")
	*now = now.Add(2 * time.Minute)
	l.Fail("user:new")

	if _, ok := l.failures["user:old"]; ok {
		t.Error("ended window for user:old was kept, want it dropped")
	}
	if l.failures["user:new"].count != 1 {
		t.Errorf("user:new count = %d, want 1", l.failures["user:new"].count)
	}
}

func TestLoginLimiterPruneDropsOnlyTheWindowsThatEnded(t *testing.T) {
	t.Parallel()

	l, now := newTestLimiter(3, time.Minute)
	l.Fail("ip:old")
	*now = now.Add(45 * time.Second)
	l.Fail("ip:recent")

	*now = now.Add(30 * time.Second) // old ended 15s ago, recent has 45s left
	if dropped := l.Prune(); dropped != 1 {
		t.Fatalf("Prune() = %d, want 1", dropped)
	}
	if len(l.failures) != 1 {
		t.Errorf("entries after Prune = %v, want only the recent one", l.failures)
	}
	if _, ok := l.failures["ip:recent"]; !ok {
		t.Error("the window still running was dropped")
	}
	if dropped := l.Prune(); dropped != 0 {
		t.Errorf("second Prune() = %d, want 0", dropped)
	}
}
