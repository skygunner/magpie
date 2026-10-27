package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/yetone/magpie/internal/testenv"
)

func TestDebugSync(t *testing.T) {
	home := t.TempDir()
	testenv.SetHome(t, home)
	
	cfg := filepath.Join(home, ".config", "opencode", "opencode.json")
	os.MkdirAll(filepath.Dir(cfg), 0o755)
	
	// Write initial config
	initial := `{"theme":"dark","provider":{"magpie":{"models":{"codex/gpt-5.5":{"variants":{
  "max":{"reasoningEffort":"max"},
  "turbo":{"reasoningEffort":"low","service_tier":"priority","textVerbosity":"low"},
  "high-fast":{"reasoningEffort":"high","service_tier":"flex"}}}}}}}`
	os.WriteFile(cfg, []byte(initial), 0o644)
	
	oc := opencode(home, filepath.Join(home, ".config"))
	fmt.Println("Before Sync:")
	b, _ := os.ReadFile(cfg)
	fmt.Println(string(b))
	
	if err := oc.Sync(); err != nil {
		t.Fatal(err)
	}
	
	b2, _ := os.ReadFile(cfg)
	fmt.Println("After Sync:")
	fmt.Println(string(b2))
}
