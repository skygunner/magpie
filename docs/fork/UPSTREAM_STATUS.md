# Fork delivery status — skygunner/magpie → yetone/magpie

Checked 2026-10-24T03:20:00Z against upstream/main `fa22bc64`. Generated from `upstream-status.json` by `docs/fork/generate.py` — edit the JSON, not this file.

## Fix units

### u-volcengine-resets — Volcengine/Ark reset times (+0800 CST) unread by resetsWhen; monthly trust window

- Issues: #3251, #3276 (product)
- Branch: `fix/volcengine-resets` base `87d703c5c263de0293091d95200cc02f741b1a47` commits `e6c2b6fb`
- Coverage: Layer 1 only: offset parsing and the monthly 35-day trust window. Layer 2 of #3251 (AccessKey plan windows into routing) is NOT implemented and remains open upstream.
- Upstream: adopted_with_changes at `8278fd9b` (v0.1.1325)
- Verification: pass_upstream — 2026-10-24, this machine: TestResetsWritten, TestMonthlyWrittenResetRestsTheMonth, TestResetsZhipu all pass on upstream/main fa22bc64 (which carries 8278fd9b) — running upstream's narrowed implementation and its own tests, not replaying the fork's e6c2b6fb. Historical: fork red-green 10-22 (TestResetsWritten ark case + TestMonthlyWrittenResetRestsTheMonth); reporter CNife confirmed the fix live 10-23 (#3251 comment); #3276 closed completed.
- **Archive candidate**: scope adopted upstream with release; no unique outstanding change; layer 2 tracked in the ledger, not on this branch earliest review 2027-01-23

### u-shared-session-folder — Two agents sharing one folder under different spellings read twice (pi/omp, qoder/qoder-cn)

- Issues: #3258, #3255 (product)
- Branch: `fix/shared-session-folder` base `be002aca` commits `4c3d6e26`
- Coverage: The two spelled pairs; #3255's exactly-identical path case was already covered by upstream 572a1757 (distinctPaths).
- Upstream: none_yet
- Verification: pass — TestSharedFolderReadOnce red->green 10-22; full nogui suite ok

### u-null-tool-calls — mimo: tool_calls:null treated as a call happened

- Issues: #3259 (product)
- Branch: `fix/null-tool-calls` base `be002aca` commits `d2abf49`
- Coverage: chattidy null==[] parity.
- Upstream: none_yet
- Verification: pass — TestChatTidyNullToolCalls red->green 10-22; full nogui suite ok

### u-account-rest-cap — rest-cap/quotaRest set on a subscription provider dropped (withPicks)

- Issues: #3076, #3176 (product)
- Branch: `fix/account-rest-cap` base `87d703c5` commits `c41b6feb`
- Coverage: The three fields carried onto accounts and AlsoOn.
- Upstream: none_yet
- Verification: pass — TestAccountKeepsItsRests red->green 10-22; validated by ggbdpq 10-22 (issue comment) as equivalent to his hunt53-d-restcap

### u-unknown-model-sent-id — 404 for unknown auto model reports the id the client sent, not a synthetic one

- Issues: #3150 (product)
- Branch: `fix/unknown-model-sent-id` base `be002aca` commits `38444c2d`
- Coverage: (none recorded)
- Upstream: none_yet
- Verification: pass — TestUnknownModelNamesTheIdSent red->green 10-22

### u-union-items — Antigravity schema unions beside items/properties; fold order-independent

- Issues: #3121, #3173, #3370 (product)
- Branch: `fix/union-items` base `0815c918` commits `6ae89820` `b2336968`
- Rebased tree: `fix/union-items-v2` base `fa22bc64` commits `d4e6998f` `99d6e6ed` `cbc6c3b3`
- Coverage: #3121 reporter's ["string","array"] shape; #3173 LIMITED shape coverage — the reporter's full 309-tool schema was never obtained, and the plain items.oneOf example was already correct before; only nested type-lists on the same path were tested. #3370 object-half keyword cleanup, both member orders.
- Upstream: none_yet
- Verification: pass — union-items-v2 cbc6c3b3: TestPlainSchemaUnionFoldIsOrderIndependent red->green (order [array,object] failed before), family green; schema-output only, no real Antigravity call

### u-queue-model-rest — Per-model queue 503 rests the model+account, not the whole account

- Issues: #3213 (product)
- Branch: `fix/queue-model-rest` base `be002aca->50eac57f (rebased 10-23)` commits `50eac57f`
- Coverage: (none recorded)
- Upstream: none_yet
- Verification: pass — TestQueuedModelRestsAlone red->green 10-23 rebase; full nogui ok

### u-qoder-ide-code — Qoder IDE detected as both VS Code and Qoder

- Issues: #3148 (product)
- Branch: `fix/qoder-ide-code` base `origin/main 10-23` commits `4aa39dbf`
- Coverage: vscodeCodeBinary exclusion using reporter-provided product.json values.
- Upstream: none_yet
- Verification: pass — red->green with real code control group; internal/agent ok

### u-batch-1023am — 10-22/10-23 batch verification chains (fix/issue-batch-20261023*, shared stack)

- Issues: #2259, #2287, #2323, #2389, #2460, #2473, #2530, #2923, #2953, #3020, #3035, #3062, #3097, #3100, #3139, #3141, #3152, #3153, #3165, #3183, #3193, #3197, #3205, #3224, #3233, #3234, #3260, #3266, #3277, #3280, #3282, #3284, #3285, #3289, #3302, #3306, #3311 (mixed)
- Branch: `fix/issue-batch-20261023* (chain heads; see branches map)` base `7aad6a50 (and later merges of upstream main)` commits `see bran`
- Coverage: Each issue's fix is its own commit inside the shared verification stack; see the branch map for which branch head carries which tail commit. Sources: 2530/3141/3152/3153/3165/3193/3205/3224 adopt other authors' patches with Co-authored-by preserved; 3020/3183 verified by ggbdpq upstream-side.
- Upstream: none_yet
- Verification: pass — per-commit red-green recorded in commit bodies; ggbdpq independently verified ~20 of the adapted branches 10-23 (cherry-pick + ablation) with no rework requested

### u-3339-disabled-models — Antigravity -disabled placeholder ids left out of the model list

- Issues: #3339, #3027 (product)
- Branch: `fix/issue-batch-1023pm (old head)` base `0815c918` commits `8b4ecbac`
- Rebased tree: `fix/issue-batch-1023pm-v2` base `fa22bc64` commits `fbaa9a67`
- Coverage: (none recorded)
- Upstream: none_yet
- Verification: pass — TestAntigravityDisabledSuffixNotOffered red->green; tier tests unchanged

### u-3374-agent-message-relay — Codex subagent task in agent_message reaches Responses-API key providers

- Issues: #3374 (product)
- Branch: `fix/issue-batch-1023pm (old head)` base `0815c918` commits `67b57000`
- Rebased tree: `fix/issue-batch-1023pm-v2` base `fa22bc64` commits `34dcacff` `a77f4eba` `c2602273` `97a8a0c2`
- Coverage: Superseded mechanism: the delivered 67b57000 bare rewrite is retired in favor of #3183's PlainAgentMessage reading (97a8a0c2), which keeps author/recipient and images. #3183's own two commits (b1ae66c7 request+reply sides, eae195b1 Copilot native) remain the product path; this unit's remaining value is the composition test through /backend-api/codex/responses.
- Upstream: none_yet
- Verification: pass — v2 97a8a0c2: TestAgentMessageRelaysAsUserTurn (From/To lead + task text through /backend-api/codex/responses), TestSealedAgentMessageIsRefusedNotForwarded (400 sealed, 0 upstream calls)

### u-3352-balance-only — Balance-only usage readings know the account

- Issues: #3352 (product)
- Branch: `fix/issue-batch-1023pm (old head)` base `0815c918` commits `4fb09e6b`
- Rebased tree: `fix/issue-batch-1023pm-v2` base `fa22bc64` commits `6e3c2bd1`
- Coverage: (none recorded)
- Upstream: none_yet
- Verification: pass — live + persisted path red-green

### u-3349-decide — decide= on signed-in/plugin providers persists and routes the picked decision model

- Issues: #3349 (product)
- Branch: `fix/issue-batch-1023pm (old head)` base `0815c918` commits `b2497dd8` `007b8b9b`
- Rebased tree: `fix/issue-batch-1023pm-v2` base `fa22bc64` commits `26500e21` `1fb9a44e` `683547f0` `1e7644d1`
- Coverage: v2 completes the loop: field saved (b2497dd8-equivalent), identity stable (683547f0), plugin's own declared decide kept when no override (1fb9a44e-equivalent), and RouteDecider/SystemOne answer the explicitly picked model end to end (1e7644d1). Real Command Code endpoint behavior was the reporter's own live-API verification; ours is local routing and protocol construction only.
- Upstream: none_yet
- Verification: pass — v2: TestAccountKeepsItsDecide, TestAccountDecideStaysOffCustomIDs (PluginID stable), TestPluginPickedDecisionModelRoutes (save->reread->RouteDecider->/v1/systemone through the plugin fetch: path, model, Bearer k1; red = the review's exact error text)

### u-3365-redact — redact: credential fields keep version-shaped values; placeholder passwords learn when real; reindex keeps learning state

- Issues: #3365 (product)
- Branch: `fix/issue-batch-1023pm (old head)` base `0815c918` commits `4b6d93d9`
- Rebased tree: `fix/issue-batch-1023pm-v2` base `fa22bc64` commits `91bbe995` `9c720bc0` `fde6695f`
- Coverage: v2: field-aware exemption (A), learned-eligibility upgrade + rebuild survival (B), both against upstream's sentKnownSpans cache (50f52b54) with fresh texts at every step.
- Upstream: none_yet
- Verification: pass — A: TestCredentialFieldsKeepVersionShapedSecrets red (n=0 as-written) then green; B: TestPlaceholderPasswordLearnsWhenReal + TestReindexKeepsLearningState red->green; TestSentTextKeepsItsMasking and TestMaskedConversationKeepsItsCachedPrefix green unchanged; whole redact package green

### u-3358-empty-chat-reply — An ordinary Chat upstream that says nothing fails the reply, not the turn

- Issues: #3358 (product)
- Branch: `fix/issue-batch-1023pm (old head)` base `0815c918` commits `600fa01c`
- Rebased tree: `fix/issue-batch-1023pm-v2` base `fa22bc64` commits `28c6dcab` `95537ceb`
- Coverage: emptyFails += Chat with title-ask exemption; v2 adds the translated route's positive controls (text, tool call, length, filter; streamed and not).
- Upstream: none_yet
- Verification: pass — TestChatEmptyReplyIsAnError red->green; TestChatEndingsKeepTheirSemanticsThroughResponses green; gateway package green except known #3280

### u-web-signal-shutdown — magpie web exits cleanly on SIGTERM/SIGINT (ported PR 1258) + plugin host reaped before reexec

- Issues: #1258, #3285 (product+test)
- Branch: `fix/web-signal-shutdown` base `upstream/main at rebuild` commits `44d850d8` `eb890865` `c5334b59`
- Coverage: Port of AmAzing129's PR #1258 with Co-authored-by preserved; #3285 is our own; the fixture cleanup commit is test-only.
- Upstream: none_yet
- Verification: pass — rebuild 10-23: cli signal/web family 22 tests + TestWebReexecReapsPluginHost green on Linux; three injected regressions red as designed (original verification)

### u-persist-gateway-rests — Gateway rest deadlines persist across restarts; save errors reported; slow rests reach the ledger

- Issues: #2008 (product)
- Branch: `fix/persist-gateway-rests` base `upstream/main at rebuild` commits `2fc84b93` `fcb0767a`
- Rebased tree: `fix/persist-gateway-rests-v2` base `fa22bc64` commits `047520f0` `ae193b86` `5035eb0b`
- Coverage: (none recorded)
- Upstream: none_yet
- Verification: pass — v2 5035eb0b: TestSlowRestReachesTheLedger red (no rests file) -> green (absolute deadline, reason, status, failures, model restored after simulated restart); TestQuotaRestSurvivesGatewayRestart, TestSlowStartsRestTheModelOnItsAccount green

### u-test-fixes-1022 — Test/CI fixes: socks-proxy Bun 1.4.3, hold under load, sign-in race, credential cache, gui timing x3, sh-echo portability (filed by us: #3280, #3284), panel log isolation

- Issues: #2207, #3095, #3100, #3174, #3229, #3277, #3280, #3284, #3311 (test)
- Branch: `various (fix/socks-proxy-test, fix/hold-test-expiry, fix/signin-plugin-hook-race, fix/claude-signin-test-cache, fix/quota-picker-save-test, fix/codex-subagent-story-test, fix/claude-harness-bytes, fix/opencode-session-test, fix/panel-test-request-isolation)` base `various` commits `d16f32c ` `665af242` `f85bd166` `873c5dfd` `cebfc283` `435c03ac` `ef857324` `30b23a3f` `4671590d`
- Coverage: (none recorded)
- Upstream: none_yet
- Verification: pass — each red->green recorded in its commit body; several verified by ggbdpq

### u-ported-prs — Ports of TryWorld2026's open PRs #1072/#1073 onto current main

- Issues: #1072, #1073 (product)
- Branch: `fix/issue-batch-20261023-x and -y (chain tails)` base `chain` commits `bcdee087` `0644f6fc`
- Coverage: (none recorded)
- Upstream: none_yet
- Verification: unverified — delivered on chain; no per-unit regression test cited — treat as unverified until re-checked

## Pointed elsewhere / no fix attempted

- #2949: upstream already fixed: b641868e (TestHeldClock zone)
- #3143, #3144: upstream already fixed: 88119d4f
- #3151: upstream already fixed: e6568888
- #3255: upstream 572a1757 covers identical-path case (v0.1.1320); different-spelling pairs remain our u-shared-session-folder
- #3345: same bug as #2932, fixed in v0.1.1320 (572a1757); reporter tested only to 1319
- #3131: belongs to the community plugins repo (magpie-community/plugins); pointed there
- #3147: waiting on reporter for conversations/*.db sample; no fix attempted

## Branch inventory (from git, this read)

89 remote branches. `batch` chains carry many units; `single` carry one; `maintenance`/`mirror` never archive.

- `` fix head `be002aca` base `be002aca` 0 commit(s)
- `dev` maintenance head `269f6fe8` base `87d703c5` 4 commit(s)
- `fix/account-rest-cap` fix head `c41b6feb` base `87d703c5` 1 commit(s)
- `fix/autovision-catalog-images` fix head `7e156213` base `7aad6a50` 48 commit(s)
- `fix/backup-unreadable-plugins` fix head `c43ab301` base `7aad6a50` 51 commit(s)
- `fix/balance-quota-left` fix head `15759adb` base `7aad6a50` 36 commit(s)
- `fix/claude-exit-drift` fix head `18c0967e` base `7aad6a50` 21 commit(s)
- `fix/claude-harness-bytes` fix head `ef857324` base `ca4a1cde` 1 commit(s)
- `fix/claude-signin-test-cache` fix head `873c5dfd` base `7aad6a50` 16 commit(s)
- `fix/claude-task-notification` fix head `b7726551` base `7aad6a50` 43 commit(s)
- `fix/codex-subagent-story-test` fix head `435c03ac` base `7aad6a50` 16 commit(s)
- `fix/codex-usage-only` fix head `a016fcee` base `7aad6a50` 46 commit(s)
- `fix/deepseek-reasoning-aliases` fix head `d5b46244` base `7aad6a50` 31 commit(s)
- `fix/desktop-credential-probe` fix head `9fd7a9d4` base `7aad6a50` 47 commit(s)
- `fix/embedding-indices` fix head `38e36a7d` base `7aad6a50` 1 commit(s)
- `fix/gemini-ollama-options` fix head `a0b3c8f6` base `7aad6a50` 37 commit(s)
- `fix/grok-plugin-allowed-tools` fix head `852182f6` base `7aad6a50` 32 commit(s)
- `fix/hardlink-test-mode` fix head `dda612bc` base `ca4a1cde` 2 commit(s)
- `fix/hermes-vision-mapping` fix head `d03d4a7b` base `7aad6a50` 50 commit(s)
- `fix/hold-test-expiry` fix head `665af242` base `7aad6a50` 38 commit(s)
- `fix/initial-single-quota-refresh` fix head `a11d605e` base `7aad6a50` 35 commit(s)
- `fix/issue-batch-1023pm` fix head `007b8b9b` base `0815c918` 7 commit(s)
- `fix/issue-batch-1023pm-v2` fix head `95537ceb` base `fa22bc64` 15 commit(s)
- `fix/issue-batch-20261023` fix head `065f0c8f` base `ca4a1cde` 11 commit(s)
- `fix/issue-batch-20261023-b` fix head `0644f6fc` base `7aad6a50` 15 commit(s)
- `fix/issue-batch-20261023-c` fix head `0437ae4c` base `7aad6a50` 18 commit(s)
- `fix/issue-batch-20261023-d` fix head `348fb756` base `7aad6a50` 20 commit(s)
- `fix/issue-batch-20261023-e` fix head `c61bf9b7` base `7aad6a50` 24 commit(s)
- `fix/issue-batch-20261023-f` fix head `3833e3f3` base `7aad6a50` 27 commit(s)
- `fix/issue-batch-20261023-g` fix head `4b137b70` base `7aad6a50` 30 commit(s)
- `fix/issue-batch-20261023-i` fix head `deefacd2` base `7aad6a50` 33 commit(s)
- `fix/issue-batch-20261023-k` fix head `a11d605e` base `7aad6a50` 35 commit(s)
- `fix/issue-batch-20261023-l` fix head `15759adb` base `7aad6a50` 36 commit(s)
- `fix/issue-batch-20261023-m` fix head `4671590d` base `7aad6a50` 40 commit(s)
- `fix/issue-batch-20261023-n` fix head `cf95652a` base `7aad6a50` 41 commit(s)
- `fix/issue-batch-20261023-o` fix head `8f931a28` base `7aad6a50` 42 commit(s)
- `fix/issue-batch-20261023-p` fix head `b7726551` base `7aad6a50` 43 commit(s)
- `fix/issue-batch-20261023-q` fix head `1805ff94` base `7aad6a50` 44 commit(s)
- `fix/issue-batch-20261023-r` fix head `2bddfe10` base `7aad6a50` 45 commit(s)
- `fix/issue-batch-20261023-s` fix head `a016fcee` base `7aad6a50` 46 commit(s)
- `fix/issue-batch-20261023-t` fix head `9fd7a9d4` base `7aad6a50` 47 commit(s)
- `fix/issue-batch-20261023-u` fix head `7e156213` base `7aad6a50` 48 commit(s)
- `fix/issue-batch-20261023-v` fix head `ee1b018b` base `7aad6a50` 49 commit(s)
- `fix/issue-batch-20261023-w` fix head `d03d4a7b` base `7aad6a50` 50 commit(s)
- `fix/issue-batch-20261023-x` fix head `c43ab301` base `7aad6a50` 51 commit(s)
- `fix/issue-batch-20261023-y` fix head `ec3c82b1` base `7aad6a50` 52 commit(s)
- `fix/json-array-index` fix head `1805ff94` base `7aad6a50` 44 commit(s)
- `fix/kimi-ref-chain` fix head `f5ec664b` base `7aad6a50` 25 commit(s)
- `fix/kimi-tool-root` fix head `2bddfe10` base `7aad6a50` 45 commit(s)
- `fix/native-codex-collaboration` fix head `eae195b1` base `7aad6a50` 29 commit(s)
- `fix/nested-group-affinity` fix head `ec3c82b1` base `7aad6a50` 52 commit(s)
- `fix/null-tool-calls` fix head `d2abf49a` base `87d703c5` 1 commit(s)
- `fix/opencode-session-test` fix head `30b23a3f` base `ca4a1cde` 1 commit(s)
- `fix/panel-test-request-isolation` fix head `4671590d` base `7aad6a50` 40 commit(s)
- `fix/path-same-tick-install` fix head `89bceb39` base `7aad6a50` 34 commit(s)
- `fix/persist-gateway-rests` fix head `4d0b0b75` base `0815c918` 2 commit(s)
- `fix/persist-gateway-rests-v2` fix head `5035eb0b` base `fa22bc64` 3 commit(s)
- `fix/persist-rest-write-errors` fix head `fcb0767a` base `f5b01574` 2 commit(s)
- `fix/pi-google-api` fix head `279e9bd1` base `ca4a1cde` 1 commit(s)
- `fix/pi-reasoning-usage` fix head `b506ff56` base `7aad6a50` 25 commit(s)
- `fix/provider-protocol-url` fix head `9829accb` base `7aad6a50` 22 commit(s)
- `fix/qoder-ide-code` fix head `4aa39dbf` base `be002aca` 1 commit(s)
- `fix/queue-model-rest` fix head `50eac57f` base `be002aca` 1 commit(s)
- `fix/quota-picker-save-test` fix head `c24a50e3` base `7aad6a50` 21 commit(s)
- `fix/reexec-plugin-host` fix head `3da2bb26` base `ca4a1cde` 2 commit(s)
- `fix/relay-reasoning-item-refusal` fix head `cf95652a` base `7aad6a50` 41 commit(s)
- `fix/rest-deadline-monotonic` fix head `b7df5885` base `ca4a1cde` 1 commit(s)
- `fix/retired-model-output-save` fix head `5e68f183` base `7aad6a50` 25 commit(s)
- `fix/sensenova-grammar-refusal` fix head `8f931a28` base `7aad6a50` 42 commit(s)
- `fix/settings-served-models` fix head `7ccf8d5d` base `7aad6a50` 1 commit(s)
- `fix/severed-chat-events` fix head `a6a2c260` base `7aad6a50` 28 commit(s)
- `fix/shared-session-folder` fix head `4c3d6e26` base `87d703c5` 1 commit(s)
- `fix/signin-plugin-hook-race` fix head `f85bd166` base `7aad6a50` 39 commit(s)
- `fix/skill-desktop-metadata` fix head `f42f29c4` base `7aad6a50` 1 commit(s)
- `fix/socks-proxy-test` fix head `d16f32c3` base `ca4a1cde` 1 commit(s)
- `fix/systemone-redaction` fix head `ee1b018b` base `7aad6a50` 49 commit(s)
- `fix/tool-call-whole-numbers` fix head `deefacd2` base `7aad6a50` 33 commit(s)
- `fix/tui-agent-fields` fix head `cebfc283` base `7aad6a50` 19 commit(s)
- `fix/union-items` fix head `b2336968` base `0815c918` 2 commit(s)
- `fix/union-items-v2` fix head `cbc6c3b3` base `fa22bc64` 3 commit(s)
- `fix/unknown-model-sent-id` fix head `38444c2d` base `be002aca` 1 commit(s)
- `fix/volcengine-resets` fix head `e6c2b6fb` base `87d703c5` 1 commit(s)
- `fix/vscode-portable-path` fix head `e4ca1b34` base `7aad6a50` 19 commit(s)
- `fix/web-signal-shutdown` fix head `c5334b59` base `0815c918` 3 commit(s)
- `fix/zcode-window-renewal` fix head `942bf679` base `7aad6a50` 16 commit(s)
- `main` fix head `be002aca` base `be002aca` 0 commit(s)
- `fix/provider-detect-test-scroll` local-only head `348fb756` base `7aad6a50` 20 commit(s)
- `fix/web-signal-shutdown-2` local-only head `0815c918` base `0815c918` 0 commit(s)
- `fix/webdav-digest-auth` local-only head `0437ae4c` base `7aad6a50` 18 commit(s)

## Archive

All units of the branch adopted/superseded AND verified AND released (where release-verifiable) AND no unique outstanding change AND no active dependent; earliest review 90 days after the confirming evidence. Candidacy is not deletion; deletion needs an explicit instruction, an annotated archive tag or bundle first.

- Candidate now: fix/volcengine-resets (u-volcengine-resets adopted_with_changes, released v0.1.1325, #3276 closed-completed; #3251 layer 2 tracked in ledger not on branch)
- Never: origin (fork mirror of upstream main)
- Never: dev (maintenance/PLAN)
