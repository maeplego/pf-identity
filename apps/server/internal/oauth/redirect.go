// Package oauth holds OAuth 2.1 / OIDC helpers that must not depend on HTTP or storage.
package oauth

import (
	"net/url"
	"strings"
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

var blockedRedirectSchemes = map[string]struct{}{
	"javascript": {},
	"data":       {},
	"file":       {},
	"vbscript":   {},
}

// ParseRedirectURI rejects empty, relative, and dangerous URIs.
// Web RPs use http(s); native/mobile clients may register custom schemes (e.g. pfhabit://callback).
func ParseRedirectURI(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme == "" {
		return nil, errInvalidRedirect
	}
	if u.Fragment != "" {
		return nil, errInvalidRedirect
	}
	if _, blocked := blockedRedirectSchemes[strings.ToLower(u.Scheme)]; blocked {
		return nil, errInvalidRedirect
	}
	switch u.Scheme {
	case "http", "https":
		if u.Host == "" {
			return nil, errInvalidRedirect
		}
		return u, nil
	default:
		if u.Host == "" && u.Path == "" && u.Opaque == "" {
			return nil, errInvalidRedirect
		}
		return u, nil
	}
}
