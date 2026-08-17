// Package oauth holds OAuth 2.1 / OIDC helpers that must not depend on HTTP or storage.
package oauth

import (
	"net/url"
)

// RedirectURIExact reports whether requested matches a registered URI.
// Spec-shaped servers compare the whole string; allowing extra query parameters
// is a classic open-redirect footgun.
func RedirectURIExact(requested string, registered []string) bool {
	if requested == "" {
		return false
	}
	for _, allowed := range registered {
		if requested == allowed {
			return true
		}
	}
	return false
}

// ParseRedirectURI rejects empty and clearly non-absolute http(s) URIs.
// Custom schemes are out of scope for this learning IdP (use claimed HTTPS for mobile).
func ParseRedirectURI(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, errInvalidRedirect
	}
	if u.Host == "" {
		return nil, errInvalidRedirect
	}
	// Fragments never reach the server; refusing them keeps the registered
	// value identical to what /token will compare against.
	if u.Fragment != "" {
		return nil, errInvalidRedirect
	}
	return u, nil
}
