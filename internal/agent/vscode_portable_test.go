package agent

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/edit"
	"github.com/yetone/magpie/internal/provider"
)

func TestVSCodePortableWritesItsProfiles(t *testing.T) {
	for _, mode := range []string{"archive", "linked", "override", "installed", "other editor", "missing product", "bad product"} {
		for _, k := range []vscodeKind{vscodeStable, vscodeInsiders, vscodiumKind} {
			t.Run(k.id+"/"+mode, func(t *testing.T) {
				home := syncHome(t)
				cfg := filepath.Join(home, "ordinary-config")
				t.Setenv("APPDATA", cfg)
				ordinary := vscodeUserDir(k, home, cfg)
				app := filepath.Join(home, "scoop", "apps", k.bin, "1.143.0")
				data := filepath.Join(app, "data")
				product := filepath.Join(app, "resources", "app", "product.json")
				if runtime.GOOS == "darwin" {
					app = filepath.Join(app, k.name+".app", "Contents", "Resources", "app")
					data = filepath.Clean(filepath.Join(app, "..", "..", "..", "..", k.bin+"-portable-data"))
					product = filepath.Join(app, "product.json")
				}
				binDir := filepath.Join(app, "bin")
				name := k.bin
				if runtime.GOOS == "windows" {
					name += ".cmd"
					t.Setenv("PATHEXT", ".CMD")
				}
				bin := filepath.Join(binDir, name)
				writeFile(t, bin, "fixture, never executed\n")
				if err := os.Chmod(bin, 0755); err != nil {
					t.Fatal(err)
				}
				writeFile(t, product, `{"applicationName":"`+k.bin+`"}`)
				if mode == "linked" {
					current := filepath.Join(home, "current")
					vscodeTestSymlink(t, app, current)
					binDir = filepath.Join(current, "bin")
				}
				t.Setenv("PATH", binDir)
				if mode == "override" {
					if err := os.MkdirAll(data, 0755); err != nil {
						t.Fatal(err)
					}
					data = filepath.Join(home, "my-portable-data")
					t.Setenv("VSCODE_PORTABLE", data)
				}
				switch mode {
				case "installed":
					writeFile(t, product, `{"applicationName":"`+k.bin+`","target":"user"}`)
				case "other editor":
					writeFile(t, product, `{"applicationName":"cursor"}`)
				case "missing product":
					if err := os.Remove(product); err != nil {
						t.Fatal(err)
					}
				case "bad product":
					writeFile(t, product, `{"applicationName":`)
				}
				dir := filepath.Join(data, "user-data", "User")
				settings := `{"editor.fontSize":17,"chat.defaultModel":"old"}`
				writeFile(t, filepath.Join(dir, "settings.json"), settings)
				work := filepath.Join(dir, "profiles", "work")
				writeFile(t, filepath.Join(work, "settings.json"), settings)
				writeFile(t, filepath.Join(dir, "globalStorage", "storage.json"), `{"userDataProfiles":[{"location":"work","name":"Work"}]}`)
				if k.exts != "" {
					writeFile(t, filepath.Join(data, "extensions", vscodeChatID+"-0.48.1", "package.json"), `{"version":"0.48.1","contributes":{"languageModelChatProviders":[{"vendor":"customoai"}]}}`)
				}
				a := vscodeOf(k, home, cfg)
				switch mode {
				case "installed", "other editor", "missing product", "bad product":
					if a.Dir != ordinary {
						t.Fatalf("ordinary config %s, want %s", a.Dir, ordinary)
					}
					return
				}
				if k.id == "vscode" {
					writeFile(t, filepath.Join(data, "extensions", zooExtension+"-3.90.0", "package.json"), `{}`)
					if z := zooCode(home, cfg); z.Dir != dir || !z.Detected() {
						t.Fatalf("portable Zoo Code: %s, detected %v", z.Dir, z.Detected())
					}
				}

				if a.Dir != dir || a.Path != filepath.Join(dir, "settings.json") {
					t.Fatalf("config %s, want portable %s", a.Path, dir)
				}
				if !a.Detected() {
					t.Fatal("portable editor not detected")
				}
				if err := a.Connect(); err != nil {
					t.Fatal(err)
				}
				if k.exts != "" {
					if g, _, _ := vscodeOurs(filepath.Join(dir, "chatLanguageModels.json")); gjson.Get(g, "vendor").String() != vscodeOAI {
						t.Fatalf("portable Copilot Chat format: %s", g)
					}
				}
				for _, d := range []string{dir, work} {
					if got, _ := edit.GetJSON(filepath.Join(d, "settings.json"), vscodeDefault); got != "relay/glm-4.6" {
						t.Fatalf("%s model %q", d, got)
					}
					if _, _, ok := vscodeOurs(filepath.Join(d, "chatLanguageModels.json")); !ok {
						t.Fatalf("no models in %s", d)
					}
				}
				if err := provider.Save(provider.Provider{ID: "relay", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"glm-4.6", "other"}}); err != nil {
					t.Fatal(err)
				}
				if err := a.Sync(); err != nil {
					t.Fatal(err)
				}
				for _, d := range []string{dir, work} {
					g, _, ok := vscodeOurs(filepath.Join(d, "chatLanguageModels.json"))
					if !ok || gjson.Get(g, "models.#").Int() != 2 {
						t.Fatalf("models not synced in %s: %s", d, g)
					}
				}
				if err := a.Disconnect(); err != nil {
					t.Fatal(err)
				}
				for _, d := range []string{dir, work} {
					if got := readFile(filepath.Join(d, "settings.json")); got != settings {
						t.Fatalf("settings not restored in %s: %s", d, got)
					}
				}
				if isDir(ordinary) {
					t.Fatal("created ordinary editor config")
				}
			})
		}
	}
}
