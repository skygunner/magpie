package gui

import (
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"

	"github.com/yetone/magpie/internal/agent"
	"github.com/yetone/magpie/internal/gateway"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/settings"
)

func cooldownRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/cooldown", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		id, ref := r.URL.Query().Get("id"), r.URL.Query().Get("ref")
		p := provider.Provider{}
		if id != "" {
			found, err := provider.Find(id)
			if err != nil {
				fail(w, err)
				return
			}
			p = *found
			if ref != "" {
				var ok bool
				ref, ok = p.AccountRefOf(ref)
				if !ok {
					fail(w, fmt.Errorf("unknown account/key"))
					return
				}
				ref = strings.ToLower(ref)
			}
		} else if ref != "" {
			fail(w, fmt.Errorf("an account/key requires a provider"))
			return
		}
		if r.Method == "POST" {
			var in settings.CooldownPolicy
			dec := json.NewDecoder(io.LimitReader(r.Body, 64<<10))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&in); err != nil {
				fail(w, err)
				return
			}
			if err := provider.CheckCooldown(in); err != nil {
				fail(w, err)
				return
			}
			if id == "" {
				if err := checkCooldownStart(in, provider.ResolveCooldown(p, ref, in)); err != nil {
					fail(w, err)
					return
				}
				if err := settings.SetCooldown(in); err != nil {
					fail(w, err)
					return
				}
			} else {
				if ref == "" {
					p.Cooldown = in
					p.RestCap, p.QuotaRest, p.QuotaResetAt, p.ModelCooldown, p.LimitRestOff = 0, 0, "", nil, false
				} else {
					p.AccountCooldown = maps.Clone(p.AccountCooldown)
					if p.AccountCooldown == nil {
						p.AccountCooldown = map[string]settings.CooldownPolicy{}
					}
					p.AccountCooldown[ref] = in
				}
				if err := checkCooldownStart(in, p.CooldownFor(ref)); err != nil {
					fail(w, err)
					return
				}
				if err := provider.Save(p); err != nil {
					fail(w, err)
					return
				}
				agent.SyncCatalogSoon()
			}
		}
		global := settings.Load().Cooldown
		raw := global
		inherited := provider.ResolveCooldown(provider.Provider{}, "", settings.CooldownPolicy{})
		if id != "" {
			inherited = provider.ResolveCooldown(provider.Provider{}, "", global)
		}
		if ref != "" {
			inherited = p.CooldownFor("")
		}
		legacy := settings.CooldownPolicy{}
		if id != "" {
			raw = p.Cooldown
			legacy = p.LegacyCooldown()
			if ref != "" {
				raw = p.AccountCooldown[ref]
				legacy = settings.CooldownPolicy{}
			}
		}
		rests := []gateway.Rest{}
		if id != "" && ref != "" {
			key := p.ID + "@" + ref
			if p.Account == nil {
				for _, k := range p.KeyList() {
					if k.ID == ref {
						key = p.RestKey(k)
						break
					}
				}
			}
			rests = gateway.CooldownRests(key)
		}
		groups := []map[string]any{}
		if id != "" {
			find := provider.GroupFinder()
			for _, g := range provider.Groups() {
				if g.Hidden {
					continue
				}
				_, members, ok := find(provider.GroupPrefix + g.ID)
				if !ok {
					continue
				}
				for _, m := range members {
					if m.Provider.ID != id {
						continue
					}
					wait := g.FirstToken
					for _, inner := range m.Via {
						if inner.FirstToken > 0 {
							wait = inner.FirstToken
						}
					}
					groups = append(groups, map[string]any{"name": g.Name, "model": m.Model, "seconds": wait})
				}
			}
		}
		writeJSON(w, map[string]any{"raw": raw, "legacy": legacy, "effective": p.CooldownFor(ref), "inherited": inherited, "rests": rests, "groups": groups})
	})
}

func checkCooldownStart(in settings.CooldownPolicy, v provider.CooldownValues) error {
	if in.RateStart != nil && *in.RateStart > v.RateCap {
		return fmt.Errorf("rateStart exceeds the effective rateCap")
	}
	if in.SlowStart != nil && *in.SlowStart > v.SlowCap {
		return fmt.Errorf("slowStart exceeds the effective slowCap")
	}
	return nil
}
