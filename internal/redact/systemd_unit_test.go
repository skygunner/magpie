package redact

import (
	"strings"
	"testing"
)

// A systemd unit, name@instance.service as a path holds it under
// ~/.config/systemd, is no address: .service is a domain that is one, so
// the rule read the unit as an address and masked it, twice over in a
// default.target.wants symlink (#3629, 321sssrt-bit). An address in
// prose, even of a .service domain, still is one.
func TestSystemdUnitIsNoEmail(t *testing.T) {
	for _, s := range []string{
		"ln -s /home/u/.config/systemd/user/demo@web.service /home/u/.config/systemd/user/default.target.wants/demo@web.service",
		"/etc/systemd/system/demo@web.service.d/override.conf",
		`{"path": "/home/u/.config/systemd/user/demo@web.service"}`,
		"cat /home/u/.config/systemd/user/demo@web.service",
	} {
		if out, n := Mask(s, Options{Personal: true}); n != 0 || strings.Contains(out, "{{") {
			t.Errorf("%q: masked %q", s, out)
		}
	}
	for _, s := range []string{
		"write to rina@kowalcky.service",
		"see /home/u/notes: rina@kowalczyk.pl",
	} {
		if out, n := Mask(s, Options{Personal: true}); n != 1 || strings.Contains(out, "rina@") {
			t.Errorf("%q: went as written %q", s, out)
		}
	}
	// the byte counters the same request held are no phones either: a
	// bare JSON number is none since fc91e58b
	for _, s := range []string{
		`{"swap_free_bytes": 10000000000}`,
		`{"swap_free_bytes": 13800138000}`,
	} {
		if out, n := Mask(s, Options{Personal: true}); n != 0 {
			t.Errorf("%q: masked %q", s, out)
		}
	}
}
