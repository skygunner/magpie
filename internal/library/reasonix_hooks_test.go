package library

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/tidwall/jsonc"
	"github.com/yetone/magpie/internal/agent"
)

func TestReasonixHookWireFormat(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "reasonix")
	sp, why := hookSpecOf(&Target{Agent: &agent.Agent{ID: "reasonix", Name: "Reasonix Studio", Dir: dir}})
	if sp == nil {
		t.Fatalf("missing hook spec: %s", why)
	}
	if sp.path != filepath.Join(dir, "settings.json") {
		t.Errorf("path %q", sp.path)
	}
	for _, event := range []string{"PreToolUse", "PostToolUse", "PostToolUseFailure", "PermissionRequest", "UserPromptSubmit", "Stop", "StopFailure", "PostLLMCall", "SessionStart", "SessionEnd", "SubagentStart", "SubagentStop", "Notification", "PreCompact"} {
		if !slices.Contains(sp.native, event) {
			t.Errorf("missing Reasonix event %s", event)
		}
	}
	h := &Hook{Event: "after-tool", Matcher: "shell", Command: "echo hi", Timeout: 4}
	event, err := sp.event("reasonix", h)
	if err != nil || event != "PostToolUse" {
		t.Fatalf("event %q err %v", event, err)
	}
	var data map[string]any
	if err := json.Unmarshal(sp.entry(h, event), &data); err != nil {
		t.Fatal(err)
	}
	if data["command"] != "echo hi" || data["match"] != "^bash$" || data["timeout"] != float64(4000) {
		t.Errorf("Reasonix direct hook fields: %+v", data)
	}
	if _, ok := data["matcher"]; ok {
		t.Errorf("Claude matcher leaked: %+v", data)
	}
	if _, ok := data["hooks"]; ok {
		t.Errorf("Claude nested hooks leaked: %+v", data)
	}
	if _, err := sp.event("reasonix", &Hook{Event: "reasonix:PreToolUse", Matcher: "["}); err == nil {
		t.Error("invalid anchored regex accepted")
	}
	if _, err := sp.event("reasonix", &Hook{Event: "reasonix:NotAnEvent"}); err == nil {
		t.Error("unknown Reasonix event accepted")
	}
}

func reasonixTestSync(t *testing.T, l *Library, dir string) *Result {
	t.Helper()
	res := &Result{}
	l.syncHooks(&Target{Agent: &agent.Agent{ID: "reasonix", Name: "Reasonix Studio", Dir: dir}}, newBackups(), res)
	return res
}

func TestReasonixHookSyncPreservesUser(t *testing.T) {
	h := sandbox(t)
	dir := filepath.Join(h, ".reasonix")
	cfg := filepath.Join(dir, "config.toml")
	file := filepath.Join(dir, "settings.json")
	write(t, cfg, "default_model = \"qwen\"\n")
	original := "{\n // own comment\n \"theme\":\"dark\",\n \"hooks\":{\n \"Stop\":[{\"command\":\"mine\"}],\n \"PostToolUse\":[{\"match\":\"read_file\",\"command\":\"my reader\"}]\n }\n}\n"
	write(t, file, original)
	l := &Library{Applied: map[string]*Applied{}, Hooks: []*Hook{
		{Name: "notify", Event: "turn-end", Command: "done", Timeout: 12, Agents: []string{"reasonix"}},
		{Name: "format", Event: "after-tool", Matcher: "edit", Command: "fmt", Timeout: 3, Agents: []string{"reasonix"}},
		{Name: "shell", Event: "before-tool", Matcher: "shell", Command: "check", Agents: []string{"reasonix"}},
		{Name: "reason", Event: "reasonix:PostLLMCall", Command: "reason-summary", Agents: []string{"reasonix"}},
	}}
	if res := reasonixTestSync(t, l, dir); len(res.Problems) != 0 {
		t.Fatalf("syncing Reasonix hooks: %+v", res.Problems)
	}
	stop := hooksOf(t, file, "Stop")
	if len(stop) != 2 || stop[0] != "{\"command\":\"mine\"}" || stop[1] != "{\"command\":\"done\",\"timeout\":12000}" {
		t.Errorf("Stop: %v", stop)
	}
	after := hooksOf(t, file, "PostToolUse")
	if len(after) != 2 || after[0] != "{\"command\":\"my reader\",\"match\":\"read_file\"}" || after[1] != "{\"command\":\"fmt\",\"match\":\"^(?:write_file|edit_file|multi_edit)$\",\"timeout\":3000}" {
		t.Errorf("PostToolUse: %v", after)
	}
	before := hooksOf(t, file, "PreToolUse")
	if len(before) != 1 || before[0] != "{\"command\":\"check\",\"match\":\"^bash$\"}" {
		t.Errorf("PreToolUse: %v", before)
	}
	native := hooksOf(t, file, "PostLLMCall")
	if len(native) != 1 || native[0] != "{\"command\":\"reason-summary\"}" {
		t.Errorf("PostLLMCall: %v", native)
	}
	body := read(t, file)
	if !strings.Contains(body, "// own comment") || gjson.GetBytes(jsonc.ToJSON([]byte(body)), "theme").String() != "dark" {
		t.Errorf("user's settings changed: %s", body)
	}
	if read(t, cfg) != "default_model = \"qwen\"\n" {
		t.Errorf("model config was changed: %s", read(t, cfg))
	}
	if !slices.Contains(hookAgents, "reasonix") || len(l.Applied["reasonix"].Hooks) != 4 {
		t.Errorf("Reasonix hook integration not registered: %+v", l.Applied["reasonix"])
	}
	l.Hooks = nil
	if res := reasonixTestSync(t, l, dir); len(res.Problems) != 0 {
		t.Fatalf("taking out Reasonix hooks: %+v", res.Problems)
	}
	if c := hooksOf(t, file, "Stop"); len(c) != 1 || c[0] != "{\"command\":\"mine\"}" {
		t.Errorf("user Stop changed: %v", c)
	}
	if c := hooksOf(t, file, "PostToolUse"); len(c) != 1 || c[0] != "{\"command\":\"my reader\",\"match\":\"read_file\"}" {
		t.Errorf("user PostToolUse changed: %v", c)
	}
	if gjson.Get(read(t, file), "hooks.PreToolUse").Exists() || gjson.Get(read(t, file), "hooks.PostLLMCall").Exists() {
		t.Errorf("magpie hooks remain: %s", read(t, file))
	}
	if canonical(jsonc.ToJSON([]byte(read(t, file)))) != canonical(jsonc.ToJSON([]byte(original))) {
		t.Errorf("user JSON changed: %s", read(t, file))
	}
}

