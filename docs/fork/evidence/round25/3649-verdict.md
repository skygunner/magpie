# #3649 disposition: EXCLUDED - not verifiable deterministically

Issue: proc.ListeningOn walks every /proc/<pid>/fd when many descriptors are open, so the GUI quit path times out and the port's holder is never named (reported on a machine with 19.68m open descriptors; a walk took ~90s while the take-over allows 30s).

## Why it is not delivered

The defect is a *wall-clock* failure: a single process's descriptor directory is enormous, and no realistic test can create millions of real file descriptors on this host, nor can a fixed clock be injected into os.ReadDir/os.Readlink. Every deterministic design I tried fails one of the objective's gates:

1. A synthetic /proc fixture behind an environment variable ("MAGPIE_TEST_PROC") that the clean base ignores: the clean base reads the real /proc, so the added test fails for the wrong reason (it walks the real machine and finds nothing). That is not a same-input RED; the objective requires the new test to fail on the clean base with only the test added, because of the defect, not because of a missing seam. Rejected.
2. Adding the seam to the production code and then demonstrating the difference: the RED would then be a compile failure or unchanged behavior at base (the base has no per-batch context check), and the "red" would be about the seam, not about the reported timeout. Also a fabricated RED. Rejected.
3. A test that opens a real listening socket plus a large number of real descriptors (a few thousand) and cancels the context: the clean base checks the context *between processes*, so it returns context.Canceled either way; the test cannot distinguish "the walk was cut between processes" (base behavior, already correct) from "the walk was cut inside one process's descriptor directory" (the defect). The distinguishing input needs millions of descriptors and a real deadline. Rejected.

## What is preserved for a human follow-up

`candidate-diff-untried.patch` holds the unfinished candidate change in the worktree /home/vincenty/Projects/magpie-issues/r25-proc-listening-bound (uncommitted, nothing pushed, nothing commented on the issue). It shows the shape a real fix would take:

- the walk's context is checked between descriptors as well as between processes, so one enormous directory cannot outlast the take-over's deadline;
- magpie-named processes are read first, so the pid the take-over actually needs is named even when the rest of the machine does not fit the budget;
- `quitMagpieOnPort` acts on a partial (possibly incomplete) pid list only after every pid's program is verified to be magpie's, which is the check it already performs before ending anything, and it gives the port-let-go wait a deadline of its own instead of reusing the walk's expired one.

None of that is a delivered fix: no RED/GREEN pair exists, nothing is pushed, and the issue is not commented. A real delivery needs a machine-scale reproduction or an upstream test seam this repo does not have.
