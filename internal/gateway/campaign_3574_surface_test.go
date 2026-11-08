package gateway

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

func TestCampaign3574GatewayMasksProsePassword(t *testing.T) {
	var got string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		got = string(b)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(chatOK))
	}))
	defer up.Close()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Models: []string{"m1"}, Chat: up.URL + "/v1"}); err != nil {
		t.Fatal(err)
	}
	s := New()
	ask := func(enabled bool) int {
		if err := settings.Save(settings.Settings{Redact: enabled}); err != nil {
			t.Fatal(err)
		}
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(`{"model":"fake/m1","messages":[{"role":"user","content":"my password: abc12"}]}`)))
		return rec.Code
	}
	if code := ask(true); code != 200 || strings.Contains(got, "abc12") || !strings.Contains(got, "{{SECRET_") {
		t.Fatalf("enabled code %d body %s", code, got)
	}
	if code := ask(false); code != 200 || !strings.Contains(got, "my password: abc12") {
		t.Fatalf("disabled code %d body %s", code, got)
	}
}
