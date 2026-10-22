package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yetone/magpie/internal/agent"
)

func TestAgentsFieldsStayReadable(t *testing.T) {
	fields := []agent.Field{}
	values := map[string]string{}
	for _, key := range []string{"model", "subagents", "smol", "slow", "plan", "vision", "advisor", "commit", "tiny", "memory"} {
		fields = append(fields, agent.Field{Key: key, Label: key})
		values[key] = "magpie/relay/" + key + "-模型"
	}
	agents := []*agent.Agent{}
	allValues := []map[string]string{}
	for i := 0; i < 30; i++ {
		agents = append(agents, &agent.Agent{ID: fmt.Sprint(i), Name: fmt.Sprintf("agent-%02d", i), Path: "/tmp/omp/config.json", Fields: fields})
		allValues = append(allValues, values)
	}
	for _, size := range []struct{ w, h int }{{80, 24}, {48, 12}, {120, 40}} {
		t.Run(fmt.Sprintf("%dx%d", size.w, size.h), func(t *testing.T) {
			m := model{w: size.w, h: size.h, agents: agents, hidden: len(agents), values: allValues, failover: make([]string, len(agents)), row: 22}
			m.failover[m.row] = "fallback-ready"
			for range fields {
				v := m.viewList()
				lines := strings.Split(v, "\n")
				if len(lines) > size.h-4 {
					t.Fatalf("%d lines won't fit %d-high terminal:\n%s", len(lines), size.h, v)
				}
				for _, line := range lines[2:] {
					if lipgloss.Width(line) > size.w {
						t.Fatalf("%d-cell agent row exceeds %d:\n%s", lipgloss.Width(line), size.w, line)
					}
				}
				if !strings.Contains(v, "▸ "+agents[m.row].Name) || !strings.Contains(v, agents[m.row].Path) {
					t.Fatalf("selected agent or path is off screen:\n%s", v)
				}
				compact := strings.Join(strings.Fields(v), "")
				if !strings.Contains(compact, values[fields[m.col].Key]) {
					t.Fatalf("selected %s is not readable:\n%s", fields[m.col].Key, v)
				}
				if size.h == 40 {
					if !strings.Contains(v, "fallback-ready") {
						t.Fatalf("failover status missing: %s", v)
					}
					for _, f := range fields {
						if !strings.Contains(compact, values[f.Key]) {
							t.Fatalf("field %s missing in tall terminal", f.Key)
						}
					}
				}
				next, _ := m.updateList(tea.KeyMsg{Type: tea.KeyTab})
				m = next.(model)
			}
		})
	}
}
