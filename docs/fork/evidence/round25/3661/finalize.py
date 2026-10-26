# /// script
# requires-python = ">=3.12"
# ///
from __future__ import annotations

import hashlib
import importlib.util
import json
import re
import subprocess
from pathlib import Path

EV = Path(__file__).resolve().parent
WT = Path('/home/vincenty/Projects/magpie-issues/r25-expired-model-not-account-fault')
BASE = '19253f67bca2310c32739a6eaac88546b3574dbc'
BRANCH = 'fix/r25-expired-model-not-account-fault'
spec = importlib.util.spec_from_file_location('checks', EV / 'run-checks.py')
assert spec is not None and spec.loader is not None
checks = importlib.util.module_from_spec(spec)
spec.loader.exec_module(checks)


def git(*args: str) -> subprocess.CompletedProcess[str]:
    result = subprocess.run(['git', *args], cwd=WT, env=checks.ENV, text=True, capture_output=True, check=False)
    with (EV / 'verify.log').open('a') as log:
        log.write('\n$ GIT_MASTER=1 git ' + ' '.join(args) + '\n' + result.stdout + result.stderr + f'returncode={result.returncode}\n')
    return result


head_result = git('rev-parse', 'HEAD')
assert head_result.returncode == 0
head = head_result.stdout.strip()
count = git('rev-list', '--count', BASE + '..HEAD')
remote = git('ls-remote', 'origin', 'refs/heads/' + BRANCH)
ancestor = git('merge-base', '--is-ancestor', BASE, 'HEAD')
status = git('status', '--short')
diff_check = git('diff', '--check', BASE, 'HEAD')
assert count.returncode == 0 and count.stdout.strip() == '1'
assert remote.returncode == 0 and remote.stdout.split()[0] == head
assert ancestor.returncode == 0 and status.returncode == 0 and not status.stdout.strip()
assert diff_check.returncode == 0
fmt = subprocess.run(['gofmt', '-l', 'internal/gateway', 'internal/plugin'], cwd=WT, env=checks.ENV, text=True, capture_output=True, check=False)
assert fmt.returncode == 0 and fmt.stdout == ''
with (EV / 'verify.log').open('a') as log:
    log.write('\n$ GOTOOLCHAIN=go1.26.8 gofmt -l internal/gateway internal/plugin\n' + fmt.stdout + fmt.stderr + f'returncode={fmt.returncode}\nempty=true\n')
    log.write('linux nogui build env: GOOS=linux GOARCH=amd64\nwindows build env: GOOS=windows GOARCH=amd64\n')
    log.write('rev-list count == 1: true\nls-remote == local head: true\nmerge-base is-ancestor: true\nworktree clean: true\n')
related = json.loads((EV / 'baseline-failures-results.json').read_text())
with (EV / 'verify.log').open('a') as log:
    log.write('\naffected package tests: ' + related[1]['command'] + '\nreturncode=' + str(related[1]['returncode']) + '\n')
results = json.loads((EV / 'green-results.json').read_text())
verify = json.loads((EV / 'verify-results.json').read_text())
red = json.loads((EV / 'red-results.json').read_text())
assert red[0]['returncode'] == 1 and results[0]['returncode'] == 0
assert related[0]['returncode'] == 1 and related[1]['returncode'] == 0
assert all(result['returncode'] == 0 for result in verify)
files = git('diff', '--name-only', BASE, 'HEAD').stdout.splitlines()
summary = {
    'issue': 3661,
    'title': json.loads((EV / 'issue.json').read_text())['title'],
    'verdict': 'fixed-scoped-with-pre-existing-red',
    'branch': BRANCH, 'base': BASE, 'head': head, 'commitCount': 1,
    'testCommands': [row['command'] for row in red + results + related + verify],
    'testsAdded': ['TestZCodeTrialQuotaRestsTheModelAlone: 7 subtests; 2 exact expired-trial responses, quota/authentication controls, mismatched-model/nontrial/status controls'],
    'filesChanged': files,
    'results': {'red': red, 'green': results, 'related': related[1], 'verify': verify},
    'preExistingRed': [{'test': 'TestClaudeClientCacheAbsentUnchanged', 'failure': 'claude_client_cache_test.go:291: ordinary prompt changed', 'headLog': 'green.log', 'baseLog': 'pre-existing-red.log', 'base': BASE, 'baseReturncode': 1, 'headPackageReturncode': 1}],
    'unverifiedBoundaries': ['No real ZCode account exercised.', 'No change to stale saved-member pruning or plugin model discovery; the full listing/skip part of #3661 remains outside this fix.', 'No native macOS/Windows runtime exercise; Windows was cross-built.', 'Full repository Go tests not run; full gateway package has one clean-base-confirmed existing failure.', 'gopls unavailable; compiler, vet and executable tests used.'],
    'verification': {'gofmtEmpty': True, 'vet': True, 'build': True, 'linuxNogui': True, 'windows': True, 'targetedRace': True, 'relatedTests': True, 'existingFailureBaseConfirmed': True, 'oneCommit': True, 'remoteMatchesHead': True, 'baseIsAncestor': True, 'worktreeClean': True},
}
(EV / 'summary.json').write_text(json.dumps(summary, ensure_ascii=False, indent=2) + '\n')
print(json.dumps({'head': head, 'verification': summary['verification']}, indent=2))
