package gateway

import (
	"encoding/json"
	"github.com/yetone/magpie/internal/provider"
	"slices"
	"testing"
)

const nvidiaParameters = "{\"message\":\"Validation: Unsupported parameter(s): `reasoning`, `enable_thinking`\"}"

func TestNvidiaRedundantReasoningCompatibility(t *testing.T) {
	f := &fake{t: t, reply: optionalFieldReply(provider.Chat)}
	f.refuse = func(b []byte) (int, string) {
		var m map[string]json.RawMessage
		json.Unmarshal(b, &m)
		if m["reasoning"] != nil || m["enable_thinking"] != nil {
			return 400, nvidiaParameters
		}
		if string(m["reasoning_effort"]) != `"high"` {
			t.Errorf("lost effort: %s", b)
		}
		return 0, ""
	}
	setup(t, provider.Chat, f)
	s := New()
	path, body := optionalFieldAsk(provider.Chat)
	request := string(withFields([]byte(body), map[string]any{"reasoning_effort": "high", "reasoning": map[string]string{"effort": "high"}, "enable_thinking": true, "thinking": map[string]string{"type": "enabled"}}))
	for _, want := range []int{2, 1} {
		before := f.calls
		code, reply := postTo(t, s, path, request)
		if code != 200 || f.calls-before != want {
			t.Fatalf("status %d calls %d want %d: %s", code, f.calls-before, want, reply)
		}
	}
	// A cached refusal must not remove a later request's only reasoning control.
	for _, reasoning := range []string{`{"effort":"high"}`, `{"effort":"low"}`, `{"effort":"high","summary":"auto"}`} {
		b := []byte(`{"reasoning":` + reasoning + `}`)
		if reasoning != `{"effort":"high"}` {
			b = withFields(b, map[string]any{"reasoning_effort": "high"})
		}
		if got := s.withoutRefused("fake", provider.Chat, b); string(got) != string(b) {
			t.Fatalf("nonredundant reasoning lost: %s -> %s", b, got)
		}
	}
}

func TestNvidiaParameterListBoundaries(t *testing.T) {
	body := []byte(`{"reasoning_effort":"high","reasoning":{"effort":"high"},"enable_thinking":true}`)
	if got := refusedOptional(400, []byte(nvidiaParameters), body); !slices.Contains(got, "reasoning") || !slices.Contains(got, "enable_thinking") {
		t.Fatalf("missed rejected fields: %v", got)
	}
	for _, msg := range []string{
		"Unsupported parameter(s): `reasoning.effort`, `enable_thinking.foo`",
		"Unsupported value: `reasoning`, `enable_thinking`",
		`{"detail":[{"loc":["body","metadata"],"msg":"Invalid value","input":"Validation: Unsupported parameter(s): ` + "`reasoning`, `enable_thinking`" + `"}]}`,
	} {
		if got := refusedOptional(400, []byte(msg), body); len(got) != 0 {
			t.Errorf("misclassified %s: %v", msg, got)
		}
	}
	for _, b := range []string{`{"reasoning":{"effort":"high"}}`, `{"reasoning_effort":"low","reasoning":{"effort":"high"}}`, `{"reasoning_effort":"high","reasoning":{"effort":"high","summary":"auto"}}`} {
		if got := refusedOptional(400, []byte(nvidiaParameters), []byte(b)); slices.Contains(got, "reasoning") {
			t.Errorf("unsafe removal: %s", b)
		}
	}
}
