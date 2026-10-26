# Heavy artifacts relocated (2026-10-27 closeout)

Regenerable heavy artifacts from the round-25 evidence (clean-base / base /
red-base / exit-proof-base / deterministic-base / sandbox / _scratch source
trees, 3620/base.tar, 3165 built binaries) were moved out of the repository
to keep the ledger commit reviewable. They live at:

    ~/Projects/magpie-issues/evidence/round25-heavy/<issue>/<name>

Every one of them is reproducible from the recorded base SHAs in each
issue's summary.json (git archive / go build). The light evidence kept here
(red/green/verify logs, diff.patch, summary.json, comment readbacks,
audit reports) is the record that matters.
