#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Execute ready-image downloads/imports with isolated GitHub/Kind/Docker
fakes."""

import fcntl
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import time
import unittest

SCRIPTS = Path(__file__).resolve().parent
ARTIFACTS = ('apiserver', 'scheduledworkflow', 'persistenceagent', 'frontend',
             'viewer-crd-controller', 'driver', 'launcher',
             'runtime-base-images')
ARTIFACT_IDS = {name: 100 + index for index, name in enumerate(ARTIFACTS)}

# Commands share a locked event log so concurrency and cancellation are tested
# across the actual shell/Python worker processes, rather than simulated calls.
FAKE_COMMAND = r'''#!/usr/bin/env python3
import fcntl
import io
import json
import os
from pathlib import Path
import stat
import subprocess
import sys
import time
import zipfile

command = Path(sys.argv[0]).name
args = sys.argv[1:]
state_path = Path(os.environ['FAKE_STATE'])
config = json.loads(Path(os.environ['FAKE_CONFIG']).read_text())


def update(callback):
    with state_path.open('r+') as stream:
        fcntl.flock(stream, fcntl.LOCK_EX)
        state = json.load(stream)
        result = callback(state)
        stream.seek(0)
        json.dump(state, stream)
        stream.truncate()
        return result


def event(state):
    state['events'].append([command, args])


update(event)
if command == 'timeout':
    assert args[:3] == ['--foreground', '--kill-after=30s', '10m'], args
    os.execvp(args[3], args[3:])
if command == 'sleep':
    sys.exit(0)
if command == 'gh':
    assert len(args) == 2 and args[0] == 'api', args
    prefix = 'repos/kubeflow/pipelines/actions/artifacts/'
    assert args[1].startswith(prefix) and args[1].endswith('/zip'), args
    artifact_id = args[1][len(prefix):-len('/zip')]
    artifact = config['artifact_ids'][artifact_id]
    destination = Path(os.environ['IMAGE_PATH']) / artifact

    def start(state):
        attempt = state['attempts'].get(artifact, 0) + 1
        state['attempts'][artifact] = attempt
        state['before_download'].append([
            artifact, attempt, sorted(p.name for p in destination.glob('*.tar'))])
        state['active'] += 1
        state['max_active'] = max(state['active'], state['max_active'])
        return attempt

    attempt = update(start)
    modes = config.get('downloads', {}).get(artifact, ['good'])
    mode = modes[min(attempt - 1, len(modes) - 1)]
    if mode == 'block':
        child = subprocess.Popen([sys.executable, '-c', 'import time; time.sleep(60)'])
        update(lambda state: state['blocked'].append([os.getpid(), child.pid]))
        time.sleep(60)
    time.sleep(0.08)
    buffer = io.BytesIO()
    with zipfile.ZipFile(buffer, 'w') as archive:
        if mode == 'unsafe':
            entry = zipfile.ZipInfo(artifact + '.tar')
            entry.create_system = 3
            entry.external_attr = (stat.S_IFLNK | 0o777) << 16
            archive.writestr(entry, config['outside'])
        elif mode not in ('missing', 'modelcar-only'):
            archive.writestr(artifact + '.tar',
                             '' if mode == 'empty' else 'artifact ID ' + artifact_id)
        if mode == 'traversal':
            archive.writestr('../outside', 'overwritten')
        if mode == 'absolute':
            archive.writestr(config['outside'], 'overwritten')
        if mode == 'duplicate':
            archive.writestr(artifact + '.tar', 'ambiguous duplicate')
        if mode == 'extra':
            archive.writestr('unrelated', 'overwritten')
        if artifact == 'runtime-base-images' and mode in ('good', 'modelcar-only'):
            archive.writestr('modelcar.tar', 'modelcar archive')
    content = buffer.getvalue()
    sys.stdout.buffer.write(content[:20] if mode in ('transport', 'corrupt') else content)
    update(lambda state: state.update(active=state['active'] - 1))
    sys.exit(1 if mode == 'transport' else 0)
if command == 'kind':
    if args[:2] == ['get', 'nodes']:
        print('node-one\nnode-two')
    if 'image-archive' in args:
        archive = Path(args[-1])
        assert archive.is_file() and archive.stat().st_size, args
        def count(state):
            count = state['imports'].get(archive.name, 0) + 1
            state['imports'][archive.name] = count
            state['loaded_content'].append(archive.read_text())
            return count
        attempt = update(count)
        if attempt <= config.get('import_failures', {}).get(archive.name, 0):
            sys.exit(42)
    if 'docker-image' in args and config.get('fail_docker_image_import'):
        sys.exit(43)
if command == 'docker':
    if args[:2] == ['image', 'load']:
        assert Path(args[-1]).is_file(), args
    if args[:1] == ['tag'] and config.get('fail_tag'):
        sys.exit(44)
    if 'inspecti' in args and config.get('fail_verify'):
        sys.exit(45)
'''


class LoadReadyImageArtifactsTest(unittest.TestCase):

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name).resolve()
        self.scripts = self.root / 'resources/scripts'
        self.scripts.mkdir(parents=True)
        for name in ('load-ready-image-artifacts.sh', 'artifact-files.sh',
                     'ci-image-artifacts.sh', 'helper-functions.sh',
                     'load-kind-image-archives.sh'):
            shutil.copy2(SCRIPTS / name, self.scripts / name)
        (self.root / 'resources/runtime-base-images.txt').write_text(
            '# runtime inventory\nregistry/runtime:one\n'
            'registry/runtime:two@sha256:1234\n',
            encoding='utf-8')
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        for name in ('gh', 'kind', 'docker', 'sleep', 'timeout'):
            executable = self.bin / name
            executable.write_text(FAKE_COMMAND, encoding='utf-8')
            executable.chmod(0o755)
        self.state_file = self.root / 'state.json'
        self.state_file.write_text(
            json.dumps({
                'events': [],
                'attempts': {},
                'before_download': [],
                'active': 0,
                'max_active': 0,
                'blocked': [],
                'imports': {},
                'loaded_content': [],
            }),
            encoding='utf-8')
        self.config_file = self.root / 'config.json'
        self.configure()
        self.images = self.root / 'images'
        self.env = {
            **os.environ,
            'PATH':
                str(self.bin) + os.pathsep + os.environ['PATH'],
            'GITHUB_REPOSITORY':
                'kubeflow/pipelines',
            'GITHUB_RUN_ID':
                '12345',
            'GH_TOKEN':
                'test-token',
            'IMAGE_PATH':
                str(self.images),
            'CLUSTER_NAME':
                'test-cluster',
            'IMAGE_REGISTRY':
                'test-registry',
            'IMAGE_TAG':
                'test-sha',
            'LOAD_MODELCAR_FIXTURE':
                'true',
            'FAKE_STATE':
                str(self.state_file),
            'FAKE_CONFIG':
                str(self.config_file),
        }
        self.script = self.scripts / 'load-ready-image-artifacts.sh'

    def configure(self, **settings):
        settings.setdefault('artifact_ids', {
            str(artifact_id): name
            for name, artifact_id in ARTIFACT_IDS.items()
        })
        self.config_file.write_text(json.dumps(settings), encoding='utf-8')

    def state(self):
        # Wait for all child processes before reading except in the cancellation
        # test, where readers take the same file lock as the command fakes.
        with self.state_file.open() as stream:
            fcntl.flock(stream, fcntl.LOCK_SH)
            return json.load(stream)

    def command(self, *artifacts):
        return [
            str(self.script), *[
                f'{name}:{ARTIFACT_IDS[name]}' if name in ARTIFACT_IDS else name
                for name in artifacts
            ]
        ]

    def run_loader(self, *artifacts):
        return subprocess.run(
            self.command(*artifacts),
            env=self.env,
            text=True,
            capture_output=True,
            timeout=15)

    def commands(self, name):
        return [
            args for command, args in self.state()['events'] if command == name
        ]

    def archive(self, artifact, filename=None):
        return self.images / artifact / (filename or artifact + '.tar')

    def assert_success(self, result):
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_selected_id_downloads_correct_duplicate_named_artifact(self):
        self.configure(artifact_ids={'100': 'apiserver', '999': 'apiserver'})
        result = self.run_loader('apiserver:999')
        self.assert_success(result)
        self.assertEqual(
            self.commands('gh'),
            [['api', 'repos/kubeflow/pipelines/actions/artifacts/999/zip']])
        self.assertEqual(self.state()['loaded_content'], ['artifact ID 999'])

    def test_duplicate_names_with_distinct_ids_fail_before_download(self):
        result = self.run_loader('apiserver:100', 'apiserver:999')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.commands('gh'), [])

    def test_corrupt_zip_retries_without_importing_partial_files(self):
        self.configure(downloads={'apiserver': ['corrupt', 'good']})
        self.assert_success(self.run_loader('apiserver'))
        self.assertEqual(self.state()['attempts'], {'apiserver': 2})
        self.assertEqual(self.state()['imports'], {'apiserver.tar': 1})

    def test_unsafe_zip_paths_and_duplicate_members_are_fatal(self):
        outside = self.images / 'outside'
        self.images.mkdir()
        outside.write_text('preserve me', encoding='utf-8')
        for mode, artifact in zip(('traversal', 'absolute', 'duplicate'),
                                  ARTIFACTS):
            with self.subTest(mode=mode):
                self.configure(
                    downloads={artifact: [mode]}, outside=str(outside))
                result = self.run_loader(artifact)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.state()['attempts'][artifact], 1)
                self.assertFalse(self.archive(artifact).exists())
                self.assertEqual(
                    outside.read_text(encoding='utf-8'), 'preserve me')
        self.assertEqual(self.commands('kind'), [])
        self.assertEqual(self.commands('sleep'), [])

    def test_extracts_only_required_files(self):
        destination = self.images / 'apiserver'
        destination.mkdir(parents=True)
        unrelated = destination / 'unrelated'
        unrelated.write_text('preserve me', encoding='utf-8')
        self.configure(downloads={'apiserver': ['extra']})
        self.assert_success(self.run_loader('apiserver'))
        self.assertEqual(unrelated.read_text(encoding='utf-8'), 'preserve me')

    def test_every_artifact_imports_with_existing_tags_and_runtime_checks(self):
        self.assertTrue(os.access(SCRIPTS / self.script.name, os.X_OK))
        result = self.run_loader(*ARTIFACTS)
        self.assert_success(result)
        state = self.state()
        self.assertEqual(state['attempts'], dict.fromkeys(ARTIFACTS, 1))
        self.assertEqual(state['max_active'], 2)
        self.assertEqual(state['active'], 0)
        for artifact in ARTIFACTS:
            self.assertFalse(self.archive(artifact).exists())
            for phase in ('download started', 'download completed',
                          'import started', 'import completed'):
                self.assertIn(f'[{artifact}] {phase}', result.stdout)
        self.assertFalse(
            self.archive('runtime-base-images', 'modelcar.tar').exists())
        kind_calls = self.commands('kind')
        docker_calls = self.commands('docker')
        for artifact in ARTIFACTS[:5] + ('runtime-base-images',):
            self.assertIn([
                '--name', 'test-cluster', 'load', 'image-archive',
                str(self.archive(artifact))
            ], kind_calls)
        self.assertIn([
            '--name', 'test-cluster', 'load', 'image-archive',
            str(self.archive('runtime-base-images', 'modelcar.tar'))
        ], kind_calls)
        for artifact in ('driver', 'launcher'):
            original = f'test-registry/{artifact}:test-sha'
            loaded = f'test-registry/{artifact}:ci'
            self.assertIn(['image', 'load', '-i',
                           str(self.archive(artifact))], docker_calls)
            self.assertIn(['tag', original, loaded], docker_calls)
            self.assertIn(
                ['--name', 'test-cluster', 'load', 'docker-image', loaded],
                kind_calls)
            self.assertIn(['image', 'rm', original], docker_calls)
            self.assertIn(['image', 'rm', loaded], docker_calls)
        for node in ('node-one', 'node-two'):
            for image in ('registry/runtime:one', 'registry/runtime:two'):
                self.assertIn(['exec', node, 'crictl', 'inspecti', image],
                              docker_calls)
        self.assertFalse(any('pull' in args for args in docker_calls))

    def test_optional_modelcar_is_not_required_or_imported(self):
        self.env['LOAD_MODELCAR_FIXTURE'] = 'false'
        self.configure(downloads={'runtime-base-images': ['base-only']})
        self.assert_success(self.run_loader('runtime-base-images'))
        self.assertEqual(self.state()['imports'],
                         {'runtime-base-images.tar': 1})

    def test_transport_failure_retries_after_removing_partial_archive(self):
        self.configure(downloads={'apiserver': ['transport', 'good']})
        self.assert_success(self.run_loader('apiserver'))
        self.assertEqual(self.state()['before_download'],
                         [['apiserver', 1, []], ['apiserver', 2, []]])
        self.assertEqual(self.commands('sleep'), [['20']])

    def test_empty_or_missing_archives_retry_and_never_import(self):
        for mode in ('empty', 'missing'):
            with self.subTest(mode=mode):
                # A separate artifact keeps attempt counters independent.
                artifact = 'apiserver' if mode == 'empty' else 'frontend'
                self.configure(downloads={artifact: [mode]})
                result = self.run_loader(artifact)
                self.assertNotEqual(result.returncode, 0)
                self.assertEqual(self.state()['attempts'][artifact], 2)
                self.assertNotIn(f'[{artifact}] import started', result.stdout)
        self.assertEqual(self.commands('kind'), [])

    def test_modelcar_is_required_before_importing_runtime_archive(self):
        self.configure(downloads={'runtime-base-images': ['base-only']})
        result = self.run_loader('runtime-base-images')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.state()['attempts']['runtime-base-images'], 2)
        self.assertEqual(self.commands('kind'), [])
        self.assertTrue(self.archive('runtime-base-images').exists())

    def test_retry_cannot_combine_files_from_incomplete_downloads(self):
        self.configure(
            downloads={'runtime-base-images': ['base-only', 'modelcar-only']})
        result = self.run_loader('runtime-base-images')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(
            self.state()['before_download'],
            [['runtime-base-images', 1, []], ['runtime-base-images', 2, []]])
        self.assertEqual(self.commands('kind'), [])
        self.assertFalse(self.archive('runtime-base-images').exists())
        self.assertTrue(
            self.archive('runtime-base-images', 'modelcar.tar').exists())

    def test_unsafe_download_is_fatal_without_retry(self):
        outside = self.root / 'outside'
        outside.write_text('preserve me', encoding='utf-8')
        self.configure(
            downloads={'apiserver': ['unsafe']}, outside=str(outside))
        result = self.run_loader('apiserver')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.state()['attempts'], {'apiserver': 1})
        self.assertEqual(self.commands('kind'), [])
        self.assertEqual(self.commands('sleep'), [])
        self.assertEqual(outside.read_text(encoding='utf-8'), 'preserve me')

    def test_unsafe_destination_fails_before_downloading(self):
        self.images.symlink_to(
            self.root / 'elsewhere', target_is_directory=True)
        result = self.run_loader('apiserver')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(self.commands('gh'), [])
        self.assertEqual(self.commands('kind'), [])

    def test_import_failure_retains_archive(self):
        self.configure(import_failures={'apiserver.tar': 99})
        result = self.run_loader('apiserver')
        self.assertNotEqual(result.returncode, 0)
        self.assertTrue(self.archive('apiserver').exists())
        self.assertNotIn('[apiserver] import completed', result.stdout)

    def test_driver_tag_or_kind_failure_retains_archive(self):
        for setting in ('fail_tag', 'fail_docker_image_import'):
            with self.subTest(setting=setting):
                self.configure(**{setting: True})
                result = self.run_loader('driver')
                self.assertNotEqual(result.returncode, 0)
                self.assertTrue(self.archive('driver').exists())
                self.assertNotIn('[driver] import completed', result.stdout)

    def test_runtime_and_modelcar_imports_keep_three_attempt_retry(self):
        self.configure(import_failures={
            'runtime-base-images.tar': 2,
            'modelcar.tar': 2
        })
        self.assert_success(self.run_loader('runtime-base-images'))
        self.assertEqual(self.state()['imports'], {
            'runtime-base-images.tar': 3,
            'modelcar.tar': 3
        })
        self.assertEqual(self.commands('sleep'), [['10']] * 4)

    def test_missing_cri_image_fails_with_diagnostics_and_retains_archive(self):
        self.configure(fail_verify=True)
        result = self.run_loader('runtime-base-images')
        self.assertNotEqual(result.returncode, 0)
        calls = self.commands('docker')
        inspections = [args for args in calls if 'inspecti' in args]
        self.assertEqual(len(inspections), 10)
        self.assertIn(['exec', 'node-one', 'crictl', 'images'], calls)
        self.assertIn(
            ['exec', 'node-one', 'ctr', '--namespace=k8s.io', 'images', 'ls'],
            calls)
        self.assertTrue(self.archive('runtime-base-images').exists())
        self.assertTrue(
            self.archive('runtime-base-images', 'modelcar.tar').exists())
        self.assertFalse(any('pull' in args for args in calls))

    def test_invalid_arguments_fail_before_starting_workers(self):
        for arguments in ((), ('unknown:100',), ('apiserver', 'apiserver'),
                          ('apiserver:',), ('apiserver:no',), ('apiserver:0',),
                          ('apiserver:01',), ('apiserver:100:200',)):
            with self.subTest(arguments=arguments):
                self.assertNotEqual(self.run_loader(*arguments).returncode, 0)
        self.assertEqual(self.commands('gh'), [])

    def test_failed_worker_stops_another_worker_and_its_descendants(self):
        self.configure(
            downloads={'apiserver': ['block']},
            import_failures={'frontend.tar': 99})
        result = self.run_loader('apiserver', 'frontend', 'driver')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(self.state()['blocked']), 1)
        self.assertNotIn('driver', self.state()['attempts'])
        self.assertIn('stopping other workers', result.stderr)
        # A live descendant inherits stdout/stderr and would prevent the
        # subprocess call from finishing before its 60-second sleep ends.

    def test_cancellation_stops_both_workers_and_their_descendants(self):
        self.configure(downloads={
            'apiserver': ['block'],
            'frontend': ['block']
        })
        process = subprocess.Popen(
            self.command('apiserver', 'frontend'),
            env=self.env,
            text=True,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE)
        try:
            deadline = time.monotonic() + 5
            while len(self.state()
                      ['blocked']) != 2 and time.monotonic() < deadline:
                time.sleep(0.02)
            self.assertEqual(len(self.state()['blocked']), 2)
            process.terminate()
            stdout, stderr = process.communicate(timeout=8)
            self.assertEqual(process.returncode, 143, stdout + stderr)
        finally:
            if process.poll() is None:
                process.kill()
            process.communicate(timeout=8)


if __name__ == '__main__':
    unittest.main()
