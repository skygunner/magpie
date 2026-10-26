#!/bin/bash
set -u
EV=/home/vincenty/Projects/magpie-issues/evidence/r25-3621
WT=/home/vincenty/Projects/magpie-issues/r25-autostart-exec-stable
BASE=19253f67bca2310c32739a6eaac88546b3574dbc
cd "$WT"
GIT_MASTER=1 git diff "$BASE..HEAD" > "$EV/diff.patch"
wc -l "$EV/diff.patch"
GIT_MASTER=1 git push origin fix/r25-autostart-exec-stable
code=$?
echo "== push_exit=$code ==" >> "$EV/verify.log"
echo "== ls-remote read-back ==" >> "$EV/verify.log"
GIT_MASTER=1 git ls-remote origin refs/heads/fix/r25-autostart-exec-stable >> "$EV/verify.log" 2>&1
tail -6 "$EV/verify.log"
