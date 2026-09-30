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
'''
        for command in ('docker', 'kind', 'sleep'):
            path = self.bin_directory / command
            path.write_text(command_stub, encoding='utf-8')
            path.chmod(0o755)

    def run_consumer(self, consumer, image=PINNED_IMAGE, failed_command=''):
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
                'bash', '-c', 'source "$1"; "$2" "$3" "$4"',
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

    def consumer_commands(self, consumer, image=IMAGE_TAG):
        if consumer == 'archive':
            return [['docker', 'save', image, '-o', str(self.archive)]]
        return [
            ['kind', '--name', 'test-cluster', 'load', 'docker-image', image],
            ['docker', 'image', 'rm', image],
        ]

    def assert_no_consumption(self, commands):
        self.assertFalse(self.archive.exists())
        self.assertFalse(any(command[0] == 'kind' for command in commands))
        self.assertFalse(
            any(command[:2] == ['docker', 'save'] for command in commands))

    def test_pinned_acquisition_preserves_fixture_tag_in_archive_and_kind(self):
        for consumer in ('archive', 'kind'):
            with self.subTest(consumer=consumer):
                result, commands = self.run_consumer(consumer)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(commands, [
                    ['docker', 'pull', PINNED_IMAGE],
                    ['docker', 'tag', PINNED_IMAGE, IMAGE_TAG],
                ] + self.consumer_commands(consumer))
                if consumer == 'archive':
                    self.assertEqual(self.archive.read_bytes(), b'archive')

    def test_unpinned_images_do_not_require_retagging(self):
        for consumer in ('archive', 'kind'):
            with self.subTest(consumer=consumer):
                result, commands = self.run_consumer(consumer, image=IMAGE_TAG)
                self.assertEqual(result.returncode, 0, result.stderr)
                self.assertEqual(commands, [['docker', 'pull', IMAGE_TAG]] +
                                 self.consumer_commands(consumer))

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
                self.assertEqual(pulls, [['docker', 'pull', PINNED_IMAGE]] * 5)
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
                self.assertEqual(commands, [
                    ['docker', 'pull', PINNED_IMAGE],
                    ['docker', 'tag', PINNED_IMAGE, IMAGE_TAG],
                ])
                self.assert_no_consumption(commands)


if __name__ == '__main__':
    unittest.main()
