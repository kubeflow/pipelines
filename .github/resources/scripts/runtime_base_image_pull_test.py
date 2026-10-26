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
"""Exercise runtime image acquisition without a registry or Kind cluster."""

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]
HELPERS = ROOT / '.github/resources/scripts/helper-functions.sh'
IMAGE_TAG = 'docker.io/library/python:3.12'
PINNED_IMAGE = f'{IMAGE_TAG}@sha256:{"a" * 64}'


class RuntimeBaseImagePullTest(unittest.TestCase):

    def setUp(self):
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.directory = Path(self.temporary_directory.name)
        self.bin_directory = self.directory / 'bin'
        self.bin_directory.mkdir()
        self.command_log = self.directory / 'commands.jsonl'
        self.inventory = self.directory / 'images.txt'
        self.archive = self.directory / 'runtime images.tar'
        command_stub = '''#!/usr/bin/env python3
import json
import os
from pathlib import Path
import sys

command = Path(sys.argv[0]).name
arguments = sys.argv[1:]
with open(os.environ['COMMAND_LOG'], 'a') as log:
    log.write(json.dumps([command, *arguments]) + '\\n')
if command == 'docker':
    if arguments[0] == os.environ['FAIL_DOCKER_COMMAND']:
        sys.exit(1)
    if arguments[0] == 'save':
        Path(arguments[arguments.index('-o') + 1]).write_bytes(b'archive')
    if arguments[0] == 'version':
        print('linux/amd64')
    if arguments[0] == 'inspect':
        print('sha256:' + arguments[-1])
    if arguments[:2] == ['image', 'inspect']:
        platforms = json.loads(os.environ.get('NODE_PLATFORMS', '{}'))
        print(platforms.get(arguments[-1], os.environ.get('NODE_PLATFORM', 'linux/arm64')))
if command == 'kind':
    if arguments[:2] == ['get', 'nodes']:
        print(os.environ.get('KIND_NODES', 'test-cluster-control-plane'))
    elif os.environ.get('FAIL_KIND_IMPORT') == 'true':
        sys.exit(1)
'''
        for command in ('docker', 'kind', 'sleep'):
            path = self.bin_directory / command
            path.write_text(command_stub, encoding='utf-8')
            path.chmod(0o755)

    def run_consumer(self,
                     consumer,
                     image=PINNED_IMAGE,
                     failed_command='',
                     **environment):
        self.command_log.write_text('', encoding='utf-8')
        self.inventory.write_text(f'{image}\n', encoding='utf-8')
        if self.archive.exists():
            self.archive.unlink()
        functions = {
            'archive': 'pull_and_save_runtime_base_images',
            'kind': 'load_runtime_base_images_into_kind',
        }
        target = str(self.archive) if consumer == 'archive' else 'test-cluster'
        result = subprocess.run(
            [
                'bash', '-c', 'set -euo pipefail; source "$1"; "$2" "$3" "$4"',
                'runtime-image-test',
                str(HELPERS), functions[consumer],
                str(self.inventory), target
            ],
            cwd=ROOT,
            env={
                **os.environ,
                'PATH':
                    f'{self.bin_directory}:{os.environ["PATH"]}',
                'COMMAND_LOG':
                    str(self.command_log),
                'FAIL_DOCKER_COMMAND':
                    failed_command,
                'TMPDIR':
                    str(self.directory),
                **environment,
            },
            capture_output=True,
            check=False,
            text=True,
            timeout=10,
        )
        commands = [
            json.loads(line)
            for line in self.command_log.read_text().splitlines()
        ]
        return result, commands

    def assert_consumed(self, consumer, commands, image=IMAGE_TAG):
        saves = [c for c in commands if c[:2] == ['docker', 'save']]
        self.assertEqual(len(saves), 1)
        platform = 'linux/amd64' if consumer == 'archive' else 'linux/arm64'
        self.assertEqual(saves[0][2:5], [f'--platform={platform}', image, '-o'])
        archive = Path(saves[0][5])
        if consumer == 'archive':
            self.assertEqual(archive, self.archive)
            self.assertEqual(archive.read_bytes(), b'archive')
        else:
            self.assertIn([
                'kind', '--name', 'test-cluster', 'load', 'image-archive',
                str(archive)
            ], commands)
            self.assertFalse(archive.exists())
            self.assertIn(['docker', 'image', 'rm', image], commands)

    def assert_no_consumption(self, commands):
        self.assertFalse(self.archive.exists())
        self.assertFalse(any('load' in command for command in commands))
        self.assertFalse(
            any(command[:2] == ['docker', 'save'] for command in commands))

    def test_pinned_acquisition_preserves_fixture_tag_in_archive_and_kind(self):
        for consumer in ('archive', 'kind'):
            with self.subTest(consumer=consumer):
                result, commands = self.run_consumer(consumer)
                self.assertEqual(result.returncode, 0, result.stderr)
                platform = ('linux/amd64'
                            if consumer == 'archive' else 'linux/arm64')
                self.assertIn(
                    ['docker', 'pull', f'--platform={platform}', PINNED_IMAGE],
                    commands)
                self.assertIn(['docker', 'tag', PINNED_IMAGE, IMAGE_TAG],
                              commands)
                self.assert_consumed(consumer, commands)

    def test_unpinned_images_do_not_require_retagging(self):
        for consumer in ('archive', 'kind'):
            with self.subTest(consumer=consumer):
                result, commands = self.run_consumer(consumer, image=IMAGE_TAG)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertFalse(
                    any(c[:2] == ['docker', 'tag'] for c in commands))
                self.assert_consumed(consumer, commands)

    def test_pull_failure_stops_before_tagging_or_consumption(self):
        for consumer in ('archive', 'kind'):
            with self.subTest(consumer=consumer):
                result, commands = self.run_consumer(
                    consumer, failed_command='pull')
                self.assertNotEqual(result.returncode, 0)
                pulls = [
                    command for command in commands
                    if command[:2] == ['docker', 'pull']
                ]
                platform = ('linux/amd64'
                            if consumer == 'archive' else 'linux/arm64')
                self.assertEqual(
                    pulls,
                    [['docker', 'pull', f'--platform={platform}', PINNED_IMAGE]
                    ] * 5)
                self.assertEqual([
                    command[1] for command in commands if command[0] == 'sleep'
                ], ['20', '40', '60', '80'])
                self.assertFalse(
                    any(command[:2] == ['docker', 'tag']
                        for command in commands))
                self.assert_no_consumption(commands)

    def test_tag_failure_stops_before_consumption(self):
        for consumer in ('archive', 'kind'):
            with self.subTest(consumer=consumer):
                result, commands = self.run_consumer(
                    consumer, failed_command='tag')
                self.assertNotEqual(result.returncode, 0)
                self.assertIn(['docker', 'tag', PINNED_IMAGE, IMAGE_TAG],
                              commands)
                self.assert_no_consumption(commands)

    def test_kind_uses_node_image_platform_even_with_other_docker_default(self):
        result, commands = self.run_consumer(
            'kind', DOCKER_DEFAULT_PLATFORM='linux/amd64')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn([
            'docker', 'inspect', '--format', '{{.Image}}',
            'test-cluster-control-plane'
        ], commands)
        inspections = [
            c for c in commands if c[:3] == ['docker', 'image', 'inspect']
        ]
        self.assertEqual(len(inspections), 1)
        self.assertEqual(inspections[0][-1],
                         'sha256:test-cluster-control-plane')
        self.assert_consumed('kind', commands)

    def test_producer_ignores_other_docker_default_platform(self):
        result, commands = self.run_consumer(
            'archive', DOCKER_DEFAULT_PLATFORM='linux/arm64')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assert_consumed('archive', commands)

    def test_kind_mixed_node_platforms_stop_before_pull(self):
        result, commands = self.run_consumer(
            'kind',
            KIND_NODES='test-cluster-control-plane\ntest-cluster-worker',
            NODE_PLATFORMS=json.dumps(
                {'sha256:test-cluster-worker': 'linux/amd64'}))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('different platforms', result.stderr)
        self.assertFalse(any(c[:2] == ['docker', 'pull'] for c in commands))
        self.assert_no_consumption(commands)

    def test_kind_missing_nodes_or_platform_stops_before_pull(self):
        for environment in ({'KIND_NODES': ''}, {'NODE_PLATFORM': ''}):
            with self.subTest(environment=environment):
                result, commands = self.run_consumer('kind', **environment)
                self.assertNotEqual(result.returncode, 0)
                self.assertFalse(
                    any(c[:2] == ['docker', 'pull'] for c in commands))
                self.assert_no_consumption(commands)

    def test_kind_save_or_import_failure_removes_temporary_archive(self):
        for environment in ({
                'failed_command': 'save'
        }, {
                'FAIL_KIND_IMPORT': 'true'
        }):
            with self.subTest(environment=environment):
                result, commands = self.run_consumer('kind', **environment)
                self.assertNotEqual(result.returncode, 0)
                saves = [c for c in commands if c[:2] == ['docker', 'save']]
                self.assertEqual(len(saves), 1)
                self.assertFalse(Path(saves[0][-1]).exists())
                self.assertNotIn(['docker', 'image', 'rm', IMAGE_TAG], commands)


if __name__ == '__main__':
    unittest.main()
