#!/usr/bin/env python3

# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest

from configure_docker_registry_mirror import configure_registry_mirror

REPOSITORY_ROOT = Path(__file__).resolve().parents[3]
CONFIGURE_MIRROR_SCRIPT = (
    REPOSITORY_ROOT /
    '.github/resources/scripts/configure-docker-hub-mirror.sh')
MIRROR_SCRIPT = (
    'bash ./.github/resources/scripts/configure-docker-hub-mirror.sh')


class DockerRegistryMirrorTest(unittest.TestCase):

    def setUp(self):
        self.temp_directory = tempfile.TemporaryDirectory()
        self.config_path = Path(self.temp_directory.name) / 'daemon.json'

    def tearDown(self):
        self.temp_directory.cleanup()

    def test_adds_mirror_without_discarding_existing_configuration(self):
        self.config_path.write_text(
            json.dumps({
                'features': {
                    'containerd-snapshotter': True
                },
                'registry-mirrors': ['https://registry.example.com'],
            }),
            encoding='utf-8',
        )
        changed = configure_registry_mirror(self.config_path,
                                            'https://mirror.gcr.io')

        self.assertTrue(changed)
        self.assertEqual(
            json.loads(self.config_path.read_text(encoding='utf-8')),
            {
                'features': {
                    'containerd-snapshotter': True
                },
                'registry-mirrors': [
                    'https://mirror.gcr.io',
                    'https://registry.example.com',
                ],
            },
        )

    def test_creates_missing_configuration(self):
        changed = configure_registry_mirror(self.config_path,
                                            'https://mirror.gcr.io')

        self.assertTrue(changed)
        self.assertEqual(
            json.loads(self.config_path.read_text(encoding='utf-8')),
            {'registry-mirrors': ['https://mirror.gcr.io']},
        )

    def test_is_idempotent_with_equivalent_trailing_slash(self):
        self.config_path.write_text(
            json.dumps({'registry-mirrors': ['https://mirror.gcr.io/']}),
            encoding='utf-8',
        )

        changed = configure_registry_mirror(self.config_path,
                                            'https://mirror.gcr.io')

        self.assertFalse(changed)
        self.assertEqual(
            json.loads(self.config_path.read_text(encoding='utf-8')),
            {'registry-mirrors': ['https://mirror.gcr.io/']},
        )

    def test_rejects_invalid_existing_mirror_configuration(self):
        self.config_path.write_text(
            json.dumps({'registry-mirrors': 'https://mirror.example.com'}),
            encoding='utf-8',
        )

        with self.assertRaisesRegex(ValueError, 'list of strings'):
            configure_registry_mirror(self.config_path, 'https://mirror.gcr.io')

    def test_rejects_non_object_daemon_configuration(self):
        self.config_path.write_text('[]', encoding='utf-8')

        with self.assertRaisesRegex(ValueError, 'JSON object'):
            configure_registry_mirror(self.config_path, 'https://mirror.gcr.io')

    def test_hosted_docker_workflows_configure_mirror(self):
        paths_and_following_steps = {
            '.github/actions/create-cluster/action.yml':
                'Restore Kind node image cache',
            '.github/actions/build-tools-images/action.yml':
                'Build api-generator',
            '.github/workflows/image-builds.yml':
                'Set up Docker Buildx',
            '.github/workflows/runtime-base-images.yml':
                'Configure runtime base images',
        }

        for relative_path, following_step in paths_and_following_steps.items():
            with self.subTest(path=relative_path):
                contents = (REPOSITORY_ROOT /
                            relative_path).read_text(encoding='utf-8')
                self.assertIn(MIRROR_SCRIPT, contents)
                self.assertLess(
                    contents.index('Configure Docker Hub mirror'),
                    contents.index(following_step),
                )

    def test_provider_managed_docker_requires_health_without_mutation(self):
        for service in ('missing', 'inactive'):
            for healthy in (True, False):
                with self.subTest(service=service, healthy=healthy):
                    with tempfile.TemporaryDirectory() as directory:
                        root = Path(directory)
                        binaries = root / 'bin'
                        binaries.mkdir()
                        (binaries / 'bash').symlink_to('/bin/bash')
                        commands = {
                            'docker':
                                '#!/bin/bash\nexit %s\n' %
                                (0 if healthy else 1),
                            'sleep':
                                '#!/bin/bash\nexit 0\n',
                            'sudo':
                                '#!/bin/bash\necho unexpected-sudo >&2\nexit 99\n',
                        }
                        if service == 'inactive':
                            commands['systemctl'] = '#!/bin/bash\nexit 1\n'
                        for name, content in commands.items():
                            path = binaries / name
                            path.write_text(content, encoding='utf-8')
                            path.chmod(0o755)
                        config = root / 'daemon.json'
                        config.write_text(
                            '{"provider": true}', encoding='utf-8')
                        result = subprocess.run(
                            ['/bin/bash',
                             str(CONFIGURE_MIRROR_SCRIPT)],
                            env={
                                **os.environ, 'PATH': str(binaries),
                                'DOCKER_DAEMON_CONFIG': str(config)
                            },
                            capture_output=True,
                            text=True,
                            check=False)
                        self.assertEqual(result.returncode, 0 if healthy else 1,
                                         result.stderr)
                        self.assertEqual(
                            config.read_text(encoding='utf-8'),
                            '{"provider": true}')
                        self.assertNotIn('unexpected-sudo', result.stderr)
                        if healthy:
                            self.assertIn('provider-managed', result.stdout)

    def test_shell_wrapper_restarts_docker_only_when_config_changes(self):
        binary_directory = Path(self.temp_directory.name) / 'bin'
        binary_directory.mkdir()
        command_log = Path(self.temp_directory.name) / 'commands.log'
        for command_name, contents in {
                'docker': (
                    '#!/usr/bin/env bash\n'
                    'echo "docker $*" >> "$COMMAND_LOG"\n'
                    'if [[ "$1" == "info" ]]; then\n'
                    '  echo "Registry Mirrors: ${DOCKER_HUB_MIRROR:-https://mirror.gcr.io}/"\n'
                    'fi\n'),
                'sudo':
                    '#!/usr/bin/env bash\nunset LD_LIBRARY_PATH\nexec "$@"\n',
                # Model a setup-python executable that cannot load after sudo
                # strips its shared-library environment.
                'python3': ('#!/usr/bin/env bash\n'
                            'echo missing-libpython >&2\nexit 127\n'),
                'systemctl': ('#!/usr/bin/env bash\n'
                              'echo "systemctl $*" >> "$COMMAND_LOG"\n'),
        }.items():
            command_path = binary_directory / command_name
            command_path.write_text(contents, encoding='utf-8')
            command_path.chmod(0o755)

        environment = {
            **os.environ,
            'COMMAND_LOG': str(command_log),
            'DOCKER_DAEMON_CONFIG': str(self.config_path),
            'PATH': f'{binary_directory}:{os.environ["PATH"]}',
        }
        for _ in range(2):
            result = subprocess.run(
                ['bash', str(CONFIGURE_MIRROR_SCRIPT)],
                capture_output=True,
                check=False,
                env=environment,
                text=True,
            )
            self.assertEqual(result.returncode, 0, result.stderr)

        commands = command_log.read_text(encoding='utf-8').splitlines()
        self.assertEqual(commands.count('systemctl restart docker'), 1)
        self.assertEqual(commands.count('docker info'), 2)


if __name__ == '__main__':
    unittest.main()
