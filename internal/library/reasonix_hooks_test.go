package library

import (
	"encoding/json"
	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/agent"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Reasonix uses flat hooks with a match regex and millisecond timeout.
// User hooks, settings and changed library entries remain untouched.
func TestReasonixHooksOwnShapeAndPreservation(t *testing.T) {
	h := sandbox(t)
	path := filepath.Join(h, ".reasonix", "settings.json")
	write(t, path, "{\n // mine\n \"theme\":\"dark\", \"hooks\":{\"Stop\":[{\"command\":\"say mine\"}]}\n}\n")
	target := &Target{Agent: &agent.Agent{ID: "reasonix", Name: "Reasonix Studio", Dir: filepath.Dir(path), Path: filepath.Join(h, ".reasonix", "config.toml")}}
	lib := &Library{Applied: map[string]*Applied{}, Hooks: []*Hook{
		{Name: "notify", Event: "turn-end", Command: "notify", Timeout: 4, Agents: []string{"reasonix"}},
		{Name: "format", Event: "after-tool", Matcher: "edit", Command: "fmt", Timeout: 2, Agents: []string{"reasonix"}},
		{Name: "own", Event: "reasonix:PostLLMCall", Command: "after", Agents: []string{"reasonix"}},
	}}
	result := &Result{}
	lib.syncHooks(target, newBackups(), result)
	if len(result.Problems) != 0 || !slices.Contains(result.Changed, "reasonix") {
		t.Fatalf("sync: %+v", result)
	}
	var entries []map[string]any
	if err := json.Unmarshal([]byte(gjson.Get(read(t, path), "hooks.PostToolUse").Raw), &entries); err != nil || len(entries) != 1 {
		t.Fatalf("post hook: %v %+v", err, entries)
	}
	e := entries[0]
	if e["command"] != "fmt" || e["timeout"] != float64(2000) || e["matcher"] != nil || e["match"] != "^(edit_file|write_file|multi_edit)$" || len(read(t, path)) == 0 {
		t.Fatalf("bad entry: %+v", e)
	}
	if got := gjson.Get(read(t, path), "hooks.Stop").Array(); len(got) != 2 || !strings.Contains(got[0].Raw, "say mine") || !strings.Contains(got[1].Raw, "notify") {
		t.Fatalf("Stop: %+v", got)
	}
	if !strings.Contains(read(t, path), "// mine") || !strings.Contains(read(t, path), "dark") {
		t.Fatal("user settings changed")
	}
	if len(gjson.Get(read(t, path), "hooks.PostLLMCall").Array()) != 1 {
		t.Fatal("native event not added")
	}
	lib.Hooks = nil
	result = &Result{}
	lib.syncHooks(target, newBackups(), result)
	if len(result.Problems) != 0 {
		t.Fatalf("remove: %+v", result)
	}
	if got := gjson.Get(read(t, path), "hooks.Stop").Array(); len(got) != 1 || !strings.Contains(got[0].Raw, "say mine") {
		t.Fatalf("user Stop deleted: %+v", got)
	}
	if gjson.Get(read(t, path), "hooks.PostToolUse").Exists() || gjson.Get(read(t, path), "hooks.PostLLMCall").Exists() {
		t.Fatal("managed entries not removed")
	}
}

func TestReasonixSharedSkillsAccounting(t *testing.T) {
	h := sandbox(t)
	a := &Target{Agent: &agent.Agent{ID: "reasonix", Name: "Reasonix"}, Skills: filepath.Join(h, ".reasonix", "skills")}
	b := &Target{Agent: &agent.Agent{ID: "goose", Name: "Goose"}, Skills: filepath.Join(h, ".agents", "skills")}
	readsWith([]*Target{a, b})
	if !slices.Contains(a.SkillsAlso, "goose") {
		t.Fatalf("shared skills not accounted: %+v", a.SkillsAlso)
	}
}
