package appdir

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// MiniMaxDirIn is MiniMax Code's data folder. The Windows CN desktop build
// records its moved parent in %APPDATA%/MiniMax/minimax-agent-cn-config.json
// (config.localRuntimeDataParentDir, #2998); other builds keep the default.
// MINIMAX_DATA_DIR takes precedence, as it does for the CLI.
func MiniMaxDirIn(home string) string {
	if d := strings.TrimSpace(Getenv("MINIMAX_DATA_DIR")); d != "" {
		return d
	}
	dir := filepath.Join(home, ".minimax")
	appdata := Getenv("APPDATA")
	if appdata == "" {
		return dir
	}
	var s struct {
		Config struct {
			Parent string `json:"localRuntimeDataParentDir"`
		} `json:"config"`
	}
	b, err := os.ReadFile(filepath.Join(appdata, "MiniMax", "minimax-agent-cn-config.json"))
	if err != nil || json.Unmarshal(b, &s) != nil {
		return dir
	}
	parent := strings.TrimSpace(s.Config.Parent)
	if parent == "" || !rooted(parent) {
		return dir
	}
	return filepath.Join(parent, ".minimax")
}
