# Round 19 evidence (2026-10-25)

- ../round19-package-suites.log: gateway/provider/proc/agent on the round-19
  merged tree (upstream/main 1b039a3c + 741237cf #2741 at the time of the run).
  Three reds, each attributed, none from the round's changes:
  - #3280 TestClaudeClientCacheAbsentUnchanged: the long-known local red, on
    every round's baseline.
  - TestHandedOverGatewayFinishesItsStream: the b027ca26-era load flake,
    attributed with raw burst logs in round 16/17 evidence (base red 1/3
    bursts, combined 2/3); this run's red is that same flake.
  - TestNoCommandBypassesProc: upstream regression, NOT ours — yetone's
    75e78a5a (keep.go, 2026-10-25 12:42 +0800) used os/exec; proven red on a
    clean 1b039a3c checkout (-count=2, both red), matches his own #3567 and CI
    run 40166079788. Fixed this round as #3567 (64bd29fa).
- Combined tree after all three fixes (9e7f5275 = upstream 1b039a3c +
  741237cf + 55dfe0a5 + 64bd29fa): pii/proc/tui all ok, and the guard test
  green at -count=3. The full gateway suite is not re-run here; #2741's
  package run (one red, #3280) is in its commit body, and gateway's other
  known reds are the two attributed above.
- 3539-typo-fix.json: the #3539 investigation comment's stray-word edit,
  content otherwise unchanged, read back with SHA256.
