import json, pathlib, tarfile
p=pathlib.Path('/home/vincenty/Projects/magpie-issues/evidence/r25-3620')
body=(p/'comment.md').read_text()
(p/'comment-request.json').write_text(json.dumps({'body':body},ensure_ascii=False))
comments=json.loads((p/'comments-before.json').read_text())
print('existing_skygunner_ids=',[c['id'] for c in comments if c['user']['login']=='skygunner'])
with tarfile.open(p/'base.tar') as tar:
    different=[]
    for entry in tar:
        if entry.isfile() and tar.extractfile(entry).read() != (p/'clean-base'/entry.name).read_bytes(): different.append(entry.name)
assert different==['internal/gui/tests/routing-levels.test.cjs'], different
print('clean_base_only_changed_test=',different)
with (p/'verify.log').open('a') as log:
    for name in ['red.log','green.log']:
        log.write('\n# Raw '+name+'\n'+(p/name).read_text())
print('json_request_matches_local=',json.loads((p/'comment-request.json').read_text())['body']==body)
