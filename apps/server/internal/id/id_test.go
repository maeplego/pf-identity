package id

import "testing"

func TestNewUnique(t *testing.T) {
	a := New()
	b := New()
	if a == b {
		t.Fatal("ULIDs collided")
	}
	if err := Parse(a); err != nil {
		t.Fatal(err)
	}
}

func TestParseRejectsEmpty(t *testing.T) {
	if err := Parse(""); err == nil {
		t.Fatal("expected error")
	}
	if err := Parse("not-a-ulid"); err == nil {
		t.Fatal("expected error")
	}
}
