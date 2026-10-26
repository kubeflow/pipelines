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

import os
from pathlib import Path
import subprocess
import tempfile
import unittest

SCRIPT = Path(__file__).with_name('load-tensorboard-image.sh')


class LoadTensorboardImageTest(unittest.TestCase):

    def run_loader(self, version, fail_load=False):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            log = root / 'commands'
            for command in ('docker', 'kind'):
                mock = root / command
                mock.write_text(
                    '#!/bin/bash\nprintf "%s\\n" "$(basename "$0") $*" >> "$COMMAND_LOG"\n'
                    'if [[ "${FAIL_LOAD:-}" == 1 && "$1" == image && "$2" == load ]]; then exit 1; fi\n'
                )
                mock.chmod(0o755)
            version_file = root / 'VERSION'
            version_file.write_text(version)
            result = subprocess.run(
                [
                    'bash',
                    str(SCRIPT), 'candidate.tar', 'local/tensorboard:candidate',
                    'test-cluster',
                    str(version_file)
                ],
                env={
                    **os.environ, 'PATH':
                        str(root) + os.pathsep + os.environ['PATH'],
                    'COMMAND_LOG':
                        str(log),
                    'FAIL_LOAD':
                        str(int(fail_load))
                },
                capture_output=True,
                text=True)
            return result, log.read_text().splitlines() if log.exists() else []

    def test_loads_candidate_under_release_default_without_registry_pull(self):
        result, commands = self.run_loader('2.18.0-rc.1\n')
        self.assertEqual(result.returncode, 0, result.stderr)
        image = 'ghcr.io/kubeflow/kfp-tensorboard:2.18.0-rc.1'
        self.assertEqual(commands, [
            'docker image load -i candidate.tar',
            f'docker tag local/tensorboard:candidate {image}',
            f'kind --name test-cluster load docker-image {image}',
            f'docker image rm {image} local/tensorboard:candidate',
        ])

    def test_invalid_version_does_not_load_an_image(self):
        result, commands = self.run_loader('../invalid')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(commands, [])

    def test_failed_archive_load_cannot_fall_back_to_a_registry_image(self):
        result, commands = self.run_loader('2.18.0', fail_load=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(commands, ['docker image load -i candidate.tar'])


if __name__ == '__main__':
    unittest.main()
