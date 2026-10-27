package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// The reporter's own bytes (#3724): a screenshot a tool gave back, sent as
// a data: URL in the tool's text. Counted as tokens it is 167,598 tokens of
// base64 in one turn, and a vendor that reads it as a prompt answers
// invalid_prompt from then on.
func TestToolResultImageURLInTextIsTheImage(t *testing.T) {
	fresh(t)
	u := &eyeUp{}
	up := httptest.NewServer(u)
	t.Cleanup(up.Close)
	if err := provider.Save(provider.Provider{ID: "probe", Name: "Probe", Chat: up.URL + "/v1", Responses: up.URL + "/v1", Key: "key", Models: []string{"text", "eye"}}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveLive("probe", up.URL+"/v1", []catalog.Model{
		{ID: "text", ImageInput: imageInputBool(false)},
		{ID: "eye", Images: true, ImageInput: imageInputBool(true)},
	}); err != nil {
		t.Fatal(err)
	}
	st := settings.Load()
	st.Vision = "probe/eye"
	if err := settings.Save(st); err != nil {
		t.Fatal(err)
	}
	// 670,392 characters of base64: the turn the reporter saw at 167,598 tokens
	big := "data:image/png;base64,iVBORw0KGgo" + strings.Repeat("A", 670376)
	call := `{"role":"user","content":[{"type":"input_text","text":"hello"}]},{"type":"function_call","name":"view_image","call_id":"c1","arguments":"{}"}`
	cases := []struct {
		name, path, body string
		from             provider.Protocol
	}{
		{"responses-string-output", "/v1/responses",
			`{"model":"probe/text","stream":false,"input":[` + call + `,{"type":"function_call_output","call_id":"c1","output":"` + big + `"}]}`,
			provider.Responses},
		{"custom-tool-string-output", "/v1/responses",
			`{"model":"probe/text","stream":false,"input":[` + call + `,{"type":"custom_tool_call_output","call_id":"c1","output":"` + big + `"}]}`,
			provider.Responses},
		{"chat-tool-string-content", "/v1/chat/completions",
			`{"model":"probe/text","messages":[{"role":"user","content":"hello"},{"role":"assistant","content":null,"tool_calls":[{"id":"c1","type":"function","function":{"name":"view_image","arguments":"{}"}}]},{"role":"tool","tool_call_id":"c1","content":"` + big + `"}]}`,
			provider.Chat},
		{"anthropic-tool-result-string", "/v1/messages",
			`{"model":"probe/text","max_tokens":16,"messages":[{"role":"user","content":[{"type":"text","text":"hello"}]},{"role":"assistant","content":[{"type":"tool_use","id":"c1","name":"view_image","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"c1","content":"` + big + `"}]}]}`,
			provider.Anthropic},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if p := promptOf(tc.from, []byte(tc.body)); p == nil || p.Tokens > 5000 {
				t.Fatalf("the turn is counted as its base64: %+v", p)
			}
			rec := httptest.NewRecorder()
			New().Handler().ServeHTTP(rec, httptest.NewRequest("POST", tc.path, strings.NewReader(tc.body)))
			if rec.Code != 200 {
				t.Fatalf("%s: %d %s", tc.path, rec.Code, rec.Body.String())
			}
			sent := u.sent("text")
			if len(sent) == 0 {
				t.Fatal("the model was sent nothing")
			}
			last := sent[len(sent)-1]
			if strings.Contains(last, "iVBORw0KGgo") {
				t.Fatalf("the base64 was sent to a model that can't see it: %d bytes", len(last))
			}
			if !strings.Contains(last, "HELLO") {
				t.Fatalf("the image wasn't described to the model: %s", last)
			}
		})
	}
}
