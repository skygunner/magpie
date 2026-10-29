# Future-batch duplicate-search results (partial)

Stopped at the first `gh` error, as requested. No GitHub search succeeded and no finding was assessed against live search results. All 63 findings in [future-dedup-gap.md](future-dedup-gap.md) remain unsearched; do not infer that any lacks a duplicate.

## Attempted search

- Finding: P2-1, non-streaming group first-response failover. The local FIX-MAP and audit report associate it with issue #3788; this was not verified on GitHub in this run.
- Exact command: `gh search issues 'repo:yetone/magpie "non-streaming" failover timeout' --limit 30 --json number,title,url,state,body`
- Exit status: 1. Output:

```text
Invalid search query "( repo:\"yetone/magpie \\\"non-streaming\\\" failover timeout\" ) type:issue".
The listed users and repositories cannot be searched either because the resources do not exist or you do not have permission to view them.
```

- Result URLs: none returned. Candidate decision: unassessed. The error does not establish whether the repository is accessible, whether issue #3788 matches, or whether other duplicates exist. No retry was attempted.

## Local context

The prior gap record identifies P2-5 with #4101, and identifies #3197 and #3444 as potentially related to P2-4 and P3-1 respectively, without claiming exact duplication. These are historical local references, not results of this search. `docs/fork/prior-comments.json` and `docs/fork/upstream-status.json` contain neither #3788 nor #4101; absence from these historical ledgers does not establish absence upstream.
