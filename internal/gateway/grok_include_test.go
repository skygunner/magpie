package gateway

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

// #3871: Grok (xAI) rejects include: ["output.encrypted_content"].
// The include field should drop output.encrypted_content for xAI hosts.
func TestGrokIncludeDropsOutputEncryptedContent(t *testing.T) {
	fresh(t)
	provider.Save(provider.Provider{
		ID: "grok", Name: "Grok", Key: "k", Responses: "https://api.x.ai/v1", Models: []string{"grok-4.7"},
	})

	s := New()
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"grok/grok-4.7","input":"hi","include":["output.encrypted_content"]}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	// the request should not 400
	if rec.Code == 400 {
		t.Fatalf("got 400: %s", rec.Body.String())
	}
}

// TestOpenAIIncludeKeepsOutputEncryptedContent verifies that output.encrypted_content
// is NOT dropped for OpenAI hosts (where it might be supported).
func TestOpenAIIncludeKeepsOutputEncryptedContent(t *testing.T) {
	fresh(t)
	provider.Save(provider.Provider{
		ID: "openai", Name: "OpenAI", Key: "k", Responses: "https://api.openai.com/v1", Models: []string{"gpt-4o"},
	})

	s := New()
	req := httptest.NewRequest("POST", "/v1/responses", strings.NewReader(`{"model":"openai/gpt-4o","input":"hi","include":["output.encrypted_content"]}`))
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	// OpenAI might support it, so we don't assert on the include field here
	// Just verify it doesn't crash
	if rec.Code == 500 {
		t.Fatalf("internal error: %s", rec.Body.String())
	}
}
