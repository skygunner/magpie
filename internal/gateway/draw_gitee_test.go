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
	"github.com/yetone/magpie/internal/settings"
)

// gitee is Gitee AI's serverless API (ai.gitee.com) as the reporter met
// it (#1105): an image model its chat completions endpoint refuses with
// "暂不支持该接口", its images API serving, and a size dialect of its own -
// width and height, with OpenAI's size refused as 参数无效.
type gitee struct {
	mu     sync.Mutex
	paths  []string
	bodies []string
}

func (g *gitee) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.mu.Lock()
	g.paths = append(g.paths, r.Method+" "+r.URL.Path)
	body, _ := io.ReadAll(r.Body)
	g.bodies = append(g.bodies, string(body))
	g.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/v1/chat/completions":
		w.WriteHeader(400)
		io.WriteString(w, `{"error":{"message":"暂不支持该接口"}}`)
	case r.URL.Path == "/v1/images/generations" || r.URL.Path == "/v1/images/edits":
		var req map[string]any
		json.Unmarshal(body, &req)
		if _, ok := req["size"]; ok {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":{"message":"参数无效 'size'"}}`)
			return
		}
		io.WriteString(w, `{"data":[{"b64_json":"`+base64.StdEncoding.EncodeToString(pngBytes)+`"}],"usage":{"output_tokens":100}}`)
	default:
		w.WriteHeader(404)
	}
}

func giteed(t *testing.T) (*Server, *gitee) {
	t.Helper()
	fresh(t)
	g := &gitee{}
	up := httptest.NewServer(g)
	t.Cleanup(up.Close)
	if err := provider.Save(provider.Provider{ID: "gitee", Name: "Gitee AI", Preset: "gitee", Chat: up.URL + "/v1", Key: "key", Models: []string{"z-image-turbo", "deepseek-v4"}}); err != nil {
		t.Fatal(err)
	}
	// the user said the model draws, as its odd name says nowhere (#1399)
	cfg := settings.Load()
	cfg.ModelKinds = map[string]string{"gitee/z-image-turbo": provider.KindImage}
	if err := settings.Save(cfg); err != nil {
		t.Fatal(err)
	}
	return New(), g
}

// TestGiteeMarkedImageDrawsOnImagesAPI: a model the user said draws, on a
// vendor that serves it only on the images API, is drawn there - not sent
// to chat completions for its 400 (#1105, GEEKCX: z-image-turbo).
func TestGiteeMarkedImageDrawsOnImagesAPI(t *testing.T) {
	s, g := giteed(t)
	code, _, raw := postImages(t, s, "/v1/images/generations", "application/json", `{"model":"gitee/z-image-turbo","prompt":"a magpie","size":"1024x1024"}`)
	if code != 200 {
		t.Fatalf("code %d: %s", code, raw)
	}
	var sent map[string]any
	json.Unmarshal([]byte(g.bodies[len(g.bodies)-1]), &sent)
	if sent["model"] != "z-image-turbo" || sent["prompt"] != "a magpie" {
		t.Fatalf("sent %s", g.bodies[len(g.bodies)-1])
	}
	// Gitee's dialect: width and height, never OpenAI's size
	if sent["width"] != float64(1024) || sent["height"] != float64(1024) {
		t.Fatalf("no width/height 1024x1024: %s", g.bodies[len(g.bodies)-1])
	}
	if _, ok := sent["size"]; ok {
		t.Fatalf("size sent, which Gitee refuses: %s", g.bodies[len(g.bodies)-1])
	}
	for _, p := range g.paths {
		if strings.HasSuffix(p, "/v1/chat/completions") {
			t.Fatalf("the drawing went to chat completions: %v", g.paths)
		}
	}
}

// a model neither marked nor of a drawing name still goes to chat, and a
// chat model is untouched by the dialect.
func TestGiteeChatModelStillOnChat(t *testing.T) {
	s, g := giteed(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"gitee/deepseek-v4","messages":[{"role":"user","content":"hi"}]}`))
	req.Header.Set("Content-Type", "application/json")
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 || !strings.Contains(rec.Body.String(), "暂不支持该接口") {
		t.Fatalf("code %d: %s", rec.Code, rec.Body.String())
	}
	if len(g.paths) != 1 || !strings.HasSuffix(g.paths[0], "/v1/chat/completions") {
		t.Fatalf("paths %v", g.paths)
	}
}
