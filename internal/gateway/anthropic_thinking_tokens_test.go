package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
)

// #3430: a Messages reply says how much of its output was thinking in
// output_tokens_details.thinking_tokens, as Copilot's message_delta did
// (output 1277, 710 of them thinking), and magpie read none of it: every
// Claude reply over the Messages API was in the ledger with reasoning 0,
// relayed or translated. A Chat or Responses upstream's reasoning went out
// to a Claude client with no thinking_tokens either.
func TestMessagesReplyCountsItsThinkingTokens(t *testing.T) {
	// relayed: Copilot's reply as it arrived, its text and signature cut
	t.Run("relayed", func(t *testing.T) {
		fresh(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w,
				"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-opus-5-5\",\"content\":[],\"usage\":{\"cache_creation_input_tokens\":3858,\"cache_read_input_tokens\":12030,\"input_tokens\":2,\"output_tokens\":1}}}\n\n"+
					"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n"+
					"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"done\"}}\n\n"+
					"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n"+
					"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"input_tokens\":2,\"output_tokens\":1277,\"output_tokens_details\":{\"thinking_tokens\":710}}}\n\n"+
					"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
		}))
		t.Cleanup(srv.Close)
		if err := provider.Save(provider.Provider{ID: "relay", Name: "Relay", Key: "k", Models: []string{"claude-opus-5-5"}, Anthropic: srv.URL}); err != nil {
			t.Fatal(err)
		}
		gw := httptest.NewServer(New().Handler())
		t.Cleanup(gw.Close)
		resp, err := http.Post(gw.URL+"/v1/messages", "application/json",
			strings.NewReader(`{"model":"relay/claude-opus-5-5","max_tokens":64,"stream":true,"messages":[{"role":"user","content":"hi"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("%d %s", resp.StatusCode, body)
		}
		records := usage.Load(time.Time{})
		if len(records) != 1 {
			t.Fatalf("ledger has %d records: %s", len(records), body)
		}
		if u := records[0]; u.Output != 1277 || u.Reasoning != 710 {
			t.Errorf("ledger says output %d, reasoning %d; want the reply's 1277 and 710", u.Output, u.Reasoning)
		}
	})

	// translated: a Chat upstream's reasoning, told to a Claude client
	t.Run("translated", func(t *testing.T) {
		fresh(t)
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			io.Copy(io.Discard, r.Body)
			w.Header().Set("Content-Type", "text/event-stream")
			io.WriteString(w,
				"data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1791500000,\"model\":\"glm-5.3\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"reasoning_content\":\"thinking it over\"}}]}\n\n"+
					"data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1791500000,\"model\":\"glm-5.3\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"done\"}}]}\n\n"+
					"data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"created\":1791500000,\"model\":\"glm-5.3\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":140,\"total_tokens\":152,\"completion_tokens_details\":{\"reasoning_tokens\":100}}}\n\n"+
					"data: [DONE]\n\n")
		}))
		t.Cleanup(srv.Close)
		if err := provider.Save(provider.Provider{ID: "z", Name: "Z", Key: "k", Models: []string{"glm-5.3"}, Chat: srv.URL + "/v1"}); err != nil {
			t.Fatal(err)
		}
		gw := httptest.NewServer(New().Handler())
		t.Cleanup(gw.Close)
		resp, err := http.Post(gw.URL+"/v1/messages", "application/json",
			strings.NewReader(`{"model":"z/glm-5.3","max_tokens":64,"stream":true,"messages":[{"role":"user","content":"hi"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Fatalf("%d %s", resp.StatusCode, body)
		}
		if !strings.Contains(string(body), `"output_tokens_details":{"thinking_tokens":100}`) {
			t.Errorf("the message_delta doesn't tell the thinking tokens: %s", body)
		}
		records := usage.Load(time.Time{})
		if len(records) != 1 || records[0].Reasoning != 100 {
			t.Errorf("ledger: %+v", records)
		}
	})
}
