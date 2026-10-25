# Round 16 evidence (2026-10-25; corrected and completed 2026-10-25 round 17)

- The full log is ../../round16-full-suite-combined.log (one level up, not in
  this folder); SHA256
  7f1be897ce0ac6dbba02a05d3d19b6836fa3c4461e450e002419970bc3d182bd, kept as it
  was written, exit=1: 69 packages ok, 3 tests failed (#3280
  TestClaudeClientCacheAbsentUnchanged, #2207 TestPluginSOCKSProxy,
  TestHandedOverGatewayFinishesItsStream), 19 no-test. It does not read as an
  all-green run.
- handedover-baseline-check.txt is the first, partial attribution attempt: it
  kept only the passing bursts' tails, not the failures' output, and does not
  carry the ratios the first README text claimed. It stands as what it is.
- The attribution has been redone with full raw output per burst (HEAD,
  command, every line, exit code):
  - handedover-rerun-baseline.log — clean upstream b027ca26 checkout: 3
    bursts of `go test ./internal/gateway -run TestHandedOverGateway
    -count=4`; 2 bursts green, burst 3 red (exit=1,
    TestHandedOverGatewayFinishesItsStream).
  - handedover-rerun-combined.log — the round-16 combined tree (merge of
    58a0f39c + 11d176bd + 4264a4b7): 1 of 3 bursts green, 2 red.
  The flake reproduces on the untouched baseline, so
  TestHandedOverGatewayFinishesItsStream stands attributed to the new
  baseline, not the round-16 merges. The rates differ run to run (this rerun:
  1/3 baseline, 2/3 combined) — a load-sensitive flake either way; no claim
  of an exact ratio is made.
- manifest.json lists the files with SHA256s.