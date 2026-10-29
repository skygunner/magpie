package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestRestoreFailureKeepsOriginalsForRetry(t *testing.T) {
	cases := []struct{ kind, marker string }{
		{"claude-model", "native-original"},
		{"claude-env", "ORIGINAL_ENV"},
		{"claude-base", "https://original.example"},
		{"claude-token", "original-token"},
		{"codex-model", "native-original"},
		{"codex-provider", "native-provider"},
		{"codex-catalog", "native-catalog.json"},
		{"codex-effort", "low"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			var path string
			var a *Agent
			if strings.HasPrefix(tc.kind, "claude") {
				home := syncHome(t)
				path = filepath.Join(home, ".claude", "settings.json")
				writeFile(t, path, `{"model":"native-original","env":{"ANTHROPIC_BASE_URL":"https://original.example","ANTHROPIC_AUTH_TOKEN":"original-token","ANTHROPIC_MODEL":"ORIGINAL_ENV"}}`)
				a = claude(home)
				if err := a.Apply("model", "relay/glm-4.6"); err != nil {
					t.Fatal(err)
				}
			} else {
				var home string
				home, _ = codexHome(t, "", "model = \"native-original\"\nmodel_provider = \"native-provider\"\nmodel_catalog_json = \"native-catalog.json\"\nmodel_reasoning_effort = \"low\"\n")
				if tc.kind != "codex-provider" {
					_ = edit.DelTOMLTop(filepath.Join(home, ".codex", "config.toml"), "model_provider")
				}
				if err := os.WriteFile(filepath.Join(home, ".codex", "native-catalog.json"), []byte(`[]`), 0600); err != nil {
					t.Fatal(err)
				}
				path = filepath.Join(home, ".codex", "config.toml")
				a = codex(home)
				if err := a.Apply("model", "fake/m1"); err != nil {
					t.Fatal(err)
				}
			}
			held := path + ".held-test"
			injected := false
			err := edit.Filtered(func(p string, data []byte) []byte {
				if p == path && strings.Contains(string(data), tc.marker) && !injected {
					injected = true
					if err := os.Rename(path, held); err != nil {
						t.Fatal(err)
					}
					if err := os.Mkdir(path, 0700); err != nil {
						t.Fatal(err)
					}
				}
				return data
			}, a.Disconnect)
			if !injected || err == nil {
				t.Fatalf("restore failure not injected: %v", err)
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(held, path); err != nil {
				t.Fatal(err)
			}
			if err := stash(map[string]string{"other.concurrent": "keep"}); err != nil {
				t.Fatal(err)
			}
			err = a.Disconnect()
			if err != nil {
				t.Fatalf("retry: %v", err)
			}
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b), tc.marker) {
				t.Fatalf("original not restored: %s", b)
			}
			if got, err := peekStash("other.concurrent"); err != nil || got != "keep" {
				t.Fatalf("concurrent stash lost: %q %v", got, err)
			}
		})
	}
}
