#!/usr/bin/python3
"""Offline tests: no SSH, real Docker, provider network, or real inference.
The real Deployment methods run against a stateful fake Docker/HTTP backend;
real tar/file permissions are exercised only inside TemporaryDirectory(/tmp).
"""
import collections
import contextlib
import copy
import datetime as dt
import fcntl
import io
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import types
import unittest
from unittest.mock import patch
import urllib.request

sys.dont_write_bytecode = True
ENTRY = Path(__file__).resolve().with_name('deploy-docker1.sh')
EMBEDDED = ENTRY.read_text().split('\n# MAGPIE_PYTHON_BEGIN\n', 1)[1].split('\n# MAGPIE_PYTHON_END\n', 1)[0]
g = types.ModuleType('gate')
g.__file__ = str(ENTRY)
exec(compile(EMBEDDED, str(ENTRY), 'exec'), g.__dict__)
OLD, NEW, FOREIGN = 'a' * 64, 'b' * 64, 'c' * 64
OLD_IMAGE, TARGET = 'sha256:' + 'd' * 64, 'sha256:' + 'e' * 64
VERSION = '0.1.2950-p1'
IMAGE = 'magpie:' + VERSION
SENTINEL = 'OFFLINE-SECRET-MUST-NOT-APPEAR'
BASE_TIME = dt.datetime(2026, 11, 4, tzinfo=dt.timezone.utc)
SNAPSHOT = {'models': ['group/fast-cheap'], 'providers': ['provider'], 'groups': ['fast-cheap'], 'cooldown': {}}


