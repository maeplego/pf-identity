package random

import (
	"encoding/base64"
	"testing"
)

func TestBytesRejectsShortLength(t *testing.T) {
	if _, err := Bytes(15); err == nil {
		t.Fatal("expected error for short length")
	}
}

func TestBytesLength(t *testing.T) {
	b, err := Bytes(32)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 32 {
		t.Fatalf("len=%d", len(b))
	}
}

func TestTokenUniqueAndDecodable(t *testing.T) {
	a, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	b, err := Token()
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("tokens should not collide")
	}
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 32 {
		t.Fatalf("decoded len=%d", len(raw))
	}
}
