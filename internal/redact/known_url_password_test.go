package redact

import (
	"strings"
	"testing"
)

// A URL's password that is also its user (postgres://paperclip:paperclip@…)
// is a public placeholder's shape: masked where it stands, but its value is
// not learned, since its word is a name in plain words too — "Open the
// paperclip repository." went to the model as "Open the {{PASSWORD_…}}
// repository." in every request after (#3365, Ethereal49).
func TestURLPasswordEqualToItsUserIsNotLearned(t *testing.T) {
	o := Options{Secrets: true}
	out, n := Mask("postgres://paperclip:paperclip@localhost:5432/paperclip", o)
	if n != 1 || !strings.Contains(out, "PASSWORD_") {
		t.Fatalf("the placeholder's password was not masked: %q", out)
	}
	out, n = Mask("Open the paperclip repository.", o)
	if n != 0 || strings.Contains(out, "PASSWORD") {
		t.Fatalf("the project's name was swapped in plain words: %q", out)
	}
	// a password of its own is learned as before: the vendor's reply that
	// names it in words is masked again
	secret := "correct-horse-batteries"
	if out, n = Mask("postgres://paperclip:"+secret+"@localhost:5432/paperclip", o); n != 1 || !strings.Contains(out, "PASSWORD_") {
		t.Fatalf("a real password was not masked: %q", out)
	}
	if out, n = Mask("the password is "+secret+" today", o); n != 1 || !strings.Contains(out, "PASSWORD_") {
		t.Fatalf("a real password was not known: %q", out)
	}
}

// A version, build or image tag is not a secret, whatever field holds it:
// fields named token and key hold them all the time (#3365).
func TestVersionIsNotASecret(t *testing.T) {
	o := Options{Secrets: true}
	for _, v := range []string{
		`{"versionToken":"2.9.0+sha.a7177d7bb704"}`,
		"tokenspeed-triton: 3.9.10.post20261008",
		"tokenspeed-runner: cu130-torch-2.14.0-flashinfer-0.6.18",
	} {
		if out, n := Mask(v, o); n != 0 || strings.Contains(out, "SECRET") {
			t.Fatalf("%q masked: %q", v, out)
		}
	}
	// a key's own value still is
	if out, n := Mask(`{"key":"oc_sk_9f2m4xq7a1l0"}`, o); n != 1 || !strings.Contains(out, "SECRET_") {
		t.Fatalf("a key's value was not masked: %q", out)
	}
}

// A repository's path or reference is not a secret either:
// owner/name#123 and owner/name under a field named key went out as
// placeholders (#3365).
func TestRepoRefIsNotASecret(t *testing.T) {
	o := Options{Secrets: true}
	for _, v := range []string{
		`{"key":"example-org/demo-project#1234"}`,
		`{"key":"example1234/demo-repository"}`,
	} {
		if out, n := Mask(v, o); n != 0 || strings.Contains(out, "SECRET") {
			t.Fatalf("%q masked: %q", v, out)
		}
	}
}
