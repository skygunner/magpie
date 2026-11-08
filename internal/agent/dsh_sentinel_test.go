package agent

import (
	"os"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A route refresh must leave another tool's top-level block and its sentinels
// byte-for-byte intact while retaining comments inside the route (#5684).
func TestDshSyncKeepsAdjacentManagedBlock(t *testing.T) {
	home, _, web := dshRouteHome(t)
	a := dsh(home)
	if err := a.Field("model").Set("magpie/deepseek/pro"); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(web)
	if err != nil {
		t.Fatal(err)
	}
	const block = "# >>> one-memory recall — managed block, do not edit >>>\n- insert:\n    - id: one-memory-recall\n      name: '@deepseek-ai/dsh-mcp-client'\n      config:\n        serverName: onememory\n# <<< one-memory recall — managed block <<<\n"
	const routeComment = "      # user route comment\n"
	s := strings.Replace(string(b), "    providers:\n", "    providers:\n"+routeComment, 1)
	if s == string(b) {
		t.Fatal("route provider mapping missing")
	}
	// Attach the external sentinel immediately before the next list entry.
	pos := strings.Index(s, "- id: llm-pi-ai")
	if pos < 0 {
		t.Fatal("route entry missing")
	}
	next := strings.Index(s[pos+1:], "\n- ")
	if next < 0 {
		s += block
	} else {
		pos += next + 2
		s = s[:pos] + block + s[pos:]
	}
	if err := os.WriteFile(web, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := provider.Save(provider.Provider{ID: "deepseek", Name: "DeepSeek", Chat: "https://api.deepseek.com/v1", Key: "k", Models: []string{"pro", "flash", "new-model"}}); err != nil {
		t.Fatal(err)
	}
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile(web)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+string(first), "\n"+block) || !strings.Contains(string(first), routeComment) {
		t.Fatalf("block or route comment changed:\n%s", first)
	}
	if a.Field("model").Get() != "magpie/deepseek/pro" || !strings.Contains(string(first), "deepseek/new-model") {
		t.Fatalf("route no longer works:\n%s", first)
	}
	if err := a.Sync(); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(web)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("second sync changed the file:\n%s\n---\n%s", first, second)
	}
	if err := a.Field("model").Set(""); err != nil {
		t.Fatal(err)
	}
	removed, err := os.ReadFile(web)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains("\n"+string(removed), "\n"+block) {
		t.Fatalf("route removal lost the adjacent block:\n%s", removed)
	}
}
