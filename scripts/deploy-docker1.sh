#!/bin/bash
# Self-contained production switch for docker1; no companion file is needed.
# Transfer the image first, then launch this entry ONCE on docker1 with nohup.
# Usage: deploy-docker1.sh magpie:<major>.<minor>.<patch>-p<N> [optional ID assertions]
# The Python transaction holds one host flock through baseline, stop/backup,
# create/start, acceptance, 2760-second observation and rollback. It never publishes.
set +x
set -uo pipefail
umask 077
script_path=$(readlink -f -- "${BASH_SOURCE[0]}") || exit 1
if [[ ${EUID} -ne 0 ]]; then
  exec sudo -n -- /bin/bash "$script_path" "$@"
fi
exec /usr/bin/python3 -I -B - "$script_path" "$@" <<'MAGPIE_EMBEDDED_PYTHON'
# MAGPIE_PYTHON_BEGIN
#!/usr/bin/python3
"""One docker1 deployment transaction, entered via deploy-docker1.sh.
No publication command. Requires an already transferred magpie:<version>-p<N> image.
The three ID arguments are optional assertions; IDs are bound under the lock.
A round directory is never reused, including after failure. All output is a fixed
allow-list of events/counts/IDs; command stderr, HTTP bodies and secrets stay private.
"""
import argparse
import collections
import datetime as dt
import fcntl
import http.cookiejar
import json
import os
from pathlib import Path
import re
import signal
import stat
import subprocess
import sys
import time
import urllib.error
import urllib.parse
import urllib.request
import uuid

# The self-contained shell entry supplies its source path before the actual arguments.
if __name__ == '__main__' and sys.argv[0] in ('-', '-c'):
    __file__ = sys.argv.pop(1)

UTC = dt.timezone.utc
DOCKER = ['/usr/bin/docker', '--host', 'unix:///var/run/docker.sock']
ROOT = Path('/var/lib/magpie-deploy')
POLL = 3.0
OBSERVE = 2760.0
COMMAND_TIMEOUT = 20
HTTP_TIMEOUT = 90
ARCHIVE_TIMEOUT = 300


class Failure(Exception):
    pass


class Cancelled(BaseException):
    pass


def require(condition, code):
    if not condition:
        raise Failure(code)


def stamp():
    return dt.datetime.now(UTC).isoformat()


def emit(event, *, strict=False, **safe_values):
    try:
        print(json.dumps(dict(event=event, at=stamp(), **safe_values)), flush=True)
    except OSError:
        if strict:
            raise
        pass  # A closed log pipe must not disable the rollback path.


def write_private(path, data):
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_TRUNC | os.O_NOFOLLOW, 0o600)
    with os.fdopen(fd, 'wb') as stream:
        os.fchmod(stream.fileno(), 0o600)
        stream.write(data)
        stream.flush()
        os.fsync(stream.fileno())


def private_directory(path):
    try:
        path.mkdir(mode=0o700)
    except FileExistsError:
        pass
    s = path.lstat()
    require(stat.S_ISDIR(s.st_mode) and s.st_uid == os.geteuid()
            and stat.S_IMODE(s.st_mode) == 0o700, 'unsafe_state_directory')


