package gateway

// A rest is a deadline, not a duration to start over when magpie restarts.
// Keep it beside providers.json, which already survives container updates.

import (
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/yetone/magpie/internal/lastgood"
	"github.com/yetone/magpie/internal/provider"
	"github.com/yetone/magpie/internal/steady"
)

type savedRest struct {
	Rest
	Agent string    `json:"agent,omitempty"`
	User  string    `json:"user,omitempty"`
	Said  string    `json:"said,omitempty"`
	Hold  time.Time `json:"hold,omitzero"`
}

var restsSaving sync.Mutex

func restsPath() string { return filepath.Join(filepath.Dir(provider.Path()), "rests.json") }

// loadRests runs once for each configuration directory in this process.
// Tests can clear restingUntil.from to model a fresh process.
func loadRests() {
	path := restsPath()
	restingUntil.Lock()
	defer restingUntil.Unlock()
	if restingUntil.from == path {
		return
	}
	restingUntil.from = path
	b, err := steady.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return
	}
	if err != nil {
		log.Printf("gateway rests: reading %s: %v", path, err)
		return
	}
	var saved map[string]savedRest
	if err := json.Unmarshal(b, &saved); err != nil {
		log.Printf("gateway rests: parsing %s: %v", path, err)
		return
	}
	now := time.Now()
	for id, v := range saved {
		if !keepRest(v.Rest, now) {
			continue
		}
		if _, present := restingUntil.m[id]; present {
			continue
		}
		r := v.Rest
		r.agent, r.user, r.said, r.hold = v.Agent, v.User, v.Said, v.Hold
		restingUntil.m[id], restingUntil.note[id] = r.Until, r
	}
}

// A rate limit's recent end remains useful to its retry backoff ladder.
func keepRest(r Rest, now time.Time) bool {
	return now.Before(r.Until) || r.Why == failRate && now.Sub(r.Until) < rateForget
}

// saveRests serializes writers while taking a fresh snapshot under the rest
// lock. Disk I/O is outside that lock so requests can continue reading rests.
func saveRests() {
	restsSaving.Lock()
	defer restsSaving.Unlock()
	now := time.Now()
	restingUntil.Lock()
	saved := make(map[string]savedRest, len(restingUntil.note))
	for id, r := range restingUntil.note {
		r.Until = restingUntil.m[id]
		if keepRest(r, now) {
			saved[id] = savedRest{Rest: r, Agent: r.agent, User: r.user, Said: r.said, Hold: r.hold}
		}
	}
	restingUntil.Unlock()
	b, err := json.Marshal(saved)
	if err != nil {
		log.Printf("gateway rests: encoding: %v", err)
		return
	}
	path := restsPath()
	if err := lastgood.KeepUnparsed(path, func(b []byte) bool {
		var v map[string]savedRest
		return json.Unmarshal(b, &v) == nil
	}); err == nil {
		err = steady.WriteFile(path, b, 0o600)
	}
	if err != nil {
		log.Printf("gateway rests: saving %s: %v", path, err)
	}
}
