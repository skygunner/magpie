package gateway

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

func TestLastStreamKeepsEvents(t *testing.T) {
	for _, tc := range []struct{ name, first, tail string }{
		{"content then overflow", `data: {"choices":[{"delta":{"content":"hello"}}]}`, `data: {"error":{"message":"Context window is full"}}`},
		{"other error", `data: {"error":{"message":"vendor refused","type":"vendor_error","code":"vendor_code"}}`, ""},
		{"CRLF content", `data: {"choices":[{"delta":{"content":"hello"}}]}`, ""},
		{"comment heartbeat", `: vendor-heartbeat`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fresh(t)
			setup(t, provider.Chat, &fake{})
			release := make(chan struct{})
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				ending := "\n\n"
				if tc.name == "CRLF content" {
					ending = "\r\n\r\n"
				}
				fmt.Fprint(w, tc.first+ending)
				w.(http.Flusher).Flush()
				select {
				case <-release:
				case <-r.Context().Done():
					return
				}
				if tc.tail != "" {
					fmt.Fprint(w, tc.tail+"\n\n")
				}
			}))
			defer up.Close()
			closed := false
			defer func() {
				if !closed {
					close(release)
				}
			}()
			if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Models: []string{"m1"}, Chat: up.URL + "/v1", PassErrors: tc.name == "comment heartbeat"}); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(New().Handler())
			defer func() {
				if !closed {
					close(release)
					closed = true
				}
				server.Close()
			}()
			client := &http.Client{Timeout: 15 * time.Second}
			res, err := client.Post(server.URL+"/v1/chat/completions", "application/json", bytes.NewBufferString(`{"model":"fake/m1","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != 200 {
				t.Fatalf("status %d", res.StatusCode)
			}
			r := bufio.NewReader(res.Body)
			line, err := r.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			if strings.TrimSpace(line) != tc.first {
				t.Fatalf("first event %q", line)
			}
			close(release)
			closed = true
			tail, err := io.ReadAll(r)
			if err != nil {
				t.Fatal(err)
			}
			if tc.tail != "" && !strings.Contains(string(tail), tc.tail) {
				t.Errorf("missing unchanged later event: %s", tail)
			}
		})
	}
}

func TestLastStreamEOF(t *testing.T) {
	for _, body := range []string{"data: {\"incomplete\":", ""} {
		name := "partial event"
		if body == "" {
			name = "empty stream"
		}
		t.Run(name, func(t *testing.T) {
			fresh(t)
			setup(t, provider.Chat, &fake{})
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, body)
			}))
			defer up.Close()
			if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Models: []string{"m1"}, Chat: up.URL + "/v1"}); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(New().Handler())
			defer server.Close()
			client := &http.Client{Timeout: 15 * time.Second}
			res, err := client.Post(server.URL+"/v1/chat/completions", "application/json", bytes.NewBufferString(`{"model":"fake/m1","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			got, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatal(err)
			}
			if body == "" {
				if res.StatusCode != http.StatusBadGateway || !strings.Contains(string(got), `"type":"bad_gateway"`) {
					t.Errorf("empty stream: status %d, body %q", res.StatusCode, got)
				}
			} else if res.StatusCode != http.StatusOK || string(got) != body {
				t.Errorf("status %d, body %q; want 200, %q", res.StatusCode, got, body)
			}
		})
	}
}
