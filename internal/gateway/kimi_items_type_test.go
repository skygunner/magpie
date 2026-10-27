package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// #3867 (yetone): a schema with items and a string enum is an array, not
// its enum's string — Kimi refused items on a string.
func TestKimiItemsTypeBeforeEnum(t *testing.T) {
	kimi := provider.Provider{ID: "kimi", Name: "Kimi", Key: "k", Chat: "https://api.kimi.com/v1", Models: []string{"m"}}
	if err := provider.Save(kimi); err != nil {
		t.Fatal(err)
	}
	got := ""
	s := New()
	s.client = &http.Client{Transport: countTransport(func(r *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(r.Body)
		got = string(b)
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}},
			Body: io.NopCloser(strings.NewReader(`{"id":"c","object":"chat.completion","model":"m","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))}, nil
	})}
	tools := `"tools":[{"type":"function","function":{"name":"t","parameters":{"type":"object","properties":{"k":{"items":{"type":"string"},"enum":["a","b"]}}}}}]`
	body := `{"model":"kimi/m","messages":[{"role":"user","content":"hi"}],` + tools + `}`
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body)))
	if rec.Code != 200 {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	if !strings.Contains(got, `"k":{"type":"array"`) {
		t.Fatalf("Kimi was sent:\n%s\nwant k typed array", got)
	}
	if strings.Contains(got, `"k":{"type":"string"`) {
		t.Fatalf("Kimi was sent:\n%s\nk must not be typed string (items on a string)", got)
	}
}
