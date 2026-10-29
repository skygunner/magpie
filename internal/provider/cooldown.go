package provider

import (
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/yetone/magpie/internal/settings"
)

type CooldownValues struct {
	RateStart     int               `json:"rateStart"`
	RateCap       int               `json:"rateCap"`
	SlowStart     int               `json:"slowStart"`
	SlowThreshold int               `json:"slowThreshold"`
	SlowCap       int               `json:"slowCap"`
	FailureCap    int               `json:"failureCap"`
	QuotaRest     int               `json:"quotaRest"`
	QuotaResetAt  string            `json:"quotaResetAt"`
	FailRest      bool              `json:"failRest"`
	LimitRest     bool              `json:"limitRest"`
	Models        map[string]int    `json:"models"`
	Source        map[string]string `json:"source"`
}

// CooldownFor resolves one identity: account email or key fingerprint. Legacy
// provider fields remain overrides over global defaults until edited in the UI.
func (p Provider) CooldownFor(ref string) CooldownValues {
	return ResolveCooldown(p, ref, settings.Load().Cooldown)
}

// ResolveCooldown resolves against supplied global defaults for unsaved previews.
func ResolveCooldown(p Provider, ref string, global settings.CooldownPolicy) CooldownValues {
	v := CooldownValues{RateStart: 2, RateCap: 64, SlowStart: 1, SlowThreshold: 2, SlowCap: 10, FailureCap: 10, QuotaRest: 30, FailRest: true, LimitRest: true, Models: map[string]int{}, Source: map[string]string{}}
	apply := func(o settings.CooldownPolicy, source string) {
		for k, pair := range map[string]struct {
			from *int
			to   *int
		}{
			"rateStart": {o.RateStart, &v.RateStart}, "rateCap": {o.RateCap, &v.RateCap},
			"slowStart": {o.SlowStart, &v.SlowStart}, "slowThreshold": {o.SlowThreshold, &v.SlowThreshold},
			"slowCap": {o.SlowCap, &v.SlowCap}, "failureCap": {o.FailureCap, &v.FailureCap}, "quotaRest": {o.QuotaRest, &v.QuotaRest},
		} {
			if pair.from != nil {
				*pair.to = *pair.from
				v.Source[k] = source
			}
		}
		if o.QuotaResetAt != nil {
			v.QuotaResetAt = *o.QuotaResetAt
			v.Source["quotaResetAt"] = source
		}
		if o.FailRest != nil {
			v.FailRest = *o.FailRest
			v.Source["failRest"] = source
		}
		if o.LimitRest != nil {
			v.LimitRest = *o.LimitRest
			v.Source["limitRest"] = source
		}
		for m, cap := range o.Models {
			v.Models[m] = cap
			v.Source["model:"+m] = source
		}
	}
	apply(global, "global")
	legacy := p.LegacyCooldown()
	apply(legacy, "legacy")
	apply(p.Cooldown, "provider")
	apply(p.AccountCooldown[strings.ToLower(ref)], "account")
	// A lower cap inherited independently cannot make the first wait exceed it.
	v.RateStart = min(max(v.RateStart, 1), max(v.RateCap, 1))
	v.RateCap = max(v.RateCap, 1)
	v.SlowStart = min(max(v.SlowStart, 1), max(v.SlowCap, 1))
	v.SlowCap = max(v.SlowCap, 1)
	v.SlowThreshold = max(v.SlowThreshold, 1)
	v.QuotaRest = max(v.QuotaRest, 1)
	v.FailureCap = max(v.FailureCap, 1)
	return v
}
func (p Provider) CooldownIdentity() string {
	if p.Account != nil {
		return strings.ToLower(p.Account.User)
	}
	if p.Key != "" {
		return KeyID(p.Key)
	}
	return ""
}
func (p Provider) ModelCooldownCap(model string) int {
	return p.CooldownFor(p.CooldownIdentity()).Models[model]
}
func (p Provider) FailureRestCap() time.Duration {
	return time.Duration(p.CooldownFor(p.CooldownIdentity()).FailureCap) * time.Minute
}

func CheckCooldown(o settings.CooldownPolicy) error {
	for k, n := range map[string]*int{"rateStart": o.RateStart, "rateCap": o.RateCap, "slowStart": o.SlowStart, "slowCap": o.SlowCap, "failureCap": o.FailureCap, "quotaRest": o.QuotaRest, "slowThreshold": o.SlowThreshold} {
		if n != nil && (*n < 1 || *n > MaxQuotaRest) {
			return fmt.Errorf("%s must be between 1 and %d", k, MaxQuotaRest)
		}
	}
	if o.RateStart != nil && o.RateCap != nil && *o.RateStart > *o.RateCap {
		return fmt.Errorf("rateStart exceeds rateCap")
	}
	if o.SlowStart != nil && o.SlowCap != nil && *o.SlowStart > *o.SlowCap {
		return fmt.Errorf("slowStart exceeds slowCap")
	}
	if o.QuotaResetAt != nil {
		if _, err := ParseResetAt(*o.QuotaResetAt); err != nil {
			return err
		}
	}
	for model, cap := range o.Models {
		if strings.TrimSpace(model) == "" || cap < 0 || cap > 60 {
			return fmt.Errorf("model cooldown requires an exact model ID and a cap from 0 (off) to 60 minutes")
		}
	}
	return nil
}
func cloneCooldownValue[T any](p *T) *T {
	if p == nil {
		return nil
	}
	v := *p
	return &v
}
func CloneCooldown(o settings.CooldownPolicy) settings.CooldownPolicy {
	o.RateStart = cloneCooldownValue(o.RateStart)
	o.RateCap = cloneCooldownValue(o.RateCap)
	o.SlowStart = cloneCooldownValue(o.SlowStart)
	o.SlowThreshold = cloneCooldownValue(o.SlowThreshold)
	o.SlowCap = cloneCooldownValue(o.SlowCap)
	o.FailureCap = cloneCooldownValue(o.FailureCap)
	o.QuotaRest = cloneCooldownValue(o.QuotaRest)
	o.QuotaResetAt = cloneCooldownValue(o.QuotaResetAt)
	o.FailRest = cloneCooldownValue(o.FailRest)
	o.LimitRest = cloneCooldownValue(o.LimitRest)
	o.Models = maps.Clone(o.Models)
	return o
}

func (p Provider) LegacyCooldown() settings.CooldownPolicy {
	legacy := settings.CooldownPolicy{Models: p.ModelCooldown}
	if p.RestCap > 0 {
		legacy.RateCap = &p.RestCap
		legacy.SlowCap = &p.RestCap
		legacy.FailureCap = &p.RestCap
	}
	if p.RestCap == RestCapOff {
		b := false
		legacy.FailRest = &b
	}
	if p.LimitRestOff {
		b := false
		legacy.LimitRest = &b
	}
	if p.QuotaRest > 0 {
		legacy.QuotaRest = &p.QuotaRest
		s := ""
		legacy.QuotaResetAt = &s
	}
	if p.QuotaResetAt != "" {
		legacy.QuotaResetAt = &p.QuotaResetAt
	}
	return legacy
}
