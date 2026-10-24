package library

import (
	"path/filepath"
	"strings"
	"testing"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/yetone/magpie/internal/agent"
)

// Reasonix Studio gets the Library's MCP servers and skills (#3401,
// FuchaZ): its servers are [[plugins]] in the config.toml magpie already
// writes its provider into — type stdio (which it takes as the default)
// beside a command, http or sse beside a url — written as its own config
// reads them, read back, found when the user put one there, and taken out
// again leaving the user's plugins and every other field of the file.
func TestReasonixPluginsMCP(t *testing.T) {
	h := sandbox(t)
	rx := filepath.Join(h, ".reasonix")
	path := filepath.Join(rx, "config.toml")
	write(t, path, `# mine
default_model = "qwen"

[[plugins]]
name = "mine"
type = "stdio"
command = "uvx"
args = ["mine"]

[[plugins]]
name = "web-mine"
type = "http"
url = "https://example.com/mcp"

	[plugins.headers]
	Authorization = "Bearer keep"

[[providers]]
name = "keep"
api_key_env = "OTHER"
`)
	a := &agent.Agent{ID: "reasonix", Name: "Reasonix Studio", Dir: rx, Path: path}
	tg := targetOf(a)
	if tg == nil || tg.MCP == nil || tg.MCP.Path != path {
		t.Fatalf("target: %+v", tg)
	}
	if tg.Skills != filepath.Join(rx, "skills") || ProjectSkillsDir("reasonix") != ".reasonix/skills" {
		t.Fatalf("skills: %q %q", tg.Skills, ProjectSkillsDir("reasonix"))
	}
	f := tg.MCP

	// the user's plugins are there to bring in, each shape read back
	found, err := f.read()
	if err != nil {
		t.Fatal(err)
	}
	if s := found["mine"]; s == nil || s.Command != "uvx" || len(s.Args) != 1 || s.Args[0] != "mine" || s.Transport != "stdio" {
		t.Fatalf("mine: %+v", s)
	}
	if s := found["web-mine"]; s == nil || s.Transport != "http" || s.URL != "https://example.com/mcp" || s.Headers["Authorization"] != "Bearer keep" {
		t.Fatalf("web-mine: %+v", s)
	}

	// magpie writes its own, as Reasonix's config takes them
	for _, s := range []Server{
		{Name: "fs", Transport: "stdio", Command: "npx", Args: []string{"-y", "@mcp/fs"}, Env: map[string]string{"K": "V"}},
		{Name: "web", Transport: "http", URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "Bearer x"}},
		{Name: "old", Transport: "sse", URL: "https://example.com/sse"},
	} {
		if err := f.supports(&s); err != nil {
			t.Fatalf("%s: %v", s.Name, err)
		}
		if err := f.put(&s, nil); err != nil {
			t.Fatalf("%s: %v", s.Name, err)
		}
	}
	back, err := f.read()
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]Server{
		"fs":  {Name: "fs", Transport: "stdio", Command: "npx", Args: []string{"-y", "@mcp/fs"}, Env: map[string]string{"K": "V"}},
		"web": {Name: "web", Transport: "http", URL: "https://example.com/mcp", Headers: map[string]string{"Authorization": "Bearer x"}},
		"old": {Name: "old", Transport: "sse", URL: "https://example.com/sse"},
	} {
		if s := back[name]; s == nil || !s.same(&want) {
			t.Fatalf("%s read back as %+v", name, s)
		}
	}
	// one of magpie's, as the file holds it: an entry of the [[plugins]]
	// array, Reasonix's own shape
	text := read(t, path)
	if !strings.Contains(text, "[[plugins]]") {
		t.Fatalf("no plugins entries:\n%s", text)
	}
	var doc struct {
		Plugins []map[string]any `toml:"plugins"`
	}
	if err := toml.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatalf("%v:\n%s", err, text)
	}
	var fs map[string]any
	for _, m := range doc.Plugins {
		if m["name"] == "fs" {
			fs = m
		}
	}
	if fs == nil || fs["type"] != "stdio" || fs["command"] != "npx" {
		t.Fatalf("fs as written: %+v", fs)
	}

	// taken out again: the user's plugins, provider and model stay
	for _, name := range []string{"fs", "web", "old"} {
		if err := f.del(name); err != nil {
			t.Fatal(err)
		}
	}
	after := read(t, path)
	for _, keep := range []string{`name = "mine"`, `name = "web-mine"`, `name = "keep"`, `default_model = "qwen"`} {
		if !strings.Contains(after, keep) {
			t.Fatalf("%s didn't stay:\n%s", keep, after)
		}
	}
	for _, gone := range []string{`"fs"`, `"web"`, `"old"`} {
		if strings.Contains(after, gone) {
			t.Fatalf("%s stayed:\n%s", gone, after)
		}
	}
}
