# /// script
# requires-python = ">=3.12"
# ///
"""Run the issue's checks with isolated homes and record exact subprocess results."""
from __future__ import annotations

import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys
import tempfile

EV = Path(__file__).resolve().parent
WT = Path('/home/vincenty/Projects/magpie-issues/r25-expired-model-not-account-fault')
BASE = '19253f67bca2310c32739a6eaac88546b3574dbc'
TARGET = '^TestZCodeTrialQuotaRestsTheModelAlone$'
ENV = dict(os.environ, GOTOOLCHAIN='go1.26.8', GIT_MASTER='1')
for key in ('GOPATH', 'GOMODCACHE', 'GOCACHE'):
    ENV[key] = subprocess.run(['go', 'env', key], env=ENV, check=True, text=True, capture_output=True).stdout.strip()


def run_check(label: str, args: list[str], cwd: Path, log_name: str, extra: dict[str, str] | None = None) -> dict[str, str | int]:
    with tempfile.TemporaryDirectory(prefix='home-', dir=EV) as home:
        environment = dict(ENV, HOME=home, USERPROFILE=home, XDG_CONFIG_HOME=home + '/config', XDG_CACHE_HOME=home + '/cache')
        environment.update(extra or {})
        with (EV / log_name).open('a', encoding='utf8') as log:
            log.write(f'\n$ GOTOOLCHAIN=go1.26.8 {shlex.join(args)}\ncwd={cwd}\nextra_env={extra or {}}\n')
            log.flush()
            result = subprocess.run(args, cwd=cwd, env=environment, stdout=log, stderr=subprocess.STDOUT, check=False)
            log.write(f'\nreturncode={result.returncode}\n')
        print(f'{label}: returncode={result.returncode}', flush=True)
        return {'label': label, 'command': shlex.join(args), 'cwd': str(cwd), 'returncode': result.returncode, 'log': log_name}


def main() -> None:
    mode = sys.argv[1]
    results: list[dict[str, str | int]] = []
    if mode == 'red':
        scratch = Path(tempfile.mkdtemp(prefix='clean-base-', dir=EV))
        archive = subprocess.Popen(['git', 'archive', BASE], cwd=WT, env=ENV, stdout=subprocess.PIPE)
        assert archive.stdout is not None
        unpack = subprocess.run(['tar', '-x', '-C', str(scratch)], stdin=archive.stdout, check=False)
        archive.stdout.close()
        assert archive.wait() == 0 and unpack.returncode == 0
        tests = ['internal/gateway/zcode_trial_test.go', 'internal/plugin/testdata/fake/index.js']
        for path in tests:
            shutil.copyfile(WT / path, scratch / path)
        (EV / 'red-manifest.json').write_text(json.dumps({'base': BASE, 'scratch': str(scratch), 'onlyCopiedFiles': tests}, indent=2) + '\n')
        results.append(run_check('red', ['go', 'test', './internal/gateway', '-run', TARGET, '-count=1', '-v'], scratch, 'red.log'))
    elif mode == 'green':
        results.append(run_check('targeted', ['go', 'test', '-race', './internal/gateway', '-run', TARGET, '-count=1', '-v'], WT, 'green.log'))
        results.append(run_check('gateway package', ['go', 'test', './internal/gateway', '-count=1', '-timeout=15m'], WT, 'green.log'))
    elif mode == 'baseline-failures':
        scratch = Path(json.loads((EV / 'red-manifest.json').read_text())['scratch'])
        results.append(run_check('baseline existing failure', ['go', 'test', './internal/gateway', '-run', '^TestClaudeClientCacheAbsentUnchanged$', '-count=1', '-v'], scratch, 'pre-existing-red.log'))
        results.append(run_check('head related package', ['go', 'test', './internal/gateway', '-run', 'ZCode|ModelRefused|Quota|Rest', '-count=1'], WT, 'green.log'))
    elif mode == 'verify':
        for label, args, extra in [
            ('vet', ['go', 'vet', './...'], {}),
            ('build', ['go', 'build', './...'], {}),
            ('linux nogui', ['go', 'build', '-tags', 'nogui', './...'], {'GOOS': 'linux', 'GOARCH': 'amd64'}),
            ('windows', ['go', 'build', './...'], {'GOOS': 'windows', 'GOARCH': 'amd64'}),
        ]:
            results.append(run_check(label, args, WT, 'verify.log', extra))
    else:
        raise ValueError(f'unknown mode {mode}')
    (EV / f'{mode}-results.json').write_text(json.dumps(results, indent=2) + '\n')


if __name__ == '__main__':
    main()
