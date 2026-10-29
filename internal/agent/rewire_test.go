package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/proc"
	"github.com/yetone/magpie/internal/settings"
)

// setPort is Settings' gateway port set to p, as the Settings page saves it.
func setPort(t *testing.T, p int) {
	t.Helper()
	s := settings.Load()
	s.Port = p
	if err := settings.Save(s); err != nil {
		t.Fatal(err)
	}
}

// filesWith are the files under home holding s.
func filesWith(home, s string) []string {
	var out []string
	filepath.WalkDir(home, func(p string, e os.DirEntry, err error) error {
		if err != nil || e.IsDir() {
			return nil
		}
		if b, _ := os.ReadFile(p); strings.Contains(string(b), s) {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func TestRewireReleasesStashAfterPanic(t *testing.T) {
	stashHome(t)
	a := &Agent{Name: "rewire-test", Join: func() (bool, error) {
		panic("rewire failed")
	}, Joined: func() bool { return true }}
	func() {
		defer func() {
			if recover() == nil {
				t.Fatal("rewire did not panic")
			}
		}()
		setAgain([]*Agent{a})
	}()
	if err := stash(map[string]string{"other.value": "saved"}); err != nil {
		t.Fatalf("stash remains blocked after panic: %v", err)
	}
	if !rewireMu.TryLock() {
		t.Fatal("rewire remains locked after panic")
	}
	rewireMu.Unlock()
}

func TestRewireRejectsStashReplacement(t *testing.T) {
	stashHome(t)
	if err := stash(map[string]string{"agent.original": "original"}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{Name: "rewire-test", Join: func() (bool, error) { return true, nil },
		joinRewire: func(g *rewireGuard) (bool, error) {
			return true, g.stash(map[string]string{"agent.original": "old gateway"})
		}, Joined: func() bool { return true }}
	moved, errs := setAgain([]*Agent{a})
	if len(errs) == 0 || len(moved) != 0 {
		t.Fatalf("unsafe replacement accepted: moved=%v errors=%v", moved, errs)
	}
	if got := stashLoad()["agent.original"]; got != "original" {
		t.Fatalf("original lost: %q", got)
	}
}

// A restart status lookup has its own stash write while rewire is active.
func TestRewireAllowsConcurrentChangedAt(t *testing.T) {
	stashHome(t)
	path := filepath.Join(t.TempDir(), "changed-at")
	if err := os.WriteFile(path, []byte("config"), 0o600); err != nil {
		t.Fatal(err)
	}
	other := &Agent{ID: "other"}
	a := &Agent{Name: "rewire-test", Join: func() (bool, error) {
		result := make(chan time.Time, 1)
		go func() { result <- other.changedAt([]string{path}, func() string { return "config" }) }()
		select {
		case at := <-result:
			if at.IsZero() {
				return false, fmt.Errorf("changed time missing")
			}
			return true, nil
		case <-time.After(5 * time.Second):
			return false, fmt.Errorf("changed time blocked")
		}
	}, Joined: func() bool { return true }}
	_, errs := setAgain([]*Agent{a})
	if len(errs) != 0 || stashLoad()["other.started"] == "" {
		t.Fatalf("independent status write failed: %v; stash %v", errs, stashLoad())
	}
}

func TestRewireAllowsUnrelatedSameProcessStash(t *testing.T) {
	stashHome(t)
	a := &Agent{ID: "rewire-test", Name: "rewire-test", Join: func() (bool, error) { return true, nil }, joinRewire: func(g *rewireGuard) (bool, error) {
		result := make(chan error, 1)
		go func() { result <- stash(map[string]string{"other.value": "saved"}) }()
		select {
		case err := <-result:
			return true, err
		case <-time.After(5 * time.Second):
			return false, fmt.Errorf("unrelated stash blocked")
		}
	}, Joined: func() bool { return true }}
	_, errs := setAgain([]*Agent{a})
	if len(errs) != 0 || stashLoad()["other.value"] != "saved" {
		t.Fatalf("unrelated stash failed: %v; stash %v", errs, stashLoad())
	}
}

// A concurrent address update must never be replaced by a stale rewire.
func TestRewireRejectsConcurrentWiredAt(t *testing.T) {
	if mode := os.Getenv("MAGPIE_REWIRE_CHILD"); mode == "wired" || mode == "wired-delete" {
		value := "concurrent"
		if mode == "wired-delete" {
			value = ""
		}
		if err := stash(map[string]string{"agent.wired_at": value}); err != nil {
			t.Fatal(err)
		}
		return
	}
	for _, tc := range []struct{ mode, initial, want string }{
		{"wired", "old", "concurrent"},
		{"wired", "", "concurrent"},
		{"wired-delete", "old", ""},
	} {
		t.Run(tc.mode+"/"+tc.initial, func(t *testing.T) {
			stashHome(t)
			if err := stash(map[string]string{"agent.wired_at": tc.initial}); err != nil {
				t.Fatal(err)
			}
			exe, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			a := &Agent{Name: "rewire-test", Join: func() (bool, error) { return true, nil }, joinRewire: func(g *rewireGuard) (bool, error) {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := proc.CommandContext(ctx, exe, "-test.run=^TestRewireRejectsConcurrentWiredAt$")
				cmd.Env = append(os.Environ(), "MAGPIE_REWIRE_CHILD="+tc.mode)
				if out, err := cmd.CombinedOutput(); err != nil {
					return false, fmt.Errorf("child: %w: %s", err, out)
				}
				return true, g.stash(map[string]string{"agent.wired_at": "rewired"})
			}, Joined: func() bool { return true }}
			moved, errs := setAgain([]*Agent{a})
			if len(errs) == 0 || len(moved) != 0 || stashLoad()["agent.wired_at"] != tc.want {
				t.Fatalf("concurrent address lost: moved=%v errors=%v stash=%v", moved, errs, stashLoad())
			}
		})
	}
}

func TestRewirePreservesConcurrentStashChanges(t *testing.T) {
	if os.Getenv("MAGPIE_REWIRE_CHILD") == "1" {
		if err := stash(map[string]string{"other.original": "concurrent", "other.added": "new"}); err != nil {
			t.Fatal(err)
		}
		return
	}
	stashHome(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if err := stash(map[string]string{"other.original": "before"}); err != nil {
		t.Fatal(err)
	}
	a := &Agent{ID: "rewire-test", Name: "rewire-test", Join: func() (bool, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := proc.CommandContext(ctx, exe, "-test.run=^TestRewirePreservesConcurrentStashChanges$")
		cmd.Env = append(os.Environ(), "MAGPIE_REWIRE_CHILD=1")
		if out, err := cmd.CombinedOutput(); err != nil {
			return false, fmt.Errorf("child: %w: %s", err, out)
		}
		return true, nil
	}, Joined: func() bool { return true }}
	_, errs := setAgain([]*Agent{a})
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	got := stashLoad()
	if got["other.original"] != "concurrent" || got["other.added"] != "new" {
		t.Fatalf("rewire discarded another update: %v", got)
	}
}

// Pi keeps the saved native defaults while another process updates the stash.
func TestRewirePiConcurrentStashAndRestore(t *testing.T) {
	if os.Getenv("MAGPIE_REWIRE_CHILD") == "pi" {
		if err := stash(map[string]string{"other.original": "concurrent"}); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Setenv("MAGPIE_ADDR", "")
	home, _ := codexHome(t, "", "")
	dir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(`{"defaultProvider":"native","defaultModel":"original"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3591)
	a := pi(home)
	if _, err := a.ConnectHow(); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	field := a.Field("model")
	set := field.setRewire
	field.setRewire = func(g *rewireGuard, v string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := proc.CommandContext(ctx, exe, "-test.run=^TestRewirePiConcurrentStashAndRestore$")
		cmd.Env = append(os.Environ(), "MAGPIE_REWIRE_CHILD=pi")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("child: %w: %s", err, out)
		}
		return set(g, v)
	}
	moved, errs := setAgain([]*Agent{a})
	url, _ := edit.GetJSON(filepath.Join(dir, "models.json"), "providers.magpie.baseUrl")
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.original"] != "concurrent" || url != "http://127.0.0.1:3592/v1" {
		t.Fatalf("rewire: moved=%v errors=%v URL=%q", moved, errs, url)
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	p, _ := edit.GetJSON(path, "defaultProvider")
	m, _ := edit.GetJSON(path, "defaultModel")
	if p != "native" || m != "original" {
		t.Fatalf("original defaults lost: %q %q", p, m)
	}
}

// OpenCode-shaped agents rewire while another process saves unrelated state.
func TestRewireOpenCodeLikeConcurrentStash(t *testing.T) {
	if os.Getenv("MAGPIE_REWIRE_CHILD") == "open-code" {
		if err := stash(map[string]string{"other.original": "concurrent"}); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Setenv("MAGPIE_ADDR", "")
	home, _ := codexHome(t, "", "")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"opencode", "mimocode", "deveco"} {
		t.Run(id, func(t *testing.T) {
			setPort(t, 3591)
			dir := filepath.Join(home, ".config", id)
			a := openCodeLike(here(home), id, id, id, id, dir, filepath.Join(home, "auth.json"), nil)
			if err := a.Apply("model", "magpie/fake/m1"); err != nil {
				t.Fatal(err)
			}
			setPort(t, 3592)
			field := a.Field("model")
			set := field.Set
			field.Set = func(v string) error {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				cmd := proc.CommandContext(ctx, exe, "-test.run=^TestRewireOpenCodeLikeConcurrentStash$")
				cmd.Env = append(os.Environ(), "MAGPIE_REWIRE_CHILD=open-code")
				if out, err := cmd.CombinedOutput(); err != nil {
					return fmt.Errorf("child: %w: %s", err, out)
				}
				return set(v)
			}
			moved, errs := setAgain([]*Agent{a})
			url, _ := edit.GetJSON(a.Path, "provider.magpie.options.baseURL")
			if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.original"] != "concurrent" || url != "http://127.0.0.1:3592/v1" {
				t.Fatalf("rewire: moved=%v errors=%v URL=%q", moved, errs, url)
			}
		})
	}
}

// The gateway's port set in Settings (Magic_zero on Discord) moves every
// agent magpie connected: each one's config is written with the gateway's
// URL on the new port, none keeps the old one, and each is still connected
// with nothing off.
func TestPortMovesEveryAgent(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, _ := codexHome(t, "", "")
	managed := claudeManaged
	claudeManaged = func() string { return filepath.Join(home, "managed-settings.json") }
	t.Cleanup(func() { claudeManaged = managed })
	almaApp := startAlma(t)
	os.MkdirAll(filepath.Join(home, ".hanako", "agents", "hana"), 0o755)
	os.WriteFile(filepath.Join(home, ".hanako", "agents", "hana", "config.yaml"), []byte("agent:\n  name: Hana\n"), 0o644)
	// AstrBot is connected once it has written its settings
	os.MkdirAll(filepath.Join(home, ".astrbot", "data"), 0o755)
	os.WriteFile(filepath.Join(home, ".astrbot", "data", "cmd_config.json"), []byte(astrbotConfig), 0o644)
	setPort(t, 3591)
	if gateway.URL() != "http://127.0.0.1:3591" {
		t.Fatalf("Settings' port isn't the gateway's: %s", gateway.URL())
	}
	var wired []*Agent
	for _, a := range All() {
		// as TestDriftUnwiredEveryAgent: agy's gateway is in the command
		// that starts it, and a WSL agent's files are beyond the sandbox
		if a.Check == nil || a.Launch != nil || a.WSL != "" {
			continue
		}
		if a.Native != nil {
			if err := a.Connect(); err != nil {
				t.Fatal(err)
			}
			wired = append(wired, a)
			continue
		}
		var key, want string
		for _, g := range a.Fields {
			for _, o := range g.Options(a.Values()) {
				if want == "" && (o.Ref == "fake/m1" || o.Value == magpieID) {
					key, want = g.Key, o.Value
				}
			}
		}
		if want == "" {
			t.Fatalf("%s: no field takes magpie's models", a.ID)
		}
		if err := a.Apply(key, want); err != nil {
			t.Fatalf("%s: %v", a.ID, err)
		}
		wired = append(wired, a)
	}
	if len(filesWith(home, "127.0.0.1:3591")) == 0 {
		t.Fatal("no agent's config has the gateway's URL")
	}
	on := OnGateway()
	for _, a := range wired {
		if !containsAgent(on, a.ID) {
			t.Errorf("%s is connected but not on the gateway: %+v", a.ID, a.Drift())
		}
	}

	setPort(t, 3592)
	moved, err := Rewire(on)
	if err != nil {
		t.Fatal(err)
	}
	// OpenChamber reads OpenCode's config, which moves with OpenCode
	if len(moved) < len(wired)-1 {
		t.Errorf("moved %d of %d: %v", len(moved), len(wired), moved)
	}
	if left := filesWith(home, "127.0.0.1:3591"); len(left) > 0 {
		t.Errorf("still on the old port: %v", left)
	}
	if almaApp.repoint("127.0.0.1:3591", "127.0.0.1:3591") > 0 {
		t.Error("Alma still has the old port")
	}
	for _, a := range wired {
		if !a.Wired() {
			t.Errorf("%s isn't connected after the move", a.ID)
		}
		if d := a.Drift(); d != nil {
			t.Errorf("%s is off after the move: %+v", a.ID, d)
		}
	}
}

func containsAgent(as []*Agent, id string) bool {
	for _, a := range as {
		if a.ID == id {
			return true
		}
	}
	return false
}

// Codex signed in with ChatGPT is joined to magpie beside its own models:
// the move writes its base URL on the new port and keeps the model it was
// on, rather than putting it on one of magpie's.
func TestPortMovesJoinedCodex(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, read := codexHome(t, `{"auth_mode":"chatgpt","tokens":{"access_token":"at","refresh_token":"rt","id_token":"x.e30.x","account_id":"acc"}}`, "model = \"gpt-5.1\"\n")
	setPort(t, 3591)
	cx := codex(home)
	c, err := cx.ConnectHow()
	if err != nil || c.How != "joined" {
		t.Fatalf("not joined: %+v %v\n%s", c, err, read())
	}
	on := OnGateway()
	setPort(t, 3592)
	if _, err := Rewire(on); err != nil {
		t.Fatal(err)
	}
	cfg := read()
	if !strings.Contains(cfg, "127.0.0.1:3592") || strings.Contains(cfg, "127.0.0.1:3591") {
		t.Fatalf("Codex not moved:\n%s", cfg)
	}
	if !strings.Contains(cfg, `model = "gpt-5.1"`) || !cx.Joined() {
		t.Fatalf("Codex's own model lost:\n%s", cfg)
	}
	if d := cx.Drift(); d != nil {
		t.Fatalf("off after the move: %+v", d)
	}
}

// A real Codex join must not claim an unrelated same-process stash write.
func TestRewireCodexAllowsConcurrentStash(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, read := codexHome(t, `{"auth_mode":"chatgpt","tokens":{"access_token":"at","refresh_token":"rt","id_token":"x.e30.x","account_id":"acc"}}`, "model = \"gpt-5.1\"\n")
	setPort(t, 3591)
	cx := codex(home)
	if _, err := cx.ConnectHow(); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
	join := cx.joinRewire
	cx.joinRewire = func(g *rewireGuard) (bool, error) {
		done := make(chan error, 1)
		go func() { done <- stash(map[string]string{"other.original": "concurrent"}) }()
		if err := <-done; err != nil {
			return false, err
		}
		return join(g)
	}
	moved, errs := setAgain([]*Agent{cx})
	if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.original"] != "concurrent" || !strings.Contains(read(), "127.0.0.1:3592") {
		t.Fatalf("Codex rewire failed: moved=%v errors=%v stash=%v config=%s", moved, errs, stashLoad(), read())
	}
}

// An agent taken off magpie before the port moved is left as it is: it is
// the user's to set again, not the move's.
func TestPortLeavesAnUnwiredAgent(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, read := codexHome(t, "", "")
	setPort(t, 3591)
	cx := codex(home)
	if err := cx.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".codex", "config.toml")
	os.WriteFile(cfg, []byte(strings.ReplaceAll(read(), "127.0.0.1:3591", "127.0.0.1:9")), 0o644)
	on := OnGateway()
	setPort(t, 3592)
	if _, err := Rewire(on); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(read(), "127.0.0.1:3592") {
		t.Fatalf("an agent taken off magpie was moved:\n%s", read())
	}
}

// A magpie MAGPIE_ADDR puts beside the installed one tells the installed
// one's wiring by the port the installed one's Settings have, not 3425.
func TestDriftInstalledMagpieOnItsPort(t *testing.T) {
	home, _ := codexHome(t, "", "")
	setPort(t, 3591)
	t.Setenv("MAGPIE_ADDR", "127.0.0.1:3426")
	cx := codex(home)
	if err := cx.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(home, ".codex", "config.toml")
	os.WriteFile(cfg, []byte("openai_base_url = \"http://127.0.0.1:3591"+gateway.CodexPath+"\"\nmodel = \"fake/m1\"\n"), 0o644)
	if d := cx.Drift(); d != nil {
		t.Fatalf("the installed magpie's wiring taken for drift: %+v", d)
	}
}

// Moved, an agent still puts back what it had before magpie when switched
// off: its own endpoint and token, not magpie's at the old port, which
// setting it again there would otherwise have noted as its own.
// Rewire the real Hermes and DSH configs alongside an independent stash writer.
func TestRewireHermesAndDSHConcurrentStash(t *testing.T) {
	for _, kind := range []string{"hermes", "dsh"} {
		t.Run(kind, func(t *testing.T) {
			t.Setenv("MAGPIE_ADDR", "")
			t.Setenv("HERMES_HOME", "")
			t.Setenv("DSH_HOME", "")
			home, _ := codexHome(t, "", "")
			var a *Agent
			var path, original string
			if kind == "hermes" {
				a = hermes(home)
				path = filepath.Join(home, ".hermes", "config.yaml")
				original = "model:\n  default: native-model\n  provider: native\n"
			} else {
				a = dsh(home)
				path = filepath.Join(home, ".dsh", "config.yaml")
				original = "- id: tools\n  config:\n    disabled: []\n"
			}
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
				t.Fatal(err)
			}
			setPort(t, 3591)
			if err := a.Apply("model", "magpie/fake/m1"); err != nil {
				t.Fatal(err)
			}
			setPort(t, 3592)
			field := a.Field("model")
			setter := field.setRewire
			if setter == nil {
				t.Fatal("missing guarded setter")
			}
			field.setRewire = func(g *rewireGuard, v string) error {
				done := make(chan error, 1)
				go func() { done <- stash(map[string]string{"other.original": "concurrent"}) }()
				if err := <-done; err != nil {
					return err
				}
				return setter(g, v)
			}
			moved, errs := setAgain([]*Agent{a})
			data, _ := os.ReadFile(path)
			if len(errs) != 0 || len(moved) != 1 || stashLoad()["other.original"] != "concurrent" || !strings.Contains(string(data), "127.0.0.1:3592") {
				t.Fatalf("rewire: moved=%v errors=%v config=%s", moved, errs, data)
			}
		})
	}
}

// Gemini rewire leaves an independent stash update and its native credentials intact.
func TestRewireGeminiAllowsConcurrentStash(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, _ := codexHome(t, "", "")
	dir := filepath.Join(home, ".gemini")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(dir, "settings.json")
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(settings, []byte(`{"security":{"auth":{"selectedType":"gemini-api-key"}},"model":{"name":"gemini-2.5-pro"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(env, []byte("GEMINI_API_KEY=mine\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3591)
	a := gemini(home)
	if err := a.Apply("model", "fake/m1"); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
	field := a.Field("model")
	setter := field.setRewire
	field.setRewire = func(g *rewireGuard, value string) error {
		done := make(chan error, 1)
		go func() { done <- stash(map[string]string{"other.original": "concurrent"}) }()
		if err := <-done; err != nil {
			return err
		}
		return setter(g, value)
	}
	moved, errs := setAgain([]*Agent{a})
	url, _ := edit.GetEnvFile(env, "GOOGLE_GEMINI_BASE_URL")
	if len(errs) != 0 || len(moved) != 1 || url != "http://127.0.0.1:3592" || stashLoad()["other.original"] != "concurrent" {
		t.Fatalf("Gemini rewire: moved=%v errors=%v URL=%q", moved, errs, url)
	}
	if err := a.Disconnect(); err != nil {
		t.Fatal(err)
	}
	key, _ := edit.GetEnvFile(env, "GEMINI_API_KEY")
	model, _ := edit.GetJSON(settings, "model.name")
	if key != "mine" || model != "gemini-2.5-pro" {
		t.Fatal("Gemini original values not restored")
	}
}

// Claude's real model setter preserves its originals alongside another stash writer.
func TestRewireClaudeAllowsConcurrentStash(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, _ := codexHome(t, "", "")
	managed := claudeManaged
	claudeManaged = func() string { return filepath.Join(home, "managed-settings.json") }
	t.Cleanup(func() { claudeManaged = managed })
	path := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"env":{"ANTHROPIC_BASE_URL":"https://relay.example","ANTHROPIC_AUTH_TOKEN":"original-token"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3591)
	cc := claude(home)
	var model string
	for _, o := range cc.Field("model").Options(cc.Values()) {
		if o.Ref == "fake/m1" {
			model = o.Value
		}
	}
	if model == "" {
		t.Fatal("fixture model missing")
	}
	if err := cc.Apply("model", model); err != nil {
		t.Fatal(err)
	}
	setPort(t, 3592)
	field := cc.Field("model")
	setter := field.setRewire
	field.setRewire = func(g *rewireGuard, value string) error {
		done := make(chan error, 1)
		go func() { done <- stash(map[string]string{"other.original": "concurrent"}) }()
		if err := <-done; err != nil {
			return err
		}
		return setter(g, value)
	}
	moved, errs := setAgain([]*Agent{cc})
	url, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL")
	if len(errs) != 0 || len(moved) != 1 || url != "http://127.0.0.1:3592" || stashLoad()["other.original"] != "concurrent" {
		t.Fatalf("Claude rewire failed: moved=%v errors=%v URL=%q", moved, errs, url)
	}
	if err := cc.Disconnect(); err != nil {
		t.Fatal(err)
	}
	url, _ = edit.GetJSON(path, "env.ANTHROPIC_BASE_URL")
	token, _ := edit.GetJSON(path, "env.ANTHROPIC_AUTH_TOKEN")
	if url != "https://relay.example" || token != "original-token" {
		t.Fatal("original Claude connection was not restored")
	}
}

func TestPortKeepsWhatAnAgentHadBefore(t *testing.T) {
	t.Setenv("MAGPIE_ADDR", "")
	home, _ := codexHome(t, "", "")
	managed := claudeManaged
	claudeManaged = func() string { return filepath.Join(home, "managed-settings.json") }
	t.Cleanup(func() { claudeManaged = managed })
	path := filepath.Join(home, ".claude", "settings.json")
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(`{"env":{"ANTHROPIC_BASE_URL":"https://relay.example","ANTHROPIC_AUTH_TOKEN":"own-token"}}`), 0o644)
	setPort(t, 3591)
	cc := claude(home)
	var want string
	for _, o := range cc.Field("model").Options(cc.Values()) {
		if o.Ref == "fake/m1" {
			want = o.Value
		}
	}
	if err := cc.Apply("model", want); err != nil {
		t.Fatal(err)
	}
	on := OnGateway()
	setPort(t, 3592)
	if _, err := Rewire(on); err != nil {
		t.Fatal(err)
	}
	if u, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL"); u != "http://127.0.0.1:3592" {
		t.Fatalf("Claude Code not moved: %q", u)
	}
	if err := cc.Disconnect(); err != nil {
		t.Fatal(err)
	}
	u, _ := edit.GetJSON(path, "env.ANTHROPIC_BASE_URL")
	tok, _ := edit.GetJSON(path, "env.ANTHROPIC_AUTH_TOKEN")
	if u != "https://relay.example" || tok != "own-token" {
		t.Fatalf("switched off, Claude Code got %q %q back, not its own", u, tok)
	}
}
