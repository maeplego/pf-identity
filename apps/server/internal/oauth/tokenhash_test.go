package oauth

import "testing"

func TestHashTokenStableAndNotPlain(t *testing.T) {
	a := HashToken("secret-value")
	b := HashToken("secret-value")
	if a != b {
		t.Fatal("hash should be deterministic")
	}
	if a == "secret-value" {
		t.Fatal("must not store plaintext")
	}
	if HashToken("other") == a {
		t.Fatal("different inputs must not collide in this fixture")
	}
}
