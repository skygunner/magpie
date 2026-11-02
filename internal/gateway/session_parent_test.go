package gateway

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"
)

func TestTitleParentSession(t *testing.T) {
	for _, tc := range []struct {
		name, kind, header, body, parent, want string
	}{
		{name: "forked title", kind: "thread_title", header: `{"forked_from_thread_id":"main-a"}`, want: "main-a"},
		{name: "reconsidered title", kind: "thread_title_reconsideration", header: `{"parent_thread_id":"main-b"}`, want: "main-b"},
		{name: "ordinary fork stays separate", header: `{"forked_from_thread_id":"main-a"}`},
		{name: "other helper stays separate", kind: "guardian", parent: "main-a"},
		{name: "canonical body", kind: "thread_title", header: `{"forked_from_thread_id":"wrong"}`, body: `{"client_metadata":{"x-codex-turn-metadata":"{\"forked_from_thread_id\":\"main-b\"}"}}`, want: "main-b"},
		{name: "object metadata", kind: "thread_title", body: `{"client_metadata":{"x-codex-turn-metadata":{"parent_thread_id":"main-a"}}}`, want: "main-a"},
		{name: "parent header", kind: "thread_title", parent: "main-c", want: "main-c"},
		{name: "flat body", kind: "thread_title", body: `{"client_metadata":{"thread_source":"thread_title","x-codex-parent-thread-id":"main-flat"}}`, want: "main-flat"},
		{name: "partially malformed", kind: "thread_title", header: `{"parent_thread_id":"wrong","thread_source":7}`},
		{name: "malformed", kind: "thread_title", header: `{broken`},
		{name: "no ancestry", kind: "thread_title", header: `{"session_id":"aux","thread_id":"aux"}`},
		{name: "self reference", kind: "thread_title", parent: "aux"},
		{name: "overlong", kind: "thread_title", parent: strings.Repeat("x", 129)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			h.Set("session_id", "aux")
			h.Set("x-codex-turn-metadata", tc.header)
			h.Set("x-codex-parent-thread-id", tc.parent)
			if got := titleParentSession(h, requestSessionMetadata(h, []byte(tc.body)), tc.kind); got != tc.want {
				t.Fatalf("parent=%q, want %q", got, tc.want)
			}
			if sessionOf(h) != "aux" {
				t.Fatal("title grouping changed the request's routing identity")
			}
		})
	}
}

func TestRequestCallKindBody(t *testing.T) {
	meta, _ := json.Marshal(map[string]string{"thread_source": "thread_title", "forked_from_thread_id": "main"})
	body, _ := json.Marshal(map[string]any{"client_metadata": map[string]string{"x-codex-turn-metadata": string(meta)}})
	if got := requestCallKind(http.Header{}, requestSessionMetadata(http.Header{}, body)); got != "thread_title" {
		t.Fatalf("body-only title kind=%q", got)
	}
}

// A thread_sample turn names a prediction by its turn_trigger, read on
// every transport thread_source is (#3903, zhouhaochengZz): the metadata
// blob in the body, as an object or a JSON string, the flat
// client_metadata projections, and the header. The report's client sent
// no trigger; a trigger other than composer_predictions is some other
// sample, not a prediction; and a subagent turn stays a subagent turn.
func TestThreadSampleTurnTrigger(t *testing.T) {
	for _, tc := range []struct{ name, trigger, want string }{
		{"no trigger", "", "composer_predictions"},
		{"the prediction's own trigger", "composer_predictions", "composer_predictions"},
		{"padded", " composer_predictions ", "composer_predictions"},
		{"another trigger", "user_submit", "thread_sample"},
	} {
		for _, transport := range []string{"blob object", "blob string", "flat", "header"} {
			t.Run(tc.name+"/"+transport, func(t *testing.T) {
				blob := `{"thread_source":"thread_sample"`
				flat := ""
				if tc.trigger != "" {
					blob += `,"turn_trigger":` + strconv.Quote(tc.trigger)
					flat = `,"turn_trigger":` + strconv.Quote(tc.trigger)
				}
				blob += `,"forked_from_thread_id":"main"}`
				h := http.Header{}
				var body []byte
				switch transport {
				case "blob object":
					body, _ = json.Marshal(map[string]any{"client_metadata": map[string]json.RawMessage{"x-codex-turn-metadata": json.RawMessage(blob)}})
				case "blob string":
					body, _ = json.Marshal(map[string]any{"client_metadata": map[string]string{"x-codex-turn-metadata": blob}})
				case "flat":
					body = []byte(`{"client_metadata":{"thread_source":"thread_sample"` + flat + `}}`)
				case "header":
					h.Set("x-codex-turn-metadata", blob)
				}
				if got := requestCallKind(h, requestSessionMetadata(h, body)); got != tc.want {
					t.Fatalf("%s: kind=%q, want %q", transport, got, tc.want)
				}
			})
		}
	}
	h := http.Header{}
	h.Set("x-codex-turn-metadata", `{"thread_source":"thread_sample","turn_trigger":"composer_predictions"}`)
	h.Set("x-openai-subagent", "review")
	if got := requestCallKind(h, requestSessionMetadata(h, nil)); got != "review" {
		t.Fatalf("a subagent turn under a prediction's names: kind=%q", got)
	}
}

