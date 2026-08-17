// Package ratelimit slows password guessing without locking a user out forever.
// Keys include the email and client address so one attacker cannot freeze every account.
package ratelimit

import (
	"sync"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
)

// Limiter is a sliding-window counter. It is process-local; multiple IdP
// replicas would need a shared store, which is out of scope for this learning IdP.
type Limiter struct {
	mu      sync.Mutex
	hits    map[string][]time.Time
	Window  time.Duration
	Max     int
	Clock   clock.Clock
}

// New returns a limiter. max is the number of failures allowed inside window.
func New(max int, window time.Duration, clk clock.Clock) *Limiter {
	if max <= 0 {
		max = 5
	}
	if window <= 0 {
		window = time.Minute
	}
	if clk == nil {
		clk = clock.Real{}
	}
	return &Limiter{hits: map[string][]time.Time{}, Window: window, Max: max, Clock: clk}
}

// Limited reports whether key has already exhausted the window.
func (l *Limiter) Limited(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.prune(key)) >= l.Max
}

// Failure records an unsuccessful attempt.
func (l *Limiter) Failure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.Clock.Now()
	l.hits[key] = append(l.prune(key), now)
}

// Success clears the window so a real user is not punished after guessing once.
func (l *Limiter) Success(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}

func (l *Limiter) prune(key string) []time.Time {
	now := l.Clock.Now()
	cut := now.Add(-l.Window)
	old := l.hits[key]
	out := old[:0]
	for _, t := range old {
		if t.After(cut) {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		delete(l.hits, key)
		return nil
	}
	l.hits[key] = out
	return out
}
