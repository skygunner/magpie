# Magpie upstream follow-up notepad — sweep all 357 replied issues
Started: 2026-11-17T01:39:10.286106Z

## Plan
1. Discovery: upstream=yetone/magpie, fork=skygunner/magpie (local /home/vincenty/Projects/magpie-fix), 414 origin/fix/* branches (mapping saved fork-branches.txt). GitHub search: 357 issues (#1105..#5713) where skygunner commented.
2. Fetch per-issue: detail + all comments + timeline (raw/<n>.json, running detached).
3. Triage: opener language (zh/en via body+title), live state, maintainer/other replies after our comment, merged cross-ref PRs. Classes A absorbed / B needs reply / C competing PR / D no response.
4. B items: update fork branch if code change needed (worktree, tests per repo harness, fast-forward push), reply in opener language, read back URL.
5. Report: report.md + final-inventory.json under this dir. Cleanup worktrees/tmp with receipts.

## Findings
- Hindsight: magpie-issues worker contract required Chinese upstream issue comments for that batch; per-issue language detection still applied.
- No magpie-prod dir locally; PATCHES.md not found at the remembered path; live GitHub data used as source of truth.

## Now
Background fetch of 357 issue JSONs (raw/<n>.json).

## Todo
- wait fetch; build triage table
- classify + language
- replies / branch updates / report / cleanup

## Findings (fetched live, 2026-11-17)
- 357 issues fetched (raw/<n>.json): 315 open, 41 closed/completed, 1 not_planned (#4952). Openers: 133 zh, 224 en. 241 issues map to fork fix branches (branch-map.json).
- 52 issues have maintainer (yetone/OWNER) replies after ours; 66 have only non-maintainer replies after ours; 239 no reply.
- yetone actively cherry-picks skygunner patches with credit ("Landed in vX (commit), authorship kept") — absorption evidence = these comments. Many leave issue OPEN awaiting reporter confirmation.
- #4144: yetone REJECTED our series with precise reasons (needs read-past-corrupt-frame resync + real damaged session file test); ggbdpq rebased/verified ours but maintainer declined. Rework needed.
- Upstream repo owner = yetone; fork = skygunner/magpie; local clone /home/vincenty/Projects/magpie-fix (origin/main at 5e3dd1d52); go1.27.4 available.
## Now
- Two reader children producing verdicts-maint.json (52) and verdicts-others.json (66).

## Progress 2026-11-17 (follow-up sweep completed)
- Replies posted+readback: #2785 (zh, regression ack) 6935745347; #3375 (en, Anarlog) 6935745936; #4945 (en, root-cause correction ack) 6935746577; #4385 (zh -> yetone, rationale ask) 6935799230; #4144 (zh, review accepted + rework direction + ask for real damaged dsh file) 6935860474.
- Final tally A=67 B=5 D=285; report.md + final-inventory.json written.
- Pending follow-up work (not blockers for the sweep): #4144 rework (read-past-corrupt-frame + real damaged file test), #4945 note-placement fix, #2785 regression triage on v0.1.3844.

## Final Audit 2026-11-17
- Objective: Survey + follow-up 357 yetone/magpie upstream issues with fix comments by skygunner.
- S1 Inventory: 357 distinct issues (#1105..#5713), all raw JSONs saved under raw/<n>.json. 241 map to fork branches.
- S2 Live Status: fetched from GitHub API, verified state (315 open, 41 closed/completed, 1 not_planned), all comments after ours, timeline events.
- S3 Triage: A=67 (absorbed), B=5 (needed reply), D=285 (no response/community). Opener languages: 133 zh, 224 en.
- S4 B-Replies: all 5 replied in opener language and read back via API API:
  - #2785 [zh]: issuecomment-6935745347
  - #3375 [en]: issuecomment-6935745936
  - #4144 [zh]: issuecomment-6935860474
  - #4385 [en]: issuecomment-6935799230
  - #4945 [en]: issuecomment-6935746577
- S5 Report & Evidence: report.md, final-inventory.json, posted-replies.json written; Hindsight document ingested. No active worktrees or temp files left behind.
