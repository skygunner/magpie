package agent

import (
	"os"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/gateway"
)

// #3329: a comment line between two patch-list entries is kept when magpie
// writes the list again. Vibe Island's managed block is bounded by comment
// markers around an entry of its own; magpie used to append each marker to
// the entry before it, and writing that entry again took the marker with it,
// leaving the closing one behind.
func TestDshCommentsBetweenEntriesSurviveRewrite(t *testing.T) {
	_, dir, web := dshRouteHome(t)
	dshKeyFixture(dir)
	// the reporter's managed block (Vibe Island 1.0.53), down to its path
	managed := "# >>> vibe-island managed block >>> do not edit; remove the whole block to uninstall\n" +
		"- insert:\n    - id: vibe-island\n      name: /Users/<user>/.vibe-island/dsh-plugin/index-w1r8.js\n" +
		"# <<< vibe-island managed block <<<\n"
	os.WriteFile(web, []byte("# Your patch layer for this dsh profile.\n"+
		dshRouteFixture("deepseek/pro")+
		managed+
		"- id: agent-default-model\n  config:\n    provider: magpie\n    model: deepseek/flash\n"), 0o644)

	// the catalog's route is not the one in the file: magpie writes its entry
	// again, as the round does every 30 seconds
	if _, err := dshRouteAgain(web, magpieModels("dsh"), gateway.URL()); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(web)
	if err != nil {
		t.Fatal(err)
	}
	if s := string(b); !strings.Contains(s, managed) {
		t.Fatalf("the managed block's comment markers lost:\n%s", s)
	}
}
