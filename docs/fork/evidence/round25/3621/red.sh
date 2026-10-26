#!/bin/bash
set -u
EV=/home/vincenty/Projects/magpie-issues/evidence/r25-3621
cd "$EV/red-base"
GOTOOLCHAIN=go1.26.8 go test ./internal/autostart/ -run TestSelf -v -count=1 > "$EV/red.log" 2>&1
code=$?
echo "exit_code=$code" >> "$EV/red.log"
tail -2 "$EV/red.log"
