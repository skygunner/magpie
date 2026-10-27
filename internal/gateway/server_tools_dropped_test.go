package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A provider-hosted tool a translated request can't carry is named in
// X-Magpie-Tools-Dropped (#3206). A Chat upstream that doesn't search was
// sent no web_search and magpie had no searcher to do it: the model
// answered as if no tool had been offered, and the 200 said nothing about
// it. The cases are ObsidianArch02's own request, web_fetch (which no
// translated API carries at all), Codex's web search and a Gemini client's
// googleSearch.
func TestServerToolsDroppedAreNamed(t *testing.T) {
	chatReply := sse(`data: {"id":"x","choices":[{"delta":{"content":"ok"},"finish_reason":"stop"}]}`, `data: [DONE]`)
	messages := `"max_tokens":1024,"stream":true,"messages":[{"role":"user","content":"What is today's date? Search the web."}]`
	for _, tc := range []struct {
		name, path, body, dropped string
		proto                     provider.Protocol
	}{
		{
			name:    "web search to a Chat upstream, with no searcher",
			path:    "/v1/messages",
			proto:   provider.Chat,
			body:    `{"model":"fake/m1",` + messages + `,"tools":[{"type":"web_search_20260209","name":"web_search","max_uses":5}]}`,
			dropped: "web_search_20260209",
		},
		{
			name:    "web fetch, which a translated request never carries",
			path:    "/v1/messages",
			proto:   provider.Chat,
			body:    `{"model":"fake/m1",` + messages + `,"tools":[{"type":"web_fetch_20250910","name":"web_fetch","max_uses":3}]}`,
			dropped: "web_fetch_20250910",
		},
		{
			name:    "the client's search and fetch together, each named once",
			path:    "/v1/messages",
			proto:   provider.Chat,
			body:    `{"model":"fake/m1",` + messages + `,"tools":[{"type":"web_search_20260209","name":"web_search"},{"type":"web_fetch_20250910","name":"web_fetch"},{"type":"web_search_20260209","name":"web_search"}]}`,
			dropped: "web_search_20260209,web_fetch_20250910",
		},
		{
			name:    "Codex's web search to a Chat upstream",
			path:    "/v1/responses",
			proto:   provider.Chat,
			body:    `{"model":"fake/m1","input":"What is today's date?","tools":[{"type":"web_search"}]}`,
			dropped: "web_search",
		},
		{
			// a Messages relay magpie doesn't know searches is built
			// again for it, the server tool left out
			name:    "a Messages relay that isn't known to search",
			path:    "/v1/messages",
			proto:   provider.Anthropic,
			body:    `{"model":"fake/m1",` + messages + `,"tools":[{"type":"web_search_20250305","name":"web_search","max_uses":5}]}`,
			dropped: "web_search_20250305",
		},
		{
			name:    "a Gemini client's own search to a Chat upstream",
			path:    "/v1beta/models/fake/m1:streamGenerateContent?alt=sse",
			proto:   provider.Chat,
			body:    `{"contents":[{"role":"user","parts":[{"text":"What is today's date?"}]}],"tools":[{"googleSearch":{}}]}`,
			dropped: "googleSearch",
		},
		{
			name:  "a function tool, which is carried, names nothing",
			path:  "/v1/messages",
			proto: provider.Chat,
			body:  `{"model":"fake/m1",` + messages + `,"tools":[{"name":"get_weather","description":"Weather.","input_schema":{"type":"object","properties":{"city":{"type":"string"}}}}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fresh(t)
			f := &fake{t: t, reply: chatReply}
			setup(t, tc.proto, f)
			rec := httptest.NewRecorder()
			New().Handler().ServeHTTP(rec, httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body)))
			if rec.Code != 200 {
				t.Fatalf("%d %s", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("X-Magpie-Tools-Dropped"); got != tc.dropped {
				t.Fatalf("X-Magpie-Tools-Dropped = %q, want %q; upstream got %s", got, tc.dropped, f.got)
			}
			if sent := strings.Contains(string(f.got), `"web_search_`) || strings.Contains(string(f.got), `"web_search"`) ||
				strings.Contains(string(f.got), `"web_fetch_`) || strings.Contains(string(f.got), `"googleSearch"`); sent {
				t.Fatalf("the upstream was sent a server tool %q says it dropped: %s", tc.dropped, f.got)
			}
			if tc.dropped == "" && !strings.Contains(string(f.got), "get_weather") {
				t.Fatalf("the function tool wasn't sent: %s", f.got)
			}
		})
	}
}
