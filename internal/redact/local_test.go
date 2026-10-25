package redact

import (
	"strings"
	"testing"
)

// A mDNS host is not an address: user@host.local was masked and counted as
// an email with Mask personal data on (#3541). A name on a real domain
// beside it is still masked.
func TestLocalHostIsNoEmail(t *testing.T) {
	o := Options{Secrets: true, Personal: true}
	for _, s := range []string{
		"ssh user@host.local",
		"curl http://pi@nas.local:8080/status",
		"open vnc://me@macbook.local",
	} {
		if out, n := Mask(s, o); n != 0 || out != s {
			t.Errorf("%q masked: %q", s, out)
		}
	}
	s := "reach user@host.local or mail rina.k@fastmail.com"
	out, n := Mask(s, o)
	if n != 1 || !strings.Contains(out, "user@host.local") || strings.Contains(out, "rina.k@fastmail.com") {
		t.Fatalf("%d masked: %q", n, out)
	}
}