class FakeSystem:
    def __init__(self, parent):
        self.parent = parent
        self.clock = 0.0
        self.calls = []
        self.inferences = 0
        self.inference_times = []
        self.snapshots = 0
        self.snapshot_keys = []
        self.baseline_key_index = 0
        self.binary_version = VERSION
        self.boundary_adjusted = False
        self.log_error_count = 20
        self.log_success_count = 0
        self.inject_usage = []
        self.initial_acceptance = None
        self.events = []
        self.log_lines = []
        self.triggered = False
        self.mode = ''
        self.step = ''
        self.after = False
        self.rollback_failure = ''
        self.candidate_failure = ''
        self.containers = {OLD: self.container(OLD, '/magpie', OLD_IMAGE, running=True)}
        self.volume = parent / 'volume'
        (self.volume / 'magpie').mkdir(parents=True)
        self.usage_path = self.volume / 'magpie/usage.jsonl'
        self.usage_path.write_text(json.dumps({'t': (BASE_TIME - dt.timedelta(minutes=1)).isoformat(), 'status': 200})+'\n')
        (self.volume / 'magpie/caller-keys.json').write_text(json.dumps([{'secret': SENTINEL}]))
        (self.volume / 'original-config').write_text('before')
        for name in ('providers.json', 'logins.json'):
            (self.volume / 'magpie' / name).write_text(json.dumps({'private': SENTINEL}))

    @staticmethod
    def container(cid, name, image, running=False, token=None):
        return dict(Id=cid, Name=name, Image=image, RestartCount=0,
                    State={'Status': 'running' if running else 'created', 'Health': {'Status': 'healthy'},
                           'OOMKilled': False, 'StartedAt': BASE_TIME.isoformat()},
                    Config={'User': 'nonroot', 'Cmd': ['web', '--addr', '0.0.0.0:3430', '--no-open'],
                            'Env': ['MAGPIE_WEB_KEY=' + SENTINEL, 'MAGPIE_PUBLIC_URL=http://example.invalid', 'PATH=/usr/bin:/bin'],
                            'Labels': {'magpie.deploy': token} if token else {}},
                    Mounts=[{'Name': 'magpie-config', 'Destination': '/config', 'RW': True}],
                    HostConfig={'RestartPolicy': {'Name': 'unless-stopped'}, 'NetworkMode': 'default',
                                'PortBindings': {p+'/tcp': [{'HostPort':p,'HostIp':''}] for p in ['3425','3430']}})

    def mono(self):
        before = self.clock
        if self.mode == 'clock_boundary' and self.clock >= g.OBSERVE - .25:
            self.clock += .2  # Time can cross the boundary between two monotonic reads.
        return before
    def sleep(self, seconds):
        if seconds < 0:
            raise ValueError('negative sleep')
        if self.mode == 'clock_boundary' and self.inferences and not self.boundary_adjusted \
                and self.clock + seconds >= g.OBSERVE:
            self.clock = g.OBSERVE - .25
            self.boundary_adjusted = True
        else:
            self.clock += seconds
        if self.inferences and not self.triggered and self.clock >= 30:
            self.inject_observation()
    def now(self): return (BASE_TIME + dt.timedelta(seconds=self.clock)).isoformat()

    def append(self, status=200, **extra):
        with self.usage_path.open('a') as f:
            f.write(json.dumps({'t': self.now(), 'status': status, **extra})+'\n')

    def inject_observation(self):
        self.triggered = True
        if self.mode == 'panic': self.log_lines.append('panic: private application detail ' + SENTINEL)
        elif self.mode == 'protocol': self.log_lines.append('protocol_error ' + SENTINEL)
        elif self.mode == 'oom': self.containers[NEW]['State']['OOMKilled'] = True
        elif self.mode == 'restart': self.containers[NEW]['RestartCount'] = 1
        elif self.mode == 'unhealthy': self.containers[NEW]['State']['Health']['Status'] = 'unhealthy'
        elif self.mode == 'transient_unhealthy': self.events.append({'Action': 'health_status: unhealthy'})
        elif self.mode == 'surge':
            for _ in range(5): self.append(503)
        elif self.mode == 'ordinary_5xx': self.append(503)
        elif self.mode == 'usage_protocol': self.append(200, err_type='protocol_error')
        elif self.mode == 'partial_usage':
            with self.usage_path.open('a') as f: f.write('{"t":')
        elif self.mode == 'bad_usage':
            with self.usage_path.open('a') as f: f.write('{bad json}\n')
        elif self.mode == 'replaced_usage':
            self.usage_path.rename(self.volume / 'saved-usage')
            self.usage_path.write_text('')
        elif self.mode == 'truncated_usage': self.usage_path.write_text('')
        elif self.mode == 'foreign': self.containers[NEW]['Name']='/displaced'; self.containers[FOREIGN]=self.container(FOREIGN,'/magpie',OLD_IMAGE,True)
        elif self.mode == 'identity': self.containers[NEW]['Image'] = OLD_IMAGE
        elif self.mode == 'new_removed': del self.containers[NEW]
        elif self.mode == 'log_usage_mismatch':
            self.log_lines += ['status=503'] * self.log_error_count + ['status=200'] * self.log_success_count
            self.append(200)
        elif self.mode == 'high_baseline':
            self.log_lines += ['status=503'] * self.log_error_count + ['status=200'] * self.log_success_count
            for status in self.inject_usage:
                self.append(status)
        elif self.mode == 'volume_writer':
            self.containers[FOREIGN] = self.container(FOREIGN, '/shared-volume-writer', OLD_IMAGE, True)
        elif self.mode == 'term': signal.raise_signal(signal.SIGTERM)
        elif self.mode == 'int': signal.raise_signal(signal.SIGINT)

    def fault(self, step, after=False):
        if step == self.step and after == self.after:
            self.step = ''
            if self.mode == 'signal_stop': signal.raise_signal(signal.SIGTERM)
            raise g.Failure('injected_command_failure')

    def run(self, argv, *, timeout=g.COMMAND_TIMEOUT, data=None, watch=None, combine=False):
        self.calls.append(list(argv))
        self.assert_safe_args(argv)
        if argv[:len(g.DOCKER)] != g.DOCKER:
            if argv[0] == '/usr/bin/tar':
                self.fault('backup' if '-czf' in argv else 'archive_check' if '-tzf' in argv else 'restore')
            if argv[0] == '/usr/bin/find': self.fault('clean_volume')
            for arg in argv:
                if arg.startswith('/') and arg != argv[0]:
                    assert str(self.parent) in arg, 'offline filesystem command escaped /tmp fixture'
            return subprocess.run(argv, check=True, stdout=subprocess.PIPE, stderr=subprocess.PIPE, timeout=5).stdout
        args = argv[len(g.DOCKER):]
        command = args[0]
        self.fault(command)
        if command == 'inspect':
            key = args[-1]
            require_present = self.containers.get(key)
            if not require_present:
                require_present = next((v for v in self.containers.values() if v['Name'] == '/' + key), None)
            if not require_present: raise g.Failure('inspect_missing')
            result = json.dumps([require_present]).encode()
        elif command == 'image':
            if self.mode == 'image_missing': raise g.Failure('missing_image')
            result = json.dumps([{'Id': OLD_IMAGE if self.mode == 'image_mismatch' else TARGET,
                                  'Config': {'Healthcheck': {'Test': ['CMD', 'true']}}}]).encode()
        elif command == 'volume':
            result = json.dumps([{'Name':'magpie-config','Driver':'local','Mountpoint':str(self.volume)}]).encode()
        elif command == 'ps':
            filter_value = args[args.index('--filter')+1]
            rows = list(self.containers.values())
            if '-a' not in args: rows = [v for v in rows if v['State']['Status']=='running']
            if filter_value.startswith('name=^/'):
                name = filter_value[len('name=^'): -1]
                rows = [v for v in rows if v['Name'] == name]
            elif filter_value.startswith('id='):
                rows=[v for v in rows if v['Id']==filter_value[3:]]
            elif filter_value.startswith('label='):
                token=filter_value.split('=',2)[-1]
                rows=[v for v in rows if v['Config']['Labels'].get('magpie.deploy')==token]
            result = ('\n'.join(v['Id'] for v in rows)+'\n').encode()
        elif command == 'stop':
            self.containers[args[-1]]['State']['Status']='exited'
            result=b''
        elif command == 'rename':
            cid, name=args[1:]
            if any(v['Name']=='/'+name for v in self.containers.values()): raise g.Failure('name_conflict')
            self.containers[cid]['Name']='/'+name
            result=b''
        elif command == 'create':
            token=args[args.index('--label')+1].split('=',1)[1]
            name=args[args.index('--name')+1]
            assert '--pull=never' in args and TARGET in args
            env=Path(args[args.index('--env-file')+1])
            assert env.stat().st_mode & 0o777 == 0o600
            assert SENTINEL in env.read_text()
            self.containers[NEW]=self.container(NEW,'/'+name,TARGET,token=token)
            result=(NEW+'\n').encode()
        elif command == 'start':
            cid=args[1]
            if cid==OLD and self.rollback_failure=='start': raise g.Failure('restore_start_failed')
            self.containers[cid]['State']['Status']='running'
            if cid == NEW:
                self.containers[cid]['State']['StartedAt']=self.now()
                (self.volume/'original-config').write_text('changed by new version')
                (self.volume/'new-only-file').write_text('new version only')
                if self.mode in ('startup_unhealthy','startup_timeout'):
                    self.containers[cid]['State']['Health']['Status']='unhealthy' if self.mode=='startup_unhealthy' else 'starting'
            result=b''
        elif command == 'rm':
            if self.rollback_failure == 'rm':
                raise g.Failure('persistent_remove_failure')
            assert args[-1] != FOREIGN
            del self.containers[args[-1]]
            result=b''
        elif command == 'exec': result=('magpie '+('WRONG' if self.mode=='bad_version' else self.binary_version)+'\n').encode()
        elif command == 'logs':
            if self.mode=='log_failure': raise g.Failure('logs_failed')
            result=('\n'.join(self.log_lines)).encode()
            self.log_lines=[]
        elif command == 'events': result='\n'.join(json.dumps(v) for v in self.events).encode(); self.events=[]
        else: raise AssertionError('unimplemented fake docker command '+command)
        self.fault(command, after=True)
        return result

    def assert_safe_args(self, argv):
        assert all(SENTINEL not in arg for arg in argv), 'secret in process arguments'

    def worker(self, action, payload, *, watch=None, timeout=g.HTTP_TIMEOUT):
        if action=='usage': return g.read_usage(payload)
        if action=='candidate-snapshot':
            assert timeout == 2
            assert NEW not in self.containers and self.containers[OLD]['State']['Status']=='exited'
            if self.candidate_failure:
                if self.candidate_failure=='timeout': self.clock+=2
                raise g.Failure('candidate_copy_failed')
            return g.copy_candidate_configs(payload)
        if action=='snapshot':
            self.snapshots+=1
            self.snapshot_keys.append(list(payload['keys']))
            if self.mode=='baseline_api' and NEW not in self.containers: raise g.Failure('baseline_api_failed')
            if self.mode=='acceptance_api' and NEW in self.containers: raise g.Failure('acceptance_api_failed')
            if self.rollback_failure=='api' and self.inferences and NEW not in self.containers: raise g.Failure('rollback_api_failed')
            snap=copy.deepcopy(SNAPSHOT)
            if self.mode=='model_list' and NEW in self.containers: snap['models'].append('unexpected-model')
            if self.mode=='final_snapshot' and NEW in self.containers and self.clock>=g.OBSERVE: snap['groups']=[]
            if self.mode=='final_panic' and NEW in self.containers and self.clock>=g.OBSERVE: self.log_lines.append('panic: final boundary')
            return {'key_index':self.baseline_key_index if self.snapshots==1 else 0,'snapshot':snap}
        assert action=='inference'
        self.inferences+=1
        self.inference_times.append(self.clock)
        if self.mode in ('inference_timeout','inference_failed'): raise g.Failure(self.mode)
        if self.mode == 'observation_probe_failed' and self.inferences == 2: raise g.Failure(self.mode)
        if self.mode == 'second_observation_probe_failed' and self.inferences == 3: raise g.Failure(self.mode)
        if self.mode=='inference_protocol':
            self.clock += 4
            self.log_lines.append('protocol_error during blocked inference')
            watch()
        self.initial_acceptance=self.clock
        return {'status':200}


