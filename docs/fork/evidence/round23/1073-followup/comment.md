follow-up on current main (`3a89dd53`): the port is now rebased and re-verified there (main moved 511 commits past `7aad6a50`, including the conflict zone this PR touches).

- rebased branch: `fix/settings-served-models` @ [`3acb51e6`](https://github.com/skygunner/magpie/commit/3acb51e62694f2f580720c684cb44b1851b6f7b9), net diff vs main = `internal/gui/api.go`, the doc, and two test files.
- all three of the maintainer's blockers from the earlier review are addressed in the port:
  1. the doc line now says the correct thing — a *new* pick is checked against `provider.Served`, a saved one the provider stopped serving is **kept** (shown as missing on the page), so another save isn't refused; that's also the doc nit @rahul05ranjan's review raised, so the port already covers it.
  2. `imageGens()` keeps `VendorPicks` per entry, as main now requires.
  3. `codex-descriptions` and `codex-predictions` get the same `modelServed` check as titles (the "set on its own" boundary is preserved per route), and the tests cover all five routes.
- tests on current main: `TestSettingsModelsMustBeServed`, `TestImageGenCodexOneEntry`, `TestCodexAutoReview`, `TestCodexTitles` — 9/9 pass; `go vet -tags nogui ./internal/gui`, `gofmt -l internal/gui` clean. The five routes still return 200 for `fake/missing` and `fake/` without the change (verified against current main before the port), so the defect is still live on `3a89dd53`.

The PR head (`2681c8b5`) is still the original and does not merge onto current main; the rebased port above is the current form. If TryWorld2026 lands the PR itself, the rebase needs `VendorPicks` kept in `imageGens()` and the two sibling routes added — the maintainer's points 2 and 3.
