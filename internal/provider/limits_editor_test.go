package provider

import (
	"maps"
	"testing"

	"github.com/yetone/magpie/internal/catalog"
	"github.com/yetone/magpie/internal/settings"
)

// A provider's editor sets its reply limits as a whole, as it sets its
// windows (ARNO on Discord: a model's maxTokens was wrong and only the
// window could be set there): those it leaves out come off, another
// provider's stay, a model it doesn't serve is refused with nothing kept,
// and a save that changes nothing tells the agents nothing.
func TestModelOutputsSetFromTheEditor(t *testing.T) {
	limitsHome(t)
	for _, id := range []string{"relay", "other"} {
		if err := Save(Provider{ID: id, Name: id, Chat: "https://relay.example/v1", Key: "k", Models: []string{"sol", "luna"}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := SetModelOutput("relay/luna", 1000); err != nil {
		t.Fatal(err)
	}
	if err := SetModelOutput("other/sol", 2000); err != nil {
		t.Fatal(err)
	}
	if got := OutputsOf("relay"); !maps.Equal(got, map[string]int{"luna": 1000}) {
		t.Fatalf("OutputsOf: %v", got)
	}
	told := 0
	catalog.Changed = func() { told++ }
	t.Cleanup(func() { catalog.Changed = nil })

	if err := SetModelOutputs("relay", map[string]int{"*": 32000, "sol": 128000}); err != nil {
		t.Fatal(err)
	}
	if told != 1 {
		t.Errorf("told the agents %d times; want once", told)
	}
	want := map[string]int{"relay/*": 32000, "relay/sol": 128000, "other/sol": 2000}
	if got := settings.Load().ModelOutputs; !maps.Equal(got, want) {
		t.Errorf("outputs %v; want %v", got, want)
	}
	if e, _ := EntryOf("relay/sol"); e.Output != 128000 {
		t.Errorf("relay/sol output %d", e.Output)
	}
	if e, _ := EntryOf("relay/luna"); e.Output != 32000 {
		t.Errorf("relay/luna output %d; want the provider's 32000", e.Output)
	}

	told = 0
	if err := SetModelOutputs("relay", map[string]int{"*": 32000, "sol": 128000}); err != nil || told != 0 {
		t.Errorf("the same again: %v, told %d", err, told)
	}
	if err := SetModelOutputs("relay", map[string]int{"typo": 5}); err == nil {
		t.Error("a model relay doesn't serve was accepted")
	}
	if got := settings.Load().ModelOutputs; !maps.Equal(got, want) {
		t.Errorf("a refused save changed them: %v", got)
	}
	if err := SetModelOutputs("relay", map[string]int{}); err != nil {
		t.Fatal(err)
	}
	if got := settings.Load().ModelOutputs; !maps.Equal(got, map[string]int{"other/sol": 2000}) {
		t.Errorf("emptied: %v", got)
	}
}

// #3305: a catalog can retire a model after its limit was saved. Carrying
// the unchanged value through an editor save must not block all other edits.
func TestEditorKeepsRetiredModelLimits(t *testing.T) {
	for _, compact := range []bool{false, true} {
		name := "output"
		if compact {
			name = "compact"
		}
		t.Run(name, func(t *testing.T) {
			limitsHome(t)
			if err := Save(Provider{ID: "relay", Name: "Relay", Chat: "https://relay.example/v1", Key: "k", Models: []string{"current"}}); err != nil {
				t.Fatal(err)
			}
			s := settings.Load()
			old := map[string]int{"relay/retired": 65536, "other/model": 1024}
			set := SetModelOutputs
			if compact {
				s.ModelCompacts = old
				set = SetModelCompacts
			} else {
				s.ModelOutputs = old
			}
			if err := settings.Save(s); err != nil {
				t.Fatal(err)
			}
			read := func() map[string]int {
				s := settings.Load()
				if compact {
					return s.ModelCompacts
				}
				return s.ModelOutputs
			}
			if err := set("relay", map[string]int{"retired": 65536, "current": 8192}); err != nil {
				t.Fatal(err)
			}
			want := map[string]int{"relay/retired": 65536, "relay/current": 8192, "other/model": 1024}
			if !maps.Equal(read(), want) {
				t.Fatalf("saved: %v", read())
			}
			for _, bad := range []map[string]int{{"retired": 1234}, {"typo": 65536}, {"retired": -1}} {
				if err := set("relay", bad); err == nil {
					t.Errorf("accepted invalid limits: %v", bad)
				}
				if !maps.Equal(read(), want) {
					t.Fatalf("refused save changed limits: %v", read())
				}
			}
			if err := set("relay", map[string]int{"retired": 0, "current": 8192}); err != nil {
				t.Fatal(err)
			}
			delete(want, "relay/retired")
			if !maps.Equal(read(), want) {
				t.Fatalf("removed: %v", read())
			}
			if err := set("relay", map[string]int{"retired": 65536}); err == nil {
				t.Error("recreated a retired model limit")
			}
		})
	}
}
