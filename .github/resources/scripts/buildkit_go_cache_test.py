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
"""Exercise the Go mount transfer and cache recovery workflow contracts."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
HELPER = ROOT / '.github/resources/scripts/buildkit-go-cache.sh'
BASE = 'golang:1.27.1-alpine@sha256:' + 'a' * 64


class BuildkitGoCacheTest(unittest.TestCase):

    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.cache = self.directory / 'go-build-cache'
        self.mounts = self.directory / 'mounts'
        self.mounts.mkdir()
        self.commands = self.directory / 'commands.jsonl'
        self.dockerfile = self.directory / 'Dockerfile'
        self.dockerfile.write_text(f'FROM {BASE} AS builder\n')
        binaries = self.directory / 'bin'
        binaries.mkdir()
        docker = binaries / 'docker'
        docker.write_text('''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import re
import shutil
import sys

args = sys.argv[1:]
dockerfile = sys.stdin.read()
with open(os.environ['COMMAND_LOG'], 'a') as log:
    log.write(json.dumps({'args': args, 'dockerfile': dockerfile}) + '\\n')
if os.environ.get('FAIL_TRANSFER') == 'true':
    sys.exit(42)
cache_id = re.search(r'type=cache,id=([^,]+),', dockerfile).group(1)
mount = Path(os.environ['MOUNTS']) / cache_id
mount.mkdir(exist_ok=True)
output = args[args.index('--output') + 1]
if output == 'type=cacheonly':
    shutil.copytree(Path(args[-1]), mount, dirs_exist_ok=True)
else:
    destination = Path(output.removeprefix('type=local,dest='))
    shutil.copytree(mount, destination)
''')
        docker.chmod(0o755)
        timeout = binaries / 'timeout'
        timeout.write_text('''#!/usr/bin/env bash
[[ "$1 $2 $3" == '--foreground --kill-after=10s 120s' ]] || exit 91
shift 3
exec "$@"
''')
        timeout.chmod(0o755)
        self.env = {
            **os.environ,
            'PATH': f'{binaries}:{os.environ["PATH"]}',
            'RUNNER_TEMP': str(self.directory),
            'TARGET_ARCH': 'amd64',
            'COMMAND_LOG': str(self.commands),
            'MOUNTS': str(self.mounts),
        }

    def run_helper(self, mode, cache_id='kfp-go-build', **env):
        return subprocess.run([
            'bash',
            str(HELPER), mode,
            str(self.dockerfile), 'test-builder', cache_id
        ],
                              env={
                                  **self.env,
                                  **env
                              },
                              capture_output=True,
                              text=True,
                              timeout=10,
                              check=False)

    def test_cold_cache_does_not_need_docker(self):
        for exists in (False, True):
            with self.subTest(exists=exists):
                if exists:
                    self.cache.mkdir()
                result = self.run_helper('restore')
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertIn('empty cache mount', result.stdout)
                self.assertFalse(self.commands.exists())

    def test_restore_and_save_transfer_the_same_mount_on_native_builder(self):
        self.cache.mkdir()
        (self.cache / 'existing').write_text('compiled package')
        result = self.run_helper('restore', TARGET_ARCH='arm64')
        self.assertEqual(result.returncode, 0, result.stderr)
        mount = self.mounts / 'kfp-go-build'
        self.assertEqual((mount / 'existing').read_text(), 'compiled package')
        # Simulate Go's update and garbage collection during compilation.
        (mount / 'existing').unlink()
        (mount / 'new').write_text('new compiled package')
        result = self.run_helper('save', TARGET_ARCH='arm64')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.cache / 'new').read_text(),
                         'new compiled package')
        self.assertFalse((self.cache / 'existing').exists())
        calls = [
            json.loads(line) for line in self.commands.read_text().splitlines()
        ]
        self.assertEqual(len(calls), 2)
        for call in calls:
            args = call['args']
            self.assertEqual(args[args.index('--builder') + 1], 'test-builder')
            self.assertEqual(args[args.index('--platform') + 1], 'linux/arm64')
            self.assertIn('--no-cache', args)
            self.assertIn(f'FROM {BASE}', call['dockerfile'])
            self.assertIn(
                'id=kfp-go-build,target=/root/.cache/go-build,sharing=locked',
                call['dockerfile'])
        self.assertEqual(
            list(self.directory.glob('go-build-cache-transfer.*')), [])

    def test_retry_mount_can_be_saved_without_reading_restored_mount(self):
        for cache_id in ('kfp-go-build', 'kfp-go-build-retry'):
            mount = self.mounts / cache_id
            mount.mkdir()
            (mount / 'compiled').write_text(cache_id)
        result = self.run_helper('save', cache_id='kfp-go-build-retry')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual((self.cache / 'compiled').read_text(),
                         'kfp-go-build-retry')

    def test_failed_export_preserves_restored_cache_and_cleans_temporary_files(
            self):
        self.cache.mkdir()
        (self.cache / 'keep').write_text('old cache')
        result = self.run_helper('save', FAIL_TRANSFER='true')
        self.assertEqual(result.returncode, 42, result.stderr)
        self.assertEqual((self.cache / 'keep').read_text(), 'old cache')
        self.assertEqual(
            list(self.directory.glob('go-build-cache-transfer.*')), [])

    def test_failed_restore_propagates_for_the_workflow_to_ignore(self):
        self.cache.mkdir()
        (self.cache / 'entry').touch()
        result = self.run_helper('restore', FAIL_TRANSFER='true')
        self.assertEqual(result.returncode, 42, result.stderr)

    def test_rejects_unpinned_or_non_go_base_before_running_docker(self):
        for base in ('golang:latest', 'alpine@sha256:' + 'a' * 64,
                     'golang:1.27.1@sha256:invalid'):
            with self.subTest(base=base):
                self.dockerfile.write_text(f'FROM {base}\n')
                result = self.run_helper('save')
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.commands.exists())

    def test_rejects_unsupported_architecture_cache_id_and_symlink(self):
        for env, cache_id in (({
                'TARGET_ARCH': 's390x'
        }, 'kfp-go-build'), ({}, 'other-cache')):
            with self.subTest(env=env, cache_id=cache_id):
                result = self.run_helper('save', cache_id=cache_id, **env)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(self.commands.exists())
        self.cache.symlink_to(self.mounts, target_is_directory=True)
        result = self.run_helper('save')
        self.assertNotEqual(result.returncode, 0)
        self.assertFalse(self.commands.exists())


class ImageBuildCacheWorkflowTest(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.workflow = yaml.safe_load(
            (ROOT / '.github/workflows/image-builds.yml').read_text())
        cls.steps = cls.workflow['jobs']['image-build']['steps']
        cls.by_id = {step['id']: step for step in cls.steps if 'id' in step}

    def test_benchmark_dispatch_preserves_manually_dispatched_callers(self):
        events = self.workflow.get('on', self.workflow.get(True))
        self.assertIsNone(events['workflow_dispatch'])
        self.assertEqual(
            events['workflow_call']['inputs']['architecture']['default'],
            'amd64')
        benchmark = self.workflow['jobs']['cache-benchmark']
        image_build = self.workflow['jobs']['image-build']
        path_test = "contains(github.workflow_ref, '/.github/workflows/image-builds.yml@')"
        self.assertEqual(
            benchmark['if'],
            "${{ github.event_name == 'workflow_dispatch' && " + path_test +
            ' }}')
        self.assertEqual(
            image_build['if'],
            "${{ github.event_name != 'workflow_dispatch' || !" + path_test +
            ' }}')
        self.assertEqual(self.workflow['jobs']['runtime-base-images']['if'],
                         "${{ inputs.architecture == 'amd64' }}")
        # The dispatch has no architecture input, while reusable callers get
        # the amd64 default even when the caller itself was dispatched manually.
        for event, workflow, expected in (('workflow_dispatch',
                                           'image-builds.yml', True),
                                          ('workflow_dispatch',
                                           'api-server-tests.yml', False),
                                          ('pull_request', 'e2e-test.yml',
                                           False)):
            with self.subTest(event=event, workflow=workflow):
                ref = f'kubeflow/pipelines/.github/workflows/{workflow}@refs/heads/master'
                self.assertEqual(
                    event == 'workflow_dispatch' and
                    '/.github/workflows/image-builds.yml@' in ref, expected)
        self.assertEqual(benchmark['runs-on'], 'ubuntu-latest')
        self.assertEqual(benchmark['env']['TARGET_ARCH'], 'amd64')
        upload = benchmark['steps'][-1]
        self.assertEqual(upload['if'], 'always()')
        self.assertEqual(upload['with']['retention-days'], 7)

    def test_layers_are_shared_per_image_and_architecture_with_bounded_export(
            self):
        build = self.by_id['save-image']['with']
        for key in ('cache-from', 'cache-to'):
            value = build[key]
            self.assertIn('type=gha,version=2', value)
            self.assertIn(
                'scope=ci-image-v1-${{ matrix.image }}-${{ inputs.architecture }}',
                value)
            self.assertIn('timeout=60s', value)
            self.assertNotIn('github.workflow', value)
        self.assertIn('mode=max', build['cache-to'])
        self.assertIn('ignore-error=true', build['cache-to'])

    def test_uncached_retry_is_isolated_from_both_external_caches(self):
        retry = self.by_id['rebuild']['with']
        self.assertTrue(retry['no-cache'])
        self.assertNotIn('cache-from', retry)
        self.assertNotIn('cache-to', retry)
        self.assertIn('GO_BUILD_CACHE_ID=kfp-go-build-retry',
                      retry['build-args'])
        extract = self.by_id['extract-go-cache']
        self.assertIn("steps.rebuild.outcome == 'success'",
                      extract['env']['CACHE_ID'])
        self.assertIn('kfp-go-build-retry', extract['env']['CACHE_ID'])

    def test_go_cache_refreshes_per_source_with_compatible_builder_fallback(
            self):
        restore = self.by_id['go-cache']
        key = restore['with']['key']
        prefix = restore['with']['restore-keys'].strip()
        self.assertEqual(key, prefix + '${{ github.sha }}')
        for discriminator in ('matrix.image', 'inputs.architecture',
                              'matrix.dockerfile', "'go.mod'", "'go.sum'",
                              "'api/go.mod'", "'kubernetes_platform/go.mod'"):
            self.assertIn(discriminator, prefix)
        save = next(step for step in self.steps
                    if step['name'] == 'Save Go compiler cache')
        self.assertEqual(save['with']['path'], restore['with']['path'])
        self.assertEqual(save['with']['key'],
                         '${{ steps.go-cache.outputs.cache-primary-key }}')

    def test_optional_cache_failure_never_gates_image_upload(self):
        cache_steps = [
            step for step in self.steps if 'Go compiler cache' in step['name']
        ]
        self.assertEqual(len(cache_steps), 4)
        for step in cache_steps:
            self.assertTrue(step['continue-on-error'], step['name'])
            self.assertLessEqual(step['timeout-minutes'], 3)
        upload = next(
            step for step in self.steps if step['name'] == 'Upload artifact')
        self.assertNotIn('cache', upload['if'])
        self.assertLess(
            self.steps.index(upload),
            self.steps.index(self.by_id['extract-go-cache']))
        self.assertIn("steps.go-cache.outputs.cache-hit != 'true'",
                      self.by_id['extract-go-cache']['if'])


if __name__ == '__main__':
    unittest.main()
