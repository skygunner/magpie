package routing

import (
	"os"
	"strings"
	"testing"
)

func TestGroupMemberDistinguishable(t *testing.T) {
	jsPath := "/home/vincenty/Projects/magpie-r28/lane2/internal/gui/assets/plugins/routing/client.js"
	b, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("read client.js: %v", err)
	}
	s := string(b)

	if strings.Contains(s, `(x) => x.provider || x.providerName || ""`) {
		t.Fatalf("modelTells still groups by provider – cross-provider same name will have no tell")
	}
	if !strings.Contains(s, `modelTells(tellsFor || [], (x) => x.name || x.id, (x) => x.id, () => "")`) {
		t.Fatalf("modelTells not using global key () => \"\"")
	}

	if strings.Contains(s, `nm.title = memberName(id)`) {
		t.Fatalf("tooltip still uses memberName without provider")
	}
	if !strings.Contains(s, `nm.title = memberLabel(d, id)`) {
		t.Fatalf("tooltip not using memberLabel")
	}
}
