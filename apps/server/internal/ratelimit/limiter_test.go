package ratelimit

import (
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
)

func TestLimiterBlocksAfterMaxFailures(t *testing.T) {
	now := time.Date(2026, 8, 17, 5, 0, 0, 0, time.UTC)
	clk := &clock.Fixed{T: now}
	l := New(3, time.Minute, clk)
	key := "127.0.0.1\nuser@example.com"
	for i := 0; i < 3; i++ {
		if l.Limited(key) {
			t.Fatalf("limited too early at %d", i)
		}
		l.Failure(key)
	}
	if !l.Limited(key) {
		t.Fatal("expected limit after 3 failures")
	}
}

func TestLimiterWindowExpiry(t *testing.T) {
	now := time.Date(2026, 8, 17, 5, 0, 0, 0, time.UTC)
	clk := &clock.Fixed{T: now}
	l := New(1, time.Minute, clk)
	l.Failure("k")
	if !l.Limited("k") {
		t.Fatal("expected limited")
	}
	clk.T = now.Add(time.Minute + time.Second)
	if l.Limited("k") {
		t.Fatal("window should have elapsed")
	}
}

func TestSuccessClearsFailures(t *testing.T) {
	l := New(1, time.Minute, clock.Real{})
	l.Failure("k")
	l.Success("k")
	if l.Limited("k") {
		t.Fatal("success should reset")
	}
}