class TransactionTests(unittest.TestCase):
    def setUp(self):
        self.tmp=tempfile.TemporaryDirectory(prefix='magpie-deploy-tests-',dir='/tmp')
        self.path=Path(self.tmp.name)
        self.io=FakeSystem(self.path)
        self.args=types.SimpleNamespace(image=IMAGE,old_container_id=OLD,old_image_id=OLD_IMAGE,target_image_id=TARGET)
        self.root=self.path/'state'
        self.sig={s:signal.getsignal(s) for s in (signal.SIGTERM,signal.SIGINT)}
        for sig in self.sig: signal.signal(sig,g.terminated)
        self.output=io.StringIO()

    def tearDown(self):
        for sig,handler in self.sig.items(): signal.signal(sig,handler)
        self.tmp.cleanup()

    def execute(self):
        self.deploy=g.Deployment(self.args,self.io,self.root)
        with contextlib.redirect_stdout(self.output): code=self.deploy.run()
        self.assertNotIn(SENTINEL,self.output.getvalue())
        self.result=json.loads((self.deploy.directory/'result.json').read_text()) if (self.deploy.directory/'result.json').exists() else None
        return code

    def assert_restored(self):
        self.assertEqual(self.result['outcome'],'FAILED_ROLLED_BACK')
        self.assertEqual(self.io.containers[OLD]['Name'],'/magpie')
        self.assertEqual(self.io.containers[OLD]['State']['Status'],'running')
        self.assertNotIn(NEW,self.io.containers)
        # Rollback must preserve real users' post-backup writes, never restore the whole volume.
        if any(c[len(g.DOCKER):][:1]==['start'] and c[-1]==NEW for c in self.io.calls):
            self.assertEqual((self.io.volume/'original-config').read_text(),'changed by new version')
            self.assertTrue((self.io.volume/'new-only-file').exists())
        self.assertFalse(any(c[0]=='/usr/bin/find' or '-xzf' in c for c in self.io.calls))
        self.assertNotIn('OBSERVATION_PASSED',self.output.getvalue())

    def test_clean_full_observation_permissions_and_no_secret_echo(self):
        self.assertEqual(self.execute(),0)
        self.assertEqual(self.result['outcome'],'PASSED')
        self.assertGreaterEqual(self.result['observation_seconds'],2760)
        self.assertEqual(self.io.inferences,3)
        self.assertGreaterEqual(self.io.inference_times[1]-self.io.inference_times[0],g.OBSERVE/10)
        self.assertGreaterEqual(self.io.inference_times[2]-self.io.inference_times[0],g.OBSERVE*.6)
        self.assertFalse((self.deploy.directory/'runtime.env').exists())
        self.assertEqual(self.root.stat().st_mode&0o777,0o700)
        self.assertEqual(self.deploy.directory.stat().st_mode&0o777,0o700)
        for file in self.deploy.directory.iterdir(): self.assertEqual(file.stat().st_mode&0o777,0o600)
        self.assertEqual(self.io.containers[OLD]['State']['Status'],'exited')

    def test_observation_probe_failure_is_not_retried(self):
        self.io.mode='observation_probe_failed'
        self.assertEqual(self.execute(),1)
        self.assertEqual(self.io.inferences,2)
        self.assert_restored()

    def test_second_observation_probe_failure_is_not_retried(self):
        self.io.mode='second_observation_probe_failed'
        self.assertEqual(self.execute(),1)
        self.assertEqual(self.io.inferences,3)
        self.assert_restored()

    def test_single_image_argument_binds_ids_under_lock(self):
        self.args.old_container_id=self.args.old_image_id=self.args.target_image_id=None
        self.assertEqual(self.execute(),0)
        self.assertEqual((self.args.old_container_id,self.args.old_image_id,self.args.target_image_id),(OLD,OLD_IMAGE,TARGET))

    def test_runtime_version_and_directory_come_from_image(self):
        self.args.image='magpie:1.2.3-p9'; self.io.binary_version='1.2.3-p9'
        self.assertEqual(self.execute(),0)
        self.assertEqual(self.result['version'],'1.2.3-p9')
        self.assertEqual(self.deploy.directory.name,'1.2.3-p9')

    def test_same_selected_caller_key_used_for_every_comparison(self):
        chosen='SECOND-OFFLINE-CALLER-KEY'
        (self.io.volume/'magpie/caller-keys.json').write_text(json.dumps([{'secret':SENTINEL},{'secret':chosen}]))
        self.io.baseline_key_index=1
        self.assertEqual(self.execute(),0)
        self.assertEqual(self.io.snapshot_keys[0],[SENTINEL,chosen])
        self.assertTrue(all(keys==[chosen] for keys in self.io.snapshot_keys[1:]))
        self.assertNotIn(chosen,self.output.getvalue())

    def test_rm_reports_error_after_removal_still_restores_old(self):
        self.io.mode='inference_failed'; self.io.step='rm'; self.io.after=True
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_rm_transient_error_before_removal_still_restores_old(self):
        self.io.mode='inference_failed'; self.io.step='rm'; self.io.after=False
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_result_fsync_failure_never_exposes_passed_during_rollback(self):
        original_write, original_fsync, original_rollback = g.write_private, g.os.fsync, g.Deployment.rollback
        flags={'armed':False,'failed':False,'seen':None}
        def writing(path,data):
            if path.name.startswith('result') and json.loads(data).get('outcome')=='PASSED': flags['armed']=True
            return original_write(path,data)
        def fsync(fd):
            if flags['armed'] and not flags['failed']:
                flags['failed']=True
                raise OSError('synthetic fsync failure')
            return original_fsync(fd)
        def rollback(deployment):
            path=deployment.directory/'result.json'
            flags['seen']=json.loads(path.read_text())['outcome'] if path.exists() else None
            return original_rollback(deployment)
        with patch.object(g,'write_private',writing), patch.object(g.os,'fsync',fsync), patch.object(g.Deployment,'rollback',rollback):
            self.assertEqual(self.execute(),1)
        self.assertTrue(flags['failed'])
        self.assertNotEqual(flags['seen'],'PASSED')
        self.assert_restored()

    def test_final_log_failure_never_publishes_success(self):
        class FailingLog(io.StringIO):
            def write(self,text):
                if 'OBSERVATION_COMPLETE_COMMITTING_RESULT' in text:
                    raise OSError('synthetic final log failure')
                return super().write(text)
        self.output=FailingLog()
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_term_at_success_rename_revokes_passed_and_rolls_back(self):
        original=g.os.replace
        flags={'sent':False}
        def replace(source,dest):
            original(source,dest)
            if Path(dest).name=='result.json' and not flags['sent'] \
                    and json.loads(Path(dest).read_text())['outcome']=='PASSED':
                flags['sent']=True
                signal.raise_signal(signal.SIGTERM)
        with patch.object(g.os,'replace',replace): self.assertEqual(self.execute(),1)
        self.assertTrue(flags['sent'])
        self.assert_restored()

    def test_lock_close_failure_after_commit_clears_passed_and_restores_old(self):
        original=g.os.fdopen
        flags={'raised':False}
        class Closing:
            def __init__(self,inner): self.inner=inner
            def __enter__(self): return self.inner.__enter__()
            def __exit__(self,*args):
                answer=self.inner.__exit__(*args)
                if not flags['raised']:
                    flags['raised']=True
                    raise OSError('synthetic lock close failure after commit')
                return answer
        def fdopen(fd,*args,**kwargs):
            inner=original(fd,*args,**kwargs)
            return Closing(inner) if args and args[0]=='r+' else inner
        with patch.object(g.os,'fdopen',fdopen): self.assertEqual(self.execute(),1)
        self.assertTrue(flags['raised'])
        self.assert_restored()

    def test_replace_error_after_effect_revokes_success(self):
        original=g.os.replace
        flags={'raised':False}
        def replace(source,dest):
            original(source,dest)
            if Path(dest).name=='result.json' and not flags['raised'] \
                    and json.loads(Path(dest).read_text())['outcome']=='PASSED':
                flags['raised']=True
                raise OSError('synthetic rename error after effect')
        with patch.object(g.os,'replace',replace): self.assertEqual(self.execute(),1)
        self.assertTrue(flags['raised'])
        self.assert_restored()

    def test_replace_after_effect_then_failed_status_write_removes_passed(self):
        original_replace, original_write = g.os.replace, g.write_private
        flags={'raised':False}
        def replace(source,dest):
            original_replace(source,dest)
            if Path(dest).name=='result.json' and not flags['raised'] \
                    and json.loads(Path(dest).read_text())['outcome']=='PASSED':
                flags['raised']=True
                raise OSError('synthetic rename error after effect')
        def write(path,data):
            if path.name.startswith('result') and json.loads(data)['outcome'] != 'PASSED':
                raise OSError('synthetic persistent failure-result write error')
            return original_write(path,data)
        with patch.object(g.os,'replace',replace), patch.object(g,'write_private',write):
            self.assertEqual(self.execute(),1)
        self.assertTrue(flags['raised'])
        self.assertIsNone(self.result)
        self.assertEqual(self.io.containers[OLD]['State']['Status'],'running')
        self.assertNotIn('OBSERVATION_PASSED',self.output.getvalue())

    def test_directory_fsync_failure_precedes_success_commit(self):
        import stat
        original=g.os.fsync
        flags={'raised':False}
        def fsync(fd):
            if stat.S_ISDIR(os.fstat(fd).st_mode) and not flags['raised']:
                flags['raised']=True
                self.assertFalse((self.root/VERSION/'result.json').exists())
                raise OSError('synthetic directory sync failure')
            return original(fd)
        with patch.object(g.os,'fsync',fsync): self.assertEqual(self.execute(),1)
        self.assertTrue(flags['raised'])
        self.assert_restored()

    def test_cleanup_failure_happens_before_publishing_passed(self):
        original=g.Path.unlink
        flags={'failed':False}
        def unlink(path,*args,**kwargs):
            if path.name=='runtime.env' and not flags['failed']:
                flags['failed']=True
                raise OSError('synthetic cleanup failure')
            return original(path,*args,**kwargs)
        with patch.object(g.Path,'unlink',unlink):
            self.assertEqual(self.execute(),1)
        self.assertTrue(flags['failed'])
        self.assert_restored()

    def test_error_after_passed_write_plus_failed_status_write_never_leaves_passed(self):
        original=g.write_private
        def write(path,data):
            if path.name.startswith('result'):
                outcome=json.loads(data)['outcome']
                if outcome=='PASSED':
                    original(path,data)
                    raise OSError('synthetic error after result write')
                raise OSError('synthetic persistent final-result failure')
            return original(path,data)
        with patch.object(g,'write_private',write):
            self.assertEqual(self.execute(),1)
        self.assertTrue(self.result is None or self.result['outcome']!='PASSED')
        self.assertEqual(self.io.containers[OLD]['State']['Status'],'running')

    def test_logs_5xx_surge_detected_even_if_usage_has_one_200(self):
        self.io.mode='log_usage_mismatch'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertLess(self.io.clock,2760)

    def test_log_surge_is_scaled_against_baseline_and_traffic(self):
        for prior_five, should_fail in ((5, True),(20, False)):
            case=self.path/str(prior_five); case.mkdir()
            self.io=FakeSystem(case); self.io.mode='log_usage_mismatch'; self.io.log_success_count=80
            self.root=case/'state'
            self.io.usage_path.write_text(''.join(json.dumps({'t':(BASE_TIME-dt.timedelta(minutes=1)).isoformat(),
                'status': 503 if i < prior_five else 200})+'\n' for i in range(100)))
            self.assertEqual(self.execute(),1 if should_fail else 0)
            if should_fail: self.assert_restored()

    def test_high_baseline_five_usage_503_triggers_rollback(self):
        self.io.mode='high_baseline'; self.io.log_error_count=0
        self.io.inject_usage=[503]*5
        self.io.usage_path.write_text(''.join(json.dumps({'t':(BASE_TIME-dt.timedelta(minutes=1)).isoformat(),
            'status':503 if i<50 else 200})+'\n' for i in range(100)))
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertEqual(self.result['reason'],'5xx_surge')

    def test_high_baseline_twenty_log_503_triggers_rollback(self):
        self.io.mode='high_baseline'; self.io.log_error_count=20
        self.io.inject_usage=[200]
        self.io.usage_path.write_text(''.join(json.dumps({'t':(BASE_TIME-dt.timedelta(minutes=1)).isoformat(),
            'status':503 if i<50 else 200})+'\n' for i in range(100)))
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertEqual(self.result['reason'],'5xx_log_surge')

    def test_high_baseline_five_503_and_twenty_error_logs_triggers_rollback(self):
        self.io.mode='high_baseline'; self.io.log_error_count=20
        self.io.inject_usage=[503]*5
        self.io.usage_path.write_text(''.join(json.dumps({'t':(BASE_TIME-dt.timedelta(minutes=1)).isoformat(),
            'status':503 if i<50 else 200})+'\n' for i in range(100)))
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_high_baseline_ratio_exactly_half_does_not_cross_cap(self):
        self.io.mode='high_baseline'; self.io.log_error_count=0
        self.io.inject_usage=[503]*5+[200]*5
        self.io.usage_path.write_text(''.join(json.dumps({'t':(BASE_TIME-dt.timedelta(minutes=1)).isoformat(),
            'status':503 if i<50 else 200})+'\n' for i in range(100)))
        self.assertEqual(self.execute(),0)
        self.assertEqual(self.result['outcome'],'PASSED')

    def test_high_baseline_four_503_does_not_reach_minimum_count(self):
        self.io.mode='high_baseline'; self.io.log_error_count=0
        self.io.inject_usage=[503]*4
        self.io.usage_path.write_text(''.join(json.dumps({'t':(BASE_TIME-dt.timedelta(minutes=1)).isoformat(),
            'status':503 if i<50 else 200})+'\n' for i in range(100)))
        self.assertEqual(self.execute(),0)
        self.assertEqual(self.result['outcome'],'PASSED')

    def test_new_volume_writer_during_observation_causes_rollback(self):
        self.io.mode='volume_writer'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertIn(FOREIGN,self.io.containers)
        self.assertEqual(self.result['reason'],'another_volume_writer')

    def test_observation_crossing_boundary_never_sleeps_negative(self):
        self.io.mode='clock_boundary'
        self.assertEqual(self.execute(),0)
        self.assertTrue(self.io.boundary_adjusted)
        self.assertGreaterEqual(self.result['observation_seconds'],2760)

    def test_candidate_snapshot_manifest_names_actual_optional_files(self):
        self.io.mode='inference_failed'
        for name in ('settings.json','plugin-auth.json','migrations.json','plugins.yml'):
            (self.io.volume/'magpie'/name).write_text(SENTINEL)
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        directory=self.deploy.directory/'candidate-config'
        manifest=json.loads((directory/'complete.json').read_text())
        expected={'providers.json','logins.json','settings.json','caller-keys.json',
                  'plugin-auth.json','migrations.json','plugins.yml'}
        self.assertEqual(set(manifest['files']),expected)
        self.assertFalse((directory/'plugin-auth.json.bak').exists())
        for name in expected:
            self.assertTrue((directory/name).is_file())
            self.assertEqual((directory/name).stat().st_mode&0o777,0o600)

    def test_optional_candidate_snapshot_files_may_be_absent(self):
        self.io.mode='inference_failed'
        (self.io.volume/'magpie/caller-keys.json').unlink()
        # Retain one baseline credential through the fake, without exposing it anywhere.
        self.io.mode='inference_failed'
        (self.io.volume/'magpie/caller-keys.json').write_text(json.dumps([{'secret':SENTINEL}]))
        original=self.io.worker
        def worker(action,payload,**kwargs):
            if action=='candidate-snapshot': (self.io.volume/'magpie/caller-keys.json').unlink()
            return original(action,payload,**kwargs)
        self.io.worker=worker
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        manifest=json.loads((self.deploy.directory/'candidate-config/complete.json').read_text())
        self.assertEqual(set(manifest['files']),{'providers.json','logins.json'})

    def test_candidate_already_removed_still_restores_old(self):
        self.io.mode='new_removed'
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_candidate_config_snapshot_private_and_non_destructive(self):
        self.io.mode='inference_failed'
        (self.io.volume/'magpie/plugin-auth.json').write_text(SENTINEL)
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        directory=self.deploy.directory/'candidate-config'
        self.assertEqual(directory.stat().st_mode&0o777,0o700)
        self.assertTrue(json.loads((directory/'complete.json').read_text())['complete'])
        self.assertEqual((directory/'plugin-auth.json').read_text(),SENTINEL)
        for file in directory.iterdir(): self.assertEqual(file.stat().st_mode&0o777,0o600)

    def test_candidate_snapshot_timeout_or_failure_does_not_block_rollback(self):
        for failure in ('timeout','failure'):
            case=self.path/failure; case.mkdir()
            self.io=FakeSystem(case); self.io.mode='inference_failed'; self.io.candidate_failure=failure
            self.root=case/'state'
            self.assertEqual(self.execute(),1)
            self.assert_restored()
            self.assertIn('CANDIDATE_CONFIG_SNAPSHOT_INCOMPLETE',self.output.getvalue())
            self.assertLessEqual(self.io.clock,2)
            self.assertFalse((self.deploy.directory/'candidate-config/complete.json').exists())

    def test_early_failures_never_stop_old(self):
        for mode in ['image_missing','image_mismatch','baseline_api']:
            with self.subTest(mode=mode):
                self.io.mode=mode
                self.root=self.path/('state-'+mode)
                self.assertEqual(self.execute(),1)
                self.assertEqual(self.result['outcome'],'FAILED_PRE_SWITCH')
                self.assertEqual(self.io.containers[OLD]['State']['Status'],'running')
        self.assertFalse(any(c[len(g.DOCKER):][:1]==['stop'] for c in self.io.calls))

    def test_wrong_expected_old_id(self):
        self.args.old_container_id=FOREIGN
        self.assertEqual(self.execute(),1)
        self.assertEqual(self.result['outcome'],'FAILED_PRE_SWITCH')
        self.assertEqual(self.io.containers[OLD]['State']['Status'],'running')

    def test_unrepresentable_env_rejected_before_stop(self):
        self.io.containers[OLD]['Config']['Env'].append('BAD=line1\nline2')
        self.assertEqual(self.execute(),1)
        self.assertEqual(self.result['outcome'],'FAILED_PRE_SWITCH')

    def test_other_volume_writer_rejected(self):
        self.io.containers[FOREIGN]=self.io.container(FOREIGN,'/another',OLD_IMAGE,True)
        self.assertEqual(self.execute(),1)
        self.assertEqual(self.result['reason'],'another_volume_writer')

    def test_stop_timeout_after_effect_rolls_back_without_deploy_retry(self):
        self.io.step,self.io.after='stop',True
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertEqual(sum(c[len(g.DOCKER):][:1]==['stop'] for c in self.io.calls),1)

    def test_backup_failure_restores_old(self):
        self.io.step='backup'
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_rename_timeout_after_effect(self):
        self.io.step,self.io.after='rename',True
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_create_timeout_discovers_and_removes_only_own_candidate(self):
        self.io.step,self.io.after='create',True
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_start_timeout_after_effect_preserves_volume(self):
        self.io.step,self.io.after='start',True
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_startup_unhealthy_and_timeout(self):
        self.io.mode='startup_unhealthy'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertEqual(self.io.inferences,0)

    def test_startup_deadline(self):
        self.io.mode='startup_timeout'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertGreaterEqual(self.io.clock,180)
        self.assertEqual(self.io.inferences,0)

    def test_acceptance_auth_failure(self):
        self.io.mode='acceptance_api'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertEqual(self.io.inferences,0)

    def test_first_inference_timeout_is_not_retried(self):
        self.io.mode='inference_timeout'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertEqual(self.io.inferences,1)

    def test_first_inference_failure_is_not_retried(self):
        self.io.mode='inference_failed'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertEqual(self.io.inferences,1)

    def test_watchdog_cancels_pending_inference_on_protocol_error(self):
        self.io.mode='inference_protocol'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertLess(self.io.clock,90)

    def test_log_read_failure_never_passes(self):
        self.io.mode='log_failure'
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_protocol_usage_failure(self):
        self.io.mode='usage_protocol'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertLess(self.io.clock,2760)

    def test_immediate_observation_failures(self):
        for mode in ['panic','protocol','oom','restart','unhealthy','transient_unhealthy','surge']:
            with self.subTest(mode=mode):
                # Independent transaction for every scenario, never a retry of one deployment.
                case=self.path/mode; case.mkdir()
                self.io=FakeSystem(case); self.io.mode=mode; self.root=case/'state'
                self.assertEqual(self.execute(),1)
                self.assert_restored()
                self.assertLessEqual(self.io.clock,33)

    def test_ordinary_single_5xx_does_not_fail(self):
        self.io.mode='ordinary_5xx'
        self.assertEqual(self.execute(),0)
        self.assertEqual(self.deploy.counts['503'],1)

    def test_usage_integrity_failures(self):
        for mode in ['partial_usage','bad_usage','replaced_usage','truncated_usage']:
            with self.subTest(mode=mode):
                case=self.path/mode; case.mkdir()
                self.io=FakeSystem(case); self.io.mode=mode; self.root=case/'state'
                self.assertEqual(self.execute(),1)
                self.assert_restored()

    def test_final_boundary_is_rechecked(self):
        self.io.mode='final_panic'
        self.assertEqual(self.execute(),1)
        self.assert_restored()
        self.assertGreaterEqual(self.io.clock,2760)

    def test_final_snapshot_is_rechecked(self):
        self.io.mode='final_snapshot'
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_version_mismatch(self):
        self.io.mode='bad_version'
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_model_list_mismatch(self):
        self.io.mode='model_list'
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_real_term_and_int_are_caught(self):
        for mode in ['term','int']:
            case=self.path/mode; case.mkdir()
            self.io=FakeSystem(case); self.io.mode=mode; self.root=case/'state'
            for sig in self.sig: signal.signal(sig,g.terminated)
            self.assertEqual(self.execute(),1)
            self.assert_restored()

    def test_signal_during_first_stop_has_recovery_installed(self):
        self.io.mode,self.io.step,self.io.after='signal_stop','stop',True
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_foreign_current_container_is_never_removed(self):
        self.io.mode='foreign'
        self.assertEqual(self.execute(),3)
        self.assertEqual(self.result['outcome'],'FAILED_ROLLBACK')
        self.assertEqual(self.io.containers[FOREIGN]['Name'],'/magpie')
        self.assertFalse(any(c[len(g.DOCKER):][:1]==['rm'] and c[-1]==FOREIGN for c in self.io.calls))

    def test_changed_target_identity_refuses_destructive_rollback(self):
        self.io.mode='identity'
        self.assertEqual(self.execute(),3)
        self.assertEqual(self.result['outcome'],'FAILED_ROLLBACK')
        self.assertIn(NEW,self.io.containers)

    def test_rollback_start_failure_is_explicit(self):
        self.io.mode='inference_failed'; self.io.rollback_failure='start'
        self.assertEqual(self.execute(),3)
        self.assertEqual(self.result['outcome'],'FAILED_ROLLBACK')
        self.assertNotIn('ROLLED_BACK_VERIFIED',self.output.getvalue())

    def test_rollback_api_failure_is_explicit(self):
        self.io.mode='inference_failed'; self.io.rollback_failure='api'
        self.assertEqual(self.execute(),3)
        self.assertEqual(self.result['outcome'],'FAILED_ROLLBACK')

    def test_rollback_remove_failure_never_claims_recovery(self):
        self.io.mode='inference_failed'; self.io.rollback_failure='rm'
        self.assertEqual(self.execute(),3)
        self.assertEqual(self.result['outcome'],'FAILED_ROLLBACK')
        self.assertEqual(self.io.containers[OLD]['State']['Status'],'exited')
        self.assertNotIn('ROLLED_BACK_VERIFIED',self.output.getvalue())

    def test_lock_covers_baseline_and_rollback(self):
        # Every fake Docker command, snapshot, and recovery step observes the same live lock.
        original_run=self.io.run
        def verify_lock(*a,**kw):
            with (self.root/'deploy.lock').open('r+') as fd:
                with self.assertRaises(BlockingIOError): fcntl.flock(fd,fcntl.LOCK_EX|fcntl.LOCK_NB)
            return original_run(*a,**kw)
        self.io.run=verify_lock
        self.io.mode='inference_failed'
        self.assertEqual(self.execute(),1)
        self.assert_restored()

    def test_busy_lock_never_touches_docker(self):
        g.private_directory(self.root)
        with (self.root/'deploy.lock').open('w') as fd:
            fcntl.flock(fd,fcntl.LOCK_EX|fcntl.LOCK_NB)
            self.assertEqual(self.execute(),2)
        self.assertFalse(self.io.calls)

    def test_repeat_run_is_refused_even_after_failure(self):
        self.io.mode='inference_failed'
        self.assertEqual(self.execute(),1)
        count=len(self.io.calls)
        self.assertEqual(self.execute(),2)
        self.assertEqual(len(self.io.calls),count)

    def test_result_write_failure_rolls_back_and_cannot_publish(self):
        original=g.write_private
        def fail_pass(path,data):
            if path.name.startswith('result') and json.loads(data)['outcome']=='PASSED': raise OSError('offline full disk')
            return original(path,data)
        with patch.object(g,'write_private',fail_pass): self.assertEqual(self.execute(),1)
        self.assert_restored()


