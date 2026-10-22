package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/yetone/magpie/internal/agentenv"
)

func isolateRefreshes(t *testing.T) {
	t.Helper()
	isolate(t)
	h := t.TempDir()
	t.Setenv("HOME", h)
	t.Setenv("USERPROFILE", h)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(h, ".config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(h, ".cache"))
	t.Setenv("PATH", h)
	for _, v := range agentenv.Vars {
		t.Setenv(v, "")
	}
	reset := func() {
		keyBalanceCache.Lock()
		keyBalanceCache.data, keyBalanceCache.at = nil, time.Time{}
		keyBalanceCache.Unlock()
		planQuotaCache.Lock()
		planQuotaCache.data, planQuotaCache.at = nil, time.Time{}
		planQuotaCache.Unlock()
		subscriptionUsageCache.Lock()
		subscriptionUsageCache.data, subscriptionUsageCache.at, subscriptionUsageCache.asked = nil, time.Time{}, false
		subscriptionUsageCache.Unlock()
		lastQuotas.Lock()
		lastQuotas.m, lastQuotas.loaded = nil, false
		lastQuotas.Unlock()
	}
	reset()
	t.Cleanup(reset)
}

func waitRefresh(t *testing.T, done <-chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("usage request did not finish")
	}
}

// The whole-page read starts first but its spare-key response arrives
// after the card's refresh. It must keep that card's new figure while
// still updating the other cards, including in the response to its caller.
func TestRefreshUsageKeepsNewerKeyReading(t *testing.T) {
	for _, kind := range []string{"balance", "plan"} {
		for _, older := range []struct {
			name           string
			whole, refused bool
		}{
			{"whole page", true, false},
			{"same card", false, false},
			{"whole page error", true, true},
			{"same card error", false, true},
		} {
			t.Run(kind+"/"+older.name, func(t *testing.T) {
				isolateRefreshes(t)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				arrived, release := make(chan struct{}), make(chan struct{})
				var once sync.Once
				var mu sync.Mutex
				calls := map[string]int{}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					key := r.Header.Get("Authorization")
					mu.Lock()
					calls[key]++
					n := calls[key]
					mu.Unlock()
					value := 0
					switch key {
					case "Bearer sk-one":
						value = 100
						if n > 1 {
							value = 90
						}
					case "Bearer sk-other":
						value = 50
						if n > 1 {
							value = 40
						}
					case "Bearer sk-two":
						value = 25
						if n == 2 {
							close(arrived)
							<-release
							if older.refused {
								w.WriteHeader(http.StatusUnauthorized)
								return
							}
						} else if n == 3 {
							value = 10
						} else if n > 3 {
							w.WriteHeader(http.StatusServiceUnavailable)
							return
						}
					}
					if kind == "balance" {
						fmt.Fprintf(w, `{"data":{"total_available":%d}}`, value*500000)
					} else {
						fmt.Fprintf(w, `{"usage":{"rolling":{"percent":%d}}}`, value)
					}
				}))
				t.Cleanup(srv.Close)
				read := KeyBalances
				expire := func() {
					keyBalanceCache.Lock()
					keyBalanceCache.at = time.Time{}
					keyBalanceCache.Unlock()
				}
				if kind == "plan" {
					oldTransport := http.DefaultClient.Transport
					http.DefaultClient.Transport = rewrite{srv}
					t.Cleanup(func() { http.DefaultClient.Transport = oldTransport })
					read = PlanQuotas
					expire = func() {
						planQuotaCache.Lock()
						planQuotaCache.at = time.Time{}
						planQuotaCache.Unlock()
					}
				}
				for _, p := range []Provider{
					{ID: "relay", Name: "Relay", Key: "sk-one", KeyName: "main", Keys: []KeyAccount{{Name: "spare", Key: "sk-two"}}},
					{ID: "other", Name: "Other", Key: "sk-other"},
				} {
					p.Chat = "https://opencode.ai/zen/go/v1"
					if kind == "balance" {
						p.Chat, p.BalanceURL, p.BalancePath = srv.URL+"/v1", srv.URL+"/balance", "$data.total_available / 500000"
					}
					if err := Save(p); err != nil {
						t.Fatal(err)
					}
				}
				assertCards := func(got []SubscriptionQuota, main, spare, other int) {
					t.Helper()
					want := map[string]int{"relay/main": main, "relay/spare": spare, "other/": other}
					for _, q := range got {
						key := q.Provider + "/" + q.User
						value := "$" + strconv.Itoa(want[key]) + ".00"
						actual := q.Balance
						if kind == "plan" {
							value = strconv.Itoa(want[key])
							if len(q.Windows) == 1 {
								actual = strconv.FormatFloat(q.Windows[0].Used, 'f', -1, 64)
							}
						}
						if _, ok := want[key]; !ok || q.Error != "" || actual != value {
							t.Errorf("card %s = %+v, want %s", key, q, value)
						}
						delete(want, key)
					}
					if len(want) > 0 {
						t.Errorf("missing cards: %v", want)
					}
				}
				assertCards(read(ctx), 100, 25, 50)
				if older.whole {
					expire()
				}
				var got []SubscriptionQuota
				done := make(chan struct{})
				go func() {
					if older.whole {
						got = read(ctx)
					} else {
						RefreshUsage(ctx, "relay", "spare")
					}
					close(done)
				}()
				t.Cleanup(func() {
					once.Do(func() { close(release) })
					<-done
				})
				waitRefresh(t, arrived)
				RefreshUsage(ctx, "relay", "Spare")
				once.Do(func() { close(release) })
				waitRefresh(t, done)
				if older.whole {
					assertCards(got, 90, 10, 40)
					assertCards(read(ctx), 90, 10, 40)
				} else {
					assertCards(read(ctx), 100, 10, 50)
				}
				// The late response must not corrupt quotas.json either: a
				// subsequent 503, after restarting its cache, uses the new value.
				lastQuotas.Lock()
				lastQuotas.m, lastQuotas.loaded = nil, false
				lastQuotas.Unlock()
				expire()
				assertCards(read(ctx), 90, 10, 40)
			})
		}
	}
}

