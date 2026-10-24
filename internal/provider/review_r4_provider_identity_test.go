package provider

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/yetone/magpie/internal/plugin"
)

// A plugin's stored decision override remains account settings after a
// local sign-out. This includes a provider renamed with the documented
// -plugin suffix because its raw id belongs to a built-in preset.
func TestReviewR4SignedOutDecisionPicksStaySettings(t *testing.T) {
	for _, rawID := range []string{"cline", "openai"} {
		t.Run(rawID, func(t *testing.T) {
			clinePlugin(t)
			t.Cleanup(plugin.Told)
			ps := plugin.Cached()
			ps[0].ID, ps[0].Name = rawID, rawID
			ps[0].Accounts = []plugin.Account{{Key: rawID, Type: "api"}}
			b, err := json.Marshal(map[string]map[string]any{rawID: {"type": "api", "key": "synthetic-key"}})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(plugin.AuthPath(), b, 0o600); err != nil {
				t.Fatal(err)
			}
			plugin.UseCached(ps)
			id := PluginID(rawID)
			want := rawID
			if rawID == "openai" {
				want += "-plugin"
			}
			if id != want {
				t.Fatalf("plugin fixture id=%q, want %q", id, want)
			}
			p, err := Find(id)
			if err != nil || !p.IsPlugin() {
				t.Fatalf("signed-in plugin fixture is missing: %+v %v", p, err)
			}
			p.Decide = "https://fake.invalid/v1"
			p.Models = []string{"typesafe/jev"}
			if err := Save(*p); err != nil {
				t.Fatal(err)
			}
			if p, err := Find(id); err != nil || !p.IsPlugin() {
				t.Fatalf("saving decide changed the signed-in identity: %+v %v", p, err)
			}
			if plugin.Running() {
				t.Fatal("fixture unexpectedly started a plugin host")
			}
			if err := plugin.SignOut(context.Background(), rawID, ""); err != nil {
				t.Fatal(err)
			}
			// Keep the installed provider's synthetic listing available;
			// sign-out invalidates it, but no host or network is needed to
			// establish that its persisted account settings stay settings.
			plugin.UseCached(ps)
			if len(plugin.Auths(rawID)) != 0 {
				t.Fatal("sign-out did not clear synthetic authentication")
			}
			if stored, ok := find(load().Providers, id); !ok || stored.Decide != "https://fake.invalid/v1" {
				t.Fatalf("sign-out discarded the stored decision settings: %+v %v", stored, ok)
			}
			if p, err := Find(id); err == nil {
				t.Fatalf("signed-out account picks became a standalone provider: id=%s plugin=%v account=%v decide=%s", p.ID, p.IsPlugin(), p.Account != nil, p.Decide)
			}
		})
	}
}

// Metadata joins explicit picks without changing the existing choice of
// a default when the plugin already declares a decision model.
func TestReviewR4NativeDecisionDefaultKeepsPriority(t *testing.T) {
	clinePlugin(t)
	ps := plugin.Cached()
	ps[0].Decide = true
	ps[0].Models = append(ps[0].Models, plugin.Model{ID: "native-decision", Name: "Native Decision", Decides: true})
	plugin.UseCached(ps)
	p, err := Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	p.Models = []string{"typesafe/jev"}
	if err := Save(*p); err != nil {
		t.Fatal(err)
	}
	p, err = Find("cline")
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Jev(); got != "native-decision" {
		t.Fatalf("explicit pick changed the feed's established default: %q", got)
	}
	if _, got, err := RouteDecider("cline"); err != nil || got != "native-decision" {
		t.Fatalf("bare provider default route: %q %v", got, err)
	}
}
