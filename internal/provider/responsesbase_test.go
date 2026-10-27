package provider

import (
	"testing"

	"github.com/yetone/magpie/internal/catalog"
)

// A model whose list names the Responses API is asked on Responses at the
// Chat base when that base is the one OpenAI's two APIs are asked under
// (…/v1), and nowhere else (#2822).
func TestWithResponsesAtChatBase(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := catalog.SaveLive("respbase", "https://relay.example/v1", []catalog.Model{
		{ID: "gpt-both", APIs: []string{"chat", "responses"}},
		{ID: "gpt-chat", APIs: []string{"chat"}},
	}); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name, chat, responses, model, want string
	}{
		{"a base both APIs share", "https://relay.example/v1", "", "gpt-both", "https://relay.example/v1"},
		{"a trailing slash", "https://relay.example/v1/", "", "gpt-both", "https://relay.example/v1"},
		{"a base under another path", "https://relay.example/openai/v1", "", "gpt-both", "https://relay.example/openai/v1"},
		{"an endpoint pasted whole", "https://relay.example/v1/chat/completions", "", "gpt-both", ""},
		{"a host with no base", "https://relay.example", "", "gpt-both", ""},
		{"a list that names Chat alone", "https://relay.example/v1", "", "gpt-chat", ""},
		{"a Responses URL of its own", "https://relay.example/v1", "https://other.example/v1", "gpt-both", "https://other.example/v1"},
	} {
		p := Provider{ID: "respbase", Chat: c.chat, Responses: c.responses}
		if got := p.WithResponsesAtChatBase(c.model).Responses; got != c.want {
			t.Errorf("%s: Responses %q, want %q", c.name, got, c.want)
		}
	}
}
