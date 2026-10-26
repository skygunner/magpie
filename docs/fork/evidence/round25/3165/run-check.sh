#!/usr/bin/env bash
set -uo pipefail
cd /home/vincenty/Projects/magpie-issues/r25-3165-reasoning-clear-test || exit 1
log=$1
shift
export GOPATH=/home/vincenty/go
export GOMODCACHE=/home/vincenty/go/pkg/mod
export GOCACHE=/home/vincenty/.cache/go-build
export GOTOOLCHAIN=local
export GOMAXPROCS=4
sandbox=$(mktemp -d /tmp/magpie-3165-home.XXXXXXXX) || exit 1
trap 'chmod -R u+w "$sandbox"; rm -rf "$sandbox"' EXIT
export HOME=$sandbox USERPROFILE=$sandbox
export XDG_CONFIG_HOME=$sandbox/.config XDG_CACHE_HOME=$sandbox/.cache XDG_DATA_HOME=$sandbox/.local/share
{
  printf 'UTC: '; date -u +%FT%TZ
  printf 'HEAD: '; GIT_MASTER=1 git rev-parse HEAD
  go version
  printf 'COMMAND:'; printf ' %q' "$@"; printf '\n'
  "$@"
  result=$?
  printf 'EXIT_CODE=%s\n' "$result"
  exit "$result"
} 2>&1 | tee "$log"
exit "${PIPESTATUS[0]}"
