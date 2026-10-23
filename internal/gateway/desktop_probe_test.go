package gateway

import (
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

func TestDesktopProbeKeepsItsModel(t *testing.T) {
	for _, c := range []struct{ off, stream bool }{{true, false}, {false, false}, {true, true}, {false, true}} {
		off := c.off
		t.Run(fmt.Sprintf("old_provider_off_%v/stream_%v", off, c.stream), func(t *testing.T) {
			f := &fake{reply: sse(
				`event: message_start`+"\n"+`data: {"type":"message_start","message":{"id":"m","type":"message","role":"assistant","model":"claude-sonnet-5","content":[],"usage":{"input_tokens":1,"output_tokens":0}}}`,
				`event: message_stop`+"\n"+`data: {"type":"message_stop"}`)}
			up := setup(t, provider.Anthropic, f)
			if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Anthropic: up.URL,
				Models: []string{"claude-sonnet-5"}}); err != nil {
				t.Fatal(err)
			}
			desktopPicked.Lock()
			desktopPicked.model, desktopPicked.from = "", ""
			desktopPicked.Unlock()
			// A probe keeps its model whether the previous provider is on or off.
			if err := provider.Save(provider.Provider{ID: "any", Name: "Any", Key: "k", Anthropic: "http://127.0.0.1:1",
				Models: []string{"claude-fable-5-1-reversed"}, Off: off}); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(settings.Dir(), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(desktopPickedPath(), []byte("any/claude-fable-5-1-reversed\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			const electron = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Claude/1.6608.0 Chrome/142.0.7444.235 Electron/39.2.7 Safari/537.36"
			f.got = nil
			req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(fmt.Sprintf(`{"model":"fake/claude-sonnet-5","max_tokens":1,"stream":%v,"messages":[{"role":"user","content":"."}]}`, c.stream)))
			req.Header.Set("x-api-key", Token+"-claude-desktop")
			req.Header.Set("User-Agent", electron)
			rec := httptest.NewRecorder()
			New().Handler().ServeHTTP(rec, req)
			if rec.Code != 200 {
				t.Fatalf("probe: %d %.200s", rec.Code, rec.Body)
			}
			if got := modelOf(f.got); got != "claude-sonnet-5" {
				t.Fatalf("probe sent %q, want claude-sonnet-5", got)
			}
			// a probe is no session: the pick stands
			if b, err := os.ReadFile(desktopPickedPath()); err != nil || strings.TrimSpace(string(b)) != "any/claude-fable-5-1-reversed" {
				t.Fatalf("pick %q, want the stale one kept (%v)", b, err)
			}
			// and a title, known by its prompt, still follows the session's pick:
			// here a session the fake serves
			if err := os.WriteFile(desktopPickedPath(), []byte("fake/claude-sonnet-5\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			desktopPicked.Lock()
			desktopPicked.model, desktopPicked.from = "", ""
			desktopPicked.Unlock()
			f.got = nil
			title := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{"model":"claude-sonnet-5","max_tokens":200,"stream":true,"system":"You write short session titles. Reply with only the tagged fields the prompt asks for.","messages":[{"role":"user","content":"hi"}]}`))
			title.Header.Set("x-api-key", Token+"-claude-desktop")
			title.Header.Set("User-Agent", electron)
			rec = httptest.NewRecorder()
			New().Handler().ServeHTTP(rec, title)
			if rec.Code != 200 {
				t.Fatalf("title: %d %.200s", rec.Code, rec.Body)
			}
			if got := modelOf(f.got); got != "claude-sonnet-5" {
				t.Fatalf("title sent %q, want the session's pick", got)
			}
		})
	}
}
