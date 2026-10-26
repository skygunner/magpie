The rule matched the mDNS name as an address: `realEmail` (internal/redact/redact.go) kept `user@host.local` because it only turned away `localhost`, the reserved examples and file-looking hosts. `.local` is a name the mDNS serves (RFC 6762), not a delegated TLD, so nothing that ends in it can be a mail address.

Fixed on [fix/redact-local-host](https://github.com/skygunner/magpie/tree/fix/redact-local-host) (`aa0e3ec7`, one commit on `01468003`): `.local` joins `localhost` and the reserved examples in `realEmail`'s list. The local model's EMAIL verdict goes through the same function (internal/redact/model.go), so it is turned away there too, and a real address beside it in the same text is still masked.

RED -> GREEN:
- clean base + only the new test: `ssh user@host.local` -> `ssh {{EMAIL_7kzhpekp}}`, and the mixed sentence masks 2 values where 1 is wanted.
- at the head: `TestLocalHostIsNoEmail` passes - `ssh user@host.local`, `curl http://pi@nas.local:8080/status` and `vnc://me@macbook.local` stay as written while `rina.k@fastmail.com` beside them is still masked.
- the whole `internal/redact` package: 83 tests pass. `go vet` and `go build` for linux, windows and `-tags nogui` pass; `gofmt -l` is empty.

Not checked: no gateway was run; the rule is exercised through `Mask` with Mask personal data on, which is the same call the gateway's masker makes. Other private suffixes (`.lan`, `.home.arpa`) are untouched - no report covers them, and they are one line each if wanted.