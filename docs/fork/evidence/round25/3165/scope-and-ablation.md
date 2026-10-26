# #3165 rework scope and independent evidence

Base: 19253f67bca2310c32739a6eaac88546b3574dbc.
Worktree: /home/vincenty/Projects/magpie-issues/r25-3165-reasoning-clear-test.
Branch: fix/r25-3165-reasoning-clear-test.

The requested base does not contain d5b46244abb5da4572fc759688281296caab3c8d (git merge-base --is-ancestor exits 1), nor foldReasoningAliases and the original two regression tests. Replay that already-delivered patch as a distinct prerequisite commit (57a507167a15025674e266340290ccca357c40f8); keep the actual rework test-only. chattidy.go stable patch-id matches the delivered change (c646cdbaedef92d87f73fb4c0ff16228ceb04899), and chat_tool_reasoning_test.go SHA-256 matches its delivered bytes.

Live issue and all comments were read through GitHub API pagination (1 page, 7 comments). Comment 6357540177 is the ablation source. The exact demonstrated vendor failure is reasoning plus reasoning_content producing 400 duplicate field `reasoning`. The vendor has not demonstrated aliasing of reasoning_details.

Independent ablations at this base plus replay:
- baseline-six.log: original six tests PASS.
- ablation-tool-cleanup-six.log: restore only withReasoningOnToolTurns to base, retain foldReasoningAliases and its call; all six tests PASS.
- ablation-fold-six.log: restore cleanup, remove only fold call; TestChatBothReasoningSpellingsRelayed FAILS with exact 400 duplicate field `reasoning`; other five pass.
- red.log: cleanup removed, fold still active, new TestChatToolTurnReasoningDetailsCarriedOnce FAILS in injected and already_present rows. Both message 1 (tool call) and 3 (assistant after result) still carry reasoning_details alongside reasoning_content.
- green.log: exact same test with cleanup restored PASS in both rows.

The single new table test goes through Server.Handler on /v1/chat/completions and records the local httptest upstream's actual request. Its structured reasoning.text fixture (text, signature:null, format:unknown, index:0) follows the existing pi fixture in chat_inbound_reasoning_test.go and kimi_reasoning_details_test.go. The already_present row has a canonical string different from the details text, so overwrite would fail. It asserts details absence and the correct canonical text, without duplicating existing reasoning/alias assertions or teaching a fake server an unproven vendor rejection rule. No external network, sleep, polling, skips, dependency changes or new production behavior.

Non-tool reasoning_content plus reasoning_details passes through both delivered helpers: no tool_calls means withReasoningOnToolTurns does not run, and foldReasoningAliases requires reasoning. This is outside the delivered cleanup scope; whether a vendor refuses it is unproven. No claim to fix it or the separate Responses failure. No real DeepSeek/JD Cloud account used.

New file: 50 nonblank, noncomment lines, one responsibility (tool-turn structured-thinking regression). Raw JSON is decoded in the test at the captured wire boundary and errors checked; no untyped any, tagged variants, suppression, redundant defenses, single-use helper, parameter bloat, negative naming, new logging, or production abstraction introduced. Existing fake/setup/postTo/sentMessages reused.

LSP diagnostic attempted: gopls unavailable. No install performed. Native, Linux nogui and Windows vet pass instead. Test/build toolchain actually used: go1.27.1-X:nodwarf5 linux/amd64; Go 1.26.8 was not installed.
