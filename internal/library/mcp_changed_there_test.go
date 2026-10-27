package library

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A server's entry changed in Codex's own config.toml is the user's: the
// library takes it, so magpie's page shows it and the other agents are given
// it, rather than the sync writing magpie's old one over Codex's at its next
// start (#3822).
func TestCodexEntryChangedThereIsTakenIntoTheLibrary(t *testing.T) {
	h := sandbox(t)
	cfg := filepath.Join(h, ".codex/config.toml")
	cl := filepath.Join(h, ".claude.json")
	srv := Server{Name: "gh", Transport: "stdio", Command: "gh-mcp", Args: []string{"serve"}, Agents: []string{"claude", "codex"}}
	ok(t)(SaveServer("", srv))
	if !strings.Contains(read(t, cl), "gh-mcp") {
		t.Fatalf("Claude Code wasn't given the server:\n%s", read(t, cl))
	}

	// the user changes it in Codex, as Codex's own MCP settings do
	write(t, cfg, `[mcp_servers.gh]
command = "gh-mcp-next"
args = ["serve", "--x"]
`)
	ok(t)(Sync())
	if e := codexEntry(t, cfg, "gh"); e["command"] != "gh-mcp-next" || !slices.Equal(strList(e["args"]), []string{"serve", "--x"}) {
		t.Fatalf("Codex's own change was written over:\n%s", read(t, cfg))
	}
	// the library has it: the page shows it, and the other agents are given it
	v, err := Read(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(v.Servers) != 1 || v.Servers[0].Command != "gh-mcp-next" || !slices.Equal(v.Servers[0].Args, []string{"serve", "--x"}) {
		t.Fatalf("the page: %+v", v.Servers)
	}
	if s := read(t, cl); !strings.Contains(s, "gh-mcp-next") {
		t.Errorf("Claude Code wasn't given the change:\n%s", s)
	}

	// magpie's own change after that goes into Codex too
	srv.Command, srv.Args = "gh-mcp-third", []string{"serve"}
	ok(t)(SaveServer("gh", srv))
	if e := codexEntry(t, cfg, "gh"); e["command"] != "gh-mcp-third" {
		t.Errorf("the library's change didn't reach Codex:\n%s", read(t, cfg))
	}

	// and once magpie's entry there is changed by hand again, that is taken
	// in too, so nothing of the user's is ever written over
	write(t, cfg, strings.Replace(read(t, cfg), `command = "gh-mcp-third"`, `command = "gh-mcp-fourth"`, 1))
	ok(t)(Sync())
	if e := codexEntry(t, cfg, "gh"); e["command"] != "gh-mcp-fourth" {
		t.Errorf("the second change was written over:\n%s", read(t, cfg))
	}
	if s := read(t, cl); !strings.Contains(s, "gh-mcp-fourth") {
		t.Errorf("Claude Code wasn't given the second change:\n%s", s)
	}
}
