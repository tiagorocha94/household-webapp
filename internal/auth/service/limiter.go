package service

import (
	"sync"
	"time"
)

// LoginLimiter throttles repeated failed logins. It counts failures per key
// in a fixed window that starts at the first failure, and refuses further
// attempts for a key once it has reached the maximum, until the window ends.
// Keys are chosen by the caller, for example one per client address and one
// per username. The state is in memory, like the sessions.
type LoginLimiter struct {
	mu       sync.Mutex
	now      func() time.Time
	max      int
	window   time.Duration
	failures map[string]failureWindow
}

type failureWindow struct {
	count int
	ends  time.Time
}

// NewLoginLimiter returns a limiter that allows max failures per key within
// each window.
func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		now:      time.Now,
		max:      max,
		window:   window,
		failures: make(map[string]failureWindow),
	}
}

// Allow reports whether an attempt may go ahead. When any of keys is over its
// limit it returns false and how long until the longest such window ends.
func (l *LoginLimiter) Allow(keys ...string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	var retryAfter time.Duration
	for _, key := range keys {
		w, ok := l.failures[key]
		if !ok || !now.Before(w.ends) || w.count < l.max {
			continue
		}
		retryAfter = max(retryAfter, w.ends.Sub(now))
	}
	return retryAfter == 0, retryAfter
}

// Fail records a failed attempt for each key. It also drops windows that have
// ended, so the map does not grow without bound.
func (l *LoginLimiter) Fail(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	l.dropEnded(now)
	for _, key := range keys {
		w := l.failures[key]
		if w.count == 0 {
			w.ends = now.Add(l.window)
		}
		w.count++
		l.failures[key] = w
	}
}

// Prune drops the windows that have ended and returns how many it dropped. Fail
// does the same, but only when another failure comes, so after a burst of
// attempts the entries would otherwise stay until then.
func (l *LoginLimiter) Prune() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.dropEnded(l.now())
}

// dropEnded removes the windows that ended by now. The caller holds the lock.
func (l *LoginLimiter) dropEnded(now time.Time) int {
	dropped := 0
	for key, w := range l.failures {
		if !now.Before(w.ends) {
			delete(l.failures, key)
			dropped++
		}
	}
	return dropped
}

// Reset forgets the failures recorded for each key, after a successful login.
func (l *LoginLimiter) Reset(keys ...string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, key := range keys {
		delete(l.failures, key)
	}
}
