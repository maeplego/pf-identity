package domain

import "errors"

var (
	// ErrNotFound is a missing row.
	ErrNotFound = errors.New("not found")
	// ErrConflict is a uniqueness violation (email, client id).
	ErrConflict = errors.New("conflict")
	// ErrUsed is a single-use token already consumed.
	ErrUsed = errors.New("already used")
)
