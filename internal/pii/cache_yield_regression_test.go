package pii

import (
	"context"
	"crypto/sha256"
	"sync/atomic"
	"testing"

	"github.com/yetone/magpie/internal/redact"
)

func TestCacheFilledWhileYielding(t *testing.T) {
	for _, bounded := range []bool{false, true} {
		t.Run(map[bool]string{false: "strict", true: "bounded"}[bounded], func(t *testing.T) {
			r := newReader(t, 12)
			texts := []string{"Alice12345", "zzzzzzzzzz"}
			var calls atomic.Int32
			r.heldBack = func() error {
				if calls.Add(1) == 3 {
					r.cache.mu.Lock()
					r.cache.put(sha256.Sum256([]byte(texts[0])), []redact.Found{{
						Start: 0, End: 5, Label: "private_person", Score: .9,
					}})
					r.cache.mu.Unlock()
				}
				return nil
			}
			ctx := context.Background()
			if bounded {
				ctx = Bounded(ctx)
			}
			found, err := r.find(ctx, texts)
			if err != nil {
				t.Fatalf("cache was filled before the next chunk: %v", err)
			}
			if len(found) != 2 || len(found[0]) != 1 ||
				found[0][0].Start != 0 || found[0][0].End != 5 {
				t.Fatalf("cached detection must be returned: %#v", found)
			}
		})
	}
}
