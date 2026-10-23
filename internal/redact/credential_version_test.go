package redact

import (
	"strings"
	"testing"
)

// A credential field's value is never let through for looking like a
// version or a repository's path: the #3365 exceptions are for identifier
// fields, and a field named password or api_key holds secrets by name
// (#3365 review: the delivered tree answered {"password":"R8c7/bT2mN4z"}
// with the value as written, count 0).
func TestCredentialFieldsKeepVersionShapedSecrets(t *testing.T) {
	o := Options{Secrets: true}
	for _, in := range []string{
		`{"password":"R8c7/bT2mN4z"}`,
		`{"api_key":"Q8c7/bT2mN4zR6f1"}`,
		`{"password":"ProdA1.2.3Z9"}`,
	} {
		if out, n := Mask(in, o); n != 1 || strings.Contains(out, "R8c7") || strings.Contains(out, "Q8c7") || strings.Contains(out, "ProdA") {
			t.Fatalf("%s went out as written: %q (n=%d)", in, out, n)
		}
	}
}
