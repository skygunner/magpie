package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestUnwireCrashBoundary(t *testing.T) {
	if kind := os.Getenv("MAGPIE_TEST_UNWIRE_CRASH"); kind != "" {
		home := os.Getenv("HOME")
		a := gemini(home)
		target := filepath.Join(home, ".gemini", ".env")
		if kind == "stepcode" {
			a = stepcode(home)
			target = filepath.Join(home, ".stepcode", "config.toml")
		}
		writes := 0
		_ = edit.Filtered(func(path string, data []byte) []byte {
			if path == target {
				writes++
				if writes == 2 {
					os.Exit(73)
				}
			}
			return data
		}, a.Disconnect)
		os.Exit(74)
	}
	for _, kind := range []string{"gemini", "stepcode"} {
		for _, userEdit := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/edit=%v", kind, userEdit), func(t *testing.T) {
				home, _ := museHome(t)
				var a *Agent
				if kind == "gemini" {
					writeFile(t, filepath.Join(home, ".gemini", "settings.json"), `{"model":{"name":"native-original"},"security":{"auth":{"selectedType":"oauth-personal"}}}`)
					writeFile(t, filepath.Join(home, ".gemini", ".env"), "GOOGLE_GEMINI_BASE_URL=https://original.example\nGEMINI_API_KEY=original-test-key\nOTHER=keep\n")
					a = gemini(home)
					if err := a.Apply("model", "deepseek/pro"); err != nil {
						t.Fatal(err)
					}
				} else {
					stepcodeWriteTestFile(t, filepath.Join(home, ".stepcode", "config.toml"), "defaultProvider = \"native\"\ndefaultModel = \"original\"\n")
					a = stepcode(home)
					if err := a.Apply("model", "magpie/deepseek/pro"); err != nil {
						t.Fatal(err)
					}
				}
				exe, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(exe, "-test.run=^TestUnwireCrashBoundary$")
				cmd.Env = append(os.Environ(), "MAGPIE_TEST_UNWIRE_CRASH="+kind)
				err = cmd.Run()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 73 {
					t.Fatalf("did not stop at write boundary: %v", err)
				}
				if kind == "gemini" {
					p := filepath.Join(home, ".gemini", ".env")
					v, _ := edit.GetEnvFile(p, "GOOGLE_GEMINI_BASE_URL")
					if v != "" {
						t.Fatalf("env not removed")
					}
					if !userEdit {
						if err := gemini(home).Disconnect(); err != nil {
							t.Fatalf("resume: %v", err)
						}
						v, _ = edit.GetEnvFile(p, "GOOGLE_GEMINI_BASE_URL")
						if v != "https://original.example" {
							t.Fatal("endpoint not restored")
						}
						return
					}
					if err := edit.SetJSON(filepath.Join(home, ".gemini", "settings.json"), edit.KV{Path: "model.name", Value: "user-new"}); err != nil {
						t.Fatal(err)
					}
					if err := gemini(home).Disconnect(); err == nil || !strings.Contains(err.Error(), "settings") {
						t.Fatalf("partial disconnect must require recovery: %v", err)
					}
					v, _ = edit.GetEnvFile(p, "GOOGLE_GEMINI_BASE_URL")
					if v != "" {
						t.Fatal("partial state unexpectedly overwritten")
					}
					if got, _ := edit.GetJSON(filepath.Join(home, ".gemini", "settings.json"), "model.name"); got != "user-new" {
						t.Fatal("user model overwritten")
					}
					if pending, err := peekStash("gemini.disconnect.pending"); err != nil || pending == "" {
						t.Fatal("recovery record lost")
					}
				} else {
					p := filepath.Join(home, ".stepcode", "config.toml")
					cfg, err := readStepcodeTOML(p)
					if err != nil {
						t.Fatal(err)
					}
					if stepcodeValue(cfg.Provider) != "native" || !strings.Contains(stepcodeValue(cfg.Model), "deepseek") {
						t.Fatal("wrong interruption boundary")
					}
					if !userEdit {
						if err := stepcode(home).Disconnect(); err != nil {
							t.Fatalf("resume: %v", err)
						}
						cfg, err = readStepcodeTOML(p)
						if err != nil || stepcodeValue(cfg.Model) != "original" {
							t.Fatal("original model not restored")
						}
						return
					}
					if err := edit.SetTOMLTopPreserving(p, edit.KV{Path: "defaultModel", Value: "user-new"}); err != nil {
						t.Fatal(err)
					}
					if err := stepcode(home).Disconnect(); err == nil || !strings.Contains(err.Error(), "changed") {
						t.Fatalf("partial disconnect must require recovery: %v", err)
					}
					cfg, err = readStepcodeTOML(p)
					if err != nil || stepcodeValue(cfg.Provider) != "native" || stepcodeValue(cfg.Model) != "user-new" {
						t.Fatal("partial state unexpectedly overwritten")
					}
					if pending, err := peekStash("stepcode.disconnect.pending"); err != nil || pending == "" {
						t.Fatal("recovery record lost")
					}
				}
			})
		}
	}
}
