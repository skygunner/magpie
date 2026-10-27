package gateway

import (
	"encoding/base64"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// A plugin in front of an OpenAI-compatible API, as ZenMux's is, lists the
// image models its catalog gives: one on the images API (openai/gpt-image-2)
// and one its capabilities say gives images (google/gemini-3-pro-image),
// which magpie asks on chat completions with modalities. Both are Settings'
// image generation picker's, the text models beside them are not, and the
// one asked on chat draws through the plugin's fetch (#1728).
func TestPluginChatImageModelDraws(t *testing.T) {
	t.Setenv("FAKE_ZENMUX", "1")
	var mu sync.Mutex
	var paths, bodies []string
	pid := besideFake(t, "zenmux", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		paths, bodies = append(paths, r.URL.Path), append(bodies, string(b))
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v1/chat/completions" {
			io.WriteString(w, `{"choices":[{"message":{"content":[{"type":"image_url","image_url":{"url":"data:image/png;base64,`+base64.StdEncoding.EncodeToString(pngBytes)+`"}}]}}],"usage":{"prompt_tokens":1,"completion_tokens":2}}`)
			return
		}
		io.WriteString(w, `{"data":[{"b64_json":"`+base64.StdEncoding.EncodeToString(pngBytes)+`"}]}`)
	}))
	p, err := provider.Find(pid)
	if err != nil {
		t.Fatal(err)
	}
	var drawers []string
	for _, m := range Drawers(*p) {
		drawers = append(drawers, m.ID)
	}
	if want := []string{"openai/gpt-image-2", "google/gemini-3-pro-image"}; !slices.Equal(drawers, want) {
		t.Fatalf("drawers %v, want %v", drawers, want)
	}
	s := New()
	code, a, raw := postImages(t, s, "/v1/images/generations", "application/json", `{"model":"`+pid+`/google/gemini-3-pro-image","prompt":"a magpie"}`)
	if code != 200 || len(a.Data) != 1 {
		t.Fatalf("%d %s", code, raw)
	}
	if b, _ := base64.StdEncoding.DecodeString(a.Data[0].B64); string(b) != string(pngBytes) {
		t.Fatalf("image %q", b)
	}
	mu.Lock()
	defer mu.Unlock()
	i := slices.Index(paths, "/v1/chat/completions")
	if i < 0 {
		t.Fatalf("asked %v, not chat completions", paths)
	}
	if !strings.Contains(bodies[i], `"model":"google/gemini-3-pro-image"`) || !strings.Contains(bodies[i], `"modalities":["image","text"]`) {
		t.Fatalf("asked on chat %s", bodies[i])
	}
}
