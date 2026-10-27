package gateway

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/maskrec"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

// A document in System One's state reaches the decision provider masked,
// on a single model and on every try of a decision group, the answer's
// placeholders come back as the client wrote them, and with masking off
// the document goes as it is. The request is Hindsight's TypeSafe
// reranker's (ati121, #2389): the candidate documents are in state.
func TestSystemOneMasksStateBeforeUpstream(t *testing.T) {
	const (
		email = "dana.reyes@northwind-logistics.com"
		phone = "+14158675309"
		ip    = "52.95.110.7"
	)
	body := func(model string) string {
		return fmt.Sprintf(`{"model":%q,"state":{"message":"which document answers this?","documents":[{"id":"d1","text":"contact %s, call %s, host %s"}]},"questions":{"intent":{"type":"choice"}}}`, model, email, phone, ip)
	}
	for _, tc := range []struct {
		name     string
		group    bool
		off      bool
		requests int
	}{
		{name: "single model", requests: 1},
		{name: "decision group fails over", group: true, requests: 2},
		{name: "masking off", off: true, requests: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fresh(t)
			var (
				mu     sync.Mutex
				bodies []string
			)
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/systemone" {
					http.NotFound(w, r)
					return
				}
				b, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				mu.Lock()
				bodies = append(bodies, string(b))
				mu.Unlock()
				if tc.group && r.Header.Get("Authorization") == "Bearer ka" {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					io.WriteString(w, `{"detail":{"message":"no capacity"}}`)
					return
				}
				var q struct{ State json.RawMessage }
				if err := json.Unmarshal(b, &q); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"model": "jev-latest",
					"answers": map[string]any{"intent": map[string]any{"type": "choice", "choice": string(q.State)}},
					"usage":   map[string]int{"input_tokens": 10, "output_tokens": 2}})
			}))
			defer up.Close()
			for _, p := range []provider.Provider{
				{ID: "ja", Name: "JA", Key: "ka", Decide: up.URL + "/v1"},
				{ID: "jb", Name: "JB", Key: "kb", Decide: up.URL + "/v1"},
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
			st := settings.Settings{}
			if !tc.off {
				st.Redact, st.RedactPersonal = true, true
				st.RedactKinds = map[string]bool{"ip": true}
			}
			if err := settings.Save(st); err != nil {
				t.Fatal(err)
			}
			var beforeKey int64
			if before, _ := maskrec.List(0, 1, maskrec.Filter{}); len(before) > 0 {
				beforeKey = before[0].Key
			}
			rec := httptest.NewRecorder()
			New().Handler().ServeHTTP(rec, httptest.NewRequest("POST", "/v1/systemone", strings.NewReader(body(model))))
			if rec.Code != http.StatusOK {
				t.Fatalf("status %d: %s", rec.Code, rec.Body)
			}
			mu.Lock()
			got := bodies
			mu.Unlock()
			if len(got) != tc.requests {
				t.Fatalf("sent %d requests, want %d", len(got), tc.requests)
			}
			for _, b := range got {
				if !json.Valid([]byte(b)) {
					t.Fatalf("invalid upstream JSON: %s", b)
				}
				for _, v := range []string{email, phone, ip} {
					if strings.Contains(b, v) == !tc.off {
						t.Fatalf("masking off=%v, upstream received %s", tc.off, b)
					}
				}
				if !tc.off && !strings.Contains(b, "{{") {
					t.Fatalf("no placeholder upstream: %s", b)
				}
			}
			if !strings.Contains(rec.Body.String(), email) || !strings.Contains(rec.Body.String(), phone) || !strings.Contains(rec.Body.String(), ip) {
				t.Fatalf("answer did not restore the values: %s", rec.Body)
			}
			rs, _ := maskrec.List(0, 1, maskrec.Filter{})
			recorded := len(rs) > 0 && rs[0].Key != beforeKey
			if tc.off {
				if recorded {
					t.Fatalf("masking off recorded a mask: %+v", rs[0])
				}
				return
			}
			if !recorded || rs[0].What != "decide" || rs[0].N < 3 {
				t.Fatalf("masking not recorded under decide: %+v", rs)
			}
			if strings.Contains(fmt.Sprint(rs[0].Values), email) || strings.Contains(fmt.Sprint(rs[0].Snips), email) {
				t.Fatalf("record holds the value: %+v", rs[0])
			}
		})
	}
}
