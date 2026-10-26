#!/bin/bash
set -eu
EV=/home/vincenty/Projects/magpie-issues/evidence/r25-3621

gh api -X DELETE repos/yetone/magpie/issues/comments/6408556560
echo "deleted wrong comment"

id=$(gh api -X POST repos/yetone/magpie/issues/3621/comments -F body=@"$EV/comment.md" --jq .id)
echo "posted comment id: $id"
echo "$id" > "$EV/comment-id.txt"

count=$(gh api repos/yetone/magpie/issues/3621/comments --jq '[.[]|select(.user.login=="skygunner")]|length')
echo "skygunner comments after: $count"

gh api repos/yetone/magpie/issues/comments/"$id" > "$EV/comment-api.json"
echo "readback written"
