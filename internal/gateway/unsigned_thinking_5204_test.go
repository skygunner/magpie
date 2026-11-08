package gateway

import (
	"context"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/provider"
)

// A Messages relay must not send another provider's unsigned thinking to a strict endpoint (#5204).
func TestMessagesRelayDropsUnsignedThinking(t *testing.T) {
	f := &fake{t: t, ctype: "application/json", reply: anthropicWholeOK, refuse: func(body []byte) (int, string) {
		for _, m := range gjson.GetBytes(body, "messages").Array() {
			for _, b := range m.Get("content").Array() {
				if b.Get("type").String() == "thinking" && b.Get("signature").String() == "" {
					return 400, `{"detail":"Parameter validation error","status":"FAILED"}`
				}
			}
		}
		return 0, ""
	}}
	up := setup(t, provider.Anthropic, f)
	if err := provider.Save(provider.Provider{ID: "strict", Name: "Strict", Key: "k", Models: []string{"claude-opus-5-5"}, Anthropic: up.URL}); err != nil {
		t.Fatal(err)
	}
	body := `{"model":"strict/claude-opus-5-5","max_tokens":128000,"thinking":{"type":"adaptive"},"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":[{"type":"thinking","thinking":"let me think","signature":""},{"type":"text","text":"hello"}]},{"role":"user","content":"again"}]}`
	code, answer := postTo(t, New(), "/v1/messages", body)
	if code != 200 || f.calls != 1 {
		t.Fatalf("strict endpoint: status %d, calls %d, reply %s, sent %s", code, f.calls, answer, f.got)
	}
	sameJSON(t, sentAnthropicMessages(t, f.got)[1], `{"role":"assistant","content":[{"type":"text","text":"hello"}]}`)

	f.calls = 0
	mixed := `{"model":"strict/claude-opus-5-5","max_tokens":128000,"thinking":{"type":"adaptive"},"tools":[{"name":"read","input_schema":{"type":"object"}}],"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":[{"type":"thinking","thinking":"gone","signature":""},{"type":"thinking","thinking":"missing"},{"type":"thinking","thinking":"signed","signature":"valid-signature"},{"type":"redacted_thinking","data":"opaque"},{"type":"text","text":"hello"},{"type":"tool_use","id":"toolu_1","name":"read","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}]}`
	code, answer = postTo(t, New(), "/v1/messages", mixed)
	if code != 200 || f.calls != 1 {
		t.Fatalf("mixed history: status %d, calls %d, reply %s", code, f.calls, answer)
	}
	ms := sentAnthropicMessages(t, f.got)
	sameJSON(t, ms[1], `{"role":"assistant","content":[{"type":"thinking","thinking":"signed","signature":"valid-signature"},{"type":"redacted_thinking","data":"opaque"},{"type":"text","text":"hello"},{"type":"tool_use","id":"toolu_1","name":"read","input":{}}]}`)
	sameJSON(t, ms[2], `{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]}`)
}

func TestDeepSeekMessagesKeepUnsignedThinking(t *testing.T) {
	for _, tc := range []struct{ name, host, model string }{
		{"model", "127.0.0.1", "deepseek-v4-pro"},
		{"host", "api.deepseek.com", "other-model"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fake{t: t, ctype: "application/json", reply: anthropicWholeOK, refuse: deepseekRefuseAnthropic}
			up := setup(t, provider.Anthropic, f)
			endpoint := up.URL
			srv := New()
			if tc.host != "127.0.0.1" {
				endpoint = strings.Replace(up.URL, "127.0.0.1", tc.host, 1)
				transport := http.DefaultTransport.(*http.Transport).Clone()
				transport.Proxy = nil
				transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
					return (&net.Dialer{}).DialContext(ctx, network, up.Listener.Addr().String())
				}
				srv.client = &http.Client{Transport: transport}
				t.Cleanup(transport.CloseIdleConnections)
			}
			if err := provider.Save(provider.Provider{ID: "ds", Name: "DS", Key: "k", Models: []string{tc.model}, Anthropic: endpoint}); err != nil {
				t.Fatal(err)
			}
			body := `{"model":"ds/` + tc.model + `","max_tokens":512,"messages":[{"role":"user","content":"hi"},{"role":"assistant","content":[{"type":"thinking","thinking":"keep me","signature":""},{"type":"tool_use","id":"toolu_1","name":"read","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"ok"}]},{"role":"assistant","content":[{"type":"text","text":"done"}]},{"role":"user","content":"again"}]}`
			code, answer := postTo(t, srv, "/v1/messages", body)
			if code != 200 || f.calls != 1 {
				t.Fatalf("status %d, calls %d, reply %s", code, f.calls, answer)
			}
			ms := sentAnthropicMessages(t, f.got)
			if !strings.Contains(string(ms[1]), `"thinking":"keep me"`) {
				t.Fatalf("unsigned thought lost: %s", ms[1])
			}
			if !strings.Contains(string(ms[3]), `"type":"thinking"`) {
				t.Fatalf("missing tool-result continuation thought: %s", ms[3])
			}
		})
	}
}
