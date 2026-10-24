package redact

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// These fields reach the existing assignment rule through "token" and
// used the former strongCredential key kinds (signing/encryption).
// Credential-name normalization must preserve the protection they had.
func TestReviewR4TokenKeyCompoundsStaySecret(t *testing.T) {
	for i, field := range []string{"token_signing_key", "token_encryption_key"} {
		for j, mode := range []string{"Mask", "MaskJSON-message", "MaskJSON-tool-file-content"} {
			t.Run(field+"/"+mode, func(t *testing.T) {
				// Unique values prevent a prior case's learning from answering
				// for this case. No live credentials or vendor calls are used.
				value := fmt.Sprintf("Rs%d%d8c7/bT2mN4z", i, j)
				assignment := fmt.Sprintf(`{"%s":"%s"}`, field, value)
				var out string
				var n int
				switch mode {
				case "Mask":
					out, n = Mask(assignment, Options{Secrets: true})
				case "MaskJSON-message":
					body, err := json.Marshal(map[string]any{"messages": []any{map[string]any{"role": "user", "content": assignment}}})
					if err != nil {
						t.Fatal(err)
					}
					masked, count := MaskJSON(body, Options{Secrets: true})
					out, n = string(masked), count
				case "MaskJSON-tool-file-content":
					args, err := json.Marshal(map[string]string{"path": "config.json", "content": assignment})
					if err != nil {
						t.Fatal(err)
					}
					body, err := json.Marshal(map[string]any{"input": []any{map[string]any{"type": "function_call", "name": "write_file", "call_id": "review_r4_key_kinds", "arguments": string(args)}}})
					if err != nil {
						t.Fatal(err)
					}
					masked, count := MaskJSON(body, Options{Secrets: true})
					out, n = string(masked), count
					if !json.Valid(masked) {
						t.Fatalf("outer body is not valid JSON: %s", masked)
					}
				}
				if n == 0 || strings.Contains(out, value) {
					t.Errorf("credential %s escaped masking: count=%d output=%s", field, n, out)
				}
			})
		}
	}
}
