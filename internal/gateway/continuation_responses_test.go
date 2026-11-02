package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// cutResponses is a Responses upstream that severs the reply mid-stream on
// the calls cut names: the events given go out with a Content-Length
// promising more, then the connection closes. Other calls answer whole.
// Bodies journals what each call was asked.
type cutResponses struct {
	mu     sync.Mutex
	bodies []map[string]any
	cuts   map[int][]string
	whole  []string
}

func (f *cutResponses) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	t.Helper()
	var body map[string]any
	b, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(b, &body); err != nil {
		t.Errorf("upstream request isn't JSON: %s", b)
	}
	f.mu.Lock()
	call := len(f.bodies)
	f.bodies = append(f.bodies, body)
	events, cut := f.cuts[call]
	whole := f.whole
	f.mu.Unlock()
	if !cut {
		events = whole
	}
	writeStream(w, events, cut, "")
}

func (f *cutResponses) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.bodies)
}

// input is what call was asked with, as Responses input items.
func (f *cutResponses) input(t *testing.T, call int) []any {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	in, _ := f.bodies[call]["input"].([]any)
	return in
}

func respCreated() string {
	return `{"type":"response.created","response":{"id":"resp_1","status":"in_progress"}}`
}

func respText(s string) string {
	return `{"type":"response.output_text.delta","item_id":"msg_1","output_index":0,"delta":` + jsonStr(s) + `}`
}

func respCompleted() string {
	return `{"type":"response.completed","response":{"id":"resp_1","status":"completed","output":[],"usage":{"input_tokens":5,"output_tokens":3}}}`
}

// continuationResponsesServer points a provider whose only API is
// Responses at f, and answers a Messages request for its model: the reply
// is translated, as a Claude client's is (#3166).
func continuationResponsesServer(t *testing.T, model string, f *cutResponses) string {
	t.Helper()
	fresh(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, w, r) }))
	t.Cleanup(up.Close)
	p := provider.Provider{ID: "up", Name: "UP", Key: "k", Models: []string{model}, Responses: up.URL + "/v1"}
	if err := provider.Save(p); err != nil {
		t.Fatal(err)
	}
	code, body := post(t, "/v1/messages", `{"model":"up/`+model+`","max_tokens":32000,"stream":true,"messages":[{"role":"user","content":"hi"}]}`)
	if code != 200 {
		t.Fatalf("status %d: %s", code, body)
	}
	return body
}

// A Responses upstream takes no prefill, so a reply it cut after some text
// goes on with the shown part sent back and a user turn asking the model
// on: the client reads one whole reply, not the cut's error.
func TestStreamCutMidReplyContinuesOnResponses(t *testing.T) {
	f := &cutResponses{
		cuts:  map[int][]string{0: {respCreated(), respText("Let me look at the")}},
		whole: []string{respCreated(), respText(" rest of it."), respCompleted()},
	}
	body := continuationResponsesServer(t, "gpt-6.1", f)
	if f.calls() != 2 {
		t.Fatalf("calls = %d, want 2", f.calls())
	}
	if got := textOf(body); got != "Let me look at the rest of it." {
		t.Fatalf("client's text = %q, want %q", got, "Let me look at the rest of it.")
	}
	if starts, stops, errs := messageStops(body); starts != 1 || stops != 1 || errs != 0 {
		t.Fatalf("starts %d, stops %d, errors %d, want 1/1/0: %s", starts, stops, errs, body)
	}
	in := f.input(t, 1)
	if len(in) < 2 {
		t.Fatalf("the second ask's input is %v, want the shown part and a user turn", in)
	}
	prev, _ := in[len(in)-2].(map[string]any)
	if prev["role"] != "assistant" {
		t.Fatalf("the second ask didn't send the shown part back as the assistant's: %v", prev)
	}
	shown, _ := prev["content"].([]any)
	if len(shown) != 1 {
		t.Fatalf("the shown part came back as %v", prev["content"])
	}
	if text, _ := shown[0].(map[string]any)["text"]; text != "Let me look at the" {
		t.Fatalf("the second ask's shown part = %q, want %q", text, "Let me look at the")
	}
	last, _ := in[len(in)-1].(map[string]any)
	said, _ := last["content"].([]any)
	if last["role"] != "user" || len(said) != 1 {
		t.Fatalf("the second ask doesn't end with a user turn asking it on: %v", last)
	}
	if text, _ := said[0].(map[string]any)["text"]; text != continueText {
		t.Fatalf("the user turn = %q, want %q", text, continueText)
	}
}

// A cut before the upstream said anything has nothing to go on from: it
// is asked again as it was, with nothing put in the model's mouth, not
// continued with a user turn after a shown part.
func TestStreamCutBeforeFirstByteResponsesNotContinued(t *testing.T) {
	f := &cutResponses{
		cuts:  map[int][]string{0: {respCreated()}},
		whole: []string{respCreated(), respText("the whole answer"), respCompleted()},
	}
	body := continuationResponsesServer(t, "gpt-6.1", f)
	if f.calls() != 2 {
		t.Fatalf("calls = %d, want the one empty reply asked again once", f.calls())
	}
	if got := textOf(body); got != "the whole answer" {
		t.Fatalf("client's text = %q, want %q", got, "the whole answer")
	}
	if _, _, errs := messageStops(body); errs != 0 {
		t.Fatalf("error events in the stream: %s", body)
	}
	in := f.input(t, 1)
	if len(in) != 1 {
		t.Fatalf("the re-ask carried a reply that never started: %v", in)
	}
	if last, _ := in[0].(map[string]any); last["role"] != "user" {
		t.Fatalf("the re-ask didn't end with the client's own user turn: %v", in)
	}
}

// A Responses reply whose every ask is cut is asked on once, not
// streamRetries times: the model may answer anew, so the shown part goes
// back only that one time.
func TestStreamCutResponsesContinuesOnlyOnce(t *testing.T) {
	f := &cutResponses{cuts: map[int][]string{
		0: {respCreated(), respText("Let me look at the")},
		1: {respCreated(), respText("Let me look at the")},
	}}
	body := continuationResponsesServer(t, "gpt-6.1", f)
	if f.calls() != 2 {
		t.Fatalf("calls = %d, want 2", f.calls())
	}
	if got := textOf(body); got != "Let me look at the" {
		t.Fatalf("client's text = %q, want the shown part once", got)
	}
	if _, _, errs := messageStops(body); errs != 1 {
		t.Fatalf("no error event after the tries ran out: %s", body)
	}
}