func TestReasonixHookBadFileAndModifiedEntry(t *testing.T) {
	h := sandbox(t)
	dir := filepath.Join(h, ".reasonix")
	file := filepath.Join(dir, "settings.json")
	write(t, filepath.Join(dir, "config.toml"), "model = \"qwen\"\n")
	l := &Library{Applied: map[string]*Applied{}, Hooks: []*Hook{
		{Name: "gate", Event: "reasonix:PreToolUse", Matcher: "^read_file$", Command: "guard", Agents: []string{"reasonix"}},
	}}
	if res := reasonixTestSync(t, l, dir); len(res.Problems) != 0 {
		t.Fatalf("native hook: %+v", res.Problems)
	}
	if c := hooksOf(t, file, "PreToolUse"); len(c) != 1 || c[0] != "{\"command\":\"guard\",\"match\":\"^read_file$\"}" {
		t.Fatalf("own matcher: %v", c)
	}
	write(t, file, strings.Replace(read(t, file), "guard", "edited by user", 1))
	l.Hooks = nil
	if res := reasonixTestSync(t, l, dir); len(res.Problems) != 0 {
		t.Fatalf("modified hook: %+v", res.Problems)
	}
	if c := hooksOf(t, file, "PreToolUse"); len(c) != 1 || !strings.Contains(c[0], "edited by user") {
		t.Errorf("user-edited hook deleted: %v", c)
	}
	bad := "{\"hooks\":{\"Stop\":[not valid JSON]}"
	write(t, file, bad)
	l.Hooks = []*Hook{{Name: "later", Event: "turn-end", Command: "notify", Agents: []string{"reasonix"}}}
	res := reasonixTestSync(t, l, dir)
	if read(t, file) != bad {
		t.Errorf("malformed file overwritten: %s", read(t, file))
	}
	if !slices.ContainsFunc(res.Problems, func(p Problem) bool { return p.Agent == "reasonix" && p.What == "hook:later" }) {
		t.Errorf("unreadable file not reported: %+v", res.Problems)
	}
	if err := (&Hook{Name: "wrong", Event: "reasonix:PostLLMCall", Command: "bad", Agents: []string{"claude"}}).check(); err == nil {
		t.Error("Reasonix native event offered to another agent")
	}
}

func TestReasonixSharedSkillAccounting(t *testing.T) {
	h := sandbox(t)
	own := filepath.Join(h, ".reasonix", "skills")
	shared := filepath.Join(h, ".agents", "skills")
	t1 := &Target{Agent: &agent.Agent{ID: "reasonix", Name: "Reasonix"}, Skills: own}
	t2 := &Target{Agent: &agent.Agent{ID: "goose", Name: "Goose"}, Skills: shared}
	readsWith([]*Target{t1, t2})
	if !slices.Contains(readsShared, "reasonix") {
		t.Error("Reasonix does not count ~/.agents/skills")
	}
	if !slices.Contains(t1.SkillsAlso, "goose") {
		t.Errorf("shared skills invisible to Reasonix: %v", t1.SkillsAlso)
	}
}
