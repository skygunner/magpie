# Delivered issues — audit v0.1.1957, Batch 1

Every row is a pushed `fix/*` branch on `skygunner/magpie` based on
`upstream/main` at `926ef769f`, with one issue or comment on `yetone/magpie`.

| # | issue | branch | commit | comment | lang |
| --- | --- | --- | --- | --- | --- |
| B1 | [#4118](https://github.com/yetone/magpie/issues/4118) | [fix/4118-pii-stale-miss](https://github.com/skygunner/magpie/tree/fix/4118-pii-stale-miss) | [ab3ed5322](https://github.com/skygunner/magpie/commit/ab3ed532271ca649b4b5a74df6d6885cc76793cc) | [评论](https://github.com/yetone/magpie/issues/4118#issuecomment-6484900780) | 中文 |
| B2 | [#4137](https://github.com/yetone/magpie/issues/4137) | [fix/4137-davsync-read-cap](https://github.com/skygunner/magpie/tree/fix/4137-davsync-read-cap) | [3ff971ddd](https://github.com/skygunner/magpie/commit/3ff971ddd56463a7bed3c8a2230ba8783ae57d23)（含前两提交） | [评论](https://github.com/yetone/magpie/issues/4137#issuecomment-6485262880) | 中文 |
| B3 | [#4136](https://github.com/yetone/magpie/issues/4136) | [fix/4136-wsl-probe-lock](https://github.com/skygunner/magpie/tree/fix/4136-wsl-probe-lock) | [788f18e38](https://github.com/skygunner/magpie/commit/788f18e38b92cdd1a3a1c1cff8cf846552850319)（含前置提交） | [评论](https://github.com/yetone/magpie/issues/4136#issuecomment-6485286748) | 中文 |
| B6 | [#4141](https://github.com/yetone/magpie/issues/4141) | [fix/4141-dav-put-version](https://github.com/skygunner/magpie/tree/fix/4141-dav-put-version) | [3452982b8](https://github.com/skygunner/magpie/commit/3452982b8ad99aa3c74f86c326a65f5f53ad1844)（含前置提交） | [评论](https://github.com/yetone/magpie/issues/4141#issuecomment-6485383967) | 中文 |
| B5 | [#4144](https://github.com/yetone/magpie/issues/4144) | [fix/4144-sessions-frame-tail](https://github.com/skygunner/magpie/tree/fix/4144-sessions-frame-tail) | [ba4f3ef69](https://github.com/skygunner/magpie/commit/ba4f3ef69a0fbfd85c0250b9e8d8dcef0ba3866f)（含前置提交） | [评论](https://github.com/yetone/magpie/issues/4144#issuecomment-6485933827) | 中文；保守缓解 |
| B4 | [#4258](https://github.com/yetone/magpie/issues/4258) | [fix/4258-agent-stash-durability](https://github.com/skygunner/magpie/tree/fix/4258-agent-stash-durability) | [9b83ba670](https://github.com/skygunner/magpie/commit/9b83ba670) | [issue 正文](https://github.com/yetone/magpie/issues/4258) + [关键文件完整 diff](https://github.com/yetone/magpie/issues/4258#issuecomment-6494205053) | 英文；单 commit；完整 diff 458 KB 超正文限制 |

Notes:

- B1 and B4 are single commits. B2, B3, B5 and B6 contain earlier commits
  needed by their branch; cherry-pick the whole branch or all commits.
- B5 is a conservative mitigation: corrupt frames stop the read and keep the
  last good result; records after the corruption are not recovered until the
  source file is repaired.
- B6 stops for manual reconciliation when the first blind write has an
  ambiguous competing update.
- B4's complete diff is ~458 KB across 114 files; the issue carries inline
  safety-critical hunks and a comment carries the complete diff of `stash.go`,
  `stepcode_recovery.go` and the restore-failure test.
