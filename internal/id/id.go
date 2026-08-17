// Package id allocates public identifiers. Sequential integers are not used
// because they leak creation order and are trivial to enumerate.
package id

import (
	"fmt"

	"github.com/oklog/ulid/v2"
)

// New returns a new ULID string (crockford base32, 26 characters).
func New() string {
	return ulid.Make().String()
}

// Parse checks that s is a ULID. Handlers use this before hitting storage.
func Parse(s string) error {
	if _, err := ulid.Parse(s); err != nil {
		return fmt.Errorf("invalid id: %w", err)
	}
	return nil
}
