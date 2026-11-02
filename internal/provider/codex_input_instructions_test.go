package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Newer Codex sends the model's base instructions as a leading developer
// message in input (marked in the passthrough metadata it sends beside
// them), not only as the top-level instructions of older versions
// (#4953). codexBody kept adding the whole template as instructions over
// them, so the backend read the base prompt twice — and a second magpie in
// front of the same backend added it once more.
func TestCodexBodyKeepsInputBaseInstructions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", dir)
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	os.WriteFile(filepath.Join(dir, "models_cache.json"), []byte(`{"models":[{"slug":"gpt-5.5","model_messages":{"instructions_template":"You are Codex, a coding agent.\nMore."}}]}`), 0o644)

	message := func(role, text string) map[string]any {
		return map[string]any{"type": "message", "role": role,
			"content": []any{map[string]any{"type": "input_text", "text": text}}}
	}
	base := "You are Codex, a coding agent.\nMore."
	user := message("user", "hello")
	marked := message("developer", "A different Codex writes this.")
	marked["internal_chat_message_metadata_passthrough"] = map[string]any{"content_item_kinds": []string{"model.base_instructions"}}
	for _, c := range []struct {
		name, own, wantInstructions string
		input                       []any
		wantFirst                   string
	}{
		{"the base in input", "", "", []any{message("developer", base), user}, base},
		{"after Lite's additional_tools", "", "", []any{map[string]any{"type": "additional_tools", "role": "developer", "tools": []any{}}, message("developer", base), user}, ""},
		{"another Codex's template, marked", "", "", []any{marked, user}, "A different Codex writes this."},
		{"the client's own kept", "You are Pi.", "", []any{message("developer", base), user}, "You are Pi."},
		{"the client's own the base whole", base, "", []any{message("developer", base), user}, base},
		{"an older client's instructions", "You are Pi.", base, []any{user}, "You are Pi."},
		{"no base in input", "", base, []any{user}, "hello"},
		{"the base after the user message", "", base, []any{user, message("developer", base)}, "hello"},
	} {
		t.Run(c.name, func(t *testing.T) {
			body, err := json.Marshal(map[string]any{"model": "gpt-5.5", "instructions": c.own, "input": c.input})
			if err != nil {
				t.Fatal(err)
			}
			var got struct {
				Instructions string           `json:"instructions"`
				Input        []map[string]any `json:"input"`
			}
			if err := json.Unmarshal(codexBody(body), &got); err != nil {
				t.Fatal(err)
			}
			if got.Instructions != c.wantInstructions {
				t.Fatalf("instructions = %q, want %q", got.Instructions, c.wantInstructions)
			}
			if c.wantFirst != "" {
				content, ok := got.Input[0]["content"].([]any)
				if !ok || len(content) != 1 || content[0].(map[string]any)["text"] != c.wantFirst {
					t.Fatalf("first input = %#v, want text %q", got.Input[0], c.wantFirst)
				}
			}
			if c.name == "the client's own the base whole" && len(got.Input) != 2 {
				t.Fatalf("the base said twice: %#v", got.Input)
			}
		})
	}
}
