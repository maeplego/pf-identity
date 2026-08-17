package clock

import (
	"testing"
	"time"
)

func TestFixedNow(t *testing.T) {
	ts := time.Date(2026, 8, 17, 4, 0, 0, 0, time.UTC)
	c := Fixed{T: ts}
	if !c.Now().Equal(ts) {
		t.Fatalf("got %s", c.Now())
	}
}
