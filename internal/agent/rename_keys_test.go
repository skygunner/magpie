package agent

import (
	"slices"
	"testing"

	"github.com/yetone/magpie/internal/access"
	"github.com/yetone/magpie/internal/provider"
)

// #3630: a gateway key's chosen models and accounts follow the provider
// renamed. A key held to some of a provider's models kept the id it had
// ("old/m1") and the editor showed them as "not currently provided", while
// a request on them was refused: the provider itself still served its
// models by the new id.
func TestRenameMovesGatewayKeyModels(t *testing.T) {
	syncHome(t)
	if err := provider.Save(provider.Provider{ID: "old", Name: "Old", Key: "k", Chat: "http://127.0.0.1:1/v1", Models: []string{"m1", "m2"}}); err != nil {
		t.Fatal(err)
	}
	if err := provider.Save(provider.Provider{ID: "other", Name: "Other", Key: "k", Chat: "http://127.0.0.1:2/v1", Models: []string{"m1", "m2"}}); err != nil {
		t.Fatal(err)
	}
	p, err := provider.Find("old")
	if err != nil {
		t.Fatal(err)
	}
	acct := p.AccountIDs()[0].ID
	secret, err := access.Update("add-key", access.Change{Name: "Laptop"})
	if err != nil {
		t.Fatal(err)
	}
	keys, err := access.List()
	if err != nil || len(keys) != 1 {
		t.Fatal(keys, err)
	}
	id := keys[0].ID
	// the key may use two of the provider's models, every model of it, a
	// group, and one model of another provider: only the first two move
	if _, err := access.Update("models-key", access.Change{Key: id, Models: []string{"old/m1", "old/*", "group/top", "other/m2"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := access.Update("accounts-key", access.Change{Key: id, Accounts: []string{"old/" + acct}}); err != nil {
		t.Fatal(err)
	}

	if _, err := RenameProvider("old", "new"); err != nil {
		t.Fatal(err)
	}
	keys, err = access.List()
	if err != nil || len(keys) != 1 {
		t.Fatal(keys, err)
	}
	if want := []string{"new/m1", "new/*", "group/top", "other/m2"}; !slices.Equal(keys[0].Models, want) {
		t.Errorf("models %v, want %v", keys[0].Models, want)
	}
	if want := []string{"new/" + acct}; !slices.Equal(keys[0].Accounts, want) {
		t.Errorf("accounts %v, want %v", keys[0].Accounts, want)
	}
	// the key allows what it allowed, of the provider by its id now, and
	// nothing more
	who, ok := access.Authenticate(secret)
	if !ok {
		t.Fatal("the key no longer authenticates")
	}
	if !who.Allows("new/m1") || !who.Allows("new/m2") || !who.Allows("new/m3") || !who.Allows("other/m2") {
		t.Errorf("allows %v: it no longer allows what it listed", who.Models)
	}
	if who.Allows("other/m1") {
		t.Errorf("allows %v: it allows what it did not list", who.Models)
	}
	if !who.AllowsAccount("new", acct) || who.AllowsAccount("new", "someone-else") {
		t.Errorf("accounts %v: it is no longer held to the account it listed", who.Accounts)
	}
}
