# Comment drafts — corrections to earlier upstream comments (NOT SENT)

Prepared 2026-10-24. These are held for later sending when the v2 heads
are pushed and confirmed; nothing here has been posted. Each names the
comment it corrects.

---

## Draft 1 — issue #3349 (decide= on plugin providers)

> Correction to [my earlier comment](https://github.com/yetone/magpie/issues/3349#issuecomment-6364310053): that fix stopped at the field. With `decide=` saved, `RouteDecider("commandcode-plan/typesafe/jev")` still answered `typesafe/jev is not a model of Command Code Plan` — a plugin's decision models were only the ones its feed marks, and the plan's feed lists 92 chat models and never its Jev, which is the shape of this issue.
>
> The completed fix (branch `fix/issue-batch-1023pm-v2`, commits `683547f0` identity, `1e7644d1` route): the model a user picks beside an explicit `decide=` on a plugin is answered as picked — `userPicked` requires both the decision API set and the model named in the picks, so nothing else opens. The end-to-end test runs the plan's shape on the plugin fixture: save → re-read (identity stable) → `RouteDecider` → `POST /v1/systemone` reaching the plan's host at `/provider/v1/systemone` with the picked model and the account's Bearer. Local routing and protocol construction only; the live-API behavior was the reporter's own verification.

## Draft 2 — issue #3370 (union type + items)

> Correction to [my earlier comment](https://github.com/yetone/magpie/issues/3370#issuecomment-6363745685): the claim that both commits cherry-pick independently is wrong — `b2336968` does not apply to a main without `6ae89820` (#3121's fix); the dependency is `6ae89820 → b2336968`. The review also found the cleanup only ran when the fold lifted another type to array, so `["array","object"]` beside both halves' keywords kept `properties`/`minProperties`/`maxProperties` on the array.
>
> Branch `fix/union-items-v2` carries the whole chain rebased: `d4e6998f` (= 6ae89820) → `99d6e6ed` (= b2336968) → `cbc6c3b3` (fold cleans by the type it ends, either order), with `TestPlainSchemaUnionFoldIsOrderIndependent` red on the `["array","object"]` order before `cbc6c3b3`. Schema-output testing only; the real Antigravity predicate was not called.

## Draft 3 — issue #3374 (agent_message relay)

> Correction to [my earlier comment](https://github.com/yetone/magpie/issues/3374#issuecomment-6364109042): the bare rewrite described there has been retired in favor of the same request path's richer reading (#3183's `PlainAgentMessage`), which keeps what the bare form dropped — the `From <author> to <recipient>` lead and image parts. The composition test runs the reporter's pair through `/backend-api/codex/responses` onto a key Responses provider and asserts the From/To lead and the task text; a sealed `agent_message` is refused 400 (sealed-task explanation) and reaches no upstream. Branch `fix/issue-batch-1023pm-v2`, commit `97a8a0c2`.

## Draft 4 — issue #2008 (rest persistence)

> Addition to [my earlier comment](https://github.com/yetone/magpie/issues/2008#issuecomment-6364369875): the review found `restSlow` wrote the rest into memory and never saved it, so a Why=slow, Status=504 rest was gone after a restart. Fixed on `fix/persist-gateway-rests-v2` (`5035eb0b`): the save runs after the lock is let go, as every other rest write does; `TestSlowRestReachesTheLedger` forms the rest through the real slow-start flow, then restores its absolute deadline, reason, status, failure count and model from the ledger after a simulated restart. No real daemon was restarted.
