package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
)

// #3852 (OrangerXu): ZCode 3.12.3 reads a model's reasoning as levels,
// defaultLevel and providerOptionsByLevel (resources/glm/zcode.cjs); the
// variants and defaultVariant magpie wrote are keys it ignores, so nothing
// ever mapped a level to thinking, and an always-thinking model was sent
// without thinking and turned away (Kimi Code's k3.1-pro: "does not support
// disabled thinking mode"). The config now carries the schema ZCode reads
// beside the one its older builds did.
func TestZCodeReasoningLevelsAndOptionsByLevel(t *testing.T) {
	home := syncHome(t)
	os.WriteFile(catalog.CachePath(), []byte(`{"zai":{"models":{"glm-4.6":{"id":"glm-4.6","name":"GLM-4.6","limit":{"context":204800},
	  "reasoning_options":[{"type":"effort","values":["none","low","high","max"]}]}}}}`), 0o644)
	catalog.Reset()
	path := filepath.Join(home, ".zcode", "v2", "config.json")
	rules := filepath.Join(home, ".zcode", "v2", "provider_config.json")
	os.MkdirAll(filepath.Dir(path), 0o755)
	// a config.json of the user's keeps the write compact, as it is for
	// every config magpie edits that came in on one line
	os.WriteFile(path, []byte(`{"provider":{"builtin:zai":{"models":{"GLM-5-Turbo":{"reasoning":{"enabled":true,"variants":["enabled","off"],"defaultVariant":"enabled"},"limit":{"context":200000,"output":64000}}}}}}`), 0o644)
	os.WriteFile(rules, []byte(`{"schemaVersion":1,"config":{"providerConfigRules":{"providerRules":[]},"modelConfigRules":{"providerModelRules":[],"manualProviderModelRules":[]}}}`), 0o600)
	if err := zcode(home).Field("provider").Set("magpie"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	for _, want := range []string{
		`"levels":["disabled","low","high","max"]`,
		`"defaultLevel":"high"`,
		`"providerOptionsByLevel":{"disabled":{"thinking":{"type":"disabled"}},` +
			`"high":{"thinking":{"type":"enabled"}},"low":{"thinking":{"type":"enabled"}},"max":{"thinking":{"type":"enabled"}}}`,
	} {
		if !strings.Contains(string(b), want) {
			t.Fatalf("config.json lacks %s (len=%d): %s", want, len(b), string(b))
		}
	}
}
