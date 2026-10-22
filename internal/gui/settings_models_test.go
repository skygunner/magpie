package gui

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// A provider that exists is not enough: a Settings pick must name one of
// its models. A value saved before the model disappeared is kept, so the
// reader can still save the rest of Settings.
func TestSettingsModelsMustBeServed(t *testing.T) {
	sandboxHome(t)
	for _, p := range []provider.Provider{
		{ID: "fake", Name: "Fake", Key: "k", Models: []string{"m1", "gpt-image-1"}, Chat: "http://127.0.0.1:9/v1"},
		{ID: "hidden", Name: "Hidden", Key: "k", Unlisted: true, Models: []string{"m1"}, Chat: "http://127.0.0.1:9/v1"},
	} {
		if err := provider.Save(p); err != nil {
			t.Fatal(err)
		}
	}
	if err := provider.SaveGroup(provider.Group{ID: "helpers", Name: "Helpers", Members: []string{"fake/m1"}}); err != nil {
		t.Fatal(err)
	}
	h := Handler(nil, nil)
	for _, tc := range []struct {
		name, path, key string
		field           func(*settings.Settings) *string
	}{
		{"vision", "/api/settings", "vision", func(s *settings.Settings) *string { return &s.Vision }},
		{"imageGen", "/api/settings", "imageGen", func(s *settings.Settings) *string { return &s.ImageGen }},
		{"titles", "/api/settings/codex-titles", "model", func(s *settings.Settings) *string { return &s.CodexTitles }},
		{"descriptions", "/api/settings/codex-descriptions", "model", func(s *settings.Settings) *string { return &s.CodexDescriptions }},
		{"predictions", "/api/settings/codex-predictions", "model", func(s *settings.Settings) *string { return &s.CodexPredictions }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			call := func(model string) *httptest.ResponseRecorder {
				t.Helper()
				body, err := json.Marshal(map[string]string{tc.key: model})
				if err != nil {
					t.Fatal(err)
				}
				r := httptest.NewRecorder()
				h.ServeHTTP(r, httptest.NewRequest("POST", tc.path, bytes.NewReader(body)))
				return r
			}
			valid := []string{"hidden/m1", "group/helpers", "fake/m1"}
			bad := []string{"fake/missing", "fake/", "missing/m1", "group/missing", "m1"}
			if tc.name == "imageGen" {
				valid = []string{"fake/gpt-image-1"}
				bad = append(bad, "fake/m1") // served for chat, but cannot draw
			}
			for _, model := range valid {
				r := call(model)
				s := settings.Load()
				if r.Code != http.StatusOK || *tc.field(&s) != model {
					t.Fatalf("valid pick %q: status %d, saved %q", model, r.Code, *tc.field(&s))
				}
			}
			kept := settings.Load()
			for _, model := range bad {
				if err := settings.Save(kept); err != nil {
					t.Fatal(err)
				}
				before, err := os.ReadFile(settings.Path())
				if err != nil {
					t.Fatal(err)
				}
				r := call(model)
				if r.Code != http.StatusBadRequest {
					t.Errorf("unavailable pick %q: status %d, want 400", model, r.Code)
				}
				after, err := os.ReadFile(settings.Path())
				if err != nil || !bytes.Equal(after, before) {
					t.Errorf("rejecting %q changed the saved settings: %v", model, err)
				}
			}
			// Another Settings save may re-send a pick whose provider is now
			// gone. It must not force the reader to choose a replacement.
			s := settings.Load()
			*tc.field(&s) = "gone/old-model"
			if err := settings.Save(s); err != nil {
				t.Fatal(err)
			}
			r := call("gone/old-model")
			s = settings.Load()
			if r.Code != http.StatusOK || *tc.field(&s) != "gone/old-model" {
				t.Errorf("saved unavailable pick: status %d, saved %q", r.Code, *tc.field(&s))
			}
		})
	}
}
