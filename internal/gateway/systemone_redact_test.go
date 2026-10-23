package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/gwmw"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// System One's state takes the same privacy policy on a direct model and
// every try of a decision group (#2389), including explicit opt-outs.
func TestSystemOneRedactsStateBeforeUpstream(t *testing.T) {
	const secret = "sk-proj-abcdEFGH1234ijklMNOP5678qrst"
	for _, tc := range []struct {
		name                                                 string
		group, firstPlain, secondPlain, off, block, noMasker bool
		status                                               int
		masked                                               bool
	}{
		{name: "single", status: 200, masked: true},
		{name: "group failover", group: true, status: 200, masked: true},
		{name: "single opted out", firstPlain: true, status: 200},
		{name: "all group members opted out", group: true, firstPlain: true, secondPlain: true, status: 200},
		{name: "mixed group", group: true, firstPlain: true, status: 200, masked: true},
		{name: "privacy off", off: true, status: 200},
		{name: "single blocked", block: true, status: 400},
		{name: "group blocked", group: true, block: true, status: 400},
		{name: "single masker absent", noMasker: true, status: 503},
		{name: "group masker absent", group: true, noMasker: true, status: 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fresh(t)
			bodies := make(chan string, 4)
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/systemone" {
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					w.WriteHeader(500)
					return
				}
				bodies <- string(b)
				w.Header().Set("Content-Type", "application/json")
				if tc.group && r.Header.Get("Authorization") == "Bearer ka" {
					w.WriteHeader(503)
					io.WriteString(w, `{"detail":{"message":"no capacity"}}`)
					return
				}
				var q struct{ State struct{ Message string } }
				if err := json.Unmarshal(b, &q); err != nil {
					t.Error(err)
					w.WriteHeader(400)
					return
				}
				json.NewEncoder(w).Encode(map[string]any{"model": "jev-latest", "answers": map[string]any{"intent": map[string]any{"type": "choice", "choice": q.State.Message}}, "usage": map[string]int{"input_tokens": 10, "output_tokens": 2}})
			}))
			defer up.Close()
			for _, p := range []provider.Provider{
				{ID: "ja", Name: "JA", Key: "ka", Decide: up.URL + "/v1", Unredacted: tc.firstPlain},
				{ID: "jb", Name: "JB", Key: "kb", Decide: up.URL + "/v1", Unredacted: tc.secondPlain},
			} {
				if err := provider.Save(p); err != nil {
					t.Fatal(err)
				}
			}
			model := "ja/jev-latest"
			if tc.group {
				if err := provider.SaveGroup(provider.Group{ID: "jevs", Name: "Jevs", Routing: provider.Ordered, Members: []string{"ja/jev-latest", "jb/jev-latest"}}); err != nil {
					t.Fatal(err)
				}
				model = "group/jevs"
			}
			st := settings.Settings{Redact: !tc.off}
			if tc.block {
				st.RedactActions = map[string]string{"API_KEY": "block"}
			}
			if err := settings.Save(st); err != nil {
				t.Fatal(err)
			}
			if tc.noMasker {
				withoutMasker(t)
				gwmw.ForgetRefreshForTest()
			}
			body := fmt.Sprintf(`{"model":%q,"state":{"message":"token %s"},"questions":{"intent":{"type":"choice"}}}`, model, secret)
			rec := httptest.NewRecorder()
			New().Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/systemone", strings.NewReader(body)))
			if rec.Code != tc.status {
				t.Fatalf("status %d want %d: %s", rec.Code, tc.status, rec.Body)
			}
			want := 0
			if tc.status == 200 {
				want = 1
				if tc.group {
					want = 2
				}
			}
			if len(bodies) != want {
				t.Fatalf("sent %d requests, want %d", len(bodies), want)
			}
			for range want {
				b := <-bodies
				if !json.Valid([]byte(b)) {
					t.Fatalf("invalid upstream JSON: %s", b)
				}
				if strings.Contains(b, secret) == tc.masked || strings.Contains(b, "{{API_KEY_") != tc.masked {
					t.Fatalf("masked=%v, upstream received %s", tc.masked, b)
				}
			}
			if tc.status == 200 && !strings.Contains(rec.Body.String(), secret) {
				t.Fatalf("answer did not restore the value: %s", rec.Body)
			}
			if tc.block && strings.Contains(rec.Body.String(), secret) {
				t.Fatalf("block error exposed the value: %s", rec.Body)
			}
		})
	}
}
