package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestRemoteMagpieGemini(t *testing.T) {
	fresh(t)
	paths := make(chan string, 4)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.Path
		if r.URL.Path != "/v1beta/models/google/gemini-test:streamGenerateContent" {
			http.Error(w, "wrong API", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"hello\"}]},\"finishReason\":\"STOP\"}]}\n\n")
	}))
	t.Cleanup(up.Close)
	if err := provider.Save(provider.Provider{ID: "office", Preset: provider.RemoteMagpiePreset, Chat: up.URL, Key: "remote-key", Models: []string{"google/gemini-test"}}); err != nil {
		t.Fatal(err)
	}
	s := New()
	for _, method := range []string{"streamGenerateContent", "generateContent"} {
		r := httptest.NewRequest("POST", "/v1beta/models/office/google/gemini-test:"+method, strings.NewReader(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`))
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), "hello") {
			t.Fatalf("%s: %d %s", method, w.Code, w.Body.String())
		}
		select {
		case path := <-paths:
			if path != "/v1beta/models/google/gemini-test:streamGenerateContent" {
				t.Fatalf("%s used %s", method, path)
			}
		default:
			t.Fatal("no upstream request")
		}
	}
}
