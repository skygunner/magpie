# /// script
# requires-python = ">=3.12"
# ///
from __future__ import annotations

import hashlib
import json
import re
import subprocess
from pathlib import Path

EV = Path(__file__).resolve().parent
ENDPOINT = 'repos/yetone/magpie/issues/3661/comments'
text = (EV / 'comment.md').read_text()
issue = json.loads((EV / 'issue.json').read_text())
source = issue['title'] + '\n' + issue['body']
cjk = len(re.findall(r'[\u3400-\u9fff]', source))
latin = len(re.findall(r'[A-Za-z]', source))
assert cjk / (cjk + latin) >= 0.15
response = subprocess.run(['gh', 'api', '--paginate', ENDPOINT], text=True, capture_output=True, check=True)
comments = json.loads(response.stdout)
ours = [comment for comment in comments if comment['user']['login'] == 'skygunner']
if ours:
    assert len(ours) == 1 and ours[0]['body'] == text, 'Existing skygunner comment differs; do not create a duplicate.'
    comment = ours[0]
else:
    posted = subprocess.run(['gh', 'api', '-X', 'POST', ENDPOINT, '--input', '-'], input=json.dumps({'body': text}), text=True, capture_output=True, check=True)
    (EV / 'comment-post-response.json').write_text(posted.stdout)
    comment = json.loads(posted.stdout)
readback = subprocess.run(['gh', 'api', f"repos/yetone/magpie/issues/comments/{comment['id']}"], text=True, capture_output=True, check=True)
remote = json.loads(readback.stdout)
local_hash = hashlib.sha256(text.encode()).hexdigest()
remote_hash = hashlib.sha256(remote['body'].encode()).hexdigest()
assert remote['user']['login'] == 'skygunner'
assert remote['created_at'].endswith('Z') and local_hash == remote_hash
all_comments = json.loads(subprocess.run(['gh', 'api', '--paginate', ENDPOINT], text=True, capture_output=True, check=True).stdout)
count = sum(item['user']['login'] == 'skygunner' for item in all_comments)
assert count == 1
record = {'id': remote['id'], 'url': remote['html_url'], 'author': remote['user']['login'], 'created_at': remote['created_at'], 'body': remote['body'], 'localSha256': local_hash, 'remoteSha256': remote_hash, 'equal': local_hash == remote_hash, 'skygunnerCommentCount': count}
(EV / 'comment-readback.json').write_text(json.dumps(record, ensure_ascii=False, indent=2) + '\n')
summary = json.loads((EV / 'summary.json').read_text())
summary.update({'language': {'cjk': cjk, 'latin': latin, 'share': cjk / (cjk + latin)}, 'commentId': remote['id'], 'commentUrl': remote['html_url'], 'commentReadbackEqual': True, 'skygunnerCommentCount': count})
summary['verification'].update({'oneComment': True, 'commentAuthor': True, 'commentUtc': True, 'commentBodySha256': True})
(EV / 'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
with (EV / 'verify.log').open('a') as log:
    log.write(f"\ncomment id: {remote['id']}\nexactly one skygunner comment: true\nauthor skygunner: true\ncreated_at UTC Z: true\ncomment body sha256 equal: true\n")
print(json.dumps({'id': remote['id'], 'url': remote['html_url'], 'equal': True, 'skygunnerCommentCount': count}, indent=2))
