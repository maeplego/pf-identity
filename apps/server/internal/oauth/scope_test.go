package oauth

import (
	"reflect"
	"testing"
)

func TestNormalizeScopes(t *testing.T) {
	got, err := NormalizeScopes("email openid profile email", true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"email", "openid", "profile"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestNormalizeScopesRejectsUnknown(t *testing.T) {
	if _, err := NormalizeScopes("openid calendar.book", true); err == nil {
		t.Fatal("expected unknown scope error")
	}
}

func TestNormalizeScopesRequiresOpenID(t *testing.T) {
	if _, err := NormalizeScopes("email profile", true); err == nil {
		t.Fatal("expected missing openid")
	}
}

func TestContains(t *testing.T) {
	if !Contains([]string{"email", "openid"}, "openid") {
		t.Fatal("expected hit")
	}
	if Contains([]string{"email"}, "offline_access") {
		t.Fatal("expected miss")
	}
}
