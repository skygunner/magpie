#!/bin/bash
set -u
EV=/home/vincenty/Projects/magpie-issues/evidence/r25-3621
WT=/home/vincenty/Projects/magpie-issues/r25-autostart-exec-stable
BASE=19253f67bca2310c32739a6eaac88546b3574dbc
cd "$WT"
V="$EV/verify.log"

echo "== gofmt -l internal/autostart/ (must be empty) ==" | tee "$V"
GOTOOLCHAIN=go1.26.8 gofmt -l internal/autostart/ 2>&1 | tee -a "$V"
echo "gofmt_exit=$?" | tee -a "$V"

echo "== go vet ./... ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 go vet ./... 2>&1 | tee -a "$V"
echo "vet_exit=$?" | tee -a "$V"

echo "== go build ./... ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 go build ./... 2>&1 | tee -a "$V"
echo "build_exit=$?" | tee -a "$V"

echo "== GOOS=linux GOARCH=amd64 go build -tags nogui ./... ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 GOOS=linux GOARCH=amd64 go build -tags nogui ./... 2>&1 | tee -a "$V"
echo "build_nogui_linux_exit=$?" | tee -a "$V"

echo "== GOOS=windows GOARCH=amd64 go build ./... ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 GOOS=windows GOARCH=amd64 go build ./... 2>&1 | tee -a "$V"
echo "build_windows_exit=$?" | tee -a "$V"

echo "== GOOS=darwin GOARCH=amd64 go build ./... (compile-only sanity for the darwin test edits) ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 GOOS=darwin GOARCH=amd64 go build ./internal/autostart/ 2>&1 | tee -a "$V"
echo "build_darwin_autostart_exit=$?" | tee -a "$V"

echo "== GOOS=darwin GOARCH=amd64 go vet ./internal/autostart/ ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 GOOS=darwin GOARCH=amd64 go vet ./internal/autostart/ 2>&1 | tee -a "$V"
echo "vet_darwin_autostart_exit=$?" | tee -a "$V"

echo "== green: new tests at head ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 go test ./internal/autostart/ -run TestSelf -v -count=1 > "$EV/green.log" 2>&1
echo "green_exit=$?" | tee -a "$V"
cat "$EV/green.log" | tee -a "$V"

echo "== affected package, full run ==" | tee -a "$V"
GOTOOLCHAIN=go1.26.8 go test ./internal/autostart/ -count=1 2>&1 | tee -a "$V"
echo "package_exit=$?" | tee -a "$V"

echo "== rev-list count BASE..HEAD (must be 1) ==" | tee -a "$V"
GIT_MASTER=1 git rev-list --count "$BASE..HEAD" 2>&1 | tee -a "$V"

echo "== merge-base --is-ancestor BASE HEAD ==" | tee -a "$V"
GIT_MASTER=1 git merge-base --is-ancestor "$BASE" HEAD
echo "ancestor_exit=$?" | tee -a "$V"

echo "== head ==" | tee -a "$V"
GIT_MASTER=1 git rev-parse HEAD | tee -a "$V"
