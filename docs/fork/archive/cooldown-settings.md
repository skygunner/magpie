# Error cooldown settings — suspended historical trial

**Status: suspended in the upstream-first integration since 2026-11-05.**
The production 0.1.3036-p2 image has no fork cooldown editor or `/api/cooldown`.
The instructions, defaults and test results below describe the earlier trial
on [`feature/cooldown-settings`](https://github.com/skygunner/magpie/tree/feature/cooldown-settings),
not current production. Current decisions: [PATCHES.md](../../../PATCHES.md).

This feature lives on `feature/cooldown-settings` in skygunner/magpie. It was
based on the production branch carrying v0.1.2051-p3 and deployed to docker1 in
v0.1.2051-p4; see [deployment history](../history.md#deployment-history).
No upstream feature request has been submitted.

## Where to use it

- Settings → Network and sharing → Error cooldown and backoff: global defaults.
- Providers → clock button (or provider context menu): provider defaults.
- Open a provider → Error cooldown and backoff beside an account/key: its overrides.

The editor shows effective values, sources, inherited placeholders, detailed
examples, model timeout/cancellation opt-ins and active rests. Clear a number to
inherit. For a model, 0 disables its opt-in and a blank inherits. The provider's
ordinary editing form and refreshes preserve these settings. A field-specific
settings save preserves the policy when unrelated preferences are changed.

Defaults are 2 → 4 → 8 → 16 → 32 → 64 minutes for repeated rate limits, and a
fixed 30-minute wait for exhausted quota without a reset. Existing explicit
values, such as quotaRest=540/1080, keep their meaning. Response timeouts and
other failures keep their existing 1-minute starts and 10-minute caps by default.

Account/key overrides provider, which overrides global defaults. Existing
provider restCap, quotaRest, quotaResetAt, modelCooldown and limitRestOff values
remain readable as overrides over global defaults. Saving the provider's policy
converts its legacy values to the explicit policy shown in the form. The old
rest-cap/quota-rest CLI commands still work, clearing the corresponding provider
policy override when changed. Replacing a key carries its policy to the new
fingerprint. No dependency or additional persistent file is introduced.

See [gateway policy contract](#historical-runtime-contract)
for the exact runtime rules and API. The UI remains available with the Routing
plugin off, which is why its shared editor belongs to the core provider/settings
UI; the script budget increases by exactly the editor's added bytes.

## Validation

All tests use isolated homes or local browser fixtures, without production
credentials or vendor requests. Local builds use the existing dependencies.

- `go test -tags nogui ./internal/provider ./internal/gateway ./internal/gui -run 'Test(Cooldown|Rate|OwnRest|RestCap|Quota|ModelCooldown|OptedIn|Slow|Canceled|SaveKeeps)' -count=1`: passed.
- `go test -race -tags nogui ./internal/provider ./internal/gateway ./internal/gui -run 'Test(Cooldown|ModelCooldown|OptedIn|CanceledSlow|TryNow|CoreScriptBudget|DatesFollowGUILanguage|SettingsSaveKeepsWhatItDoesNotSend)' -count=3`: passed.
- `TestCooldownDefaultRateLadder` copied alone to the unmodified base failed with a real behavior assertion: `rest 59.999…s; want 2m`. It passes on the feature branch.
- `cooldown-settings.test.cjs`, `provider-limit-rest.test.cjs` and `i18n-keys.test.cjs`: 74 checks passed. Chromium and WebKit, English, simplified/traditional Chinese, Japanese and German. The new form checks global/provider/key/account entry points, save failure retention, model caps, persistence, inheritance, numeric validation, provider refreshes and 420px layout with fixed title/footer.
- Existing provider concurrency browser tests passed separately. A Chinese narrow-screen fixture screenshot was visually inspected.
- Native/Linux/Windows vet and builds passed; Windows was cross-compiled, not run on Windows.
- `gofmt -l` and `git diff --check`: clean.

The complete Go suite and complete GUI suite were also attempted. They cannot
be called green on this base. Unmodified-base reproductions identify unrelated
failures in TestAgyShell, TestClaudeClientCacheAbsentUnchanged,
TestCodexFittedImagesPastTheBudget, TestMarshalRawIsEncodingJSON,
TestServeWSLListensBesideLoopback, TestPluginSOCKSProxy and
TestOpenCodeSessionOnEveryCall (also reproduced with 30 repetitions on the base). The full GUI attempt
was stopped after agent-panel-name-width.test.cjs failed; running that file on
the unmodified base reproduces its panel-text truncation failures. Consequently
the remainder of the 991-file GUI suite is not claimed as passing. These baseline
problems are recorded here instead of expanding this fork-only feature to change
them. On this machine tests must use umask 022; 027 also breaks existing
permission assertions in internal/edit and internal/testenv.

Local verification logs are `/tmp/cooldown-*.log`; they are not part of the
persisted configuration. Runtime cooldown counters remain process-local.

## Historical runtime contract

`settings.CooldownPolicy` exposes the gateway's error rests in Settings →
Network and sharing → Error cooldown and backoff, on provider rows, and on
individual account/key rows. `/api/cooldown?id=<provider>&ref=<account-or-key>`
reads or replaces that scope's optional overrides; omitting both selects global
settings. A key is addressed by its fingerprint, never its secret. Reads include
resolved values, each value's source, active account/model rests and the provider's
routing-group first-response waits. The response-wait control links to Routing.

The resolution order in `Provider.CooldownFor` is built-in defaults, global
`settings.json.cooldown`, legacy provider fields, provider `cooldown`, then
`accountCooldown[lowercase account email or key fingerprint]`. Empty fields
inherit. Explicit false overrides a parent switch; a model's explicit zero
turns its opt-in off, while removing that model's entry inherits. Subscription
and plugin accounts receive these fields through `withPicks`, including sibling
accounts in `AlsoOn`. Replacing a key moves its overrides to the new fingerprint.

Defaults in this fork are rateStart=2, rateCap=64, slowStart=1,
slowThreshold=2, slowCap=10, failureCap=10, quotaRest=30 (all times in minutes).
Rate-limit rests therefore run 2, 4, 8, 16, 32, 64, 64 minutes. Ordinary unknown
failures still start at one minute; repeated first-response timeouts start
resting on the second occurrence. Upstream 504s continue through ordinary
failure classification; gateway first-response deadline 504s use the model's
slow-start ladder. Backoff doubles only below its cap, avoiding duration overflow
for large custom starts. The first rest is shortened if an inherited cap is
lower; the editor rejects an explicitly entered start larger than its cap.

`quotaRest` retains its existing meaning: a fixed wait for exhausted quota with
no recognized reset/window, repeated without doubling. `quotaResetAt` replaces
that fixed wait with the next daily time; an explicit empty string clears an
inherited daily reset. Times without an offset use the gateway server's clock;
`00:00+08:00` is explicit UTC+8. Recognized vendor resets/windows retain their
existing priority and bounds. A vendor wait can exceed the locally calculated
rate/backoff cap; Retry-After retains its one-hour bound. User-set 9/18-hour
quota rests are preserved; only the unset default changes from 15 to 30 minutes.

`failRest=false` is the legacy `restCap off` switch and disables unknown-failure,
slow-start and opted-in model cancellation rests. `limitRest=false` is the
inverse of `limitRestOff` and disables quota, credit and rate-limit rests.
`models` is the scoped editor for legacy `modelCooldown`: exact model IDs mapped
to caps from 1 to 60 minutes, zero to disable. Enabled models retain first-event
1-minute starts, shared timeout/499 counts, hard holds and 503 + Retry-After when
none can be tried. Manual client cancellations count too. Runtime state is
process-local and reset by restart; successful answers reset the relevant ladder.

Legacy RestCap/QuotaRest/QuotaResetAt/ModelCooldown/LimitRestOff remain readable
and retain priority over global settings. The provider policy editor displays
them as existing overrides and converts them into the scoped policy only when
that provider's policy is saved; other provider form saves preserve the policy.
The old provider form's limit-rest toggle also updates a scoped limit-rest switch
when one exists. The policy editor can clear any override to restore inheritance.
A global policy save edits just that settings field under the settings file lock.

Saving changes subsequent failure handling; existing rest deadlines are not
rewritten. Account/key policy panels list active rests and reuse `gateway/unrest`
for Try now. This releases a rest while retaining failure counts, including
rate-limit counts; replacing a key still clears its old rest records. Background probes and vendor-reset parsing otherwise retain their
existing rules. These are routing policies, not rate pacing (MaxRPM), concurrency
limits or the group's first-response deadline.

Verification: `TestCooldown*` in provider, gateway and GUI packages covers the
new defaults, per-identity inheritance, legacy settings, signed-in picks,
validation and save/reset behavior. `cooldown-settings.test.cjs` exercises the
real settings form, failures, persistence, numeric validation and inheritance in
all five languages and both browser engines. Existing rate/backoff tests pin the
updated public defaults.