class PrimitiveTests(unittest.TestCase):
    def test_command_absolute_timeout_and_secret_not_in_exception(self):
        with self.assertRaises(g.Failure) as caught:
            g.System().run([sys.executable,'-I','-B','-c','import time; time.sleep(10)'],timeout=.1)
        self.assertEqual(str(caught.exception),'operation_timeout')

    def test_nonzero_output_is_not_echoed(self):
        # Synthetic secret written only to a captured pipe by an offline child.
        with self.assertRaises(g.Failure) as caught:
            g.System().run([sys.executable,'-I','-B','-c','import sys; sys.stderr.write(sys.stdin.read()); sys.exit(9)'],data=SENTINEL.encode())
        self.assertNotIn(SENTINEL,str(caught.exception))

    def test_watchdog_failure_kills_inflight_worker(self):
        calls=[]
        def watch(): calls.append(True); raise g.Failure('offline_anomaly')
        with self.assertRaises(g.Failure):
            g.System().run([sys.executable,'-I','-B','-c','import time; time.sleep(10)'],timeout=5,watch=watch)
        self.assertTrue(calls)

    def test_usage_records_after_requested_cutoff_are_not_lost(self):
        with tempfile.TemporaryDirectory(dir='/tmp',prefix='magpie-usage-test-') as temp:
            path=Path(temp)/'usage.jsonl'
            path.write_text(json.dumps({'t':(BASE_TIME+dt.timedelta(seconds=1)).isoformat(),'status':503})+'\n')
            result=g.read_usage({'path':str(path),'start':BASE_TIME.isoformat(),'stop':BASE_TIME.isoformat()})
            self.assertEqual(result['counts'],{'503':1})

    def test_redirects_cannot_leak_key_or_repeat_post(self):
        handler=g.LocalRedirect()
        req=urllib.request.Request('http://127.0.0.1:3430/?k='+SENTINEL)
        with self.assertRaises(g.Failure):
            handler.redirect_request(req,None,302,'',{},'http://elsewhere.invalid/')
        req=urllib.request.Request('http://127.0.0.1:3425/v1/chat/completions',data=b'{}')
        with self.assertRaises(g.Failure):
            handler.redirect_request(req,None,307,'',{},req.full_url)

    def test_real_http_code_uses_one_post_and_private_auth(self):
        requests=[]
        class Response:
            status=200
            def __enter__(self): return self
            def __exit__(self,*a): pass
            def read(self,n): return json.dumps({'choices':[{'message':{'content':'OK'}}]}).encode()
        class Opener:
            def open(self,request,timeout): requests.append(request); return Response()
        with patch.object(g.urllib.request,'build_opener',return_value=Opener()):
            self.assertEqual(g.http_work('inference',{'key':SENTINEL}),{'status':200})
        self.assertEqual(len(requests),1)
        self.assertEqual(requests[0].get_method(),'POST')
        self.assertNotIn(SENTINEL,requests[0].full_url)

    def test_bounded_optional_copy_has_no_blocking_reap(self):
        start=g.time.monotonic()
        with self.assertRaises(g.Failure):
            g.System().run([sys.executable,'-I','-B','-c','import time; time.sleep(10)'],timeout=.1,quick_kill=True)
        self.assertLess(g.time.monotonic()-start,1)

    def test_self_contained_entry_payload_and_worker(self):
        with tempfile.TemporaryDirectory(dir='/tmp',prefix='magpie-embedded-worker-') as tmp:
            path=Path(tmp)/'usage.jsonl'
            path.write_text(json.dumps({'t':BASE_TIME.isoformat(),'status':200})+'\n')
            payload={'path':str(path),'start':BASE_TIME.isoformat(),'stop':BASE_TIME.isoformat()}
            io_backend=g.System()
            self.assertEqual(io_backend.source.rstrip(), EMBEDDED.rstrip())
            result=io_backend.worker('usage',payload,timeout=5)
            self.assertEqual(result['counts'],{'200':1})

    def test_snapshot_skips_200_key_that_cannot_see_inference_model(self):
        import urllib.error
        requests=[]
        class Response:
            status=200
            def __init__(self,value,raw=False):
                self.body=value if raw else json.dumps(value).encode()
            def __enter__(self): return self
            def __exit__(self,*a): pass
            def read(self,n): return self.body
        class Opener:
            authenticated=False
            def open(self,request,timeout):
                url=request.full_url; requests.append((url,request.get_header('Authorization'),request.get_method()))
                if url.endswith('/api/providers') and not self.authenticated:
                    raise urllib.error.HTTPError(url,401,'offline',{},None)
                if '/?k=' in url:
                    self.authenticated=True
                    return Response(b'ok',raw=True)
                if url.endswith('/api/providers'): return Response({'providers':[{'id':'p'}]})
                if url.endswith('/api/groups'): return Response({'groups':[{'id':'g'}]})
                if url.endswith('/api/cooldown'): return Response({'effective':{}})
                if url.endswith('/v1/models'):
                    model='group/unrelated' if request.get_header('Authorization')=='Bearer FIRST' else 'group/fast-cheap'
                    return Response({'data':[{'id':model}]})
                raise AssertionError('unexpected offline HTTP call')
        with patch.object(g.urllib.request,'build_opener',return_value=Opener()):
            result=g.http_work('snapshot',{'web_key':SENTINEL,'keys':['FIRST','SECOND']})
        self.assertEqual(result['key_index'],1)
        self.assertEqual([r[1] for r in requests if r[0].endswith('/v1/models')],['Bearer FIRST','Bearer SECOND'])
        self.assertFalse(any(r[2]=='POST' for r in requests))

    def test_entry_help_runs_without_docker_or_network(self):
        result=subprocess.run([sys.executable,'-I','-B','-c',EMBEDDED,str(ENTRY),'--help'],
                              stdout=subprocess.PIPE,stderr=subprocess.PIPE,env={'PATH':'/usr/bin:/bin'},timeout=10)
        self.assertEqual(result.returncode,0)
        self.assertNotIn(SENTINEL,result.stdout.decode()+result.stderr.decode())


if __name__=='__main__':
    unittest.main(verbosity=2)
