package gateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/tidwall/gjson"
	"github.com/yetone/magpie/internal/provider"
)

func TestOpenAIReasoningTemperature(t *testing.T) {
	fresh(t)
	for _, tc := range []struct {
		proto               provider.Protocol
		host, model, effort string
		keep                bool
	}{
		{provider.Chat, "api.openai.com", "gpt-6-luna", "medium", false},
		{provider.Chat, "api.openai.com", "gpt-6.1-terra", "", false},
		{provider.Responses, "api.openai.com", "gpt-6.1-terra", "high", false},
		{provider.Chat, "api.openai.com", "gpt-6-luna", "none", true},
		{provider.Responses, "api.openai.com", "gpt-6.1-terra", "none", true},
		{provider.Chat, "api.openai.com", "gpt-4.1", "", true},
		{provider.Chat, "relay.example", "gpt-6-luna", "medium", true},
	} {
		t.Run(fmt.Sprintf("%s/%s/%s/%s", tc.proto, tc.host, tc.model, tc.effort), func(t *testing.T) {
			sent := make(chan []byte, 1)
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				sent <- b
				w.WriteHeader(200)
			}))
			defer proxy.Close()
			base := "http://" + tc.host + "/v1"
			p := provider.Provider{ID: "key", Chat: base, Responses: base, Proxy: proxy.URL, Key: "test"}
			body := []byte(fmt.Sprintf(`{"model":%q,"temperature":0.1,"top_p":0.8}`, tc.model))
			if tc.effort != "" {
				if tc.proto == provider.Responses {
					body = withFields(body, map[string]any{"reasoning": map[string]string{"effort": tc.effort}})
				} else {
					body = withBodyEffort(tc.proto, body, tc.effort)
				}
			}
			res, err := New().forwardOnce(context.Background(), p, tc.proto, upstreamPath(p, tc.proto, tc.model), body, nil)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			got := <-sent
			if temp := gjson.GetBytes(got, "temperature"); temp.Exists() != tc.keep || tc.keep && temp.Float() != 0.1 || bodyEffort(tc.proto, got) != tc.effort || gjson.GetBytes(got, "top_p").Float() != 0.8 || !gjson.GetBytes(body, "temperature").Exists() {
				t.Fatalf("unexpected wire request: %s", got)
			}
		})
	}
}