// Ordinary thread metadata must still allow the header-based helper labels.
func TestRequestCallKindFallbacks(t *testing.T) {
	for _, source := range []string{"user", "subagent", ""} {
		for _, transport := range []string{"body", "header"} {
			for _, tc := range []struct{ name, header, value, want string }{
				{"ordinary", "", "", ""},
				{"reserve", "x-openai-codex-luna-reserve", "1", "luna_reserve"},
				{"search", "User-Agent", SearchAgent, "web_search"},
				{"memory", "x-openai-memgen-request", "1", "memgen"},
				{"subagent", "x-openai-subagent", "review", "review"},
			} {
				t.Run(source+"/"+transport+"/"+tc.name, func(t *testing.T) {
					h := http.Header{}
					if tc.header != "" {
						h.Set(tc.header, tc.value)
					}
					meta, _ := json.Marshal(sessionMetadata{Source: source})
					var body []byte
					if transport == "body" {
						body, _ = json.Marshal(map[string]any{"client_metadata": map[string]string{"x-codex-turn-metadata": string(meta)}})
					} else {
						h.Set("x-codex-turn-metadata", string(meta))
					}
					m := requestSessionMetadata(h, body)
					if got := requestCallKind(h, m); got != tc.want {
						t.Fatalf("kind=%q, want %q", got, tc.want)
					}
				})
			}
		}
	}
	for _, tc := range []struct{ header, value, want string }{
		{"", "", "thread_title"},
		{"x-openai-codex-luna-reserve", "1", "thread_title"},
		{"x-openai-memgen-request", "1", "memgen"},
		{"x-openai-subagent", "review", "review"},
	} {
		h := http.Header{}
		if tc.header != "" {
			h.Set(tc.header, tc.value)
		}
		if got := requestCallKind(h, sessionMetadata{Source: "thread_title"}); got != tc.want {
			t.Fatalf("helper precedence: %q, want %q", got, tc.want)
		}
	}
}

func BenchmarkRequestSessionMetadata(b *testing.B) {
	body := []byte(`{"input":"` + strings.Repeat("x", 5800000) + `","client_metadata":{"x-codex-turn-metadata":{"thread_source":"thread_title","parent_thread_id":"main"}}}`)
	h := http.Header{}
	h.Set("session_id", "title")
	b.SetBytes(int64(len(body)))
	for b.Loop() {
		m := requestSessionMetadata(h, body)
		kind := requestCallKind(h, m)
		if titleParentSession(h, m, kind) != "main" {
			b.Fatal("lost parent")
		}
	}
}

// Only client_metadata is decoded now (#3394); what a body that isn't the
// usual shape gives stays what decoding the whole body gave.
func TestSessionMetadataOddBodies(t *testing.T) {
	h := http.Header{}
	h.Set("x-codex-turn-metadata", `{"turn_id":"from-header","parent_thread_id":"head"}`)
	long := `"input":"` + strings.Repeat("x", 300000) + `",`
	for _, tc := range []struct{ name, body, turn, parent string }{
		{"metadata a string", `{"client_metadata":"x"}`, "from-header", "head"},
		{"metadata null", `{"client_metadata":null}`, "from-header", "head"},
		{"body cut short", `{"client_metadata":{"parent_thread_id":"body"}`, "from-header", "head"},
		{"body an array", `[{"client_metadata":{"parent_thread_id":"body"}}]`, "from-header", "head"},
		{"nested name only", `{"input":[{"client_metadata":{"parent_thread_id":"body"}}]}`, "from-header", "head"},
		{"last of two", `{"client_metadata":{"parent_thread_id":"a"},"client_metadata":{"parent_thread_id":"b"}}`, "", "b"},
		{"long body", `{` + long + `"client_metadata":{"turn_id":"t","parent_thread_id":"body"}}`, "t", "body"},
		{"long body cut short", `{` + long + `"client_metadata":{"turn_id":"t","parent_thread_id":"body"}`, "from-header", "head"},
	} {
		m := requestSessionMetadata(h, []byte(tc.body))
		if m.Turn != tc.turn || m.Parent != tc.parent {
			t.Errorf("%s: turn %q parent %q, want %q %q", tc.name, m.Turn, m.Parent, tc.turn, tc.parent)
		}
	}
}
