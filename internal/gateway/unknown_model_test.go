package gateway

import (
	"net/http"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// MOMO on Discord asked for "Zen-Space-Bunny", the name a picker showed,
// and got a 404 that named neither the id meant nor how to name one.
func TestUnknownModelSaysWhatToAsk(t *testing.T) {
	fresh(t)
	serveOn(t, "opencode-zen-free", "k", []string{"big-pickle", "space-bunny-free", "exo-free"}, &keyed{})
	serveOn(t, "opencode-go", "kg", []string{"glm-5.3", "kimi-k3"}, &keyed{})
	if err := provider.SetModelName("opencode-zen-free/space-bunny-free", "Space Bunny Free"); err != nil {
		t.Fatal(err)
	}
	s := New()
	code, body := postAs(t, s, "", `{"model":"Zen-Space-Bunny","messages":[{"role":"user","content":"hi"}]}`)
	if code != http.StatusNotFound {
		t.Fatalf("%d %s", code, body)
	}
	if !strings.Contains(body, "did you mean opencode-zen-free/space-bunny-free") {
		t.Fatalf("the id meant isn't named first: %s", body)
	}
	for _, form := range []string{"provider/model", "routing group's id or name", "bare model id", "Providers order"} {
		if !strings.Contains(body, form) {
			t.Fatalf("doesn't say %q: %s", form, body)
		}
	}
	// the provider's name alone matches nothing
	if _, body := postAs(t, s, "", `{"model":"zen-nothing","messages":[{"role":"user","content":"hi"}]}`); strings.Contains(body, "did you mean") {
		t.Fatalf("a provider's name alone was a match: %s", body)
	}
}

// A model's display name counts before an id that spells the same words.
func TestClosestByNameFirst(t *testing.T) {
	fresh(t)
	serveOn(t, "a", "ka", []string{"m1"}, &keyed{})
	serveOn(t, "b", "kb", []string{"space-bunny-old"}, &keyed{})
	if err := provider.SetModelName("a/m1", "Space Bunny"); err != nil {
		t.Fatal(err)
	}
	got := provider.Closest("space bunny", 3)
	if len(got) != 2 || got[0] != "a/m1" || got[1] != "b/space-bunny-old" {
		t.Fatalf("Closest = %v", got)
	}
}

// The refusal for an unknown model names the id the client sent, not the
// id the gateway goes by when it differs (a found group's auto/<model> as
// group/auto-<slug>, dashes for dots): that one exists nowhere a client
// can look — not in /v1/models, not in magpie groups — so a search of the
// error string finds nothing (#3150, xiachaoxia-glitch). An id the gateway
// keeps as it is echoes as before.
func TestUnknownModelNamesTheIdSent(t *testing.T) {
	fresh(t)
	serveOn(t, "opencode-zen-free", "k", []string{"deepseek-v4.1-flash-free"}, &keyed{})
	s := New()
	for _, c := range []struct{ sent, not string }{
		{"auto/deepseek-v4.1-flash", "group/auto-deepseek-v4-1-flash"},
		{"nosuchgroup/foo", ""},
	} {
		code, body := postAs(t, s, "", `{"model":"`+c.sent+`","messages":[{"role":"user","content":"hi"}]}`)
		if code != http.StatusNotFound {
			t.Fatalf("%s: %d %s", c.sent, code, body)
		}
		if !strings.Contains(body, `no model \"`+c.sent+`\"`) {
			t.Fatalf("%s: the id sent isn't named: %s", c.sent, body)
		}
		if c.not != "" && strings.Contains(body, c.not) {
			t.Fatalf("%s: the synthesized id is named: %s", c.sent, body)
		}
	}
	if _, body := postAs(t, s, "", `{"model":"auto/deepseek-v4.1-flash","messages":[{"role":"user","content":"hi"}]}`); !strings.Contains(body, "did you mean opencode-zen-free/deepseek-v4.1-flash-free") {
		t.Fatalf("the suggestions are gone: %s", body)
	}
}
