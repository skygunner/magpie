package gateway

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/usage"
)

// A search magpie sends a plan's search service or a search API itself
// spends that plan's or key's allowance, and left no row in Usage ›
// Requests (#3528). Each is a call with Kind web_search, under its
// provider (a search API by its name), with no tokens: no model was asked,
// and the tokens Usage's totals add up don't move.
func TestSearchesWriteARequestRow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	setHome(t, t.TempDir())

	rows := func() []usage.Record { return usage.Load(time.Time{}) }
	tokens := func() string {
		var in, out, cacheRead, cacheWrite, reasoning int
		for _, r := range rows() {
			in, out, cacheRead, cacheWrite, reasoning = in+r.Input, out+r.Output, cacheRead+r.CacheRead, cacheWrite+r.CacheWrite, reasoning+r.Reasoning
		}
		return fmt.Sprint(in, out, cacheRead, cacheWrite, reasoning)
	}

	kimi := newKimiPlan(t)
	kp := provider.Provider{ID: "kimi", Name: "Kimi Code", Preset: "kimi-code-cn", Key: "sk-kimi",
		Chat: kimi.URL + "/coding/v1", Models: []string{"kimi-for-coding"}}
	wbSite := httptest.NewServer(&wbSearchSite{})
	t.Cleanup(wbSite.Close)
	wb := provider.Provider{ID: "workbuddy", Name: "WorkBuddy", Chat: wbSite.URL + "/v1",
		Account: &provider.Account{Agent: "workbuddy", User: "a@x.com"}}
	apis := newSearchAPIs(t)
	if err := provider.SetSearchAPI(apis.api("tavily", "tvly-k")); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name     string
		search   func() error
		provider string
	}{
		{"Kimi Code's search service", func() error {
			_, _, err := New().kimiSearch(context.Background(), kp, "latest go")
			return err
		}, "kimi"},
		{"WorkBuddy's search service", func() error {
			_, _, err := New().wbSearch(context.Background(), wb, "Go 1.26 release notes")
			return err
		}, "workbuddy"},
		{"a search API", func() error {
			_, _, err := New().apiSearch(context.Background(), "latest go")
			return err
		}, "Tavily"},
	} {
		t.Run(c.name, func(t *testing.T) {
			before, tokensBefore := len(rows()), tokens()
			if err := c.search(); err != nil {
				t.Fatalf("search: %v", err)
			}
			after := rows()
			if len(after) != before+1 {
				t.Fatalf("%d rows written, want 1", len(after)-before)
			}
			row := after[len(after)-1]
			if row.Kind != "web_search" || row.Provider != c.provider || row.Status != 200 {
				t.Fatalf("row %+v, want a web_search of %s with 200", row, c.provider)
			}
			if row.Input != 0 || row.Output != 0 || row.CacheRead != 0 || row.CacheWrite != 0 || row.Reasoning != 0 {
				t.Fatalf("row %+v, want no tokens", row)
			}
			if got := tokens(); got != tokensBefore {
				t.Fatalf("tokens %s → %s", tokensBefore, got)
			}
		})
	}
}
