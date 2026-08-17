package password

import "testing"

func TestHashRejectsEmpty(t *testing.T) {
	if _, err := Hash(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestVerifyRoundTrip(t *testing.T) {
	h, err := Hash("correct horse")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := Verify("correct horse", h)
	if err != nil || !ok {
		t.Fatalf("verify true: ok=%v err=%v", ok, err)
	}
	ok, err = Verify("wrong", h)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("wrong password accepted")
	}
}

func TestVerifyEmptyIsFalse(t *testing.T) {
	h, err := Hash("x")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := Verify("", h)
	if err != nil || ok {
		t.Fatalf("empty password must not verify: ok=%v err=%v", ok, err)
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	ok, err := Verify("x", "not-phc")
	if err == nil || ok {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
}

func TestHashesDiffer(t *testing.T) {
	a, err := Hash("same")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Hash("same")
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatal("salts should make encodings unique")
	}
}
