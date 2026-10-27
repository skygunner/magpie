package provider

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/settings"
)

// A plugin's models hook that couldn't read its vendor's list answers the
// plugin's short defaults (plugin.Provider.FellBack, "Cursor in WSL listing
// Auto alone"), and the editor says so over the short list. A pick saved
// from that list is not a pick of every model the vendor lists: with the
// short list kept as PickedFrom, every model the real list then has reads
// as listed since, joins the picks and is served — Qoder CN's user picked
// the free models alone on such a list and found every model picked again
// after Save (#3235). A list that fell back says nothing of what the vendor
// lists, as nothing is dropped from the picks for one (plugin_relist.go).
func TestPluginPicksSavedFromAShortListStayThePicks(t *testing.T) {
	claudeHome(t)
	t.Setenv("MAGPIE_BUN", "")
	spec := "@magpie-community/opencode-qoder-auth"
	write := func(name string, v any) {
		t.Helper()
		b, _ := json.Marshal(v)
		os.MkdirAll(settings.Dir(), 0o700)
		if err := os.WriteFile(filepath.Join(settings.Dir(), name), b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("plugins.json", map[string]any{"plugins": []map[string]any{{"spec": spec}}})
	write("plugin-auth.json", map[string]map[string]any{"qoder-cn": {"type": "api", "key": "qoder-key-1234"}})
	// the keys Qoder CN's listing has, and the two it serves free (the
	// plugin's packages/qoder: qfmodel 0x, dfmodel 0.1x, the rest paid)
	free := listedModels(t, `[
		{"id": "qfmodel", "name": "Qwen3.8-Flash", "free": true},
		{"id": "dfmodel", "name": "DeepSeek-Flash", "free": true}]`)
	whole := listedModels(t, `[
		{"id": "qfmodel", "name": "Qwen3.8-Flash", "free": true},
		{"id": "qmodel_38max", "name": "Qwen3.8-Max"},
		{"id": "dfmodel", "name": "DeepSeek-Flash", "free": true},
		{"id": "dmodel", "name": "DeepSeek-V4-Pro"},
		{"id": "gmodel", "name": "GLM-5.3"},
		{"id": "gfmodel", "name": "GLM-5.3-Flash"},
		{"id": "kmodel_latest", "name": "Kimi-K3"},
		{"id": "mmodel", "name": "MiniMax-M3"}]`)
	why := "Qoder CN couldn't list its models: the service answered 503"
	serve := func(models []plugin.Model, fellBack string) {
		t.Helper()
		pp := plugin.Provider{ID: "qoder-cn", Spec: spec, Name: "Qoder CN",
			Methods:  []plugin.Method{{Type: "api", Label: "API key"}},
			Accounts: []plugin.Account{{Key: "qoder-cn", Type: "api"}},
			Models:   models}
		if fellBack != "" {
			pp.FellBack, pp.ListError = true, fellBack
			pp.Accounts[0].FellBack, pp.Accounts[0].ListError = true, fellBack
		}
		plugin.UseCached([]plugin.Provider{pp})
	}
	t.Cleanup(func() { plugin.UseCached(nil) })
	saveLogins(t, savedLogin{Agent: "plugin:qoder-cn", User: "me@example.com", Home: "qoder-cn", On: true})
	picks := []string{"qfmodel", "dfmodel"}
	check := func(when string) {
		t.Helper()
		p, err := Find("qoder-cn")
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Available()) != len(whole) {
			t.Fatalf("%s: the provider lists %d models, want %d", when, len(p.Available()), len(whole))
		}
		if got := p.Picks(); !slices.Equal(got, picks) {
			t.Errorf("%s: the picks read back as %v, want %v", when, got, picks)
		}
		if n := len(p.Exposed()); n != len(picks) {
			t.Errorf("%s: magpie serves %d models, want the %d picked", when, n, len(picks))
		}
	}

	// the editor's Save pressed with Free only: the two the short list has
	serve(free, why)
	if err := Save(Provider{ID: "qoder-cn", Models: picks}); err != nil {
		t.Fatal(err)
	}
	// the vendor's list read again, this time whole
	serve(whole, "")
	check("first save")

	// and a later Save made while it fell back again: the picks are the same
	serve(free, why)
	if p, err := Find("qoder-cn"); err == nil {
		if err := Save(*p); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}
	serve(whole, "")
	check("saved again")
}
