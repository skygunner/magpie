package redact

import (
	"strings"
	"testing"
)

// A value first met as a placeholder URL's password (masked, not learned)
// becomes a real known credential the first time it is met in a context
// that says so: the same value under a user of its own is learned, and
// from then on it is masked in plain words too (#3365 review).
func TestPlaceholderPasswordLearnsWhenReal(t *testing.T) {
	resetSent()
	forgetKnown(t)
	o := Options{Secrets: true}
	const v = "gZ4mKq2wXx8b"
	out, n := Mask(`postgres://`+v+`:`+v+`@localhost/db`, o)
	if n != 1 || !strings.Contains(out, "PASSWORD_") {
		t.Fatalf("the placeholder password was not masked: %q (n=%d)", out, n)
	}
	// fresh words, so the sent-text cache cannot answer for it
	out, n = Mask(`postgres://realuser:`+v+`@localhost/db`, o)
	if n != 1 || strings.Contains(out, v) {
		t.Fatalf("the real credential was not masked: %q", out)
	}
	out, n = Mask("the service wrote "+v+" into its log", o)
	if n != 1 || strings.Contains(out, v) {
		t.Fatalf("a learned value was not masked in plain words: %q", out)
	}
}

// Learning eligibility survives the known rebuild: a value learned stays
// matched after reindexKnown; one only ever masked as a placeholder's
// password does not start matching after it (#3365 review).
func TestReindexKeepsLearningState(t *testing.T) {
	resetSent()
	forgetKnown(t)
	o := Options{Secrets: true}
	const real, plain = "tR9wLp5vNn3d", "qW6yHk2xBxc7"
	if out, n := Mask(`postgres://`+plain+`:`+plain+`@localhost/db`, o); n != 1 || !strings.Contains(out, "PASSWORD_") {
		t.Fatalf("the placeholder password was not masked: %q (n=%d)", out, n)
	}
	if out, n := Mask(`DB_PASSWORD=`+real, o); n != 1 || strings.Contains(out, real) {
		t.Fatalf("the real credential was not masked: %q", out)
	}
	mu.Lock()
	reindexKnown()
	mu.Unlock()
	// fresh text for both, the cache reset above notwithstanding
	if out, n := Mask("notes mention "+plain+" twice", o); n != 0 || strings.Contains(out, "PASSWORD") {
		t.Fatalf("a placeholder-only value matched after the rebuild: %q (n=%d)", out, n)
	}
	if out, n := Mask("notes mention "+real+" twice", o); n != 1 || strings.Contains(out, real) {
		t.Fatalf("a learned value was lost by the rebuild: %q (n=%d)", out, n)
	}
}

// forgetKnown starts the test with no values known, as ids_test's
// forgetIDs does: no other test's learned value can answer for these.
func forgetKnown(t *testing.T) {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	values = map[string]string{}
	learned = map[string]bool{}
	reindexKnown()
}
