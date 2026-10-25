package redact

import (
	"strings"
	"testing"
)

// A password said in words in a message is masked, a short one too: "my
// password: abc12" went to the vendor as written (#3574). Code that names a
// password (password: String, password: str, password = getPassword()) and
// a sentence about one (the password is required) stay as they are.
func TestProsePasswordMasked(t *testing.T) {
	cases := []struct {
		in, gone string
		kept     []string
	}{
		{"my password: abc12", "abc12", []string{"my password: "}},
		{"my password is abc12", "abc12", []string{"my password is "}},
		{"My password is hunter2.", "hunter2", []string{"My password is ", "."}},
		{"Your password: q8w7", "q8w7", []string{"Your password: "}},
		{"the password is 1234", "1234", []string{"the password is "}},
		{"our wifi password is abc12!", "abc12", []string{"our wifi password is "}},
		{"password is abc12", "abc12", []string{"password is "}},
		{`{"text":"my password: abc12"}`, "abc12", []string{`"text":"my password: `}},
		// a long one is masked as it was, by the rule that has it
		{"the password is Tr0ub4dor&3xq, do not share", "Tr0ub4dor&3xq", []string{"the password is ", ", do not share"}},
	}
	for _, c := range cases {
		out, n := Mask(c.in, Options{Secrets: true})
		if n != 1 || strings.Contains(unmarked(out), c.gone) {
			t.Errorf("%q: %d masked → %q", c.in, n, out)
		}
		for _, k := range c.kept {
			if !strings.Contains(out, k) {
				t.Errorf("%q: %q gone from %q", c.in, k, out)
			}
		}
		if back := Restore(out, false); back != c.in {
			t.Errorf("round trip: %q → %q → %q", c.in, out, back)
		}
	}
	// in a request body, as the gateway masks one
	body := `{"messages":[{"role":"user","content":"my password: abc12"}]}`
	if out, n := MaskJSON([]byte(body), Options{Secrets: true}); n != 1 || strings.Contains(string(out), "abc12") || string(RestoreJSON(out)) != body {
		t.Errorf("body: %d %s", n, out)
	}
	// what isn't a password: code naming one, a sentence about one, a
	// placeholder, too short, no value at all
	for _, in := range []string{
		"password: String", "password: str", `{"password": "abc12"}`, `"password": "String"`,
		"password = getPassword()", "const password = sha256Hex (input)",
		"the password is required", "the password is required to be 8 characters", "password is wrong",
		"my password is expired", "your password was changed", "Password is case-sensitive",
		"Reset your password", "my password: ${DB_PASS}", "my password: ****", "password: abc",
		"my password is set in Settings", "the password is stored hashed",
	} {
		if out, n := Mask(in, Options{Secrets: true}); n != 0 {
			t.Errorf("%q masked: %q", in, out)
		}
	}
}
