package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/provider"
)

// nestedRemote is whether the id already carries a remote-magpie
// qualification, written out here as the prefix itself, not taken from
// the package: the list over the wire carries it, and a test that
// followed a constant could not tell it changed.
func nestedRemote(id string) bool { return strings.HasPrefix(id, "remote-magpie/") }

// lostornot's field report (#3834): a Remote magpie provider that
// configuration sync had left pointing into its own chain grew the remote
// model cache by one remote-magpie/ prefix on every model each round of
// model discovery, ids 478 prefixes deep in eleven minutes, until the
// cache was megabytes and the provider's editor unusable. A model whose
// id already carries a remote-magpie qualification is neither kept from
// another magpie's list nor served to one, so a catalog cannot compound:
// this magpie serving itself keeps the list it had.
func TestRemoteMagpieNestingDoesNotCompound(t *testing.T) {
	fresh(t)
	// a relay this magpie serves its own agents from: a chat model and an
	// image one, which the loop below would feed on
	if err := provider.Save(provider.Provider{ID: "relay-a", Name: "Relay A", Key: "k", Models: []string{"claude-sonnet-5"}, Chat: "http://127.0.0.1:1/v1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveLive("relay-a", "http://127.0.0.1:1/v1", []catalog.Model{
		{ID: "claude-sonnet-5", Name: "Claude Sonnet 5"}, {ID: "seedream-4", Name: "Seedream 4", Draws: true},
	}); err != nil {
		t.Fatal(err)
	}
	h := New().Handler()
	self := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	}))
	t.Cleanup(self.Close)
	// the provider as sync left it: its endpoint is this same magpie
	id, err := provider.Add(provider.Provider{ID: "remote-magpie", Name: "Remote magpie", Key: "k", Preset: provider.RemoteMagpiePreset, Chat: strings.TrimPrefix(self.URL, "http://")})
	if err != nil {
		t.Fatal(err)
	}
	p, _ := provider.Find(id)

	// rounds of model discovery: each fetch is this magpie's own list
	// asked by itself, which is what compounded before
	var had []string
	for round := 1; round <= 3; round++ {
		if _, err := p.Fetch(context.Background()); err != nil {
			t.Fatalf("round %d: %v", round, err)
		}
		live, _, _ := catalog.Live("remote-magpie")
		var ids []string
		for _, m := range append(slices.Clone(live), catalog.LiveDrawers("remote-magpie")...) {
			ids = append(ids, m.ID)
		}
		slices.Sort(ids)
		if !slices.Contains(ids, "relay-a/claude-sonnet-5") || !slices.Contains(ids, "relay-a/seedream-4") {
			t.Errorf("round %d: the relay's models aren't both in the list: %v", round, ids)
		}
		for _, got := range ids {
			if nestedRemote(got) {
				t.Errorf("round %d: %s kept, which already carries a remote-magpie qualification", round, got)
			}
		}
		if round >= 3 && !slices.Equal(ids, had) {
			t.Errorf("round %d: the list grew to %v, from %v", round, ids, had)
		}
		had = ids
	}
}

// A model whose id already carries a remote-magpie qualification — one
// kept by a magpie from before the guard, or held from one that still
// serves it — is not served to another magpie, which would list it as its
// own remote's and grow the prefix each round. This magpie's own agents
// are shown it still.
func TestRemoteMagpieNestedNotServed(t *testing.T) {
	fresh(t)
	if err := provider.Save(provider.Provider{ID: "office", Name: "Office", Key: "k", Preset: provider.RemoteMagpiePreset, Chat: "http://127.0.0.1:1/v1"}); err != nil {
		t.Fatal(err)
	}
	if err := catalog.SaveLive("office", "http://127.0.0.1:1/v1", []catalog.Model{
		{ID: "relay-a/claude-sonnet-5", Name: "Claude Sonnet 5"}, {ID: "relay-a/seedream-4", Name: "Seedream 4", Draws: true},
		{ID: "remote-magpie/relay-a/claude-sonnet-5", Name: "Once nested"}, {ID: "remote-magpie/relay-a/seedream-4", Name: "Once nested", Draws: true},
	}); err != nil {
		t.Fatal(err)
	}
	h := New().Handler()
	ids := func(magpie bool) map[string]bool {
		req := httptest.NewRequest("GET", "/v1/models", nil)
		if magpie {
			req.Header.Set(provider.DrawersHeader, "1")
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var out struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		got := map[string]bool{}
		for _, m := range out.Data {
			got[m.ID] = true
		}
		return got
	}
	another := ids(true)
	for _, id := range []string{"office/relay-a/claude-sonnet-5", "office/relay-a/seedream-4"} {
		if !another[id] {
			t.Errorf("%s isn't served to another magpie: %v", id, another)
		}
	}
	for _, id := range []string{"office/remote-magpie/relay-a/claude-sonnet-5", "office/remote-magpie/relay-a/seedream-4"} {
		if another[id] {
			t.Errorf("%s is served to another magpie, its id already carrying a remote-magpie qualification", id)
		}
	}
	if local := ids(false); !local["office/remote-magpie/relay-a/claude-sonnet-5"] {
		t.Errorf("this magpie's own agents aren't shown the nested model: %v", local)
	}
}
