// Package clock exists so tests can freeze "now" around code expiry and sessions.
package clock

import "time"

// Clock returns the current time in UTC.
type Clock interface {
	Now() time.Time
}

// Real uses the process clock.
type Real struct{}

// Now implements Clock.
func (Real) Now() time.Time { return time.Now().UTC() }

// Fixed returns the same instant every call.
type Fixed struct{ T time.Time }

// Now implements Clock.
func (f Fixed) Now() time.Time { return f.T.UTC() }