class System:
    def __init__(self):
        self.source_path = str(Path(__file__).resolve())
        self.source = Path(self.source_path).read_text()
        if self.source_path.endswith('.sh'):
            self.source = self.source.split('\n# MAGPIE_PYTHON_BEGIN\n', 1)[1].split('\n# MAGPIE_PYTHON_END\n', 1)[0]

    mono = staticmethod(time.monotonic)
    sleep = staticmethod(time.sleep)
    now = staticmethod(stamp)

    def run(self, argv, *, timeout=COMMAND_TIMEOUT, data=None, watch=None, combine=False, quick_kill=False):
        """Absolute wall-clock deadline, including a slow HTTP response body.
        Never expose CalledProcessError, argv, stderr or stdout on failure.
        Worker input (including credentials) goes through a pipe, not argv/env.
        """
        deadline = self.mono() + timeout
        process = subprocess.Popen(argv, stdin=subprocess.PIPE if data is not None else subprocess.DEVNULL,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE, start_new_session=True,
                                   env={'PATH': '/usr/bin:/bin', 'HOME': '/root', 'LC_ALL': 'C'})
        first = True
        try:
            while True:
                left = deadline - self.mono() - (0.05 if quick_kill else 0)
                require(left > 0, 'operation_timeout')
                try:
                    out, err = process.communicate(input=data if first else None, timeout=min(1.0, left))
                    break
                except subprocess.TimeoutExpired:
                    first = False
                    if watch:
                        watch()
            require(process.returncode == 0, 'command_failed')
            return out + err if combine else out
        finally:
            if process.poll() is None:
                try:
                    os.killpg(process.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                if quick_kill:
                    # Reap only inside the reserved part of the same 2s budget; never add 5s.
                    for pipe in (process.stdin, process.stdout, process.stderr):
                        if pipe:
                            pipe.close()
                    try:
                        process.wait(timeout=max(0, deadline - self.mono()))
                    except subprocess.TimeoutExpired:
                        pass
                else:
                    process.communicate(timeout=5)

    def worker(self, action, payload, *, watch=None, timeout=HTTP_TIMEOUT):
        # Only static reviewed code and a path enter argv; credential payload uses stdin.
        raw = self.run([sys.executable, '-I', '-B', '-c', self.source, self.source_path, '--worker', action],
                       data=json.dumps(payload).encode(), timeout=timeout, watch=watch,
                       quick_kill=action == 'candidate-snapshot')
        result = json.loads(raw)
        require(isinstance(result, dict) and result.get('ok') is True, 'worker_failed')
        return result['value']


def parse_time(value):
    value = dt.datetime.fromisoformat(value.replace('Z', '+00:00'))
    require(value.tzinfo is not None, 'usage_time_without_timezone')
    return value


def read_usage(payload):
    """Read a fixed file-size snapshot. No skipped bad/partial lines, inode changes or truncation."""
    path = Path(payload['path'])
    counts = collections.Counter()
    protocol = 0
    with path.open('rb') as stream:
        s = os.fstat(stream.fileno())
        identity = [s.st_dev, s.st_ino]
        require(stat.S_ISREG(s.st_mode), 'usage_not_regular')
        if payload.get('identity') is not None:
            require(identity == payload['identity'], 'usage_replaced')
        offset = payload.get('offset', 0)
        require(0 <= offset <= s.st_size, 'usage_truncated')
        stream.seek(offset)
        start, stop = parse_time(payload['start']), parse_time(payload['stop'])
        while stream.tell() < s.st_size:
            line = stream.readline(min(1024 * 1024, s.st_size - stream.tell()))
            require(line.endswith(b'\n'), 'usage_partial_or_oversize_line')
            row = json.loads(line)
            require(isinstance(row, dict), 'usage_bad_record')
            at = parse_time(row['t'])
            status = row['status']
            require(type(status) is int and 0 <= status <= 599, 'usage_bad_status')
            require(isinstance(row.get('err_type', ''), str), 'usage_bad_error_type')
            # The size snapshot may include writes newer than the caller's cutoff.
            # Count them now: advancing the offset while skipping them loses them forever.
            if at >= start:
                if status:
                    counts[str(status)] += 1
                protocol += row.get('err_type') == 'protocol_error'
        end = os.fstat(stream.fileno())
        current = path.stat()
        require([current.st_dev, current.st_ino] == identity and end.st_size >= s.st_size,
                'usage_changed_during_read')
    return dict(identity=identity, offset=s.st_size, counts=dict(counts), protocol=protocol)


def copy_candidate_configs(payload):
    destination = Path(payload['destination'])
    private_directory(destination)
    copied = []
    files = ('providers.json', 'logins.json', 'plugin-auth.json', 'plugin-auth.json.bak',
             'migrations.json', 'plugins.yml', 'settings.json', 'caller-keys.json')
    for name in files:
        try:
            fd = os.open(Path(payload['volume']) / 'magpie' / name, os.O_RDONLY | os.O_NOFOLLOW)
        except FileNotFoundError:
            if name in ('providers.json', 'logins.json'):
                raise
            continue
        with os.fdopen(fd, 'rb') as source:
            require(stat.S_ISREG(os.fstat(source.fileno()).st_mode), 'candidate_config_not_regular')
            target = destination / (name + '.partial')
            out = os.open(target, os.O_WRONLY | os.O_CREAT | os.O_EXCL | os.O_NOFOLLOW, 0o600)
            with os.fdopen(out, 'wb') as sink:
                while True:
                    chunk = source.read(65536)
                    if not chunk:
                        break
                    sink.write(chunk)
                sink.flush()
                os.fsync(sink.fileno())
            target.replace(destination / name)
            copied.append(name)
    write_private(destination / 'complete.json', json.dumps({'complete': True, 'files': copied}).encode())
    return {'files': copied}


class LocalRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        require(urllib.parse.urlsplit(newurl).netloc == urllib.parse.urlsplit(req.full_url).netloc
                and urllib.parse.urlsplit(newurl).scheme == 'http', 'cross_origin_redirect')
        require(req.data is None, 'inference_redirect')
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def http_work(action, payload):
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), LocalRedirect(),
                                         urllib.request.HTTPCookieProcessor(http.cookiejar.CookieJar()))

    def request(url, key=None, body=None, raw=False):
        headers = {'Authorization': 'Bearer ' + key} if key else {}
        encoded = None
        if body is not None:
            headers['Content-Type'] = 'application/json'
            encoded = json.dumps(body).encode()
        req = urllib.request.Request(url, data=encoded, headers=headers)
        with opener.open(req, timeout=HTTP_TIMEOUT) as response:
            data = response.read(32 * 1024 * 1024 + 1)
            require(len(data) <= 32 * 1024 * 1024 and response.status == 200, 'http_bad_response')
            return data if raw else json.loads(data)

    gateway, console = 'http://127.0.0.1:3425', 'http://127.0.0.1:3430'
    if action == 'inference':
        # Exactly one POST; neither redirects nor application retries are allowed.
        data = request(gateway + '/v1/chat/completions', key=payload['key'], body={
            'model': 'group/fast-cheap', 'messages': [{'role': 'user', 'content': 'Reply OK'}],
            # The cap includes reasoning: 16 tokens ended before any answer.
            'max_tokens': 256, 'stream': False})
        choices = data.get('choices')
        require(isinstance(choices, list) and choices and isinstance(choices[0], dict)
                and isinstance(choices[0].get('message', {}).get('content'), str)
                and choices[0]['message']['content'].strip(), 'inference_no_content')
        return {'status': 200}
    require(action == 'snapshot', 'invalid_worker_action')
    try:
        request(console + '/api/providers')
    except urllib.error.HTTPError as error:
        status = error.code
        error.close()
        require(status == 401, 'unauthenticated_status')
    else:
        raise Failure('unauthenticated_access_allowed')
    request(console + '/?' + urllib.parse.urlencode({'k': payload['web_key']}), raw=True)
    providers = request(console + '/api/providers')
    groups = request(console + '/api/groups')
    try:
        cooldown = request(console + '/api/cooldown')['effective']
        require(isinstance(cooldown, dict), 'cooldown_schema')
    except urllib.error.HTTPError as error:
        status = error.code
        error.close()
        require(status == 404, 'cooldown_http_error')
        cooldown = None  # The upstream-first build has no fork cooldown editor.

    def ids(rows):
        require(isinstance(rows, list), 'snapshot_schema')
        result = [r['id'] for r in rows]
        require(all(isinstance(v, str) and v for v in result) and len(result) == len(set(result)),
                'snapshot_ids')
        return sorted(result)

    for index, key in enumerate(payload['keys']):
        try:
            models = request(gateway + '/v1/models', key=key)
        except urllib.error.HTTPError as error:
            status = error.code
            error.close()
            require(status in (401, 403), 'model_list_http_error')
            continue
        model_ids = ids(models['data'])
        if 'group/fast-cheap' not in model_ids:
            continue  # A models 200 can still be a key without this inference permission.
        return {'key_index': index, 'snapshot': {'models': model_ids,
                'providers': ids(providers['providers']), 'groups': ids(groups['groups']),
                'cooldown': cooldown}}
    raise Failure('no_enabled_caller_key')


