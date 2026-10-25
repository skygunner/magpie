# Round 18 combined suite notes (2026-10-25)

Merged tree 5985a6b1 (7aa04408 #2866 + 69eaf094 #2777 + b8f2a4c8 #3303 on
upstream/main 42976f9a), serial go test -tags nogui ./..., GOTOOLCHAIN=go1.26.8,
umask 022: 68 packages ok, 3 FAIL, 19 no-test, exit=1.

- #3280 TestClaudeClientCacheAbsentUnchanged and #2207 TestPluginSOCKSProxy:
  the two long-known local reds, red on every round's baseline too.
- TestOpenCodeSessionOnEveryCall is #3284, filed by this fork: red once under
  full-suite load across rounds (round 7 recorded it green in the package at
  the then head), green when the package runs alone - rerun at this merged
  head, single test, count=1: ok. Load flake, not a round-18 regression.

The original log stands as written, exit=1; it does not read as all-green.
