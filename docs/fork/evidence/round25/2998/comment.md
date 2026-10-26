Fixed on `fix/r25-minimax-moved-data` (fork `skygunner/magpie`, head `4cad21e120deced2560e904f8b9ba845a471e3c9`, one commit on top of `19253f67`): magpie now reads the desktop app's own record of the move and reaches the file the app reads.

**What changed.** `appdir.MiniMaxDirIn` reads `config.localRuntimeDataParentDir` from `%APPDATA%/MiniMax/minimax-agent-cn-config.json` and appends `.minimax`. `internal/agent.MiniMaxDir` (config.yaml) and `internal/sessions.MiniMaxDir` (sessions) both resolve through it, and the library's skills follow the resolved agent dir. `MINIMAX_DATA_DIR` stays the override. A missing, unreadable, malformed, blank or relative record keeps `~/.minimax`, so the old behavior survives anything magpie can't trust.

**Proof.** A new public-API test (`TestMiniMaxDirDesktopRecord`) drives both callers against the same genuine settings file - 14 cases covering the moved parent, whitespace, override precedence, missing/malformed/wrong-typed records and relative/missing APPDATA:

- RED on a pristine `git archive` of base `19253f67` (only the test added): both callers returned `~/.minimax`, exit 1, compile clean.
- GREEN after the fix: 14/14, exit 0.
- `go test -race -shuffle=on -count=1 ./internal/{agent,appdir,sessions,library}` all ok.
- `gofmt`, `go vet ./...`, `GOOS=linux go vet -tags nogui ./...`, `GOOS=windows go vet ./...`, native/Linux-nogui/Windows builds all clean.
- Real `magpie ls` in a sandbox HOME with the reporter's settings shape shows `~/minimax data/.minimax/config.yaml`.

Full `go test -tags nogui ./...` exits 1 in `internal/{edit,gateway,plugin,provider,testenv}`; the same failures reproduce on a clean base archive, so they are pre-existing and untouched by this change.

**Not verified.** No Windows desktop runtime here, so the desktop-side end-to-end is not re-run; the Windows work was a cross-build and vet only. Only the CN build's settings file is read - other builds keep `~/.minimax` unless they set the env var. Previous data parents are not chased; the current parent determines the active directory.

Source: [shared resolver](https://github.com/skygunner/magpie/blob/4cad21e120deced2560e904f8b9ba845a471e3c9/internal/appdir/minimax.go), [public regression test](https://github.com/skygunner/magpie/blob/4cad21e120deced2560e904f8b9ba845a471e3c9/internal/agent/minimax_dir_test.go). Raw RED/GREEN, race, vet/build, CLI and baseline logs are retained locally at `/home/vincenty/Projects/magpie-issues/evidence/r25-2998/`; these logs are not published in the fork. The commit message’s docs/fork evidence path is inaccurate; this is the correct location.