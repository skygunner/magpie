## Summary

In Magpie 0.1.1906, a concurrent cache update can make `backend.find` use a stale `miss` list after yielding and reacquiring its model turn. In strict mode this can produce a false failure:

```
masking failed, so nothing was sent: the local masking model has no text to read for 1 texts
```

With a synthetic bounded-mode unit test, the same stale state also returns without a newly cached detection. This is a unit-test result; I did not inspect any real outbound request or establish that real user data was exposed. Please retain fail-closed behavior for an actual classifier failure.

## Environment and evidence

- Magpie v0.1.1906, source commit `6104e88aa958ec39e974bab740c70f41a98bb55f`, Windows, Go 1.26.8.
- Observed the error above with local-model redaction enabled. The real occurrence cannot be proven from its log alone to have used the interleaving below.
- On the exact tag, the two tests below fail without the candidate change and pass with it. The tests deterministically inject the cache update at the held-back check; they simulate the interleaving rather than launching a real concurrent model server.
- The same relevant `known → missing → lenient/joinAt` sequence was still present when inspecting main, but I have not run the tests on main.

## Reproduction

Add this synthetic test file as `internal/pii/cache_yield_regression_test.go` on the tag (it uses the existing `newReader` helper):

```go
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
                // The third check follows the first chunk and reacquisition.
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
```

Run `go test ./internal/pii -run '^TestCacheFilledWhileYielding$' -count=1`. On the unmodified tag, the equivalent strict test fails with `the local masking model has no text to read for 1 texts`, while the bounded test returns no cached span.

## Suspected cause and candidate

After `b.known(texts, got)` produces `miss`, another request can populate the remaining entries while this request gives up the model turn. Then `b.missing(texts, miss, got)` is empty even though `miss` still contains indices. Strict mode reaches the empty-`chunks` error; bounded mode can break before refreshing `out`.

A local candidate refreshes `out, miss = b.known(texts, got)` if `units` is empty, before either mode's branch. It breaks only when no misses remain; otherwise it retains an error. It must not turn a classifier failure into success or discard known spans.

The affected package tests, masking and redact tests, selected gateway tests, and the three targeted tests (strict interleaving, bounded cached span, and classifier failure) passed on the local candidate; the targeted tests were also repeated 100 times each. This is not a claim that the proposed patch has been reviewed or production-validated.

Only synthetic test strings are included. AI assistance was used to investigate and prepare this report.
