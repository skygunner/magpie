# Error cooldown settings — fork trial

This feature lives on `feature/cooldown-settings` in skygunner/magpie. It is
based on the production branch carrying v0.1.2051-p3. It has not been deployed to
docker1 and no upstream feature request has been submitted.

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

See [gateway policy contract](../subsystems/gateway-routing.md#fork-editable-error-cooldown-policy)
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
