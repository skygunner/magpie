package gateway

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
)

// nestedKeys is a vendor that answers as each key, and refuses as told
// for each, out of its allowance as ChatGPT says it.
type nestedKeys struct {
	mu    sync.Mutex
	tried []string
	fail  map[string]bool // key → refuse with a 429 out of allowance
}

func (k *nestedKeys) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.ReadAll(r.Body)
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	k.mu.Lock()
	k.tried = append(k.tried, key)
	refuse := k.fail[key]
	k.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if refuse {
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"error":{"type":"usage_limit_reached","message":"The usage limit has been reached","resets_in_seconds":7200}}`)
		return
	}
	io.WriteString(w, `{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"from `+key+`"},"finish_reason":"stop"}]}`)
}

// A nested group's own Affinity holds the requests that reach it through
// the group asked for, as its own firstToken (#2246) and Sink (#2255) do:
// only the group the agent named had its Affinity read, so an inner
// group's strict let the conversation drift to another key.
func TestNestedGroupStrictAffinity(t *testing.T) {
	fresh(t)
	k := &nestedKeys{fail: map[string]bool{}}
	up := httptest.NewServer(k)
	t.Cleanup(up.Close)
	if err := provider.Save(provider.Provider{ID: "gw", Name: "GW", Key: "k1", Models: []string{"m"}, Chat: up.URL + "/v1",
		Keys: []provider.KeyAccount{{Key: "k2"}}}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{ID: "inner", Name: "Inner", Members: []string{"gw/m"},
		Affinity: provider.AffinityStrict, Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	if err := provider.SaveGroup(provider.Group{ID: "outer", Name: "Outer", Members: []string{"group/inner"},
		Routing: provider.Ordered}); err != nil {
		t.Fatal(err)
	}
	s := New()
	const turn = `{"model":"group/outer","messages":[{"role":"user","content":"hi"}]}`
	if code, body := postAs(t, s, "s1", turn); code != 200 || len(k.tried) != 1 {
		t.Fatalf("first turn: %d %s tried %v", code, body, k.tried)
	}
	first := k.tried[0]
	// the key the conversation is on runs out: the inner group's strict
	// keeps the conversation on it anyway, and its error is the agent's —
	// not a move to the other key
	k.mu.Lock()
	k.fail[first] = true
	k.tried = nil
	k.mu.Unlock()
	code, body := postAs(t, s, "s1", turn)
	k.mu.Lock()
	tried := strings.Join(k.tried, ",")
	k.mu.Unlock()
	if code != 429 || tried != first {
		t.Fatalf("second turn, the inner group's strict: %d %s tried %s", code, body, tried)
	}
	a := s.trace.routes[len(s.trace.routes)-1].Affinity
	if a == nil || a.Why != "strict" {
		t.Fatalf("affinity %+v, want strict", a)
	}
}

// Every explicit nested mode wins over the outer mode. Rotate still
// decides which key begins a new turn when affinity doesn't keep it.
func TestNestedAffinityModes(t *testing.T) {
	for _, tc := range []struct {
		name, inner, outer, routing, why string
	}{
		{"off", "off", "strict", provider.Rotate, "off"},
		{"session", "session", "off", provider.Rotate, "session"},
		{"turn", "turn", "session", provider.Rotate, "new-turn"},
		{"strict", "strict", "off", provider.Rotate, "strict"},
		{"inherited", "", "session", provider.Rotate, "session"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fresh(t)
			v := &keyed{}
			serveOn(t, "nested", "k1", []string{"m"}, v, "k2")
			for _, g := range []provider.Group{
				{ID: "inner", Members: []string{"nested/m"}, Affinity: tc.inner, Routing: provider.Ordered},
				{ID: "outer", Members: []string{"group/inner"}, Affinity: tc.outer, Routing: tc.routing},
			} {
				if err := provider.SaveGroup(g); err != nil {
					t.Fatal(err)
				}
			}
			s := New()
			const first = `{"model":"group/outer","messages":[{"role":"user","content":"hi"}]}`
			const again = `{"model":"group/outer","messages":[{"role":"user","content":"hi"},{"role":"assistant","content":"yo"},{"role":"user","content":"more"}]}`
			for _, body := range []string{first, again} {
				if code, out := postAs(t, s, "nested-session", body); code != 200 {
					t.Fatalf("%d %s", code, out)
				}
			}
			if why := lastWhy(t, s); why != tc.why {
				t.Fatalf("why %q, want %q", why, tc.why)
			}
			kept := tc.why == "session" || tc.why == "strict"
			a := s.trace.routes[len(s.trace.routes)-1].Affinity
			if a.Kept != kept {
				t.Fatalf("affinity %+v, kept %v", a, kept)
			}
			if kept && v.tried[0] != v.tried[1] || tc.why == "new-turn" && v.tried[0] == v.tried[1] {
				t.Fatalf("keys %v, affinity %+v", v.tried, a)
			}
		})
	}
}

// A cap or credit hold must use the same nested mode as a ready member,
// including the exact model/effort when an account serves several groups.
func TestNestedAffinityAtCap(t *testing.T) {
	for _, held := range []bool{false, true} {
		for _, mode := range []string{"strict", "off", "session", "turn", ""} {
			t.Run(fmt.Sprintf("held=%v/%s", held, mode), func(t *testing.T) {
				fresh(t)
				c := candidate{p: provider.Provider{ID: "nested", Key: "k1"}, rest: "nested", model: "m", effort: "high", affinity: mode}
				other := candidate{p: provider.Provider{ID: "other", Key: "k2"}, rest: "other", model: "m"}
				const scope = "group/outer"
				body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
				sticks.Lock()
				sticks.m[scope+"|"+conversationID(nil, body)] = stick{who: c.who(), model: c.model, effort: c.effort, at: time.Now()}
				sticks.Unlock()
				w := Weighed{ID: c.rest, Model: c.model, Fixed: c.effort, Capped: 70}
				pl := planned{order: []Weighed{{ID: other.rest, Model: other.model}}, left: []Weighed{w}, capped: []candidate{c}}
				if held {
					pl.capped = nil
					pl.held = []candidate{c}
					pl.left[0].NoCredits = true
				}
				// Outer strict exercises explicit inner modes overriding it;
				// inner strict also works with outer off.
				outer := "strict"
				if mode == "strict" {
					outer = "off"
				}
				cs, _, strictHeld := strictKeep(scope, outer, nil, body, []candidate{other}, pl)
				strict := mode == "strict" || mode == ""
				if held && strict {
					if len(cs) != 0 || len(strictHeld) != 1 {
						t.Fatalf("held strict: %v %v", cs, strictHeld)
					}
				} else if strict {
					if len(cs) != 2 || cs[0].who() != c.who() {
						t.Fatalf("strict didn't restore its account: %v", cs)
					}
				} else if len(cs) != 1 || cs[0].who() != other.who() || strictHeld != nil {
					t.Fatalf("inner %s bypassed the cap/hold: %v %v", mode, cs, strictHeld)
				}
			})
		}
	}
}

func TestNestedAffinityUsesTheAnsweringModel(t *testing.T) {
	fresh(t)
	base := candidate{p: provider.Provider{ID: "nested", Key: "k1"}, rest: "nested", model: "m", effort: "high", affinity: "off"}
	wrongModel, wrongEffort := base, base
	wrongModel.model, wrongModel.affinity = "another", "strict"
	wrongEffort.effort, wrongEffort.affinity = "low", "strict"
	cs := []candidate{wrongModel, wrongEffort, base}
	body := []byte(`{"messages":[{"role":"user","content":"hi"}]}`)
	const scope = "group/outer"
	sticks.Lock()
	sticks.m[scope+"|"+conversationID(nil, body)] = stick{who: base.who(), model: base.model, effort: base.effort, at: time.Now()}
	sticks.Unlock()
	_, _, a, _ := affine(scope, "strict", false, nil, provider.Chat, body, cs, planned{order: make([]Weighed, len(cs))})
	if a.Mode != "off" || a.Why != "off" || a.Kept {
		t.Fatalf("used a sibling model's mode: %+v", a)
	}
	// The exact model is capped too: a sibling's strict must not restore
	// this off conversation before affine gets to inspect it.
	other := candidate{p: provider.Provider{ID: "other", Key: "k2"}, rest: "other", model: "m"}
	left := []Weighed{{ID: base.rest, Model: base.model, Fixed: base.effort, Capped: 70}}
	got, _, held := strictKeep(scope, "strict", nil, body, []candidate{other}, planned{capped: cs, left: left})
	if len(got) != 1 || got[0].who() != other.who() || held != nil {
		t.Fatalf("a sibling model restored the capped account: %v %v", got, held)
	}
}

func TestNestedAffinityHeldAccountThroughGateway(t *testing.T) {
	for _, creditHeld := range []bool{false, true} {
		for _, mode := range []string{"strict", "off"} {
			t.Run(fmt.Sprintf("credits=%v/%s", creditHeld, mode), func(t *testing.T) {
				b := strictSetup(t)
				capUsage(t, map[string]float64{"me@example.com": 60, "spare@example.com": 20})
				if err := provider.SetAccountCap("codex", "me@example.com", 70); err != nil {
					t.Fatal(err)
				}
				outer := "off"
				if mode == "off" {
					outer = "strict"
				}
				for _, g := range []provider.Group{
					{ID: "inner", Members: []string{"codex/gpt-5.5"}, Routing: provider.Ordered, Affinity: mode},
					{ID: "outer", Members: []string{"group/inner"}, Routing: provider.Ordered, Affinity: outer},
				} {
					if err := provider.SaveGroup(g); err != nil {
						t.Fatal(err)
					}
				}
				s := New()
				const body = `{"model":"group/outer","messages":[{"role":"user","content":"hi"}]}`
				if code, out := postAs(t, s, "cap-session", body); code != 200 {
					t.Fatalf("first: %d %s", code, out)
				}
				tried, _ := b.seen()
				if tried != "acct-1" {
					t.Fatalf("first tried %s", tried)
				}
				used := 75.0
				if creditHeld {
					used = 100
					if err := provider.SetAccountCap("codex", "me@example.com", 0); err != nil {
						t.Fatal(err)
					}
					if err := provider.SetCodexCredits("me@example.com", false); err != nil {
						t.Fatal(err)
					}
				}
				capUsage(t, map[string]float64{"me@example.com": used, "spare@example.com": 20})
				b.mu.Lock()
				b.tried = nil
				b.mu.Unlock()
				code, out := postAs(t, s, "cap-session", body)
				tried, _ = b.seen()
				want, status := "acct-2", 200
				if mode == "strict" {
					want = "acct-1"
					if creditHeld {
						want, status = "", 429
					}
				}
				if code != status || tried != want {
					t.Fatalf("held: %d %s tried %q, want %d %q", code, out, tried, status, want)
				}
				if mode == "strict" && creditHeld && !strings.Contains(out, "Stays: Strict") {
					t.Fatalf("missing strict refusal: %s", out)
				}
			})
		}
	}
}
