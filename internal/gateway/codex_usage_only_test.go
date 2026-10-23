package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestCodexUsageOnlyCatalog(t *testing.T) {
	codexSignedIn(t)
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"models":[{"slug":"gpt-6-sol","priority":1,"visibility":"list"},{"slug":"gpt-5.5","priority":2,"visibility":"list"},{"slug":"codex-auto-review","priority":3,"visibility":"hide"},{"slug":"gpt-reserve","priority":4,"visibility":"hide"}]}`)
	}))
	t.Cleanup(up.Close)
	was := provider.CodexBase
	provider.CodexBase = up.URL + "/backend-api/codex"
	t.Cleanup(func() { provider.CodexBase = was })
	if err := provider.Save(provider.Provider{ID: "fake", Name: "Fake", Key: "k", Models: []string{"m1"}, Chat: up.URL + "/v1"}); err != nil {
		t.Fatal(err)
	}
	gw := httptest.NewServer(New().Handler())
	t.Cleanup(gw.Close)
	for _, tc := range []struct {
		name string
		p    provider.Provider
		want []string
	}{
		{"vendor list", provider.Provider{ID: "codex"}, []string{"gpt-6-sol", "gpt-5.5"}},
		{"usage only", provider.Provider{ID: "codex", NoModels: true}, nil},
		{"picked again", provider.Provider{ID: "codex", Models: []string{"gpt-6-sol"}}, []string{"gpt-6-sol"}},
		{"off keeps native list", provider.Provider{ID: "codex", Off: true, NoModels: true}, []string{"gpt-6-sol", "gpt-5.5"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := provider.Save(tc.p); err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest(http.MethodGet, gw.URL+CodexPath+"/models?client_version=0.163.0", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Authorization", "Bearer chatgpt-token")
			res, err := gw.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			var list struct {
				Models []struct {
					Slug string `json:"slug"`
				} `json:"models"`
			}
			if err := json.NewDecoder(res.Body).Decode(&list); err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != http.StatusOK {
				t.Fatalf("model list status %d", res.StatusCode)
			}
			var got []string
			for _, m := range list.Models {
				got = append(got, m.Slug)
			}
			want := append(slices.Clone(tc.want), "codex-auto-review", "gpt-reserve", "fake/m1")
			slices.Sort(got)
			slices.Sort(want)
			if !slices.Equal(got, want) {
				t.Fatalf("models = %v, want %v", got, want)
			}
		})
	}
}
