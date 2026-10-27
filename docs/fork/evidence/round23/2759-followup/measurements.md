# #2759 follow-up — 2026-10-27

Rebased `fix/import-source-row` onto current `upstream/main` `11dcc743`
(the delivery was based on `c0985122`, 172 commits back). The branch is now
one commit over main: `f45e9fa3` (cherry-pick of `0483a01f`), same two CSS
lines plus the one test file. `git diff 11dcc743` is 2 files, +83/-2.

## Red → green re-proved on the new base

`red-on-11dcc743.log`: clean `11dcc743` with only the test file — 8/8 fail
(Chromium/WebKit × en/zh × 390px/900px), e.g.
`chromium en 900px … OpenCode1 is cut off: 507.6..642.7 of 564.0`.
`green.log`: head — 8/8 pass.

## The boundary the delivery said was unasserted

The delivery note said: "on a short viewport the row takes three lines, and
whether the dialog still scrolls fully is not asserted". Measured, not
asserted (Chromium, 390px wide, stub app with 5 sources):

| window height | tab row | sheet content | dialog scroll | Import bar |
| --- | --- | --- | --- | --- |
| 900 | 104px (3 lines) | 353px | none needed | inside |
| 600 | 104px | 353px | none needed | inside |
| 400 | 104px | 353px | none needed | inside |
| 300 | 104px | 284px | none needed | inside |
| 250 | 104px | 244px | none needed | inside |

With 8 providers per source the `.applist` scrolls inside the sheet
(`listOverflow` 118px at 600px, 230px at 400px) and the bar stays inside the
window; `.dialog` still never needs its own scroll. On the base the row was
36px, so wrapping costs 68px of height and nothing else changes: the sheet
fits in every height measured down to 250px.

An assertion for this was written and dropped: it passed on the base and it
still passed with `.modal .dialog`'s `overflow-y` forced to `hidden`, so it
could not fail for the reason it claimed. It is not in the branch.

## Suite on the new base

- `internal/gui/tests/import-source-tabs.test.cjs`: 8/8 pass.
- import/discovery GUI tests (import-opencode, import-collision,
  provider-discovery): 66/66 pass.
- `go test -tags nogui ./internal/gui`: ok (426 tests).
- `go vet ./internal/gui`, `gofmt -l internal/gui`: clean.
- `go build ./...`, `GOOS=linux -tags nogui`, `GOOS=windows`: exit 0.
