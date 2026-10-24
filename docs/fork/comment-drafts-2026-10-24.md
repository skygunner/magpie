# Comment drafts — corrections after the second independent review (NOT SENT)

Prepared 2026-10-24, round 3. Supersedes the earlier draft file where they
overlap. Nothing here has been posted; the r3 heads named below are pushed
and read back. Permalinks of the comments being corrected are kept.

---

## Draft 1 — issue #3365 (redact overreach), correcting #issuecomment-6364536473

> Correction to [my earlier comment](https://github.com/yetone/magpie/issues/3365#issuecomment-6364536473): that fix was not complete. The second review's 24-case matrix (8 credential-ish field names × Mask / MaskJSON message / MaskJSON tool-file-content) showed hyphenated names (api-key, x-api-key, access-key, private-key), compound tokens (access_token, refresh_token) and values inside JSON-escaped tool content still taking the version/repo exceptions — `{"x-api-key":"Rq308c7/bT2mN4z"}` went out as written.
>
> Fixed on `r3-batch` in two steps, and a third corrected the second step's own regression. `a0b2faa8` made the exception read the field's name the way the ID rules already do (`fieldNameWords`/`nameWords`) and take it only for fields that do not name a credential outright; `c7acc938` restored the full kind lists that step had dropped — the compact check kept only apikey/accesskey/privatekey and six token compounds, so `token_signing_key` and `token_encryption_key` (kinds the assignment rules always took) leaked again. The final rule: the name, lower-cased with separators gone, carrying password/passwd/pwd/secret/credential, a key compound (apikey, accesskey, privatekey, authkey, signingkey, encryptionkey, encryptkey, decryptionkey) or a token compound (accesstoken, refreshtoken, authtoken, bearertoken, sessiontoken, apitoken, credentialstoken, credentialtoken, secrettoken) is a credential whatever it looks like. versionToken, tokenspeed-* and a bare `key` keep the exceptions, as the issue requires. The 24/24 matrix, the original three inputs, the eight compact/suffixed fields, the six token-key fields and the benign samples are green at head `4e136a4b`.

## Draft 2 — issue #3349 (decide= on plugin providers), correcting #issuecomment-6364310053 and #issuecomment-6364831515

> Two corrections to the earlier comments, both from the second review:
>
> 1. The route was not the whole story — `Deciders()`/`OfferedDeciders()` and `Jev()` still read only what the plugin's feed marks, so the explicitly configured Jev was routable but undiscoverable and not a default. Fixed on `r3-batch` (5b4e0d5f-equivalent, now `1e7644d1`-lineage rebased): a decision-shaped pick beside an explicit `decide=` joins the pickers and is the default. The route test now asserts discovery and the default beside the request path, and checks the feed through the catalog rather than the picks slice.
> 2. The pick-acceptance rule was too wide: any `Models` member counted, so selecting an ordinary chat model on a provider with a `decide=` reclassified it (`cline-pass/glm-5.3-flash` left the catalog). A pick now has to name a model a decision API answers by shape (a Jev, Bailian's, a Clef); chat picks stay chat. And an account's stored entry is told from an independent provider by its id — an id magpie knows as an account's or an installed plugin's (its sign-in may be out; the settings wait), including the plugin's public `…-plugin` form — so the identity fix hides nothing hand-written, with or without a name or key (a local System One server is asked without a key by contract).

## Draft 3 — issue #3370 (union type + items), correcting #issuecomment-6363745685

> Correction to [my earlier comment](https://github.com/yetone/magpie/issues/3370#issuecomment-6363745685): two things were wrong there. The claim that both commits cherry-pick independently is false — the hardening needs #3121's fold beneath it (`6ae89820 → b2336968` in the v2 tree, `d4e6998f → 99d6e6ed` in the r3 rebase of the same changes), and no commit of any of these chains picks alone. The second review also found the cleanup only ran when the fold lifted another type to array, so `["array","object"]` beside both halves' keywords kept `properties`/`minProperties`/`maxProperties` on the array.
>
> Current branch: `r3-union`, base `b437e4ad`, chain `ad7beadd → 45d9eb16 → 275b7a2e` (head, pushed 2026-10-24). It supersedes the earlier trees of the same three fixes: `fix/union-items` (v1, tip `b2336968`) and `fix/union-items-v2` (v2 rebuild, `d4e6998f → 99d6e6ed → cbc6c3b3`). Within the chain, the later commits need the earlier ones beneath them (`ad7beadd` is #3121's fold; `45d9eb16` and `275b7a2e` build on it) — take the chain in order. `TestPlainSchemaUnionFoldIsOrderIndependent` is red on the parent for the `["array","object"]` order and green on the head. Schema-output testing only; the real Antigravity predicate was not called.

## Draft 4 — issue #3374 (agent_message relay), correcting #issuecomment-6364109042

> Correction to [my earlier comment](https://github.com/yetone/magpie/issues/3374#issuecomment-6364109042): the bare rewrite described there has been retired in favor of the same request path's richer reading (#3183's `PlainAgentMessage`), which keeps the `From <author> to <recipient>` lead and image parts that the bare form dropped. The composition test runs the reporter's pair through `/backend-api/codex/responses` onto a key Responses provider and asserts the lead and the task text; a sealed `agent_message` is refused 400 with the sealed-task explanation and reaches no upstream. Branch `r3-batch`.

## Draft 5 — issue #2008 (rest persistence), correcting #issuecomment-6364369875

> Scope note on [my earlier comment](https://github.com/yetone/magpie/issues/2008#issuecomment-6364369875): the restart evidence is a simulated restart (in-memory rests cleared, ledger loaded again), not a real daemon restart; and the second review found `restSlow` wrote the rest into memory without saving it, so a Why=slow, Status=504 rest was gone after one. Fixed on `r3-rests`, base `b437e4ad`, chain `71b9805c → 3567cbf5 → d639c8d7` (head, pushed 2026-10-24; it supersedes the original `fix/persist-gateway-rests` chain `8d72abec → fcb0767a` and the v2 rebuild `2fc84b93 → 4d0b0b75` (tip `5035eb0b`)): the save runs after the lock is let go, and `TestSlowRestReachesTheLedger` restores the absolute deadline, reason, status, failure count and model from the ledger after the simulated restart. Cherry-picking: the chain's later commits need the earlier ones beneath them — take the whole chain in order (each commit names its base), not any commit alone.
