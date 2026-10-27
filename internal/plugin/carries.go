package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Carries returns the provider names the plugin declares it carries,
// from its package.json's magpie.carries field. Returns an empty slice
// if the file is missing, unparsable, or the field is missing/empty.
// Each name is trimmed and limited to 48 runes; at most 8 names are kept.
func Carries(spec string) []string {
	target := Target(spec)
	if target == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(target, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Magpie struct {
			Carries []string `json:"carries,omitempty"`
		} `json:"magpie"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return nil
	}
	var out []string
	for _, c := range pkg.Magpie.Carries {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		// limit each name to 48 runes
		r := []rune(c)
		if len(r) > 48 {
			c = string(r[:48])
		}
		if !slices.Contains(out, c) {
			out = append(out, c)
		}
		if len(out) >= 8 {
			break
		}
	}
	return out
}
