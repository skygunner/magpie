package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/yetone/magpie/internal/agent"
)

// An agent with many model fields (omp) is wider than the terminal: its row
// carries on under the name, indented, so no field is clipped past the right
// edge and the picked one stays visible (#3234).
func TestAgentsRowWrapsItsFields(t *testing.T) {
	labels := []string{"model", "subagents", "smol", "slow", "plan", "vision", "advisor", "commit", "tiny", "memory"}
	var fields []agent.Field
	values := map[string]string{}
	for i, lbl := range labels {
		values[lbl] = "magpie/opencode-go/deepseek-v4.1-flash-" + string(rune('a'+i))
		fields = append(fields, agent.Field{Key: lbl, Label: lbl, Set: func(string) error { return nil }})
	}
	a := &agent.Agent{ID: "omp", Name: "omp", Path: "/opt/omp", Fields: fields}
	m := model{
		w:        80,
		h:        40,
		agents:   []*agent.Agent{a},
		values:   []map[string]string{values},
		failover: make([]string, 1),
		col:      8,
	}
	out := m.viewList()
	lines := strings.Split(out, "\n")

	// the picked marker opens the row, and every value is on some line of it
	if !strings.HasPrefix(lines[2], "  ▸ omp") {
		t.Fatalf("the row doesn't start with the marker and name:\n%s", out)
	}
	for _, lbl := range labels {
		if !strings.Contains(out, values[lbl]) {
			t.Fatalf("%s (%q) is clipped out of the row:\n%s", lbl, values[lbl], out)
		}
	}

	// no line runs past the terminal, and each continuation is indented
	// under the name, so the row reads as one row
	indent := "         "
	if w := lipgloss.Width(lines[2]); w > m.w {
		t.Fatalf("the first line is %d wide in an %d-wide terminal:\n%s", w, m.w, out)
	}
	for _, l := range lines[3:] {
		if w := lipgloss.Width(l); w > m.w {
			t.Fatalf("line %d wide in an %d-wide terminal: %q", w, m.w, l)
		}
		if !strings.HasPrefix(l, indent) {
			t.Fatalf("continuation not indented under the name: %q", l)
		}
	}

	// a row that fits is still one line
	m.w = 400
	m.agents = []*agent.Agent{{ID: "codex", Name: "codex", Path: "/opt/codex", Fields: fields[:1]}}
	m.values = []map[string]string{{"model": values["model"]}}
	if n := strings.Count(m.viewList(), "\n"); n != 2 {
		t.Fatalf("a row that fits took %d lines:\n%s", n-1, m.viewList())
	}
}
