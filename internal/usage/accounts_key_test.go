package usage

import (
	"slices"
	"testing"
	"time"
)

// A key plan's requests are told apart by the key that answered them (#3838):
// kimi-code's two keys are choices of the Account filter, each under
// KeyAccountPrefix so a key isn't taken for a sign-in, and the filter keeps
// only that key's calls. A key the provider gives no name is its fingerprint.
func TestRequestPageAccountFiltersAProviderKey(t *testing.T) {
	pageHome(t)
	now := time.Now()
	// the reporter's rows: kimi-code's two keys, named new and old, and no
	// providerAccount on either
	recs := []Record{
		{Time: now, Agent: "pi", Provider: "kimi-code", Host: "api.kimi.ai", ProviderKeyID: "0f1e2d3c4b", ProviderKeyName: "new", Model: "k3.1-pro", Input: 100, Output: 10, Status: 200},
		{Time: now.Add(time.Minute), Agent: "pi", Provider: "kimi-code", Host: "api.kimi.ai", ProviderKeyID: "5a6b7c8d9e", ProviderKeyName: "old", Model: "k3.1-pro", Input: 20, Output: 2, Status: 200},
		{Time: now.Add(2 * time.Minute), Agent: "pi", Provider: "deepseek", Host: "api.deepseek.com", ProviderKeyID: "9f8e7d6c5b", Model: "deepseek-v4", Input: 5, Output: 1, Status: 200},
		{Time: now.Add(3 * time.Minute), Agent: "codex", Provider: "codex", Host: "chatgpt.com as me@example.com", ProviderAccount: "me@example.com", Model: "gpt-5", Input: 7, Status: 200},
	}
	for _, r := range recs {
		Append(r)
	}
	old := KeyAccountPrefix + "old"
	page := QueryPage(All, Filter{Account: old}, 0, 100)
	if page.Total != 1 || len(page.Rows) != 1 || page.Rows[0].ProviderKeyName != "old" {
		t.Fatalf("one provider key's calls: total %d, %+v", page.Total, page.Rows)
	}
	equalPage(t, page, pageFromLedger(All, Filter{Account: old}, 0, 100, LedgerOf(All, Filter{})))
	var ids []string
	for _, g := range page.Accounts {
		ids = append(ids, g.ID)
	}
	slices.Sort(ids)
	if want := []string{"codex@me@example.com", "deepseek@key:9f8e7d6c5b", "kimi-code@key:new", "kimi-code@key:old"}; !slices.Equal(ids, want) {
		t.Fatalf("account choices: %v, want %v", ids, want)
	}
	// a key named like a sign-in is not that account's calls
	if p := QueryPage(All, Filter{Account: "old"}, 0, 100); p.Total != 0 {
		t.Fatalf("a key's name kept %d rows as an account's: %+v", p.Total, p.Rows)
	}
	// the picker shows the key by its own name, told apart from a sign-in
	if name, ok := KeyAccountName(old); !ok || name != "old" {
		t.Fatalf("a provider key's choice: %q, %v", name, ok)
	}
	if _, ok := KeyAccountName("me@example.com"); ok {
		t.Fatalf("an account was read as a provider key")
	}
}
