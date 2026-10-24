package gateway

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// ChatGPT's Codex images backend ignores the images API's size: a 1792x1024
// ask came back 1254x1254 (kilo-s, #3513), while the same prompt with the
// resolution said in words drew at the asked ratio. Codex CLI writes the
// resolution into the prompt itself; magpie says it there for the Codex
// account's drawings too.
func TestCodexImageSaysItsSizeInWords(t *testing.T) {
	codexSignedIn(t)
	var mu sync.Mutex
	var bodies []string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		bodies = append(bodies, r.URL.Path+" "+string(b))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"created":1,"output_format":"png","data":[{"b64_json":"`+base64.StdEncoding.EncodeToString(pngBytes)+`"}]}`)
	}))
	defer up.Close()
	was := provider.CodexBase
	provider.CodexBase = up.URL + "/backend-api/codex"
	defer func() { provider.CodexBase = was }()
	s := New()
	for _, c := range []struct {
		path, body, want string
	}{
		{"/v1/images/generations", `{"model":"codex/gpt-image-2.5","prompt":"a flat green triangle on white","n":1,"size":"1792x1024","quality":"low"}`,
			"Generate at 1792x1024 resolution. a flat green triangle on white"},
		{"/v1/images/edits", `{"model":"codex/gpt-image-2.5","prompt":"make the triangle red","n":1,"size":"1024x1536","image":"data:image/png;base64,` + base64.StdEncoding.EncodeToString(pngBytes) + `"}`,
			"Generate at 1024x1536 resolution. make the triangle red"},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", c.path, strings.NewReader(c.body))
		req.Header.Set("Content-Type", "application/json")
		s.Handler().ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("%s: code %d: %s", c.path, rec.Code, rec.Body.String())
		}
		mu.Lock()
		last := bodies[len(bodies)-1]
		mu.Unlock()
		var sent map[string]any
		json.Unmarshal([]byte(strings.SplitN(last, " ", 2)[1]), &sent)
		if sent["prompt"] != c.want {
			t.Fatalf("%s: prompt %v, want %q", c.path, sent["prompt"], c.want)
		}
	}
	// a drawing without a size says nothing of one
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/images/generations", strings.NewReader(`{"model":"codex/gpt-image-2.5","prompt":"a cat","n":1}`))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("no size: code %d: %s", rec.Code, rec.Body.String())
	}
	mu.Lock()
	last := bodies[len(bodies)-1]
	mu.Unlock()
	if !strings.Contains(last, `"prompt":"a cat"`) {
		t.Fatalf("a drawing without a size gained words: %s", last)
	}
}
