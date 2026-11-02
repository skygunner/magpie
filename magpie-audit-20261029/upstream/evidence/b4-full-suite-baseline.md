# B4 full-suite baseline comparison

A later B4 `go test -tags nogui ./...` run exited 1. Its filtered failure list is in the implementation worker session `st_01a18590`, command `bash_286`; a standalone complete raw log was not located.

The following six top-level failures also occurred in focused runs on `/tmp/mg-fix-pii-stale-miss` (agent code unchanged from upstream/main baseline `926ef769f`) in this environment, with `umask 0022` and `SHELL=/bin/bash`:

- `internal/gateway.TestClaudeClientCacheAbsentUnchanged`
- `internal/gateway.TestCodexFittedImagesPastTheBudget`
- `internal/gateway.TestMarshalRawIsEncodingJSON`
- `internal/gateway.TestServeWSLListensBesideLoopback`
- `internal/gateway.FuzzMarshalRaw` (baseline seed #7)
- `internal/plugin.TestPluginSOCKSProxy`

Commands, both exit 1:

```sh
go test -tags nogui ./internal/gateway -run '^(TestClaudeClientCacheAbsentUnchanged|TestCodexFittedImagesPastTheBudget|TestMarshalRawIsEncodingJSON|TestServeWSLListensBesideLoopback|FuzzMarshalRaw)$' -count=1 -timeout 2m
go test -tags nogui ./internal/plugin -run '^TestPluginSOCKSProxy$' -count=1 -timeout 2m
```

The baseline comparison establishes the failures are reproducible on the baseline in this environment; it does not establish their causes. Full repository tests are not claimed green. No Windows tests were run.