class Deployment:
    def __init__(self, args, system=None, root=ROOT):
        self.a, self.io, self.root = args, system or System(), root
        self.version = args.image.removeprefix('magpie:')
        self.directory = root / self.version
        self.token = uuid.uuid4().hex
        self.candidate = 'magpie-candidate-' + self.token
        self.old_name = 'magpie-pre-' + self.version + '-' + self.token
        self.old = self.new_id = self.volume = None
        self.armed = False
        self.claimed = False
        self.started = self.cursor = None
        self.usage_position = None
        self.counts = collections.Counter()
        self.protocol = self.log_fives = self.log_total = 0
        self.baseline_ratio = 0.0
        self.next_check = 0.0

    def docker(self, *args, timeout=COMMAND_TIMEOUT, combine=False):
        return self.io.run(DOCKER + list(args), timeout=timeout, combine=combine)

    def inspect(self, ref):
        result = json.loads(self.docker('inspect', '--type', 'container', ref))
        require(isinstance(result, list) and len(result) == 1, 'inspect_schema')
        return result[0]

    def named(self, name):
        ids = self.docker('ps', '-a', '--no-trunc', '--filter', 'name=^/' + name + '$',
                          '--format', '{{.ID}}').decode().split()
        require(len(ids) <= 1, 'ambiguous_container_name')
        return ids[0] if ids else None

    def old_checked(self):
        info = self.inspect(self.a.old_container_id)
        require(info['Id'] == self.a.old_container_id and info['Image'] == self.a.old_image_id,
                'old_identity_changed')
        require(info['Name'] in ('/magpie', '/' + self.old_name), 'old_name_changed')
        require(len(info['Mounts']) == 1 and info['Mounts'][0]['Name'] == 'magpie-config'
                and info['Mounts'][0]['Destination'] == '/config' and info['Mounts'][0]['RW'],
                'unsupported_volume_layout')
        return info

    def volume_idle(self, permitted):
        ids = set(self.docker('ps', '--no-trunc', '--filter', 'volume=magpie-config',
                              '--format', '{{.ID}}').decode().split())
        require(ids <= set(permitted), 'another_volume_writer')

    def usage(self, start, *, reset=False):
        payload = dict(path=str(self.volume / 'magpie/usage.jsonl'), start=start,
                       stop=self.io.now(), offset=0, identity=None)
        if not reset and self.usage_position:
            payload.update(self.usage_position)
        result = self.io.worker('usage', payload, timeout=COMMAND_TIMEOUT)
        self.usage_position = {k: result[k] for k in ('identity', 'offset')}
        return result

    def snapshot(self, watch=None):
        # Baseline may find one usable key. Every later comparison uses exactly that key.
        keys = [self.key] if hasattr(self, 'key') else self.keys
        return self.io.worker('snapshot', {'web_key': self.web_key, 'keys': keys}, watch=watch)

    def preflight(self):
        require(not (self.root / 'BLOCKED').exists(), 'deployment_blocked')
        current = self.named('magpie')
        require(current and re.fullmatch(r'[0-9a-f]{64}', current), 'current_container_missing')
        require(not self.a.old_container_id or current == self.a.old_container_id, 'unexpected_current_container')
        observed = self.inspect(current)
        require(not self.a.old_image_id or observed['Image'] == self.a.old_image_id, 'unexpected_old_image')
        self.a.old_container_id, self.a.old_image_id = current, observed['Image']
        self.old = self.old_checked()
        state, config, host = self.old['State'], self.old['Config'], self.old['HostConfig']
        require(state['Status'] == 'running' and state.get('Health', {}).get('Status') == 'healthy'
                and not state.get('OOMKilled'), 'old_not_healthy')
        image = json.loads(self.docker('image', 'inspect', self.a.image))[0]
        require(not self.a.target_image_id or image['Id'] == self.a.target_image_id, 'target_image_missing_or_changed')
        require(re.fullmatch(r'sha256:[0-9a-f]{64}', image['Id']), 'invalid_target_image_id')
        self.a.target_image_id = image['Id']
        require(self.a.old_image_id != self.a.target_image_id, 'target_already_running')
        require(image['Config'].get('Healthcheck'), 'target_has_no_healthcheck')
        # Refuse configurations the small run template does not reproduce.
        require(config['User'] == 'nonroot' and config['Cmd'] ==
                ['web', '--addr', '0.0.0.0:3430', '--no-open'], 'unsupported_old_command')
        require(host['RestartPolicy']['Name'] == 'unless-stopped'
                and host['NetworkMode'] in ('default', 'bridge')
                and not host.get('Privileged') and not host.get('CapAdd')
                and not host.get('ExtraHosts') and not host.get('SecurityOpt'), 'unsupported_old_host_config')
        bindings = host['PortBindings']
        require(set(bindings) == {'3425/tcp', '3430/tcp'}, 'unsupported_old_ports')
        for port in ('3425', '3430'):
            require(len(bindings[port + '/tcp']) == 1 and bindings[port + '/tcp'][0]['HostPort'] == port
                    and bindings[port + '/tcp'][0]['HostIp'] in ('', '0.0.0.0'), 'unsupported_old_binding')
        volume = json.loads(self.docker('volume', 'inspect', 'magpie-config'))[0]
        require(volume['Name'] == 'magpie-config' and volume.get('Driver') == 'local', 'unsupported_volume')
        self.volume = Path(volume['Mountpoint'])
        require(self.volume.is_absolute() and self.volume.is_dir() and not self.volume.is_symlink(),
                'invalid_volume_path')
        self.volume_idle([self.a.old_container_id])
        env = config['Env']
        require(isinstance(env, list) and all(isinstance(v, str) and re.fullmatch(r'[A-Za-z_][A-Za-z0-9_]*=[^\x00\r\n]*', v)
                                             for v in env), 'env_file_unrepresentable')
        values = dict(v.split('=', 1) for v in env)
        require(values.get('MAGPIE_WEB_KEY'), 'missing_web_key')
        self.web_key = values['MAGPIE_WEB_KEY']
        entries = json.loads((self.volume / 'magpie/caller-keys.json').read_bytes())
        require(isinstance(entries, list), 'caller_keys_schema')
        self.keys = [v['secret'] for v in entries if isinstance(v, dict) and isinstance(v.get('secret'), str)
                     and v['secret'] and not v.get('disabled', False)]
        require(self.keys, 'no_caller_keys')
        result = self.snapshot()
        self.prior, self.key = result['snapshot'], self.keys[result['key_index']]
        self.expected = self.prior
        if getattr(self.a, 'expected_snapshot', None):
            review = json.loads(Path(self.a.expected_snapshot).read_bytes())
            require(review['target_image_id'] == self.a.target_image_id
                    and review['baseline'] == self.prior, 'snapshot_review_stale')
            self.expected = review['candidate']
            require(isinstance(self.expected, dict) and set(self.expected) == set(self.prior),
                    'snapshot_review_schema')
            write_private(self.directory / 'review.json', json.dumps(review).encode())
        then = (parse_time(self.io.now()) - dt.timedelta(seconds=OBSERVE)).isoformat()
        usage = self.usage(then, reset=True)
        self.baseline_ratio = self.ratio(usage['counts'])
        write_private(self.directory / 'container.json', json.dumps(self.old).encode())
        write_private(self.directory / 'runtime.env', ('\n'.join(env) + '\n').encode())
        write_private(self.directory / 'baseline.json', json.dumps(dict(snapshot=self.prior,
                      captured=self.io.now(), usage=usage, old_id=self.a.old_container_id,
                      target_image_id=self.a.target_image_id)).encode())

    @staticmethod
    def ratio(counts):
        return sum(v for k, v in counts.items() if 500 <= int(k) <= 599) / max(1, sum(counts.values()))

    def owned_new(self):
        # A timed-out CREATE can only leave a stopped candidate with our unique label.
        if not self.new_id:
            ids = self.docker('ps', '-a', '--no-trunc', '--filter', 'label=magpie.deploy=' + self.token,
                              '--format', '{{.ID}}').decode().split()
            require(len(ids) <= 1, 'ambiguous_candidate')
            if not ids:
                return None
            self.new_id = ids[0]
        else:
            ids = self.docker('ps', '-a', '--no-trunc', '--filter', 'id=' + self.new_id,
                              '--format', '{{.ID}}').decode().split()
            require(ids in ([], [self.new_id]), 'candidate_id_lookup_mismatch')
            if not ids:
                return None  # Confirmed removal is recoverable; a failed read is not absence.
        info = self.inspect(self.new_id)
        require(info['Id'] == self.new_id and info['Image'] == self.a.target_image_id
                and info['Config'].get('Labels', {}).get('magpie.deploy') == self.token
                and info['Name'] in ('/' + self.candidate, '/magpie'), 'new_identity_changed')
        return info

    def switch(self):
        require(self.named('magpie') == self.a.old_container_id, 'current_changed_before_stop')
        self.old_checked()
        self.armed = True  # Set BEFORE the first stop, including a timed-out or interrupted stop.
        self.docker('stop', '--time', '10', self.a.old_container_id, timeout=30)
        require(self.old_checked()['State']['Status'] == 'exited', 'old_stop_not_confirmed')
        self.volume_idle([])
        self.usage(self.io.now())  # Validate the old log tail and fix the new run's starting offset.
        archive = self.directory / 'volume.tgz'
        self.io.run(['/usr/bin/tar', '--numeric-owner', '-C', str(self.volume), '-czf', str(archive), '.'],
                    timeout=ARCHIVE_TIMEOUT)
        os.chmod(archive, 0o600)
        self.io.run(['/usr/bin/tar', '-tzf', str(archive)], timeout=ARCHIVE_TIMEOUT)
        self.docker('rename', self.a.old_container_id, self.old_name)
        # CREATE + START is Docker run split locally to bind the immutable ID BEFORE it can write.
        self.docker('create', '--pull=never', '--name', self.candidate,
                    '--label', 'magpie.deploy=' + self.token, '--restart', 'unless-stopped', '--user', 'nonroot',
                    '-p', '3425:3425', '-p', '3430:3430', '-v', 'magpie-config:/config:z',
                    '--env-file', str(self.directory / 'runtime.env'), self.a.target_image_id,
                    'web', '--addr', '0.0.0.0:3430', '--no-open')
        self.owned_new()
        require(self.new_id is not None, 'candidate_missing')
        require(self.named('magpie') is None, 'name_taken_before_start')
        self.docker('rename', self.new_id, 'magpie')
        require(self.named('magpie') == self.new_id, 'new_name_mismatch')
        self.docker('start', self.new_id)
        info = self.owned_new()
        self.started = info['State']['StartedAt']
        parse_time(self.started)
        self.cursor = self.started
        end = self.io.mono() + 180
        while True:
            self.check(starting=True)
            if self.owned_new()['State'].get('Health', {}).get('Status') == 'healthy':
                break
            require(self.io.mono() < end, 'startup_timeout')
            self.io.sleep(POLL)

    def check(self, *, starting=False, final=False):
        self.volume_idle([self.new_id])
        require(self.named('magpie') == self.new_id, 'current_container_replaced')
        info = self.owned_new()
        state = info['State']
        health = state.get('Health', {}).get('Status')
        require(state['Status'] == 'running' and not state.get('OOMKilled') and info['RestartCount'] == 0
                and state['StartedAt'] == self.started, 'container_state_anomaly')
        require(health == 'healthy' or starting and health == 'starting', 'container_health_anomaly')
        now = self.io.now()
        require(parse_time(now) >= parse_time(self.cursor), 'wall_clock_moved_backwards')
        logs = self.docker('logs', '--timestamps', '--since', self.cursor, '--until', now,
                           self.new_id, combine=True).decode('utf-8', errors='strict')
        require(not re.search(r'(?i)\b(?:panic|fatal)\b|protocol_error', logs), 'log_anomaly')
        statuses = re.findall(r'(?:status=|"status"\s*:\s*)([1-5]\d\d)\b', logs)
        self.log_fives += sum(int(status) >= 500 for status in statuses)
        self.log_total += len(statuses)
        events = self.docker('events', '--since', self.cursor, '--until', now, '--filter',
                             'container=' + self.new_id, '--format', '{{json .}}').decode()
        for line in events.splitlines():
            event = json.loads(line)
            action = event.get('Action', event.get('status', ''))
            require(action not in ('die', 'oom', 'restart', 'stop', 'kill', 'destroy', 'pause',
                                   'health_status: unhealthy'), 'docker_event_anomaly')
        usage = self.usage(self.started)
        self.counts.update(usage['counts'])
        self.protocol += usage['protocol']
        require(self.protocol == 0, 'usage_protocol_error')
        fives = sum(v for k, v in self.counts.items() if 500 <= int(k) <= 599)
        threshold = min(.50, max(.10, self.baseline_ratio * 2))
        require(not (fives >= 5 and self.ratio(self.counts) > threshold), '5xx_surge')
        # Logs remain an independent source when usage is sparse. Use the larger
        # observed traffic count, not a single successful usage row as an exemption.
        log_ratio = self.log_fives / max(1, self.log_total, sum(self.counts.values()))
        require(not (self.log_fives >= 5 and log_ratio > threshold), '5xx_log_surge')
        self.cursor = now
        self.next_check = self.io.mono() + POLL
        # IDs and health are checked again AFTER potentially slow log/file reads.
        end = self.owned_new()
        self.volume_idle([self.new_id])
        require(self.named('magpie') == self.new_id and end['State']['Status'] == 'running'
                and not end['State'].get('OOMKilled') and end['RestartCount'] == 0
                and end['State']['StartedAt'] == self.started
                and (end['State'].get('Health', {}).get('Status') == 'healthy'
                     or starting and end['State'].get('Health', {}).get('Status') == 'starting'),
                'final_state_anomaly')
        if final:
            require(health == 'healthy', 'final_health_not_healthy')

    def watch(self):
        if self.io.mono() >= self.next_check:
            self.check()

    def accept_and_observe(self):
        require(self.docker('exec', self.new_id, 'magpie', 'version').decode().strip() == 'magpie ' + self.version,
                'binary_version_mismatch')
        self.compare_snapshot('acceptance')
        self.io.worker('inference', {'key': self.key}, watch=self.watch)
        emit('REAL_INFERENCE', phase='acceptance', status=200)
        self.check()
        accepted = self.io.mono()
        emit('ACCEPTED', version=self.version)
        probe_times = (OBSERVE / 10, OBSERVE * .6)
        probes = sample = 0
        while self.io.mono() - accepted < OBSERVE:
            self.io.sleep(min(POLL, max(0, OBSERVE - (self.io.mono() - accepted))))
            self.check()
            if self.io.mono() - accepted >= sample * (OBSERVE / 10):
                emit('SAMPLE', sample=sample, seconds=round(self.io.mono()-accepted, 1),
                     health='healthy', restarts=0, usage_counts=dict(self.counts),
                     log_5xx=self.log_fives, protocol_error=self.protocol)
                sample += 1
            if probes < len(probe_times) and self.io.mono() - accepted >= probe_times[probes]:
                self.io.worker('inference', {'key': self.key}, watch=self.watch)
                probes += 1
                emit('REAL_INFERENCE', phase='observation', probe=probes, status=200)
                self.check()
        require(probes == len(probe_times), 'observation_probes_missing')
        self.compare_snapshot('final')
        self.check(final=True)
        require(self.io.mono() - accepted >= OBSERVE, 'observation_too_short')
        return round(self.io.mono() - accepted, 3)

    def compare_snapshot(self, phase):
        actual = self.snapshot(self.watch)['snapshot']
        write_private(self.directory / (phase + '.json'), json.dumps(actual).encode())
        delta = {}
        for key in self.expected.keys() | actual.keys():
            before, after = self.expected.get(key), actual.get(key)
            if before == after:
                continue
            delta[key] = ({'added': sorted(set(after) - set(before)),
                           'removed': sorted(set(before) - set(after))}
                          if isinstance(before, list) and isinstance(after, list)
                          else {'before': before, 'after': after})
        write_private(self.directory / (phase + '-diff.json'), json.dumps(delta).encode())
        require(not delta, phase + '_snapshot_changed')

    def rollback(self):
        old = self.old_checked()  # Validate BOTH identities before any destructive recovery.
        current = self.named('magpie')
        require(current in (None, self.a.old_container_id, self.new_id), 'foreign_current_container')
        new = self.owned_new()
        if new:
            # A failed/timed-out CLI may already have removed the container. Reconcile
            # its exact ID before deciding; at most one safe cleanup retry, never redeploy.
            for _ in range(2):
                if self.owned_new() is None:
                    break
                require(self.named('magpie') in (None, self.a.old_container_id, self.new_id),
                        'foreign_current_container')
                try:
                    self.docker('rm', '-f', self.new_id, timeout=30)
                except Exception:
                    emit('CANDIDATE_REMOVE_REPORTED_ERROR')
            require(self.owned_new() is None, 'new_remove_not_confirmed')
            require(self.named('magpie') in (None, self.a.old_container_id), 'new_remove_not_confirmed')
        old = self.old_checked()
        if new is not None or self.started is not None:
            try:
                self.io.worker('candidate-snapshot', {'volume': str(self.volume),
                               'destination': str(self.directory / 'candidate-config')}, timeout=2)
                emit('CANDIDATE_CONFIG_SNAPSHOT_COMPLETE')
            except BaseException:
                emit('CANDIDATE_CONFIG_SNAPSHOT_INCOMPLETE')
        # Preserve all shared-volume writes made during the observation. The archive
        # is recovery material for a human; restoring it here would discard user data.
        # An incompatible migration therefore yields FAILED_ROLLBACK, never a volume overwrite.
        current = self.named('magpie')
        require(current in (None, self.a.old_container_id), 'foreign_current_container')
        if current is None:
            self.docker('rename', self.a.old_container_id, 'magpie')
        old = self.old_checked()
        if old['State']['Status'] != 'running':
            self.docker('start', self.a.old_container_id)
        deadline = self.io.mono() + 180
        while True:
            old = self.old_checked()
            require(self.named('magpie') == self.a.old_container_id, 'rollback_name_mismatch')
            if old['State']['Status'] == 'running' and not old['State'].get('OOMKilled') \
                    and old['State'].get('Health', {}).get('Status') == 'healthy':
                # Authentication and model/catalog/cooldown parity on the restored service too.
                require(self.snapshot()['snapshot'] == self.prior, 'rollback_api_mismatch')
                old = self.old_checked()
                require(self.named('magpie') == self.a.old_container_id and old['State']['Status'] == 'running'
                        and old['State'].get('Health', {}).get('Status') == 'healthy'
                        and not old['State'].get('OOMKilled'), 'rollback_final_state')
                return
            require(self.io.mono() < deadline, 'rollback_health_timeout')
            self.io.sleep(POLL)

    def run(self):
        try:
            return self._run_locked()
        except BaseException as error:
            # Include failures when leaving the lock context, after a result was committed.
            if not self.claimed:
                emit('FAILED_PRE_SWITCH', error_type=type(error).__name__)
                return 1
            self.invalidate_result()
            try:
                fd = os.open(self.root / 'deploy.lock', os.O_RDWR | os.O_NOFOLLOW)
                with os.fdopen(fd, 'r+') as lock:
                    fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
                    return self.failed(error)
            except BaseException:
                outcome = 'FAILED_ROLLBACK' if self.armed else 'FAILED_PRE_SWITCH'
                try:
                    self.result(outcome, 'finalization_or_lock_failure', None)
                except BaseException:
                    self.invalidate_result()
                emit(outcome, reason='finalization_or_lock_failure')
                return 3 if self.armed else 1

    def _run_locked(self):
        """One lock encloses baseline, switch, checks, result and any rollback."""
        private_directory(self.root)
        fd = os.open(self.root / 'deploy.lock', os.O_CREAT | os.O_RDWR | os.O_NOFOLLOW, 0o600)
        with os.fdopen(fd, 'r+') as lock:
            try:
                fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError:
                emit('BUSY_NO_DEPLOYMENT')
                return 2
            try:
                self.directory.mkdir(mode=0o700)  # Permanent, per-version claim: no retry/redeployment.
            except FileExistsError:
                emit('ROUND_EXISTS_NO_DEPLOYMENT')
                return 2
            self.claimed = True
            seconds, committed = None, False
            try:
                self.preflight()
                self.switch()
                seconds = self.accept_and_observe()
                # Cleanup, log output and all fsync work precede the success rename.
                (self.directory / 'runtime.env').unlink(missing_ok=True)
                emit('OBSERVATION_COMPLETE_COMMITTING_RESULT', strict=True, version=self.version,
                     seconds=seconds, container_id=self.new_id)
                self.result('PASSED', 'clean_observation', seconds)
                committed = True
                return 0
            except BaseException as error:
                return self.failed(error, seconds)
            finally:
                if not committed:
                    try:
                        (self.directory / 'runtime.env').unlink(missing_ok=True)
                    except OSError:
                        emit('ENV_FILE_CLEANUP_FAILED')

    def invalidate_result(self):
        # Never let a stale successful result survive a later failure, even if its
        # replacement cannot be written/fsynced. An absent result also forbids publication.
        try:
            (self.directory / 'result.json').unlink(missing_ok=True)
        except OSError:
            emit('RESULT_INVALIDATION_FAILED')

    def failed(self, error, seconds=None):
        self.invalidate_result()
        code = str(error) if isinstance(error, Failure) else type(error).__name__
        outcome = 'FAILED_PRE_SWITCH'
        if self.armed:
            signal.signal(signal.SIGTERM, signal.SIG_IGN)
            signal.signal(signal.SIGINT, signal.SIG_IGN)
            try:
                self.rollback()
                outcome = 'FAILED_ROLLED_BACK'
                emit('ROLLED_BACK_VERIFIED', container_id=self.a.old_container_id)
            except BaseException as recovery_error:
                outcome = 'FAILED_ROLLBACK'
                emit('ROLLBACK_FAILED', error_type=type(recovery_error).__name__)
        try:
            self.result(outcome, code, seconds)
        except BaseException:
            self.invalidate_result()
            emit('RESULT_WRITE_FAILED')
        emit(outcome, reason=code)
        return 3 if outcome == 'FAILED_ROLLBACK' else 1

    def result(self, outcome, reason, seconds):
        data = dict(outcome=outcome, reason=reason, at=self.io.now(), version=self.version,
                    old_id=self.a.old_container_id, target_image_id=self.a.target_image_id,
                    new_id=self.new_id, observation_seconds=seconds,
                    usage_counts=dict(self.counts), log_5xx=self.log_fives,
                    protocol_error=self.protocol, baseline_5xx_ratio=self.baseline_ratio)
        pending = self.directory / 'result.pending.json'
        write_private(pending, json.dumps(data).encode())
        fd = os.open(self.directory, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
        try:
            os.fsync(fd)
        finally:
            os.close(fd)
        # The publication point is the LAST fallible operation: no fsync or cleanup after it.
        # A crash may lose the rename; absence never authorizes publication.
        os.replace(pending, self.directory / 'result.json')


def terminated(signum, frame):
    raise Cancelled()


def main():
    os.umask(0o077)
    if len(sys.argv) == 3 and sys.argv[1] == '--worker':
        try:
            payload = json.load(sys.stdin)
            if sys.argv[2] == 'usage':
                value = read_usage(payload)
            elif sys.argv[2] == 'candidate-snapshot':
                value = copy_candidate_configs(payload)
            else:
                value = http_work(sys.argv[2], payload)
            sys.stdout.write(json.dumps({'ok': True, 'value': value}))
            return 0
        except BaseException:
            # Never stringify HTTP exceptions: they can include a login URL with its key.
            sys.stdout.write('{"ok":false}')
            return 1
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('image', help='already transferred magpie:<major>.<minor>.<patch>-p<N>')
    parser.add_argument('--target-image-id', help='optional assertion, full sha256 ID')
    parser.add_argument('--old-container-id', help='optional assertion, full container ID')
    parser.add_argument('--old-image-id', help='optional assertion, full sha256 ID')
    parser.add_argument('--expected-snapshot', help='private reviewed baseline/candidate snapshots bound to target image ID')
    args = parser.parse_args()
    signal.signal(signal.SIGTERM, terminated)
    signal.signal(signal.SIGINT, terminated)
    try:
        require(os.geteuid() == 0, 'run_entry_with_sudo_n')
        require(re.fullmatch(r'magpie:\d+\.\d+\.\d+-p\d+', args.image), 'invalid_image_tag')
        require((not args.old_container_id or re.fullmatch(r'[0-9a-f]{64}', args.old_container_id))
                and all(not v or re.fullmatch(r'sha256:[0-9a-f]{64}', v)
                        for v in (args.target_image_id, args.old_image_id)), 'full_ids_required')
        script = Path(__file__).resolve().stat()
        require(not stat.S_IMODE(script.st_mode) & 0o022, 'script_must_not_be_publicly_writable')
        return Deployment(args).run()
    except BaseException as error:
        emit('FAILED_PRE_SWITCH', error_type=type(error).__name__)
        return 1


if __name__ == '__main__':
    sys.exit(main())

# MAGPIE_PYTHON_END
MAGPIE_EMBEDDED_PYTHON
