package gateway

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

const imageCountError = `{"errors":[{"message":"AiError: AiError: Invalid data for image - reason too many images in request (max 32) (00000000-0000-0000-0000-000000000000)","code":8006}],"success":false,"result":{},"messages":[]}`

func TestImageCountGroupFallback(t *testing.T) {
	fresh(t)
	first := make(chan string, 8)
	second := make(chan string, 8)
	serveOn(t, "a", "key", []string{"limited", "other"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		first <- string(b)
		w.Header().Set("Content-Type", "application/json")
		if modelOf(b) == "limited" {
			w.WriteHeader(400)
			fmt.Fprint(w, imageCountError)
			return
		}
		fmt.Fprint(w, chatOK)
	}))
	serveOn(t, "b", "key", []string{"m"}, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		second <- string(b)
		if strings.Contains(string(b), `"stream":true`) {
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprint(w, "data: {\"id\":\"ok\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hello\"},\"finish_reason\":null}]}\n\ndata: {\"id\":\"ok\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, chatOK)
	}))
	if err := provider.SaveGroup(provider.Group{ID: "images", Name: "images", Members: []string{"a/limited", "b/m"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	var parts []string
	var anthroParts []string
	parts = append(parts, `{"type":"text","text":"count"}`)
	anthroParts = append(anthroParts, `{"type":"text","text":"count"}`)
	for i := 0; i < 33; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		img.Set(0, 0, color.RGBA{R: uint8(i), A: 255})
		var encoded bytes.Buffer
		if err := png.Encode(&encoded, img); err != nil {
			t.Fatal(err)
		}
		data := base64.StdEncoding.EncodeToString(encoded.Bytes())
		parts = append(parts, fmt.Sprintf(`{"type":"image_url","image_url":{"url":"data:image/png;base64,%s"}}`, data))
		anthroParts = append(anthroParts, fmt.Sprintf(`{"type":"image","source":{"type":"base64","media_type":"image/png","data":"%s"}}`, data))
	}
	content := "[" + strings.Join(parts, ",") + "]"
	body := `{"model":"group/images","messages":[{"role":"user","content":` + content + `}]}`
	s := New()
	for i := 0; i < 2; i++ {
		code, got := postAs(t, s, "", body)
		if code != 200 || !strings.Contains(got, "hello") {
			t.Fatalf("group: %d %s", code, got)
		}
		a, b := <-first, <-second
		var reqA, reqB struct {
			Messages json.RawMessage `json:"messages"`
		}
		if err := json.Unmarshal([]byte(a), &reqA); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(b), &reqB); err != nil {
			t.Fatal(err)
		}
		want := `[{"role":"user","content":` + content + `}]`
		if string(reqA.Messages) != want || string(reqB.Messages) != want {
			t.Fatal("upstream did not receive the complete original messages")
		}
		r := s.trace.routes[len(s.trace.routes)-1]
		if len(r.Tries) != 2 || r.Tries[0].Fail != failShape || r.Tries[0].Rest != nil {
			t.Fatalf("tries: %+v", r.Tries)
		}
	}
	if err := provider.SaveGroup(provider.Group{ID: "same", Name: "same", Members: []string{"a/limited", "a/other"}, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	code, got := postAs(t, s, "", strings.Replace(body, "group/images", "group/same", 1))
	if code != 200 || !strings.Contains(got, "hello") {
		t.Fatalf("same account fallback: %d %s", code, got)
	}
	<-first
	<-first
	anthroBody := `{"model":"group/images","max_tokens":100,"messages":[{"role":"user","content":[` + strings.Join(anthroParts, ",") + `]}]}`
	anthro := httptest.NewRecorder()
	s.Handler().ServeHTTP(anthro, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(anthroBody)))
	if anthro.Code != 200 || !strings.Contains(anthro.Body.String(), "hello") {
		t.Fatalf("messages: %d %s", anthro.Code, anthro.Body.String())
	}
	a, b := <-first, <-second
	var reqA, reqB struct {
		Messages json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal([]byte(a), &reqA); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(b), &reqB); err != nil {
		t.Fatal(err)
	}
	if string(reqA.Messages) != string(reqB.Messages) || strings.Count(string(reqB.Messages), "data:image/png;base64,") != 33 {
		t.Fatal("translated request did not preserve all images")
	}
	code, got = postAs(t, s, "", strings.Replace(body, "group/images", "a/other", 1))
	if code != 200 || !strings.Contains(got, "hello") {
		t.Fatalf("same provider other model: %d %s", code, got)
	}
	<-first
	code, got = postAs(t, s, "", strings.Replace(body, "group/images", "a/limited", 1))
	if code != 400 || !strings.Contains(got, "too many images in request") {
		t.Fatalf("direct: %d %s", code, got)
	}
	<-first
	passOn(t, "a")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body)))
	if rec.Code != 400 || rec.Body.String() != imageCountError {
		t.Fatalf("errors=pass: %d %s", rec.Code, rec.Body.String())
	}
	<-first
	select {
	case got := <-second:
		t.Fatalf("unexpected fallback: %s", got)
	default:
	}
}

func TestImageCountRefusalControls(t *testing.T) {
	for _, tc := range []struct {
		status       int
		body         string
		shape, retry bool
	}{
		{400, imageCountError, true, true},
		{422, imageCountError, true, true},
		{400, `{"error":{"message":"max_tokens exceeds limit"}}`, false, false},
		{429, `{"error":{"message":"rate limit exceeded"}}`, false, true},
		{400, `{"error":{"message":"Invalid data for image - reason invalid image"}}`, false, false},
		{401, `{"error":{"message":"invalid API key"}}`, false, true},
		{400, `{"error":{"message":"too many images in request (max 32); quota exceeded"}}`, false, true},
	} {
		if got := shapeRefused(tc.status, []byte(tc.body)); got != tc.shape {
			t.Errorf("shape %d %s: %v", tc.status, tc.body, got)
		}
		if got := retryable(tc.status, []byte(tc.body)); got != tc.retry {
			t.Errorf("retry %d %s: %v", tc.status, tc.body, got)
		}
	}
}
