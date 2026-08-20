package oauth

import "testing"

func TestRedirectURIExact(t *testing.T) {
	reg := []string{"https://app.example/cb", "http://localhost:3000/cb"}
	if !RedirectURIExact("https://app.example/cb", reg) {
		t.Fatal("exact match should pass")
	}
	if RedirectURIExact("https://app.example/cb?x=1", reg) {
		t.Fatal("extra query must fail")
	}
	if RedirectURIExact("http://localhost:3001/cb", reg) {
		t.Fatal("port change must fail")
	}
	if RedirectURIExact("https://app.example/cb/", reg) {
		t.Fatal("trailing slash must fail")
	}
	if RedirectURIExact("", reg) {
		t.Fatal("empty must fail")
	}
}

func TestParseRedirectURI(t *testing.T) {
	if _, err := ParseRedirectURI("https://rp.example/cb"); err != nil {
		t.Fatal(err)
	}
	if _, err := ParseRedirectURI("pfhabit://callback"); err != nil {
		t.Fatal("native scheme", err)
	}
	if _, err := ParseRedirectURI("javascript:alert(1)"); err == nil {
		t.Fatal("javascript scheme")
	}
	if _, err := ParseRedirectURI("/relative"); err == nil {
		t.Fatal("relative")
	}
	if _, err := ParseRedirectURI("https://rp.example/cb#oops"); err == nil {
		t.Fatal("fragment")
	}
}
