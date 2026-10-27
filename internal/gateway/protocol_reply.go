package gateway

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/yetone/magpie/internal/provider"
)

// Check only the bounded copy already captured at the client boundary. This
// records a gateway-observed failure; it cannot know whether the client used it.
func (w *captureResponseWriter) protocolError(proto provider.Protocol, streams bool) string {
	if streams || w.status < 200 || w.status >= 300 {
		return ""
	}
	switch proto {
	case provider.Chat, provider.Responses, provider.Anthropic:
	default:
		return ""
	}
	if strings.HasPrefix(strings.ToLower(w.header.Get("Content-Type")), "text/event-stream") {
		return "protocol error: non-streaming request received SSE instead of JSON"
	}
	// A partial or encoded diagnostic copy is not evidence of malformed JSON.
	if w.body.truncated || w.header.Get("Content-Encoding") != "" && w.header.Get("Content-Encoding") != "identity" {
		return ""
	}
	body := bytes.TrimSpace(w.body.buf.Bytes())
	if bytes.HasPrefix(body, []byte("data:")) || bytes.HasPrefix(body, []byte("event:")) || bytes.HasPrefix(body, []byte(":")) {
		return "protocol error: non-streaming request received SSE instead of JSON"
	}
	var reply map[string]json.RawMessage
	if json.Unmarshal(body, &reply) != nil || reply == nil {
		return "protocol error: non-streaming response is not a JSON object"
	}
	field := "choices"
	switch proto {
	case provider.Responses:
		field = "output"
	case provider.Anthropic:
		field = "content"
	}
	var items []json.RawMessage
	if json.Unmarshal(reply[field], &items) != nil || items == nil || proto == provider.Chat && len(items) == 0 {
		return "protocol error: non-streaming response has missing or invalid " + field
	}
	return ""
}
