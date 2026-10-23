package redact

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// These synthetic credentials differ by case so an earlier case's learning
// cannot make a later one pass. The same probe also compiles on 0815c918.
func TestReviewV2CredentialAssignments(t *testing.T) {
	fields := []string{"password", "api_key", "api-key", "x-api-key", "access-key", "private-key", "access_token", "refresh_token"}
	modes := []string{"Mask", "MaskJSON-message", "MaskJSON-tool-file-content"}
	for fi, field := range fields {
		for mi, mode := range modes {
			t.Run(field+"/"+mode, func(t *testing.T) {
				value := fmt.Sprintf("Rq%d%d8c7/bT2mN4z", fi, mi)
				assignment := fmt.Sprintf(`{"%s":"%s"}`, field, value)
				var out string
				var n int
				switch mode {
				case "Mask":
					out, n = Mask(assignment, Options{Secrets: true})
				case "MaskJSON-message":
					body, _ := json.Marshal(map[string]any{"messages": []any{map[string]any{"role": "user", "content": assignment}}})
					masked, count := MaskJSON(body, Options{Secrets: true})
					out, n = string(masked), count
				case "MaskJSON-tool-file-content":
					args, _ := json.Marshal(map[string]string{"path": "config.json", "content": assignment})
					body, _ := json.Marshal(map[string]any{"input": []any{map[string]any{"type": "function_call", "name": "write_file", "call_id": "call_review", "arguments": string(args)}}})
					masked, count := MaskJSON(body, Options{Secrets: true})
					out, n = string(masked), count
					if !json.Valid(masked) {
						t.Fatalf("outer body is not JSON: %s", masked)
					}
				}
				if n == 0 || strings.Contains(out, value) {
					t.Errorf("explicit %s credential escaped masking: count=%d output=%s", field, n, out)
				}
			})
		}
	}
}
