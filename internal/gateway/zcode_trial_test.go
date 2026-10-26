package gateway

import (
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/yetone/magpie/internal/provider"
)

func TestZCodeTrialQuotaRestsTheModelAlone(t *testing.T) {
	for _, tc := range []struct {
		name, model, body string
		status            int
		accountRest       bool
	}{
		{"expired flash trial", "GLM-5.3-Flash-Trial", `{"type":"error","error":{"type":"invalid_request_error","message":"ZCode has no active limited quota for GLM-5.3-Flash-Trial"}}`, 400, false},
		{"expired trial", "GLM-5.3-Trial", `{"type":"error","error":{"type":"invalid_request_error","message":"ZCode has no active limited quota for GLM-5.3-Trial"}}`, 400, false},
		{"account quota", "GLM-5.3", `{"error":{"type":"rate_limit_error","message":"quota exhausted"}}`, 429, true},
		{"account authentication", "GLM-5.3", `{"error":{"message":"invalid api key"}}`, 401, true},
		{"another model named", "GLM-5.3-Flash-Trial", `{"error":{"message":"ZCode has no active limited quota for GLM-5.3-Trial"}}`, 400, true},
		{"nontrial model", "GLM-5.3", `{"error":{"message":"ZCode has no active limited quota for GLM-5.3"}}`, 400, true},
		{"quota status", "GLM-5.3-Flash-Trial", `{"error":{"message":"ZCode has no active limited quota for GLM-5.3-Flash-Trial"}}`, 429, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given: the real plugin dispatch path, with no free-model pool to mask an account rest.
			t.Setenv("FAKE_ZCODE", "1")
			t.Setenv("FAKE_ZCODE_TRIAL", "1")
			t.Setenv("FAKE_NO_FREE", "1")
			var mu sync.Mutex
			var asked []string
			good := &adaptiveVendor{}
			movedFake(t, "zcode", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					w.WriteHeader(500)
					return
				}
				model := modelOf(body)
				mu.Lock()
				asked = append(asked, model)
				mu.Unlock()
				if model == tc.model {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tc.status)
					io.WriteString(w, tc.body)
					return
				}
				r.Body = io.NopCloser(strings.NewReader(string(body)))
				good.ServeHTTP(w, r)
			}))
			p, err := provider.Find("zcode")
			if err != nil {
				t.Fatal(err)
			}
			c := candidate{p: *p, rest: "zcode", model: tc.model}
			old := allowances
			allowances = func(string) map[string]provider.Allowance {
				return map[string]provider.Allowance{p.Account.User: {{Used: 10}}}
			}
			t.Cleanup(func() { allowances = old })
			b := &scripted{replies: []reply{{200, "", chatOK}}}
			scriptedOn(t, "b", provider.Chat, b)
			next := "zcode/GLM-5.3"
			if tc.accountRest {
				next = "b/m"
			}
			if err := provider.SaveGroup(provider.Group{Name: "G", Members: []string{"zcode/" + tc.model, next}, Routing: provider.Ordered}); err != nil {
				t.Fatal(err)
			}
			s := New()

			// When: a refusal is followed by another member, then the group is asked again.
			for i := range 2 {
				code, body := postAs(t, s, "", `{"model":"group/g","messages":[{"role":"user","content":"hi"}]}`)
				if code != 200 || !(strings.Contains(body, "ok") || strings.Contains(body, "hello")) {
					t.Fatalf("request %d: %d %s", i, code, body)
				}
				if i == 0 {
					tries := s.trace.routes[len(s.trace.routes)-1].Tries
					if len(tries) != 2 || tries[0].Status != tc.status || tries[0].Rest == nil {
						t.Fatalf("tries: %+v", tries)
					}
					want := c.restID()
					if tc.accountRest {
						want = c.restKey()
					}
					if tries[0].Rest.Key != want {
						t.Fatalf("refusal rested %q, want %q", tries[0].Rest.Key, want)
					}
				}
			}

			// Then: the trial rests alone; real account faults keep the entire account out.
			if _, ok := restOf(c.restKey()); ok != tc.accountRest {
				t.Fatalf("account resting = %v, want %v", ok, tc.accountRest)
			}
			if _, ok := restOf(c.restID()); ok == tc.accountRest {
				t.Fatalf("model resting = %v, account fault = %v", ok, tc.accountRest)
			}
			mu.Lock()
			defer mu.Unlock()
			if tc.accountRest {
				if len(asked) != 1 || b.n != 2 {
					t.Fatalf("account fault: ZCode %v, fallback %d", asked, b.n)
				}
			} else if len(asked) != 3 || asked[0] != tc.model || asked[1] != "GLM-5.3" || asked[2] != "GLM-5.3" {
				t.Fatalf("trial refusal affected subscription: %v", asked)
			}
		})
	}
}