func TestRefreshUsageKeepsNewerSubscription(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(fmt.Sprintf("late error=%t", refused), func(t *testing.T) {
			isolateRefreshes(t)
			home := signIn(t)
			if err := os.RemoveAll(filepath.Join(home, ".config", "github-copilot")); err != nil {
				t.Fatal(err)
			}
			arrived, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			var mu sync.Mutex
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				calls++
				n := calls
				mu.Unlock()
				value := 80
				if n == 1 {
					value = 20
					close(arrived)
					<-release
					if refused {
						w.WriteHeader(http.StatusUnauthorized)
						return
					}
				} else if n > 2 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				fmt.Fprintf(w, `{"plan_type":"pro","rate_limit":{"primary_window":{"used_percent":%d,"limit_window_seconds":18000}}}`, value)
			}))
			t.Cleanup(srv.Close)
			old := CodexBase
			CodexBase = srv.URL + "/backend-api/codex"
			t.Cleanup(func() { CodexBase = old })
			c := &subscriptionUsageCache
			c.Lock()
			c.data = []SubscriptionQuota{{Provider: "codex", User: "me@example.com", Windows: []QuotaWindow{{Used: 10}}}}
			c.Unlock()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			SubscriptionUsage(ctx)
			c.Lock()
			done := c.pending
			c.Unlock()
			t.Cleanup(func() {
				once.Do(func() { close(release) })
				<-done
			})
			waitRefresh(t, arrived)
			RefreshUsage(ctx, "codex", "me@example.com")
			once.Do(func() { close(release) })
			waitRefresh(t, done)
			got := SubscriptionUsage(ctx)
			if len(got) != 1 || got[0].Error != "" || len(got[0].Windows) != 1 || got[0].Windows[0].Used != 80 {
				t.Fatalf("after late subscription response: %+v, want 80%%", got)
			}
			lastQuotas.Lock()
			lastQuotas.m, lastQuotas.loaded = nil, false
			lastQuotas.Unlock()
			c.Lock()
			c.at, c.asked = time.Time{}, true
			c.Unlock()
			got = SubscriptionUsage(ctx)
			if len(got) != 1 || got[0].Error != "" || len(got[0].Windows) != 1 || got[0].Windows[0].Used != 80 || got[0].AsOf == nil {
				t.Fatalf("subscription fallback: %+v, want saved 80%%", got)
			}
		})
	}
}

// #3139: a successful first card refresh is fresh even before the page
// has ever fetched the full list. It must not fall back to a dated disk copy.
func TestFirstClaudeCardRefreshStaysFresh(t *testing.T) {
	isolateRefreshes(t)
	out, _ := claudeUsageCards(t, "Current session: 37% used", nil)
	ctx := context.Background()
	check := func(want float64) {
		t.Helper()
		q := cardOf(CachedCards(time.Now()), "claude", "a@example.com")
		if q == nil || q.Error != "" || q.AsOf != nil || q.ReadAt == nil || len(q.Windows) != 1 || q.Windows[0].Used != want {
			t.Fatalf("single refresh: %+v, want fresh %.0f%%", q, want)
		}
	}
	RefreshUsage(ctx, "claude", "a@example.com")
	check(37)
	out.Store("Current session: 38% used")
	RefreshUsage(ctx, "claude", "a@example.com")
	check(37)          // the CLI's minimum refresh interval still holds
	retryClaudeUsage() // advance the retry clock, not a real 30-second sleep
	RefreshUsage(ctx, "claude", "a@example.com")
	check(38)
	c := &subscriptionUsageCache
	c.Lock()
	at := c.at
	c.Unlock()
	if !at.IsZero() {
		t.Fatal("a single card refresh marked the whole page fresh")
	}
	AskUsage()
	SubscriptionUsage(Asked(ctx))
	check(38)
	out.Store("Current session: 39% used")
	retryClaudeUsage()
	RefreshUsage(ctx, "claude", "a@example.com")
	check(39)
}

func TestQuotaMergeStartsWithNoCards(t *testing.T) {
	q := SubscriptionQuota{Provider: "fixture", User: "a", readSeq: 2}
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprintf("partial=%t", partial), func(t *testing.T) {
			var kept, got []SubscriptionQuota
			if partial {
				got = []SubscriptionQuota{q}
			} else {
				kept = []SubscriptionQuota{q} // newer than an empty whole-page read
			}
			out := cacheCards(kept, got, 1, partial)
			if len(out) != 1 || out[0].Provider != q.Provider {
				t.Fatalf("lost the card while merging into nil: %+v", out)
			}
		})
	}
}
