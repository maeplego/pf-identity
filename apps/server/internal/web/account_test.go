package web

import "testing"

func TestSafeContinue(t *testing.T) {
	ok := "/authorize?client_id=a&redirect_uri=https%3A%2F%2Frp%2Fcb"
	if safeContinue(ok) != ok && safeContinue(ok) == "" {
		// url.Parse may reorder query; we only require a non-empty authorize path.
		if got := safeContinue(ok); got == "" {
			t.Fatalf("got empty for %q", ok)
		}
	}
	if safeContinue("https://evil.example/steal") != "" {
		t.Fatal("absolute url")
	}
	if safeContinue("//evil.example") != "" {
		t.Fatal("protocol relative")
	}
	if safeContinue("/logout") != "" {
		t.Fatal("non-authorize path")
	}
	if safeContinue("") != "" {
		t.Fatal("empty stays empty")
	}
}
