package agent

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/plugin"
	"github.com/yetone/magpie/internal/settings"
)

// #3282: Google plugin models use Code Assist internally, but Pi must
// send Gemini requests to the gateway, with its key rather than the plugin's.
func TestPiGooglePluginModels(t *testing.T) {
	home := syncHome(t)
	t.Setenv("MAGPIE_BUN", "")
	const spec = "google-plugin-test"
	writeFile(t, filepath.Join(settings.Dir(), "plugins.json"), `{"plugins":[{"spec":"`+spec+`"}]}`)
	writeFile(t, filepath.Join(settings.Dir(), "plugin-auth.json"), `{"google-plugin":{"type":"api","key":"plugin-secret"}}`)
	plugin.UseCached([]plugin.Provider{{ID: "google-plugin", Spec: spec, Name: "Google plugin",
		Models: []plugin.Model{{ID: "gemini-3-flash", Name: "Gemini 3 Flash", NPM: "@ai-sdk/google"}}}})
	t.Cleanup(plugin.Settle)
	const gw = "http://127.0.0.1:17325"
	t.Setenv("MAGPIE_ADDR", "127.0.0.1:17325")
	for _, a := range []*Agent{pi(home), omo(home), primeAgent(home), stepcode(home)} {
		t.Run(a.ID, func(t *testing.T) {
			if err := a.Field("model").Set("magpie/google-plugin/gemini-3-flash"); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(filepath.Dir(a.Path), "models.json")
			// Existing files must acquire the correction during normal sync too.
			p := providerAt(t, path, "providers")
			for _, raw := range p["models"].([]any) {
				m := raw.(map[string]any)
				delete(m, "api")
				delete(m, "baseUrl")
			}
			b, _ := json.Marshal(map[string]any{"providers": map[string]any{"magpie": p}})
			writeFile(t, path, string(b))
			if err := a.Sync(); err != nil {
				t.Fatal(err)
			}
			p = providerAt(t, path, "providers")
			if p["api"] != "openai-completions" || p["baseUrl"] != gw+"/v1" || p["apiKey"] == "plugin-secret" {
				t.Fatalf("provider defaults or gateway authentication changed: %v", p)
			}
			found := false
			for _, raw := range p["models"].([]any) {
				m := raw.(map[string]any)
				if m["id"] != "google-plugin/gemini-3-flash" {
					continue
				}
				found = true
				if m["api"] != "google-generative-ai" || m["baseUrl"] != gw+"/v1beta" {
					t.Errorf("Google model still inherits Chat: %v", m)
				}
			}
			if !found {
				t.Fatal("Google plugin model not offered")
			}
		})
	}
	for _, m := range pencilProviderJSON(gw)["models"].([]map[string]any) {
		if m["api"] != nil || m["baseUrl"] != nil {
			t.Errorf("Pencil no longer uses its provider's Chat API: %v", m)
		}
	}
}
