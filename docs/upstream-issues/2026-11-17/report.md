# yetone/magpie upstream follow-up — 2026-11-17

Repo: yetone/magpie (upstream); fork: skygunner/magpie (local clone /home/vincenty/Projects/magpie-fix).
Scope: every upstream issue on which the fork campaign posted a fix comment (GitHub search commenter:skygunner = **357 issues**, #1105..#5713). All 357 were fetched live from the GitHub API (raw/<n>.json): state, all comments, timeline.

## Tally
| Class | Count | Meaning |
|---|---:|---|
| A | 67 | absorbed upstream (closed completed, or maintainer comment cites a landed commit/version) |
| B | 5 | needs our reply/action — all 5 replied on 2026-11-17 and read back |
| D | 285 | no response / nothing asked of us (includes 6 open issues with competing open PRs: #2207 #3280 #3285 #3680 #5150 #5348) |

Opener languages: 133 Chinese, 224 English — every reply written in the opener's language.

## B items — replies posted (all read back via API)
| issue | opener | what happened | reply |
|---|---|---|---|
| #2785 [zh] | pluo | regression report on v0.1.3844: window restarts on the wrong monitor | issuecomment-6935745347 |
| #3375 [en] | kevin-wamae | Anarlog key-verification follow-up, no endpoint named yet | issuecomment-6935745936 |
| #4144 [zh] | skygunner | maintainer rejected our series with 3 specific reasons; we accepted the review, stated the read-past-corrupt-frame rework direction, asked for a real damaged dsh file for the test | issuecomment-6935860474 |
| #4385 [en] | yetone | maintainer "not taking it" on our WebKit wheel fix without stated reasons; we asked for the decision rationale | issuecomment-6935799230 |
| #4945 [en] | barryzzz | ekoooli corrected the root cause (placeholder note lands between function_call and its output); we acknowledged, disowned the id-suffix theory, and outlined the note-placement fix | issuecomment-6935746577 |

Reply texts are in replies/<n>.md; posted-replies.json holds URL + readback.

## A evidence highlights (maintainer yetone cherry-picked skygunner patches with credit)
Examples from the 67: #2530 (v0.1.1691, 32f7c4de1), #3043 (v0.1.1585, 27ca0bef5), #3062 (v0.1.2747, f822689ad), #3097 (v0.1.2583, 9a4eaab82), #3141 (v0.1.2855, 62f612fd4), #3193 (v0.1.2583, 63cccbb47), #3228 (v0.1.1831, 1dc20e73f), #3251/#3276 (v0.1.1325, 8278fd9b0), #3282 (v0.1.2042, 7eb7fa540), #3650 (v0.1.1608, 9733ed1b4), #3656 (v0.1.2569, 9097687a3), #3661 (v0.1.1662, 65ee8c65c), #3664 (v0.1.2124, e863f520c), #4921 (v0.1.2766, c86fa2a34), #5271 (363adb352), #5433 (855b97a40), #5690 (28a79bc33). Several A items remain open only awaiting the reporter's confirmation after upgrading (e.g. #3062, #3147, #3193 reopened, #3430, #4872) — those waits belong to the issue opener, not to skygunner, so no reply was posted.

## Method notes
- Absorption evidence = maintainer (yetone/OWNER) comment citing a landed commit+version, or issue closed/completed. No commit-subject '#<issue>' grep: upstream subjects do not carry issue numbers.
- A comment was only treated as "addressed to skygunner" when it @mentions skygunner AND is not a pure verification endorsement (verified/yielding/pass/credit patterns filtered out).
- 241 of 357 issues map to a fork fix branch (branch-map.json, from fork-branches.txt); the rest were comment-only interactions.
- No worktree was created in this sweep (no branch update was required yet); no temp dirs beyond this data directory.

## Per-issue table
| issue | opener | lang | state | class | detail | branch | our comment | reply |
|---|---|---|---|---|---|---|---|---|
| #1105 | Ti800 | zh | open/reopened | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/1105#issuecomment-6389275629 |  |
| #1139 | byhooi | zh | open | D | no upstream comments after ours | fix/1139-issue-1139 | https://github.com/yetone/magpie/issues/1139#issuecomment-6451020676 |  |
| #1579 | yetone | en | open | D | no upstream comments after ours | fix/1579-test-tencent-cloud-test-cjs | https://github.com/yetone/magpie/issues/1579#issuecomment-6458832003 |  |
| #1682 | loosekite | en | open | D | no upstream comments after ours | fix/1682-pi-provider-plugins-skip-custom | https://github.com/yetone/magpie/issues/1682#issuecomment-6454729421 |  |
| #1728 | hh0592821 | zh | open | D | no upstream comments after ours | fix/1728-zenmux | https://github.com/yetone/magpie/issues/1728#issuecomment-6454499584 |  |
| #1747 | ShineBreaker | en | open | D | community comments after ours (2), no direct ask to skygunner | fix/1747-linux-guix-startup-writes-a | https://github.com/yetone/magpie/issues/1747#issuecomment-6587089526 |  |
| #1841 | yetone | en | open | D | no upstream comments after ours | fix/1841-tests-internal-agent-tests-reset | https://github.com/yetone/magpie/issues/1841#issuecomment-6454419596 |  |
| #1848 | yetone | en | open | D | no upstream comments after ours | fix/1848-gateway-deepseek-tool-calls-written | https://github.com/yetone/magpie/issues/1848#issuecomment-6454739644 |  |
| #1903 | yetone | en | open | D | no upstream comments after ours | fix/1903-setaside-leftbehind | https://github.com/yetone/magpie/issues/1903#issuecomment-6438165549 |  |
| #1955 | NorethSea | en | closed/completed | A | closed completed upstream | fix/1955-codex-native-model-list-falls | https://github.com/yetone/magpie/issues/1955#issuecomment-6454522452 |  |
| #1971 | yetone | en | open | D | no upstream comments after ours | fix/1971-rate-limit-test-clock | https://github.com/yetone/magpie/issues/1971#issuecomment-6727639484 |  |
| #2008 | Gaudi-XIX | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/2008#issuecomment-6364369875 |  |
| #2099 | yetone | en | open | D | no upstream comments after ours | fix/2099-risk-wrap | https://github.com/yetone/magpie/issues/2099#issuecomment-6438166156 |  |
| #2117 | yetone | zh | open | D | no upstream comments after ours | fix/2117-web-in-a-container-leave | https://github.com/yetone/magpie/issues/2117#issuecomment-6454205181 |  |
| #2207 | yetone | en | open | D | no upstream comments after ours | fix/2207-socks-proxy-test-deterministic | https://github.com/yetone/magpie/issues/2207#issuecomment-6450306730 |  |
| #2259 | yetone | en | open | D | community comments after ours (3), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/2259#issuecomment-6355087485 |  |
| #2287 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/2287#issuecomment-6354911428 |  |
| #2301 | LukeZ1988 | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/2301-minimax-code | https://github.com/yetone/magpie/issues/2301#issuecomment-6457668321 |  |
| #2323 | sryimnoob123 | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/2323-hermes-vision-mapping | https://github.com/yetone/magpie/issues/2323#issuecomment-6354772359 |  |
| #2389 | vianbara | en | open | D | no upstream comments after ours | fix/2389-systemone-state-redaction | https://github.com/yetone/magpie/issues/2389#issuecomment-6450452162 |  |
| #2460 | qiangxinglin | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/2460#issuecomment-6354479109 |  |
| #2473 | EchoTint | zh | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/2473#issuecomment-6354354731 |  |
| #2530 | SavitarC | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/2530#issuecomment-6354217288 |  |
| #2541 | yetone | en | open | D | no upstream comments after ours | fix/2541-queue-ledger-identity, fix/2541-queue-test-budget | https://github.com/yetone/magpie/issues/2541#issuecomment-6726493334 |  |
| #2569 | yetone | en | open | D | no upstream comments after ours | fix/2569-sessions-opencode-zcode-and-dsh | https://github.com/yetone/magpie/issues/2569#issuecomment-6451904777 |  |
| #2621 | yetone | en | open | D | no upstream comments after ours | fix/2621-tests-three-guarantees-lost-their | https://github.com/yetone/magpie/issues/2621#issuecomment-6454008224 |  |
| #2629 | yetone | en | open | D | no upstream comments after ours | fix/2629-gui-tests-agent-bulk-visibility | https://github.com/yetone/magpie/issues/2629#issuecomment-6451442864 |  |
| #2661 | yetone | en | open | D | no upstream comments after ours | fix/2661-tests-most-packages-don-t | https://github.com/yetone/magpie/issues/2661#issuecomment-6453842465 |  |
| #2669 | yetone | en | open | D | no upstream comments after ours | fix/2669-gui-after-the-codex-editor | https://github.com/yetone/magpie/issues/2669#issuecomment-6450818537 |  |
| #2674 | yetone | en | open | D | no upstream comments after ours | fix/2674-blocklist-tail | https://github.com/yetone/magpie/issues/2674#issuecomment-6403620529 |  |
| #2705 | Celestia-Milara | zh | open | D | no upstream comments after ours | fix/2705-usage-sessions-claude-desktop-3p | https://github.com/yetone/magpie/issues/2705#issuecomment-6453996858 |  |
| #2728 | yetone | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/2728-docs-strategy-test-budget | https://github.com/yetone/magpie/issues/2728#issuecomment-6727025790 |  |
| #2729 | real-jacket | zh | open | D | no upstream comments after ours | fix/2729-gateway-deepseek-s-search-api | https://github.com/yetone/magpie/issues/2729#issuecomment-6457738348 |  |
| #2730 | liangxyz | zh | open | D | no upstream comments after ours | fix/2730-deepseek | https://github.com/yetone/magpie/issues/2730#issuecomment-6451551419 |  |
| #2741 | geeknonerd | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/2741#issuecomment-6394163102 |  |
| #2744 | Goalonez | zh | open | D | no upstream comments after ours | fix/2744-issue-2744 | https://github.com/yetone/magpie/issues/2744#issuecomment-6457739011 |  |
| #2746 | kudi68 | en | open | D | no upstream comments after ours | fix/2746-unlisted-by-group | https://github.com/yetone/magpie/issues/2746#issuecomment-6438167437 |  |
| #2755 | tianxia3111 | zh | open | D | no upstream comments after ours | fix/2755-led-format-tokens | https://github.com/yetone/magpie/issues/2755#issuecomment-6438168091 |  |
| #2759 | kindmesdilabso | zh | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/2759#issuecomment-6403875879 |  |
| #2761 | yetone | en | open | D | no upstream comments after ours | fix/2761-filememo-clock-seam | https://github.com/yetone/magpie/issues/2761#issuecomment-6438168723 |  |
| #2768 | li1how | en | open | D | no upstream comments after ours | fix/2768-vs-code-copilot-chat-in | https://github.com/yetone/magpie/issues/2768#issuecomment-6456351674 |  |
| #2771 | li1how | en | open | D | no upstream comments after ours | fix/2771-windows-magpie-does-not-discover | https://github.com/yetone/magpie/issues/2771#issuecomment-6458322138 |  |
| #2773 | kumamaki | en | open | D | no upstream comments after ours | fix/2773-droid-magpie-written-custom-models | https://github.com/yetone/magpie/issues/2773#issuecomment-6450873244 |  |
| #2777 | a1286743184 | en | open | A | upstream landed/fixed: Fixed in 69626580b, in v0.1.1797. Every check-in now goes through the account's own proxy, or the |  | https://github.com/yetone/magpie/issues/2777#issuecomment-6393185272 |  |
| #2785 | pluo | zh | open/reopened | B | user @pluo addressed skygunner: @skygunner it seems to regress on v0.1.3844 (-ish). window size is memorized, but the wi | fix/2785-magpie-does-not-memorize-its, fix/2785-window-monitor | https://github.com/yetone/magpie/issues/2785#issuecomment-6477047667 | https://github.com/yetone/magpie/issues/2785#issuecomment-6935745347 |
| #2789 | turtle4512 | zh | open | D | no upstream comments after ours | fix/2789-textcall-xml | https://github.com/yetone/magpie/issues/2789#issuecomment-6438169361 |  |
| #2804 | mtul0729 | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/2804-linux-duplicate-tray-icon-on | https://github.com/yetone/magpie/issues/2804#issuecomment-6456020397 |  |
| #2808 | Kyou12138 | en | open | D | no upstream comments after ours | fix/2808-keychain-windows-reds-v2, fix/2808-test-eight-codex-keychain-tests | https://github.com/yetone/magpie/issues/2808#issuecomment-6586091790 |  |
| #2822 | JJ-Lin | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/2822-chat-only-v1-models-responses | https://github.com/yetone/magpie/issues/2822#issuecomment-6454047284 |  |
| #2836 | kinhit | zh | open | D | no upstream comments after ours | fix/2836-google-openai-images-generations-imagen | https://github.com/yetone/magpie/issues/2836#issuecomment-6453606327 |  |
| #2838 | yiyiSmallti | en | open | D | no upstream comments after ours | fix/2838-claude-subscription-upgraded-to-max | https://github.com/yetone/magpie/issues/2838#issuecomment-6726091186 |  |
| #2866 | ReneXiong | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/2866#issuecomment-6393086996 |  |
| #2867 | yetone | en | open | D | no upstream comments after ours | fix/2867-drift-detail-i18n | https://github.com/yetone/magpie/issues/2867#issuecomment-6403924259 |  |
| #2868 | WTHH031230 | zh | open | D | community comments after ours (2), no direct ask to skygunner | fix/2868-claude | https://github.com/yetone/magpie/issues/2868#issuecomment-6456125099 |  |
| #2878 | gtk96 | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/2878-qoder-qoder | https://github.com/yetone/magpie/issues/2878#issuecomment-6453683594 |  |
| #2923 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/2923#issuecomment-6352725039 |  |
| #2949 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/2949#issuecomment-6343163674 |  |
| #2953 | yetone | en | open | D | community comments after ours (3), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/2953#issuecomment-6352556608 |  |
| #2998 | NkAntony777 | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/2998#issuecomment-6415246814 |  |
| #3005 | yugaosi140 | zh | open | D | no upstream comments after ours | fix/3005-local-session-rows | https://github.com/yetone/magpie/issues/3005#issuecomment-6453413072 |  |
| #3013 | ianfksyd | zh | open | D | community comments after ours (2), no direct ask to skygunner | fix/3013-glm-window-quota | https://github.com/yetone/magpie/issues/3013#issuecomment-6449674916 |  |
| #3017 | marsxxl | zh | open | D | no upstream comments after ours | fix/3017-cindy-provider-suffix | https://github.com/yetone/magpie/issues/3017#issuecomment-6455774650 |  |
| #3020 | WardLu | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3020#issuecomment-6352398582 |  |
| #3027 | shenghsi | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3027#issuecomment-6364113600 |  |
| #3035 | jasonQin6 | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3035#issuecomment-6397596850 |  |
| #3043 | paulwalker9315-debug | zh | open | A | upstream landed/fixed: 已合入 27ca0bef5，发在 v0.1.1585（感谢 @skygunner 的修复，作者署名保留；感谢 @shupivot 的测量）。 MCP Registry 搜索现在用自己的 60 秒 |  | https://github.com/yetone/magpie/issues/3043#issuecomment-6392132968 |  |
| #3058 | chengbot-ai | zh | open | D | no upstream comments after ours | fix/3058-copilot-keychain-ask | https://github.com/yetone/magpie/issues/3058#issuecomment-6453393915 |  |
| #3062 | jasonQin6 | en | open | A | upstream landed/fixed: Landed as f822689ad in v0.1.2747, with ggbdpq as the author: the relay's "Referenced reasoning it |  | https://github.com/yetone/magpie/issues/3062#issuecomment-6352083061 |  |
| #3074 | zzjcool | en | open | D | no upstream comments after ours | fix/3074-plugin-a-request-s-body | https://github.com/yetone/magpie/issues/3074#issuecomment-6453212820 |  |
| #3076 | binbinah | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3076#issuecomment-6347697853 |  |
| #3095 | yetone | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3095#issuecomment-6351919564 |  |
| #3097 | Davyricardo | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3097#issuecomment-6351918961 |  |
| #3100 | yetone | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3100#issuecomment-6351920135 |  |
| #3107 | kongxiangyiren | zh | open | D | no upstream comments after ours | fix/3107-issue-3107 | https://github.com/yetone/magpie/issues/3107#issuecomment-6451910324 |  |
| #3108 | realnovicedev | en | open | D | no upstream comments after ours | fix/3108-panic-when-autostarted-at-login | https://github.com/yetone/magpie/issues/3108#issuecomment-6451008422 |  |
| #3117 | WardLu | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3117#issuecomment-6392132800 |  |
| #3121 | MurmuredVelpro | en | open | D | community comments after ours (2), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3121#issuecomment-6343428670 |  |
| #3122 | chengbot-ai | zh | open | D | no upstream comments after ours | fix/3122-issue-3122 | https://github.com/yetone/magpie/issues/3122#issuecomment-6451632933 |  |
| #3128 | long1973m | zh | open | D | no upstream comments after ours | fix/3128-provider-workbuddy | https://github.com/yetone/magpie/issues/3128#issuecomment-6450451081 |  |
| #3131 | yugaosi140 | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3131#issuecomment-6726369443 |  |
| #3135 | ARCJ137442 | zh | open | D | no upstream comments after ours | fix/3135-codex-responses-eof-502 | https://github.com/yetone/magpie/issues/3135#issuecomment-6587084138 |  |
| #3136 | OldSuns | zh | open | A | upstream landed/fixed: v0.1.1878（e50d73aa3）合入了 @skygunner 分支 `fix/3136-agent-magpie-preview-86-mb` 的修复，感谢 @OldSuns、@gggg | fix/3136-agent-magpie-preview-86-mb | https://github.com/yetone/magpie/issues/3136#issuecomment-6449984993 |  |
| #3137 | cjyrainbow | zh | open | D | no upstream comments after ours | fix/3137-proxy-advice-address | https://github.com/yetone/magpie/issues/3137#issuecomment-6438169999 |  |
| #3139 | 321sssrt-bit | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3139#issuecomment-6386993902 |  |
| #3141 | yinwu01 | zh | closed/completed | A | closed completed upstream | fix/3141-proc-look-clock | https://github.com/yetone/magpie/issues/3141#issuecomment-6351191858 |  |
| #3143 | hibaliqya-yonma | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3143#issuecomment-6343416196 |  |
| #3144 | hibaliqya-yonma | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3144#issuecomment-6343415582 |  |
| #3147 | cyjme | en | open | A | upstream landed/fixed, maintainer asks reporter to confirm after upgrading |  | https://github.com/yetone/magpie/issues/3147#issuecomment-6343445315 |  |
| #3148 | Henvy-Mango | en | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3148#issuecomment-6360371525 |  |
| #3150 | xiachaoxia-glitch | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3150#issuecomment-6343313763 |  |
| #3151 | ObsidianArch02 | en | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3151#issuecomment-6343163165 |  |
| #3152 | 1024MrL | en | open | A | upstream landed/fixed: The whole-number fix from gua2100's (ggbdpq) `hunt58-h-3152-whole-float` is ported to main in v0. |  | https://github.com/yetone/magpie/issues/3152#issuecomment-6351033700 |  |
| #3153 | yetone | en | open | A | upstream landed/fixed: Landed ggbdpq's patch in ca183561f, released in v0.1.2489. An `allowed_tools` tool_choice now rea |  | https://github.com/yetone/magpie/issues/3153#issuecomment-6351033261 |  |
| #3156 | AvatarGanymede | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3156#issuecomment-6434934970 |  |
| #3157 | MisonL | en | open | D | no upstream comments after ours | fix/3157-reduced-motion-decorative | https://github.com/yetone/magpie/issues/3157#issuecomment-6460279326 |  |
| #3163 | chengbot-ai | zh | open | D | no upstream comments after ours | fix/3163-issue-3163 | https://github.com/yetone/magpie/issues/3163#issuecomment-6450828448 |  |
| #3165 | wmfan2002-cloud | zh | open | D | no upstream comments after ours | fix/3165-reasoning-faces, fix/3165-reasoning-faces-current | https://github.com/yetone/magpie/issues/3165#issuecomment-6729316836 |  |
| #3166 | Parcheal | en | open | D | no upstream comments after ours | fix/3166-responses-continuation | https://github.com/yetone/magpie/issues/3166#issuecomment-6587087629 |  |
| #3169 | Sealt | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3169#issuecomment-6403679182 |  |
| #3173 | huaiqing-afk | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3173#issuecomment-6347698356 |  |
| #3174 | yetone | en | closed/completed | A | closed completed upstream | fix/3174-claude-signin-test-cache | https://github.com/yetone/magpie/issues/3174#issuecomment-6477045522 |  |
| #3176 | Arcadia822 | en | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3176#issuecomment-6343065990 |  |
| #3183 | john-zeng-233 | zh | open | A | upstream landed/fixed: v0.1.3526 (a45dd9c65): a namespaced collaboration call a Responses relay returns without `encrypt |  | https://github.com/yetone/magpie/issues/3183#issuecomment-6726672103 |  |
| #3188 | wtj-0527 | en | closed/completed | A | closed completed upstream | fix/3188-sub2api-percent, fix/3188-sub2api-percent-r2 | https://github.com/yetone/magpie/issues/3188#issuecomment-6663477178 |  |
| #3191 | LuwenOosten | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/3191-gateway-request-body-write-can | https://github.com/yetone/magpie/issues/3191#issuecomment-6457342022 |  |
| #3193 | pc418 | en | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3193#issuecomment-6351631831 |  |
| #3194 | mylastfree | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3194#issuecomment-6416184531 |  |
| #3197 | YonQua | en | open | D | community comments after ours (4), no direct ask to skygunner | fix/3197-orphan-event | https://github.com/yetone/magpie/issues/3197#issuecomment-6718027883 |  |
| #3202 | nodlles | zh | closed/completed | A | closed completed upstream | fix/3202-issue-3202 | https://github.com/yetone/magpie/issues/3202#issuecomment-6453172802 |  |
| #3205 | yetone | en | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3205#issuecomment-6350369324 |  |
| #3206 | ObsidianArch02 | en | open | D | no upstream comments after ours | fix/3206-gateway-server-managed-tools-web | https://github.com/yetone/magpie/issues/3206#issuecomment-6455559185 |  |
| #3213 | liubai0106 | en | open | D | community comments after ours (2), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3213#issuecomment-6360371924 |  |
| #3224 | IceLine389 | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3224#issuecomment-6350064542 |  |
| #3228 | wizd | zh | open | A | upstream landed/fixed: @wizd 这个在 **v0.1.1831** 已修（1dc20e73f，另有 4c1fb7758 清掉旧版本留下的副本）：Cursor 运行时（state.vscdb 旁有非空 -wal 和  | fix/3228-windows-ntfs-cursor-480gb-500gb | https://github.com/yetone/magpie/issues/3228#issuecomment-6453090351 |  |
| #3229 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3229#issuecomment-6394721247 |  |
| #3233 | 99iii | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3233#issuecomment-6350063562 |  |
| #3234 | marshalys | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3234#issuecomment-6349585899 |  |
| #3235 | x0rz4 | zh | open | D | no upstream comments after ours | fix/3235-qoder-cn | https://github.com/yetone/magpie/issues/3235#issuecomment-6452550618 |  |
| #3237 | fflrichard | zh | open | D | no upstream comments after ours | fix/3237-remote-gemini | https://github.com/yetone/magpie/issues/3237#issuecomment-6463236918 |  |
| #3246 | qu-mou | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3246#issuecomment-6403466440 |  |
| #3248 | stabruriss | en | open | D | no upstream comments after ours | fix/3248-codeassist-restart | https://github.com/yetone/magpie/issues/3248#issuecomment-6403504636 |  |
| #3251 | CNife | zh | open | A | upstream landed/fixed: 第 1 层已在 8278fd9b0ce1083e71ae4651bce5d138f940d95e 修复（main，尚未打 tag）。参考了 @skygunner 分支 `fix/volcengi |  | https://github.com/yetone/magpie/issues/3251#issuecomment-6342260029 |  |
| #3254 | jstar0 | en | open | A | upstream landed/fixed: Landed in v0.1.4130 as 284b22172, @ggbdpq's patch unchanged with authorship kept. Before it remov | fix/3254-accounts-switching-to-or-forgetting | https://github.com/yetone/magpie/issues/3254#issuecomment-6452997534 |  |
| #3255 | feiyuxiaoThu | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3255#issuecomment-6342498748 |  |
| #3258 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3258#issuecomment-6342495758 |  |
| #3259 | mawag | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3259#issuecomment-6663476508 |  |
| #3260 | YOUIMARK | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3260#issuecomment-6349585359 |  |
| #3266 | hibaliqya-yonma | zh | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3266#issuecomment-6586090139 |  |
| #3276 | wyx1818 | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3276#issuecomment-6342270787 |  |
| #3277 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3277#issuecomment-6349008616 |  |
| #3278 | yetone | en | open | D | no upstream comments after ours | fix/3278-day-window-budget | https://github.com/yetone/magpie/issues/3278#issuecomment-6438170609 |  |
| #3280 | skygunner | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3280#issuecomment-6348173813 |  |
| #3282 | XYenon | zh | open | A | upstream landed/fixed: Ported in 7eb7fa540 (v0.1.2042), with credit to @skygunner, whose patch (279e9bd1 on `fix/pi-goog |  | https://github.com/yetone/magpie/issues/3282#issuecomment-6348173228 |  |
| #3284 | skygunner | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3284#issuecomment-6348174237 |  |
| #3285 | Moody-Sin | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3285#issuecomment-6726127469 |  |
| #3289 | wjq990112 | en | open | A | upstream landed/fixed: Landed in v0.1.2497: - 18387b93b (ggbdpq): a shorter failure (the 1s 429 rest) no longer overwrit |  | https://github.com/yetone/magpie/issues/3289#issuecomment-6348172755 |  |
| #3302 | hheei | en | open | D | no upstream comments after ours | fix/3302-embedding-indices | https://github.com/yetone/magpie/issues/3302#issuecomment-6460004538 |  |
| #3303 | Moody-Sin | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3303#issuecomment-6393262850 |  |
| #3306 | Adamcf123 | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3306#issuecomment-6350368871 |  |
| #3308 | yetone | en | open | D | no upstream comments after ours | fix/3308-panel-subagent-effort | https://github.com/yetone/magpie/issues/3308#issuecomment-6429785833 |  |
| #3310 | JavaBoy6 | zh | open | D | no upstream comments after ours | fix/3310-copilot-403-forbidden-magpie | https://github.com/yetone/magpie/issues/3310#issuecomment-6455987745 |  |
| #3311 | skygunner | en | open | D | no upstream comments after ours | fix/3311-panel-request-isolation | https://github.com/yetone/magpie/issues/3311#issuecomment-6477046375 |  |
| #3313 | kinhit | zh | open | D | no upstream comments after ours | fix/3313-magpie-mcp-image-nano-banana | https://github.com/yetone/magpie/issues/3313#issuecomment-6455637052 |  |
| #3320 | Glanci-0525 | zh | open | D | no upstream comments after ours | fix/3320-rtk-verdict | https://github.com/yetone/magpie/issues/3320#issuecomment-6429784843 |  |
| #3322 | StudentWeis | zh | open | D | no upstream comments after ours | fix/3322-issue-3322 | https://github.com/yetone/magpie/issues/3322#issuecomment-6452351091 |  |
| #3324 | yetone | en | open | D | no upstream comments after ours | fix/3324-history-viewer-day | https://github.com/yetone/magpie/issues/3324#issuecomment-6429780051 |  |
| #3329 | gushan1wood | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3329#issuecomment-6663475768 |  |
| #3331 | westlinkin | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/3331-issue-3331 | https://github.com/yetone/magpie/issues/3331#issuecomment-6451239351 |  |
| #3339 | dreamfarer-space | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3339#issuecomment-6364113316 |  |
| #3345 | Sumeriasns | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3345#issuecomment-6363786237 |  |
| #3349 | Arcadia822 | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3349#issuecomment-6364831515 |  |
| #3352 | 1234fsdc | zh | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3352#issuecomment-6364210763 |  |
| #3358 | Latias94 | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3358#issuecomment-6364831214 |  |
| #3365 | Ethereal49 | zh | open | D | no upstream comments after ours | fix/3365-known-common-words | https://github.com/yetone/magpie/issues/3365#issuecomment-6586091319 |  |
| #3370 | yxr1995-maker | zh | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3370#issuecomment-6363745685 |  |
| #3374 | bigTsingLung | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3374#issuecomment-6364109042 |  |
| #3375 | kevin-wamae | en | open | B | user @kevin-wamae addressed skygunner: Hi @skygunner, I totally forgot to follow up on this and I have just posted it on |  | https://github.com/yetone/magpie/issues/3375#issuecomment-6726393455 | https://github.com/yetone/magpie/issues/3375#issuecomment-6935745936 |
| #3392 | kongkongklng | zh | open | D | no upstream comments after ours | fix/3392-xai-required-empty | https://github.com/yetone/magpie/issues/3392#issuecomment-6718027988 |  |
| #3397 | yinwu01 | zh | open | D | no upstream comments after ours | fix/3397-bun-proxy-default-port | https://github.com/yetone/magpie/issues/3397#issuecomment-6727014316 |  |
| #3399 | kupakupar | zh | open | A | upstream landed/fixed: 2ac423f4f (v0.1.2267) covers this: a Kimi Code plan model that Kimi's /coding/v1/models marks sup | fix/3399-kimi-thinking, fix/3399-kimi-thinking-2cdbe92 | https://github.com/yetone/magpie/issues/3399#issuecomment-6429783755 |  |
| #3401 | FuchaZ | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3401#issuecomment-6387030804 |  |
| #3417 | skygunner | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3417#issuecomment-6372200533 |  |
| #3430 | mdwsk88 | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3430#issuecomment-6403129004 |  |
| #3433 | obstrux | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3433#issuecomment-6380630767 |  |
| #3434 | e2720515-cloud | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3434#issuecomment-6403682869 |  |
| #3439 | yxd-ym | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3439#issuecomment-6726391952 |  |
| #3440 | Sinnhu | zh | open | A | upstream landed/fixed: Landed the copy in 50d5fcaef, with your authorship, in **v0.1.2583**: the Long conversations line |  | https://github.com/yetone/magpie/issues/3440#issuecomment-6383458761 |  |
| #3444 | lostornot | zh | open | D | community comments after ours (6), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3444#issuecomment-6380789247 |  |
| #3447 | HuHong-Tao | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3447#issuecomment-6380807714 |  |
| #3449 | cyrushuang1995-cmyk | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3449#issuecomment-6726155860 |  |
| #3455 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3455#issuecomment-6389279618 |  |
| #3456 | szu-fan | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/3456-codearts-model-benefit | https://github.com/yetone/magpie/issues/3456#issuecomment-6726690800 |  |
| #3460 | Yaka88 | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3460#issuecomment-6403185254 |  |
| #3470 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3470#issuecomment-6392134728 |  |
| #3475 | mdwsk88 | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3475#issuecomment-6587086165 |  |
| #3487 | skygunner | en | open | D | no upstream comments after ours | fix/3487-quota-span-fixture-1767 | https://github.com/yetone/magpie/issues/3487#issuecomment-6478388118 |  |
| #3513 | kilo-s | zh | open | D | community comments after ours (3), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3513#issuecomment-6389279389 |  |
| #3514 | stofancy | zh | open | D | no upstream comments after ours | fix/3514-subscription-claude-dsh-messages | https://github.com/yetone/magpie/issues/3514#issuecomment-6459418529 |  |
| #3519 | Tinkler-i | zh | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/3519#issuecomment-6389270118 |  |
| #3528 | yetone | en | open | A | upstream landed/fixed: c420559fd (v0.1.2308): each ask of a search API, and each search sent to Kimi Code's or WorkBuddy | fix/3528-search-usage-rows | https://github.com/yetone/magpie/issues/3528#issuecomment-6403927302 |  |
| #3529 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3529#issuecomment-6392564252 |  |
| #3530 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3530#issuecomment-6392564071 |  |
| #3531 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3531#issuecomment-6392783800 |  |
| #3539 | voidcraft-dev | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3539#issuecomment-6394204236 |  |
| #3541 | yetone | en | open | D | no upstream comments after ours | fix/3541-redact-local-hostname | https://github.com/yetone/magpie/issues/3541#issuecomment-6726192274 |  |
| #3547 | erickguan | en | closed/completed | A | closed completed upstream | fix/3547-marshalraw-go127-line-separators | https://github.com/yetone/magpie/issues/3547#issuecomment-6462376928 |  |
| #3549 | erickguan | en | closed/completed | A | closed completed upstream | fix/3549-image-budget-count | https://github.com/yetone/magpie/issues/3549#issuecomment-6462832924 |  |
| #3553 | greenfinchgreen-gif | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3553#issuecomment-6403129264 |  |
| #3555 | yetone | en | open | D | no upstream comments after ours | fix/3555-unrest-owner | https://github.com/yetone/magpie/issues/3555#issuecomment-6429784291 |  |
| #3556 | yetone | en | open | D | no upstream comments after ours | fix/3556-quota-retry-after-short | https://github.com/yetone/magpie/issues/3556#issuecomment-6461841136 |  |
| #3561 | yuancjun | en | open | D | no upstream comments after ours | fix/3561-privacy-model-proxy | https://github.com/yetone/magpie/issues/3561#issuecomment-6461586836 |  |
| #3566 | congee949 | zh | open | D | no upstream comments after ours | fix/3566-kimi-typed-ref-untyped-union | https://github.com/yetone/magpie/issues/3566#issuecomment-6663693365 |  |
| #3567 | yetone | en | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3567#issuecomment-6394559364 |  |
| #3572 | Yaka88 | en | open | D | no upstream comments after ours | fix/3572-antigravity-thought-signature | https://github.com/yetone/magpie/issues/3572#issuecomment-6663476158 |  |
| #3574 | yetone | zh | open | D | no upstream comments after ours | fix/3574-prose-password | https://github.com/yetone/magpie/issues/3574#issuecomment-6725932984 |  |
| #3585 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3585#issuecomment-6403818575 |  |
| #3595 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3595#issuecomment-6401147159 |  |
| #3597 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3597#issuecomment-6400908682 |  |
| #3598 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3598#issuecomment-6400912188 |  |
| #3601 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3601#issuecomment-6400908955 |  |
| #3614 | zisen123 | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3614#issuecomment-6403290176 |  |
| #3615 | easonlao | en | open | D | community comments after ours (3), no direct ask to skygunner | fix/3615-loop-guard-does-not-cover | https://github.com/yetone/magpie/issues/3615#issuecomment-6455474291 |  |
| #3619 | zisen123 | en | open | D | no upstream comments after ours | fix/3619-model-scoped-rate-limit | https://github.com/yetone/magpie/issues/3619#issuecomment-6663692894 |  |
| #3620 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3620#issuecomment-6410020815 |  |
| #3621 | yinwu01 | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3621#issuecomment-6408564893 |  |
| #3622 | gkd2323c | zh | open | A | upstream landed/fixed: 已在 v0.1.1705 修复（6cb1cfb79），补丁来自 coffemon（@JoeyWu-tech），保留了原作者。 - `pruneDist` 不再删掉 bun.lock 的 pack |  | https://github.com/yetone/magpie/issues/3622#issuecomment-6410545136 |  |
| #3623 | skygunner | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3623#issuecomment-6500351837 |  |
| #3624 | skygunner | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3624#issuecomment-6400631669 |  |
| #3628 | erickguan | en | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3628#issuecomment-6422550349 |  |
| #3629 | 321sssrt-bit | zh | open | D | no upstream comments after ours | fix/3629-masking-false-positives | https://github.com/yetone/magpie/issues/3629#issuecomment-6461767382 |  |
| #3630 | loscoy | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3630#issuecomment-6403245263 |  |
| #3632 | yinwu01 | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3632#issuecomment-6441745881 |  |
| #3634 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3634#issuecomment-6410456648 |  |
| #3635 | skygunner | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3635#issuecomment-6400628945 |  |
| #3636 | skygunner | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3636#issuecomment-6400629149 |  |
| #3645 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3645#issuecomment-6408257043 |  |
| #3647 | yinwu01 | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3647#issuecomment-6408145744 |  |
| #3648 | yinwu01 | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3648#issuecomment-6413258899 |  |
| #3650 | MYCF711 | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3650#issuecomment-6403822501 |  |
| #3655 | hibaliqya-yonma | zh | open | D | no upstream comments after ours | fix/3655-pending-wait-not-a-loop | https://github.com/yetone/magpie/issues/3655#issuecomment-6729290373 |  |
| #3656 | faketrader | zh | closed/completed | A | closed completed upstream |  | https://github.com/yetone/magpie/issues/3656#issuecomment-6408578981 |  |
| #3657 | Jandaes | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3657#issuecomment-6408073510 |  |
| #3661 | orion-nebula-dev | zh | open | A | upstream landed/fixed: 已在 **v0.1.1662**（65ee8c65c）修复“试用模型拒绝连带暂停账号”这一部分，感谢 @orion-nebula-dev 的报告和 @skygunner 的补丁（已移植并署名）。 |  | https://github.com/yetone/magpie/issues/3661#issuecomment-6411723766 |  |
| #3664 | NietzscheLi | zh | closed/completed | A | closed completed upstream | fix/3664-intent-effort-failover | https://github.com/yetone/magpie/issues/3664#issuecomment-6453028526 |  |
| #3665 | lan5635 | zh | closed/completed | A | closed completed upstream | fix/3665-bug-protocol-detection-for-custom | https://github.com/yetone/magpie/issues/3665#issuecomment-6455443425 |  |
| #3668 | zane668 | en | open | A | upstream landed/fixed: Shipped in v0.1.2673 (fa6725ee7), using @skygunner's patch from their branch plus the provider na |  | https://github.com/yetone/magpie/issues/3668#issuecomment-6411424119 |  |
| #3669 | kshq1996 | en | open | D | no upstream comments after ours | fix/3669-minimax-cn-url | https://github.com/yetone/magpie/issues/3669#issuecomment-6429777197 |  |
| #3674 | chenyanzhang000 | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3674#issuecomment-6410685355 |  |
| #3680 | 0xJchen | en | open | D | no upstream comments after ours | fix/3680-wsl-probe-fresh-connection | https://github.com/yetone/magpie/issues/3680#issuecomment-6462833280 |  |
| #3683 | wyx1818 | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/3683-mcp-oauth-callback | https://github.com/yetone/magpie/issues/3683#issuecomment-6450614859 |  |
| #3684 | Terryli246 | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3684#issuecomment-6726189148 |  |
| #3690 | yinwu01 | en | open | D | community comments after ours (2), no direct ask to skygunner | fix/3690-claude-room-freed | https://github.com/yetone/magpie/issues/3690#issuecomment-6429778471 |  |
| #3698 | coderbiq | en | open | D | community comments after ours (3), no direct ask to skygunner | fix/3698-prefix-off | https://github.com/yetone/magpie/issues/3698#issuecomment-6429783205 |  |
| #3701 | yetone | en | open | D | no upstream comments after ours | fix/3701-held-clause | https://github.com/yetone/magpie/issues/3701#issuecomment-6429777819 |  |
| #3703 | SunflowersLwtech | en | open | D | no upstream comments after ours | fix/3703-bug-a-pinned-member-that-starts-failing- | https://github.com/yetone/magpie/issues/3703#issuecomment-6457321665 |  |
| #3705 | yetone | en | open | D | no upstream comments after ours | fix/3705-remote-key-limit | https://github.com/yetone/magpie/issues/3705#issuecomment-6429778980 |  |
| #3708 | yinwu01 | en | open | D | no upstream comments after ours | fix/3708-agy-settle | https://github.com/yetone/magpie/issues/3708#issuecomment-6429781699 |  |
| #3713 | xdeers2 | zh | open/reopened | D | no upstream comments after ours | fix/3713-canonical-slug | https://github.com/yetone/magpie/issues/3713#issuecomment-6477047104 |  |
| #3716 | fzh365 | zh | open | D | no upstream comments after ours | fix/3716-options-field | https://github.com/yetone/magpie/issues/3716#issuecomment-6429779510 |  |
| #3721 | NoneMore | en | closed/completed | A | closed completed upstream | fix/3721-siwc-quota | https://github.com/yetone/magpie/issues/3721#issuecomment-6429782671 |  |
| #3722 | yinwu01 | en | open | D | no upstream comments after ours | fix/3722-pii-a-masking-request-in | https://github.com/yetone/magpie/issues/3722#issuecomment-6455259232 |  |
| #3723 | yinwu01 | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/3723-import-stamps | https://github.com/yetone/magpie/issues/3723#issuecomment-6429781209 |  |
| #3724 | fanyuxaun386 | en | open | D | no upstream comments after ours | fix/3724-xfyun-spark-x2-6-a | https://github.com/yetone/magpie/issues/3724#issuecomment-6455208188 |  |
| #3738 | hheei | en | open | D | no upstream comments after ours | fix/3738-free-model-rest-id | https://github.com/yetone/magpie/issues/3738#issuecomment-6438171195 |  |
| #3742 | yinwu01 | en | open | D | no upstream comments after ours | fix/3742-late-interrupt | https://github.com/yetone/magpie/issues/3742#issuecomment-6429780633 |  |
| #3745 | Zachary-wW | en | open | A | upstream landed/fixed: Ported @yinwu01's patch: `c0dd5838c`, in v0.1.2855. The Library's write-failure card has a × that | fix/3745-bug-library-write-failure-banner-cannot- | https://github.com/yetone/magpie/issues/3745#issuecomment-6457515456 |  |
| #3751 | yinwu01 | zh | closed/completed | A | closed completed upstream | fix/3751-codex-meta-length, fix/3751-codex-move-lock | https://github.com/yetone/magpie/issues/3751#issuecomment-6429782208 |  |
| #3754 | SunRt233 | zh | open | D | community comments after ours (2), no direct ask to skygunner | fix/3754-wsl-profile-probe | https://github.com/yetone/magpie/issues/3754#issuecomment-6452799987 |  |
| #3775 | yetone | en | closed/completed | A | closed completed upstream | fix/3775-token-count | https://github.com/yetone/magpie/issues/3775#issuecomment-6429791341 |  |
| #3776 | yetone | en | open | D | no upstream comments after ours | fix/3776-jp-phone | https://github.com/yetone/magpie/issues/3776#issuecomment-6429790548 |  |
| #3785 | yetone | en | open | A | upstream landed/fixed: Shipped in v0.1.1771 (f2b4b9894): skygunner's patch, picked as is apart from the code comment (cr | fix/3785-pool-tooltip | https://github.com/yetone/magpie/issues/3785#issuecomment-6438171807 |  |
| #3788 | skygunner | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3788#issuecomment-6452850862 |  |
| #3790 | skyArony | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/3790-tencent-cloud-tokenhub-international-guangzhou | https://github.com/yetone/magpie/issues/3790#issuecomment-6449948075 |  |
| #3791 | skygunner | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3791#issuecomment-6432514682 |  |
| #3792 | yinwu01 | en | open | D | no upstream comments after ours | fix/3792-chat-error-no-start | https://github.com/yetone/magpie/issues/3792#issuecomment-6438172447 |  |
| #3793 | yetone | en | open | D | no upstream comments after ours | fix/3793-as-sent-spawn-agent | https://github.com/yetone/magpie/issues/3793#issuecomment-6438173089 |  |
| #3794 | skygunner | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/3794#issuecomment-6425493334 |  |
| #3805 | yetone | en | open | D | no upstream comments after ours | fix/3805-ready-not-models | https://github.com/yetone/magpie/issues/3805#issuecomment-6438173724 |  |
| #3812 | zx3332352 | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/3812-workdir-enterable | https://github.com/yetone/magpie/issues/3812#issuecomment-6438174287 |  |
| #3814 | yetone | en | open | D | no upstream comments after ours | fix/3814-inline-object-bytes | https://github.com/yetone/magpie/issues/3814#issuecomment-6438174941 |  |
| #3816 | kudi68 | en | open | D | no upstream comments after ours | fix/3816-plan-quota-readat | https://github.com/yetone/magpie/issues/3816#issuecomment-6438175575 |  |
| #3821 | MengXiangxi | zh | open | A | upstream landed/fixed: Landed from Coffemon's patch, authorship kept: 7a1bfaf2f and ef7f55fe9 on main, in v0.1.2181. Wha | fix/3821-hooks-discover | https://github.com/yetone/magpie/issues/3821#issuecomment-6438176207 |  |
| #3822 | 99zhangpengxiang | zh | open | D | no upstream comments after ours | fix/3822-codex-mcp-config | https://github.com/yetone/magpie/issues/3822#issuecomment-6455166497 |  |
| #3826 | royluoh | zh | open | D | no upstream comments after ours | fix/3826-zcode-levels-order | https://github.com/yetone/magpie/issues/3826#issuecomment-6438176803 |  |
| #3828 | cloudusa | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/3828-opencode-bin-path | https://github.com/yetone/magpie/issues/3828#issuecomment-6438177401 |  |
| #3834 | nickfan | zh | open | D | no upstream comments after ours | fix/3834-config-sync-can-create-a | https://github.com/yetone/magpie/issues/3834#issuecomment-6587085163 |  |
| #3838 | davisjiahao | zh | open | D | no upstream comments after ours | fix/3838-requests-key-plan-identity | https://github.com/yetone/magpie/issues/3838#issuecomment-6452733392 |  |
| #3846 | yinwu01 | en | open | D | no upstream comments after ours | fix/3846-mainview-view-param | https://github.com/yetone/magpie/issues/3846#issuecomment-6587087185 |  |
| #3848 | yinwu01 | en | open | D | no upstream comments after ours | fix/3848-probe-timeout-flaky | https://github.com/yetone/magpie/issues/3848#issuecomment-6587088614 |  |
| #3849 | yorhaha | zh | open | D | community comments after ours (2), no direct ask to skygunner | fix/3849-codex-compaction-custom-tool-call | https://github.com/yetone/magpie/issues/3849#issuecomment-6587088118 |  |
| #3851 | 7Sageer | en | closed/completed | A | closed completed upstream | fix/3851-codex-identity-consistency | https://github.com/yetone/magpie/issues/3851#issuecomment-6587089070 |  |
| #3878 | yetone | en | open | D | no upstream comments after ours | fix/3878-redact-false-alarms | https://github.com/yetone/magpie/issues/3878#issuecomment-6726579174 |  |
| #3886 | Peterpig | zh | open | D | no upstream comments after ours | fix/3886-fix | https://github.com/yetone/magpie/issues/3886#issuecomment-6457115042 |  |
| #3899 | yetone | en | open | D | no upstream comments after ours | fix/3899-gui-omp-roles-5s-locator | https://github.com/yetone/magpie/issues/3899#issuecomment-6456134183 |  |
| #3903 | jobscn | zh | open | D | no upstream comments after ours | fix/3903-input-prediction-off | https://github.com/yetone/magpie/issues/3903#issuecomment-6587084660 |  |
| #3909 | yxd-ym | en | open | D | no upstream comments after ours | fix/3909-google-upstreams-large-requests-fail, fix/3909-stale-upload-test-comment | https://github.com/yetone/magpie/issues/3909#issuecomment-6477044902 |  |
| #3914 | fromb612 | zh | open | D | no upstream comments after ours | fix/3914-seedream-ark | https://github.com/yetone/magpie/issues/3914#issuecomment-6587085676 |  |
| #3918 | lghnb | zh | open | D | community comments after ours (2), no direct ask to skygunner | fix/3918-omp-effort-routing | https://github.com/yetone/magpie/issues/3918#issuecomment-6457515754 |  |
| #3925 | ati121 | en | open | D | no upstream comments after ours | fix/3925-rerank-record-name | https://github.com/yetone/magpie/issues/3925#issuecomment-6457516082 |  |
| #3927 | yinwu01 | en | open | D | no upstream comments after ours | fix/3927-plugins-magpie-plugin-add-a | https://github.com/yetone/magpie/issues/3927#issuecomment-6454874359 |  |
| #3928 | yinwu01 | en | open | D | no upstream comments after ours | fix/3928-gui-an-npm-card-installs | https://github.com/yetone/magpie/issues/3928#issuecomment-6452110741 |  |
| #3930 | yinwu01 | en | open | D | no upstream comments after ours | fix/3930-gui-a-repository-s-card | https://github.com/yetone/magpie/issues/3930#issuecomment-6450133579 |  |
| #3984 | yinwu01 | en | open | D | no upstream comments after ours | fix/3984-cancel-pending-mcp-signin | https://github.com/yetone/magpie/issues/3984#issuecomment-6727813099 |  |
| #4118 | xiaobaifly7 | en | open | D | no upstream comments after ours | fix/4118-pii-stale-miss | https://github.com/yetone/magpie/issues/4118#issuecomment-6726308927 |  |
| #4136 | skygunner | zh | open | D | no upstream comments after ours | fix/4136-wsl-probe-lock | https://github.com/yetone/magpie/issues/4136#issuecomment-6485286748 |  |
| #4137 | skygunner | zh | open | D | no upstream comments after ours | fix/4137-davsync-read-cap | https://github.com/yetone/magpie/issues/4137#issuecomment-6485262880 |  |
| #4141 | skygunner | zh | open | D | no upstream comments after ours | fix/4141-dav-put-version | https://github.com/yetone/magpie/issues/4141#issuecomment-6485383967 |  |
| #4144 | skygunner | zh | open | B | maintainer rejected/declined our patch — needs rework or acknowledgment | fix/4144-sessions-frame-tail | https://github.com/yetone/magpie/issues/4144#issuecomment-6485933827 | https://github.com/yetone/magpie/issues/4144#issuecomment-6935860474 |
| #4234 | yetone | en | open | D | no upstream comments after ours | fix/4234-account-list-test-budget | https://github.com/yetone/magpie/issues/4234#issuecomment-6727235408 |  |
| #4235 | yetone | en | open | D | no upstream comments after ours | fix/4235-busy-test-handler-completion | https://github.com/yetone/magpie/issues/4235#issuecomment-6726980305 |  |
| #4258 | skygunner | en | open | A | upstream landed/fixed: Face 1 landed in v0.1.4130: @ggbdpq's 1dbd3bb07 (authorship kept), plus my f62e93b8a. Claude Code | fix/4258-agent-stash-durability | https://github.com/yetone/magpie/issues/4258#issuecomment-6494205053 |  |
| #4333 | yetone | en | open | D | no upstream comments after ours | fix/4333-desktop-native-session-ids | https://github.com/yetone/magpie/issues/4333#issuecomment-6726991549 |  |
| #4385 | yetone | en | open | B | maintainer rejected/declined our patch — needs rework or acknowledgment |  | https://github.com/yetone/magpie/issues/4385#issuecomment-6654123271 | https://github.com/yetone/magpie/issues/4385#issuecomment-6935799230 |
| #4428 | yetone | en | open | D | no upstream comments after ours | fix/4428-aside-gemini-keeps-chat | https://github.com/yetone/magpie/issues/4428#issuecomment-6728383540 |  |
| #4592 | yetone | en | open | D | no upstream comments after ours | fix/4592-claude-switch-credential-first | https://github.com/yetone/magpie/issues/4592#issuecomment-6727158843 |  |
| #4603 | yetone | en | open | D | no upstream comments after ours | fix/4603-dpkg-fixture-windows-name | https://github.com/yetone/magpie/issues/4603#issuecomment-6727973226 |  |
| #4672 | yetone | en | open | D | no upstream comments after ours | fix/4672-portable-vscode-extension-detection | https://github.com/yetone/magpie/issues/4672#issuecomment-6727595279 |  |
| #4675 | yetone | en | open | D | no upstream comments after ours | fix/4675-last-stream-context-overflow | https://github.com/yetone/magpie/issues/4675#issuecomment-6727547912 |  |
| #4821 | wsyzxjn | zh | open | D | no upstream comments after ours | fix/4821-quota-all-pools | https://github.com/yetone/magpie/issues/4821#issuecomment-6588798863 |  |
| #4825 | yetone | en | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/4825#issuecomment-6588799977 |  |
| #4829 | yetone | en | open | D | no upstream comments after ours | fix/4829-plugin-run-lifetime-no-rest | https://github.com/yetone/magpie/issues/4829#issuecomment-6588797560 |  |
| #4831 | yetone | en | open | D | no upstream comments after ours | fix/4831-wallet-balance-no-line | https://github.com/yetone/magpie/issues/4831#issuecomment-6588798303 |  |
| #4843 | z1edge | en | open | D | no upstream comments after ours | fix/4843-status-lists-bound-addr | https://github.com/yetone/magpie/issues/4843#issuecomment-6602606384 |  |
| #4854 | jsk644102-art | en | open | D | community comments after ours (1), no direct ask to skygunner |  | https://github.com/yetone/magpie/issues/4854#issuecomment-6588796052 |  |
| #4855 | MYCF711 | en | closed/completed | A | closed completed upstream | fix/4855-relay-error-frame | https://github.com/yetone/magpie/issues/4855#issuecomment-6726615266 |  |
| #4870 | yetone | en | open | D | no upstream comments after ours | fix/4870-minimax-moved-datadir | https://github.com/yetone/magpie/issues/4870#issuecomment-6602607627 |  |
| #4872 | OldKing-lion | en | open | A | upstream landed/fixed, maintainer asks reporter to confirm after upgrading | fix/4872-wsl-standin-gate | https://github.com/yetone/magpie/issues/4872#issuecomment-6602607001 |  |
| #4874 | JJasonSun | en | open | D | community comments after ours (2), no direct ask to skygunner | fix/4874-overlap-text-block-start | https://github.com/yetone/magpie/issues/4874#issuecomment-6588792384 |  |
| #4878 | yetone | en | open | D | no upstream comments after ours | fix/4878-rune-clip-current, fix/4878-rune-safe-clip | https://github.com/yetone/magpie/issues/4878#issuecomment-6726147491 |  |
| #4904 | yetone | en | open | D | no upstream comments after ours | fix/4904-import-folds-endpoint | https://github.com/yetone/magpie/issues/4904#issuecomment-6588797078 |  |
| #4906 | MystryMocha | en | closed/completed | A | closed completed upstream | fix/4906-mistral-translated-reasoning | https://github.com/yetone/magpie/issues/4906#issuecomment-6588792902 |  |
| #4921 | cth123456 | zh | open | A | upstream landed/fixed: 已合入 v0.1.2766（c86fa2a34）。采用的是 skygunner 的版本（fcea11649），没有用 ggbdpq 的 57dc7b420。原因：57dc7b420 只在请求的不 | fix/4921-citation-offsets | https://github.com/yetone/magpie/issues/4921#issuecomment-6588794598 |  |
| #4923 | Nine-Days06 | zh | open | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/4923#issuecomment-6588799501 |  |
| #4942 | Ricinix | en | open | D | no upstream comments after ours | fix/4942-title-function-call | https://github.com/yetone/magpie/issues/4942#issuecomment-6588793536 |  |
| #4945 | barryzzz | en | open | B | user @ekoooli addressed skygunner: @skygunner 补一条更正和根因定位（上一条里关于 id 后缀的推测是错的，原因见下）。 ## 根因定位（更正上一条）：magpie 自己插入的 placehold | fix/4945-deepseek-parallel-tool-uses | https://github.com/yetone/magpie/issues/4945#issuecomment-6726385771 | https://github.com/yetone/magpie/issues/4945#issuecomment-6935746577 |
| #4952 | skygunner | en | closed/not_planned | D | no upstream comments after ours |  | https://github.com/yetone/magpie/issues/4952#issuecomment-6584528128 |  |
| #4953 | FjlI5 | zh | open | D | no upstream comments after ours | fix/4953-codex-input-base-instructions | https://github.com/yetone/magpie/issues/4953#issuecomment-6588795088 |  |
| #4955 | yetone | en | open | D | no upstream comments after ours | fix/4955-seedance-not-chat | https://github.com/yetone/magpie/issues/4955#issuecomment-6588796600 |  |
| #4956 | yetone | zh | open | D | no upstream comments after ours | fix/4956-4854-bridge-deferred-tool-names | https://github.com/yetone/magpie/issues/4956#issuecomment-6588795549 |  |
| #5018 | yetone | en | open | D | no upstream comments after ours | fix/5018-cached-search-model-wire | https://github.com/yetone/magpie/issues/5018#issuecomment-6727406412 |  |
| #5063 | yetone | en | open | D | no upstream comments after ours | fix/5063-bun-standard-stream-guard | https://github.com/yetone/magpie/issues/5063#issuecomment-6727594983 |  |
| #5069 | yetone | en | open | D | no upstream comments after ours | fix/5069-desktop-test-windows-skip | https://github.com/yetone/magpie/issues/5069#issuecomment-6728035992 |  |
| #5150 | turingcat | en | open | D | no upstream comments after ours | fix/5150-lane-test-owner-isolation | https://github.com/yetone/magpie/issues/5150#issuecomment-6726860674 |  |
| #5178 | yetone | en | open | D | no upstream comments after ours | fix/5178-provider-account-recovery | https://github.com/yetone/magpie/issues/5178#issuecomment-6727428610 |  |
| #5204 | donob-z | zh | open | D | no upstream comments after ours | fix/5204-messages-unsigned-thinking | https://github.com/yetone/magpie/issues/5204#issuecomment-6727525566 |  |
| #5239 | yinwu01 | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/5239-opencode-fixture-git-boundary | https://github.com/yetone/magpie/issues/5239#issuecomment-6729115638 |  |
| #5240 | rongyuio | en | open | D | no upstream comments after ours | fix/5240-ignore-browser-skill-runtime-lock | https://github.com/yetone/magpie/issues/5240#issuecomment-6727491853 |  |
| #5264 | yetone | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/5264-signin-marker-complete | https://github.com/yetone/magpie/issues/5264#issuecomment-6726771708 |  |
| #5271 | yetone | en | open | A | upstream landed/fixed: Ported in 363adb352 with credit to @skygunner's 94d8550c8. Ran -count=20 green on Linux (wsl) and | fix/5271-auth-lock-event-loop-witness | https://github.com/yetone/magpie/issues/5271#issuecomment-6728787186 |  |
| #5290 | AnacondaKC | en | open | D | no upstream comments after ours | fix/5290-library-fixture-gotmpdir | https://github.com/yetone/magpie/issues/5290#issuecomment-6726939500 |  |
| #5327 | yetone | en | closed/completed | A | closed completed upstream | fix/5327-i18n-comment-ownership | https://github.com/yetone/magpie/issues/5327#issuecomment-6726730978 |  |
| #5334 | yetone | en | open | D | no upstream comments after ours | fix/5334-closed-pipe-ending | https://github.com/yetone/magpie/issues/5334#issuecomment-6728957122 |  |
| #5341 | yetone | en | open | D | no upstream comments after ours | fix/5341-connectex-keeps-last-quota | https://github.com/yetone/magpie/issues/5341#issuecomment-6728466078 |  |
| #5342 | yetone | en | open | D | no upstream comments after ours | fix/5342-real-absolute-image-fixture | https://github.com/yetone/magpie/issues/5342#issuecomment-6728669541 |  |
| #5343 | yetone | en | open | D | no upstream comments after ours | fix/5343-mise-script-permissions | https://github.com/yetone/magpie/issues/5343#issuecomment-6728759905 |  |
| #5344 | yetone | en | open | D | no upstream comments after ours | fix/5344-windows-claude-script-expectation | https://github.com/yetone/magpie/issues/5344#issuecomment-6728744145 |  |
| #5348 | hhh2210 | en | open | D | no upstream comments after ours | fix/5348-json-runtime-parity | https://github.com/yetone/magpie/issues/5348#issuecomment-6726778451 |  |
| #5359 | neighbads | en | open | D | no upstream comments after ours | fix/5359-plugin-account-header-fields | https://github.com/yetone/magpie/issues/5359#issuecomment-6728187899 |  |
| #5376 | yetone | en | open | D | no upstream comments after ours | fix/5376-translated-phantom-tool-finish | https://github.com/yetone/magpie/issues/5376#issuecomment-6727317867 |  |
| #5425 | yetone | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/5425-nav-timeout-siblings | https://github.com/yetone/magpie/issues/5425#issuecomment-6728890326 |  |
| #5433 | yetone | en | closed/completed | A | closed completed upstream | fix/5433-register-mcp-stream-before-send | https://github.com/yetone/magpie/issues/5433#issuecomment-6727318197 |  |
| #5435 | yetone | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/5435-subagent-kept-fewest | https://github.com/yetone/magpie/issues/5435#issuecomment-6729026994 |  |
| #5522 | yetone | en | open | D | community comments after ours (2), no direct ask to skygunner | fix/5522-node-fetch-browser-classification | https://github.com/yetone/magpie/issues/5522#issuecomment-6727341707 |  |
| #5526 | yetone | en | open | D | no upstream comments after ours | fix/5526-codex-image-relay-records | https://github.com/yetone/magpie/issues/5526#issuecomment-6727983978 |  |
| #5528 | yetone | en | open | D | no upstream comments after ours | fix/5528-persist-update-check-time | https://github.com/yetone/magpie/issues/5528#issuecomment-6727503378 |  |
| #5541 | yetone | en | open | D | no upstream comments after ours | fix/5541-overlapping-held-catalog-refresh | https://github.com/yetone/magpie/issues/5541#issuecomment-6727805718 |  |
| #5587 | yetone | en | open | D | no upstream comments after ours | fix/5587-agent-test-running-isolation | https://github.com/yetone/magpie/issues/5587#issuecomment-6727055810 |  |
| #5628 | yetone | en | open | D | no upstream comments after ours | fix/5628-reset-history-preservation | https://github.com/yetone/magpie/issues/5628#issuecomment-6727039101 |  |
| #5656 | Mingtha | zh | open | D | no upstream comments after ours | fix/5656-claude-bridge-history | https://github.com/yetone/magpie/issues/5656#issuecomment-6728862107 |  |
| #5676 | yetone | en | open | D | no upstream comments after ours | fix/5676-gateway-key-badges-no-shift | https://github.com/yetone/magpie/issues/5676#issuecomment-6728005909 |  |
| #5681 | yetone | en | open | D | no upstream comments after ours | fix/5681-session-tests-timezones | https://github.com/yetone/magpie/issues/5681#issuecomment-6727662540 |  |
| #5684 | wang1078564969 | zh | open | D | community comments after ours (1), no direct ask to skygunner | fix/5684-dsh-trailing-sentinels | https://github.com/yetone/magpie/issues/5684#issuecomment-6727116312 |  |
| #5688 | iuiiui | en | open | D | no upstream comments after ours | fix/5688-plugin-page-options-limit | https://github.com/yetone/magpie/issues/5688#issuecomment-6727691785 |  |
| #5690 | yetone | en | closed/completed | A | closed completed upstream | fix/5690-gateway-stats-virtual-midnight | https://github.com/yetone/magpie/issues/5690#issuecomment-6727180940 |  |
| #5696 | mikehasa | en | open | D | community comments after ours (1), no direct ask to skygunner | fix/5696-image-count-fallback | https://github.com/yetone/magpie/issues/5696#issuecomment-6726822036 |  |
| #5708 | yetone | en | open | D | no upstream comments after ours | fix/5708-tray-motion-reduced-inflight | https://github.com/yetone/magpie/issues/5708#issuecomment-6728178163 |  |
| #5713 | wudilyy999 | zh | open | D | community comments after ours (4), no direct ask to skygunner | fix/5713-search-history-chat | https://github.com/yetone/magpie/issues/5713#issuecomment-6726797345 |  |
