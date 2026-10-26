package sessions

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
)

// Codex reads a rollout by the byte offsets it recorded: a fork's
// history_base, and each turn's start and end in thread_history_1.sqlite.
// Rewriting a session_meta with a provider id of another length moved every
// offset after it, and a fork then failed to resume: "cutoff byte offset is
// past the source rollout" for a shorter one, offsets into the middle of
// lines for a longer one (#3751). So a rewritten line keeps its length, and
// a longer id isn't written at all.
func TestSetMetaProviderKeepsByteOffsets(t *testing.T) {
	meta := func(provider string) string {
		return `{"timestamp":"2026-09-20T15:48:28.100Z","type":"session_meta","payload":{"id":"` + mgCX + `","model_provider":"` + provider + `","cwd":"/work"}}` + "\n"
	}
	other := `{"timestamp":"2026-09-20T15:48:29.000Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"hi"}]}}` + "\n"
	for _, c := range []struct {
		name    string
		from    string
		to      string
		written bool
	}{
		{"shorter by 9", "fake-provider", "fake", true},
		{"same length", "provider", "magpie-x", true},
		{"longer by 9", "fake", "fake-provider", false},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "rollout-2026-09-20T15-48-28-"+mgCX+".jsonl")
			before := meta(c.from) + other
			if err := os.WriteFile(path, []byte(before), 0o644); err != nil {
				t.Fatal(err)
			}
			written, err := rewriteCodexProvider(path, c.to)
			if err != nil {
				t.Fatal(err)
			}
			if written != c.written {
				t.Fatalf("written %v, want %v", written, c.written)
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !c.written {
				if string(after) != before {
					t.Fatalf("a file left as it was became\n%s", after)
				}
				return
			}
			if len(after) != len(before) {
				t.Fatalf("the file is %d bytes, was %d", len(after), len(before))
			}
			firstLine := bytes.IndexByte([]byte(before), '\n') + 1
			if !bytes.Equal(after[:firstLine], append([]byte(nil), after[:firstLine]...)) ||
				!bytes.HasPrefix(after[firstLine:], []byte(other)) {
				t.Fatalf("the second line doesn't start at byte %d:\n%s", firstLine, after)
			}
			// the line is still JSON, its provider the one asked for
			line := strings.SplitN(string(after), "\n", 2)[0]
			if p := gjson.Get(line, "payload.model_provider").String(); p != c.to {
				t.Fatalf("provider %q, want %q", p, c.to)
			}
			var v map[string]any
			if err := json.Unmarshal([]byte(line), &v); err != nil {
				t.Fatalf("the padded line isn't JSON: %v\n%s", err, line)
			}
		})
	}
}
