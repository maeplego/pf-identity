package account

import "errors"

// errInvalidCreds is shared for unknown users, bad passwords, and disabled accounts
// so login responses do not leak which emails exist.
var errInvalidCreds = errors.New("invalid email or password")

// ErrInvalidCredentials is the exported login failure.
func ErrInvalidCredentials() error { return errInvalidCreds }

// IsInvalidCredentials reports whether err is a failed login.
func IsInvalidCredentials(err error) bool {
	return errors.Is(err, errInvalidCreds)
}
