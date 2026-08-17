package session

import (
	"context"
	"testing"
	"time"

	"github.com/portfolio/pf-identity-server/internal/clock"
	"github.com/portfolio/pf-identity-server/internal/domain"
	"github.com/portfolio/pf-identity-server/internal/store/memory"
)

func TestStartLookupEnd(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	svc := &Service{Sessions: memory.NewStore(), Clock: clock.Fixed{T: now}, TTL: time.Hour}
	plain, exp, err := svc.Start(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !exp.Equal(now.Add(time.Hour)) {
		t.Fatalf("exp %s", exp)
	}
	uid, err := svc.Lookup(context.Background(), plain)
	if err != nil || uid != "user-1" {
		t.Fatalf("lookup %q %v", uid, err)
	}
	sess, err := svc.LookupSession(context.Background(), plain)
	if err != nil || sess.SID == "" || sess.UserID != "user-1" {
		t.Fatalf("lookup session %+v %v", sess, err)
	}
	if err := svc.End(context.Background(), plain); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Lookup(context.Background(), plain); err != domain.ErrNotFound {
		t.Fatalf("after end: %v", err)
	}
}

func TestLookupExpired(t *testing.T) {
	now := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	store := memory.NewStore()
	svc := &Service{Sessions: store, Clock: clock.Fixed{T: now}, TTL: time.Minute}
	plain, _, err := svc.Start(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	svc.Clock = clock.Fixed{T: now.Add(2 * time.Minute)}
	if _, err := svc.Lookup(context.Background(), plain); err != domain.ErrNotFound {
		t.Fatalf("expired: %v", err)
	}
}

func TestLookupEmpty(t *testing.T) {
	svc := &Service{Sessions: memory.NewStore(), Clock: clock.Real{}, TTL: time.Hour}
	if _, err := svc.Lookup(context.Background(), ""); err != domain.ErrNotFound {
		t.Fatalf("err=%v", err)
	}
}
