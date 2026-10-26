#!/bin/bash
set -eu
EV=/home/vincenty/Projects/magpie-issues/evidence/r25-3621

before=$(gh api repos/yetone/magpie/issues/3621/comments --jq '[.[]|select(.user.login=="skygunner")]|length')
echo "skygunner comments before: $before"

id=$(gh api -X POST repos/yetone/magpie/issues/3621/comments -f body=@"$EV/comment.md" --jq .id)
echo "posted comment id: $id"
echo "$id" > "$EV/comment-id.txt"

count=$(gh api repos/yetone/magpie/issues/3621/comments --jq '[.[]|select(.user.login=="skygunner")]|length')
echo "skygunner comments after: $count"

gh api repos/yetone/magpie/issues/comments/"$id" > "$EV/comment-api.json"
echo "readback written"
