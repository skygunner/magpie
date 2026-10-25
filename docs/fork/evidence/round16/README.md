# Round 16 evidence (2026-10-25)

- full-suite-combined.log: serial `go test -tags nogui ./...` at the combined
  head bc193d08 (merge of 58a0f39c #3117, 11d176bd #3043, 4264a4b7 #3470
  follow-up on upstream/main b027ca26). 69 packages ok, 3 FAIL, 19 no-test:
  #3280 and #2207 (the two long-known local reds) plus
  TestHandedOverGatewayFinishesItsStream.
- handedover-baseline-check.txt records the flake attribution: the same test
  run in bursts on a clean upstream/main b027ca26 checkout fails the same way
  (2 of 3 bursts red, including runs with no round-16 code in the tree), so
  it is a pre-existing load flake of the new baseline, not a merge
  regression. Single runs pass on both trees; -count=10 on the combined tree
  red once, on the base 0 of 10 in one burst and 2 of 12 in burst runs.
- per-branch package suites and their red/green records are in each commit
  message (58a0f39c, 11d176bd, 4264a4b7).
