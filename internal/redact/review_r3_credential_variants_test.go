package redact

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// The existing assignment rule accepts these credential names. A spelling
// without an underscore, acronym camel case, or a deployment/version suffix
// must not turn a secret into a repository reference.
func TestReviewR3CredentialNamesKeepTheirMeaning(t *testing.T) {
	for i, field := range []string{"apikey", "APIKey", "api_key_v2", "password_primary"} {
		for j, mode := range []string{"Mask", "MaskJSON-message"} {
			t.Run(field+"/"+mode, func(t *testing.T) {
				value := fmt.Sprintf("Rr%d%d8c7/bT2mN4z", i, j)
				assignment := fmt.Sprintf(`{"%s":"%s"}`, field, value)
				var out string
				var n int
				if mode == "Mask" {
					out, n = Mask(assignment, Options{Secrets: true})
				} else {
					body, err := json.Marshal(map[string]any{"messages": []any{map[string]any{"role": "user", "content": assignment}}})
					if err != nil {
						t.Fatal(err)
					}
					masked, count := MaskJSON(body, Options{Secrets: true})
					out, n = string(masked), count
				}
				if n == 0 || strings.Contains(out, value) {
					t.Errorf("explicit %s credential escaped masking: count=%d output=%s", field, n, out)
				}
			})
		}
	}
}
