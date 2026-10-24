package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// urlCiteBlind is the blind model of #669 whose answer names the page the
// search found, as magpie's web_search tool asks it to (#3455).
func urlCiteBlind(t *testing.T) *httptest.Server {
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/models") {
			io.WriteString(w, `{"data":[{"id":"glm-flash"}]}`)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		var q struct {
			Messages []map[string]any `json:"messages"`
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &q)
		w.Header().Set("Content-Type", "text/event-stream")
		if last := q.Messages[len(q.Messages)-1]; last["role"] == "tool" {
			io.WriteString(w, sse(
				`data: {"id":"d","choices":[{"index":0,"delta":{"role":"assistant","content":"上海今天小雨，22/17℃，见 https://www.weather.com.cn/weather/101020100.shtml 。"}}]}`,
				`data: {"id":"d","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":5}}`,
				`data: [DONE]`))
			return
		}
		io.WriteString(w, sse(
			`data: {"id":"c","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"web_search","arguments":"{\"query\":\"今天上海天气\"}"}}]}}]}`,
			`data: {"id":"c","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":30,"completion_tokens":7}}`,
			`data: [DONE]`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func urlCiteSandbox(t *testing.T) {
	searchSandbox(t)
	ds := newFakeDeepSeek(t)
	glm := urlCiteBlind(t)
	searchesOn(t, provider.Responses, ds.URL)
	save(t,
		provider.Provider{ID: "ds", Name: "DeepSeek", Key: "k", Chat: ds.URL + "/v1", Responses: ds.URL + "/v1", Models: []string{"deepseek-flash"}},
		provider.Provider{ID: "glm", Name: "GLM", Key: "k", Chat: glm.URL, Models: []string{"glm-flash"}})
}

const urlCiteAsk = `{"model":"glm/glm-flash","messages":[{"role":"user","content":"用 web_search 工具搜索：今天上海天气"}],"web_search_options":{}}`

// After a search magpie ran for a model, a Responses reply's output_text
// cites the pages its text names with OpenAI's url_citation, streamed as
// response.output_text.annotation.added and kept on the part (#3455).
func TestResponsesSearchCitesURLs(t *testing.T) {
	urlCiteSandbox(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"glm/glm-flash","input":"用 web_search 工具搜索：今天上海天气","stream":true,"tools":[{"type":"web_search"}]}`))
	req.Header.Set("Accept", "text/event-stream")
	New().Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	s := rec.Body.String()
	if !strings.Contains(s, `"type":"response.output_text.annotation.added"`) {
		t.Fatalf("no annotation.added event:\n%.1200s", s)
	}
	var ann map[string]any
	for _, line := range strings.Split(s, "\n") {
		data, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		var ev struct {
			Annotation map[string]any `json:"annotation"`
		}
		if json.Unmarshal([]byte(data), &ev) == nil && ev.Annotation != nil {
			ann = ev.Annotation
		}
	}
	if ann["type"] != "url_citation" || ann["url"] != "https://www.weather.com.cn/weather/101020100.shtml" {
		t.Fatalf("annotation %v", ann)
	}
	// the span counts the text's characters: the address stands after
	// "上海今天小雨，22/17℃，见 " (16 characters) and is 50 long
	if ann["start_index"] != float64(16) || ann["end_index"] != float64(66) {
		t.Fatalf("annotation span %v..%v", ann["start_index"], ann["end_index"])
	}
	if !strings.Contains(s, `"annotations":[{"end_index":66`) {
		t.Fatalf("the part's done carries no annotations:\n%s", s)
	}
}

// A Chat reply carries the same citations on its message (#3455).
func TestChatSearchCitesURLs(t *testing.T) {
	urlCiteSandbox(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(urlCiteAsk))
	New().Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var reply struct {
		Choices []struct {
			Message struct {
				Annotations []map[string]any `json:"annotations"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &reply); err != nil {
		t.Fatalf("%d not JSON: %.400s", rec.Code, rec.Body.String())
	}
	if len(reply.Choices) != 1 || len(reply.Choices[0].Message.Annotations) != 1 {
		t.Fatalf("annotations: %s", rec.Body.String())
	}
	a := reply.Choices[0].Message.Annotations[0]
	if a["type"] != "url_citation" || a["url"] != "https://www.weather.com.cn/weather/101020100.shtml" || a["start_index"] != float64(16) || a["end_index"] != float64(66) {
		t.Fatalf("annotation %v", a)
	}
}
