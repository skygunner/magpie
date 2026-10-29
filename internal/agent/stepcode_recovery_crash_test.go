package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/yetone/magpie/internal/edit"
)

func TestStepCodeRecoveryCrashWrites(t *testing.T) {
	if n := os.Getenv("STEPCODE_CRASH_WRITE"); n != "" {
		limit, err := strconv.Atoi(n)
		if err != nil {
			t.Fatal(err)
		}
		home := os.Getenv("HOME")
		config := filepath.Join(home, ".stepcode", "config.toml")
		models := filepath.Join(home, ".stepcode", "models.json")
		count := 0
		_ = edit.Filtered(func(path string, data []byte) []byte {
			if path == config || path == models {
				count++
				if count == limit {
					os.Exit(73)
				}
			}
			if limit == 5 && path == stashPath() {
				var m map[string]string
				if json.Unmarshal(data, &m) == nil {
					var marker map[string]json.RawMessage
					if json.Unmarshal([]byte(m["stepcode.disconnect.pending"]), &marker) == nil && string(marker["stepcode_done"]) == "true" {
						os.Exit(73)
					}
				}
			}
			return data
		}, stepcode(home).Disconnect)
		os.Exit(74)
	}
	for n := 1; n <= 5; n++ {
		for _, change := range []string{"none", "config", "models", "stash", "unrelated"} {
			t.Run(fmt.Sprintf("write=%d/%s", n, change), func(t *testing.T) {
				home, _ := museHome(t)
				config := filepath.Join(home, ".stepcode", "config.toml")
				models := filepath.Join(home, ".stepcode", "models.json")
				stepcodeWriteTestFile(t, config, "defaultProvider = \"native\"\ndefaultModel = \"original\"\nenabledModels = [\"native/one\"]\n")
				stepcodeWriteTestFile(t, models, `{"providers":{"native":{"name":"native"}}}`)
				if err := stepcode(home).Apply("model", "magpie/deepseek/pro"); err != nil {
					t.Fatal(err)
				}
				exe, err := os.Executable()
				if err != nil {
					t.Fatal(err)
				}
				cmd := exec.Command(exe, "-test.run=^TestStepCodeRecoveryCrashWrites$")
				cmd.Env = append(os.Environ(), "STEPCODE_CRASH_WRITE="+strconv.Itoa(n))
				err = cmd.Run()
				if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 73 {
					t.Fatalf("crash boundary: %v", err)
				}
				switch change {
				case "config":
					if err := edit.SetTOMLTopPreserving(config, edit.KV{Path: "defaultModel", Value: "user-new"}); err != nil {
						t.Fatal(err)
					}
				case "models":
					if err := edit.SetJSON(models, edit.KV{Path: "providers.magpie", Value: map[string]string{"name": "user-new"}}); err != nil {
						t.Fatal(err)
					}
				case "stash":
					if err := stash(map[string]string{"stepcode.wiring": `{"user":"new"}`}); err != nil {
						t.Fatal(err)
					}
				case "unrelated":
					if err := stash(map[string]string{"other.user": "keep"}); err != nil {
						t.Fatal(err)
					}
				}
				err = stepcode(home).Disconnect()
				if change == "none" || change == "unrelated" {
					if err != nil {
						t.Fatalf("resume: %v", err)
					}
					cfg, err := readStepcodeTOML(config)
					if err != nil || stepcodeValue(cfg.Provider) != "native" || stepcodeValue(cfg.Model) != "original" {
						t.Fatalf("config not restored: %+v %v", cfg, err)
					}
					if change == "unrelated" {
						if got, err := peekStash("other.user"); err != nil || got != "keep" {
							t.Fatalf("unrelated stash lost: %q %v", got, err)
						}
					}
					return
				}
				if err == nil {
					t.Fatal("expected conflict")
				}
				switch change {
				case "config":
					cfg, _ := readStepcodeTOML(config)
					if stepcodeValue(cfg.Model) != "user-new" {
						t.Fatal("model overwritten")
					}
				case "models":
					got, _ := edit.GetJSON(models, "providers.magpie.name")
					if got != "user-new" {
						t.Fatal("provider overwritten")
					}
				case "stash":
					got, _ := peekStash("stepcode.wiring")
					var v map[string]string
					if json.Unmarshal([]byte(got), &v) != nil || v["user"] != "new" {
						t.Fatal("stash overwritten")
					}
				}
			})
		}
	}
}
