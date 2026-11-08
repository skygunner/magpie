package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

const codeartsBenefit = `{"code":"InferHub.4004.200","message":"benefit not found"}`

func TestCodeArtsBenefitErrorBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, id, body string
		status         int
		want           bool
	}{
		{"raw", "codearts", codeartsBenefit, 400, true},
		{"plugin wrapper", "codearts-plugin", `{"error":{"code":"InferHub.4004.200","message":"benefit not found (InferHub.4004.200)"}}`, 400, true},
		{"gateway wrapper", "codearts", `{"error":{"code":"InferHub.4004.200","message":"CodeArts: benefit not found (InferHub.4004.200)"}}`, 400, true},
		{"invalid parameter", "codearts", `{"code":"InferHub.001001005.400","message":"The request param is invalid"}`, 400, false},
		{"other provider", "other", codeartsBenefit, 400, false},
		{"unauthorized", "codearts", codeartsBenefit, 401, false},
		{"rate limit", "codearts", codeartsBenefit, 429, false},
		{"quoted payload", "codearts", `{"error":{"code":"invalid_request","message":"bad request","input":{"code":"InferHub.4004.200","message":"benefit not found"}}}`, 400, false},
		{"quoted message", "codearts", `{"error":{"code":"InferHub.4004.200","message":"input contains benefit not found"}}`, 400, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := codeartsBenefitMissing(provider.PluginForTest(tc.id, strings.TrimSuffix(tc.id, "-plugin")), tc.status, []byte(tc.body)); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	if codeartsBenefitMissing(provider.Provider{ID: "codearts", PassErrors: true}, 400, []byte(codeartsBenefit)) {
		t.Fatal("a non-plugin provider with the same id was classified")
	}
}

func TestCodeArtsBenefitFallsOverAndRestsOnlyTheModel(t *testing.T) {
	var mu sync.Mutex
	var models []string
	up := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		model := modelOf(b)
		mu.Lock()
		models = append(models, model)
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if model == "fake-1" {
			w.WriteHeader(400)
			io.WriteString(w, codeartsBenefit)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, `data: {"type":"response.output_text.delta","delta":"hello"}`+"\n\n"+`data: {"type":"response.completed","response":{"id":"r","status":"completed","output":[{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}]}}`+"\n\n")
	})
	pid := besideFake(t, "codearts", up)
	serveOn(t, "spare", "k", []string{"m"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, chatOK)
	}))
	if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{pid + "/fake-1", "spare/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	gw := httptest.NewServer(s.Front())
	defer gw.Close()
	ask := func(model string) (int, string) {
		t.Helper()
		res, err := gw.Client().Post(gw.URL+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"`+model+`","messages":[{"role":"user","content":"hi"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode, string(b)
	}
	if status, body := ask("group/g"); status != 200 || !strings.Contains(body, "hello") {
		t.Fatalf("group did not fall over: %d %s", status, body)
	}
	route := lastRoute(s)
	if len(route.Tries) != 2 || route.Tries[0].Rest == nil || !strings.HasSuffix(route.Tries[0].Rest.Key, "/fake-1") {
		t.Fatalf("model rest: %+v", route.Tries)
	}
	if status, body := ask(pid + "/fake-resp"); status != 200 || !strings.Contains(body, "hello") {
		t.Fatalf("other model unavailable: %d %s", status, body)
	}
	if status, body := ask(pid + "/fake-1"); status != 400 || !strings.Contains(body, "InferHub.4004.200") {
		t.Fatalf("direct refusal lost its code: %d %s", status, body)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(models) != 3 || models[0] != "fake-1" || models[1] != "fake-resp" || models[2] != "fake-1" {
		t.Fatalf("upstream models %v", models)
	}
}
