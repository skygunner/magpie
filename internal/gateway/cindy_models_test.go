package gateway

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/provider"
)

// Cindy's menu shows a model by its name alone: the provider it took magpie
// as is Magpie on every row, whatever model it is. So /v1/models names each
// model the way the agents' lists do — its provider after it, by the
// Provider in model names setting (#3017) — and two that still read alike
// say their whole id. The id a model is asked by does not change.
func TestCindyModelsNameTheirProvider(t *testing.T) {
	fresh(t)
	os.MkdirAll(filepath.Dir(catalog.CachePath()), 0o755)
	data := `{"a":{"models":{"glm":{"id":"glm","name":"GLM-5.3"}}},"twin":{"models":{"x":{"id":"x","name":"Twin"},"x2":{"id":"x2","name":"Twin"}}}}`
	if err := os.WriteFile(catalog.CachePath(), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog.Reset()
	t.Cleanup(catalog.Reset)
	for _, p := range []provider.Provider{
		{ID: "a", Name: "WorkBuddy", Catalog: "a", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"glm"}},
		{ID: "twin", Name: "Twinco", Catalog: "twin", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"x", "x2"}},
	} {
		if err := provider.Save(p); err != nil {
			t.Fatal(err)
		}
	}
	// a name the user gave a model: with the provider after it on the
	// settings' default, as "Sol" · OpenAI in the agents' lists
	if err := provider.SetModelName("a/glm", "GLM 5.3"); err != nil {
		t.Fatal(err)
	}
	list := func() map[string]string {
		t.Helper()
		// as Cindy asks: through the provider its import link gives it,
		// keyed magpie-cindy (cindy.go's CindyLink), whichever of its
		// runtimes inside asks
		req := httptest.NewRequest("GET", "/v1/models", nil)
		req.Header.Set("x-api-key", TokenFor("cindy"))
		req.Header.Set("Authorization", "Bearer "+TokenFor("cindy"))
		req.Header.Set("User-Agent", "cindy/0.1.110")
		rec := httptest.NewRecorder()
		New().Handler().ServeHTTP(rec, req)
		var out struct {
			Data []struct {
				ID   string `json:"id"`
				Name string `json:"display_name"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, m := range out.Data {
			got[m.ID] = m.Name
		}
		return got
	}
	var ids []string
	for id := range list() {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	if want := []string{"a/glm", "twin/x", "twin/x2"}; !slices.Equal(ids, want) {
		t.Fatalf("Cindy is listed %v, want %v", ids, want)
	}
	for _, c := range []struct{ mode, glm, x, x2 string }{
		{provider.SuffixOn, "GLM 5.3 · WorkBuddy", "Twin · Twinco (twin/x)", "Twin · Twinco (twin/x2)"},
		{provider.SuffixOwn, "GLM 5.3", "Twin · Twinco (twin/x)", "Twin · Twinco (twin/x2)"},
		{provider.SuffixOff, "GLM 5.3", "Twin · Twinco (twin/x)", "Twin · Twinco (twin/x2)"},
	} {
		if err := provider.SetSuffixMode(c.mode); err != nil {
			t.Fatal(err)
		}
		got := list()
		for id, want := range map[string]string{"a/glm": c.glm, "twin/x": c.x, "twin/x2": c.x2} {
			if got[id] != want {
				t.Fatalf("%s: %s named %q, want %q (all: %v)", c.mode, id, got[id], want, got)
			}
		}
	}
	// with the bare ids on, the same model is listed twice (settings'
	// Model ids without the provider), and reads alike in both rows
	if err := provider.SetSuffixMode(provider.SuffixOn); err != nil {
		t.Fatal(err)
	}
	if err := provider.SetBareModelIDs(true); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { provider.SetBareModelIDs(false) })
	if got := list(); got["a/glm"] != "GLM 5.3 · WorkBuddy" || got["glm"] != "GLM 5.3 · WorkBuddy" {
		t.Fatalf("bare ids: %v", got)
	}
}
