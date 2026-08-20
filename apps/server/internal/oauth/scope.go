package oauth

import (
	"fmt"
	"sort"
	"strings"
)

// Allowed scopes for the first release. Fine-grained product scopes are added
// when a relying party actually needs them; listing dozens here makes consent unusable.
var allowed = map[string]struct{}{
	"openid":         {},
	"profile":        {},
	"email":          {},
	"offline_access": {},
	"org":            {},
}

// NormalizeScopes splits a scope string, drops duplicates, and rejects unknown names.
// openid is required for OIDC authorization requests.
func NormalizeScopes(raw string, requireOpenID bool) ([]string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("scope is required")
	}
	seen := map[string]struct{}{}
	var out []string
	for _, s := range strings.Fields(raw) {
		if _, ok := allowed[s]; !ok {
			return nil, fmt.Errorf("unsupported scope %q", s)
		}
		if _, dup := seen[s]; dup {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	if requireOpenID {
		if _, ok := seen["openid"]; !ok {
			return nil, fmt.Errorf("openid scope is required")
		}
	}
	sort.Strings(out)
	return out, nil
}

// Contains reports whether scope list includes name.
func Contains(scopes []string, name string) bool {
	for _, s := range scopes {
		if s == name {
			return true
		}
	}
	return false
}
