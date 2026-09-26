# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Exercise bounded pull retries and the archive's tag identity contract."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[3]
HELPER = ROOT / '.github/resources/scripts/helper-functions.sh'


class RuntimeImagePullsTest(unittest.TestCase):

    def run_helper(self, body, scenario='success'):
        with tempfile.TemporaryDirectory() as directory:
            script = r'''
set -euo pipefail
source "$HELPER"
docker() {
  echo "docker $*" >> "$COMMANDS"
  if [[ "$1" == pull ]]; then
    local count
    count=$(wc -l < "$COMMANDS")
    case "$SCENARIO" in
      quota) echo '429 Too Many Requests: Data limit exceeded' >&2; return 1;;
      exhausted) echo '429 Too Many Requests' >&2; return 1;;
      transient) if [[ "$count" -lt 3 ]]; then echo '429 Too Many Requests' >&2; return 1; fi;;
    esac
  fi
  if [[ "$1" == exec && "$SCENARIO" == missing ]]; then return 1; fi
}
sleep() { echo "sleep $1"; }
kind() {
  if [[ "$*" == 'get nodes --name test' ]]; then
    if [[ "$SCENARIO" != empty ]]; then printf 'node1\nnode2\n'; fi
  else
    echo "kind $*" >> "$COMMANDS"
  fi
}
''' + body
            commands = Path(directory) / 'commands'
            commands.touch()
            images = Path(directory) / 'images'
            images.write_text('example/image:ci@sha256:abc\nplain/image:v1\n')
            result = subprocess.run(
                ['bash', '-c', script],
                env={
                    **os.environ,
                    'HELPER': str(HELPER),
                    'COMMANDS': str(commands),
                    'IMAGES': str(images),
                    'SCENARIO': scenario,
                },
                capture_output=True,
                text=True,
                check=False,
            )
            return result, commands.read_text().splitlines()

    def test_success_never_sleeps(self):
        result, commands = self.run_helper('pull_image_with_backoff image:v1')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(commands, ['docker pull image:v1'])
        self.assertNotIn('sleep', result.stdout)

    def test_throttle_recovers_with_exponential_jitter(self):
        result, commands = self.run_helper('pull_image_with_backoff image:v1',
                                           'transient')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(len(commands), 3)
        delays = [
            int(line.split()[1])
            for line in result.stdout.splitlines()
            if line.startswith('sleep ')
        ]
        self.assertEqual(len(delays), 2)
        self.assertTrue(20 <= delays[0] <= 30)
        self.assertTrue(40 <= delays[1] <= 50)

    def test_exhaustion_stops_at_five_attempts_without_final_sleep(self):
        result, commands = self.run_helper('pull_image_with_backoff image:v1',
                                           'exhausted')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(commands), 5)
        delays = [
            int(line.split()[1])
            for line in result.stdout.splitlines()
            if line.startswith('sleep ')
        ]
        self.assertEqual(len(delays), 4)
        self.assertTrue(80 <= delays[2] <= 90)
        self.assertEqual(delays[3], 120)
        self.assertIn('429 Too Many Requests', result.stderr)
        self.assertIn('after 5 attempts', result.stderr)

    def test_data_quota_fails_without_pointless_retries(self):
        result, commands = self.run_helper('pull_image_with_backoff image:v1',
                                           'quota')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(len(commands), 1)
        self.assertNotIn('sleep', result.stdout)
        self.assertIn('Data limit exceeded', result.stderr)
        self.assertIn('shared cache or an authenticated registry',
                      result.stderr)

    def test_archive_preserves_fixture_tag_after_digest_pinned_pull(self):
        result, commands = self.run_helper(
            'pull_and_save_runtime_base_images "$IMAGES" images.tar')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(commands, [
            'docker pull example/image:ci@sha256:abc',
            'docker tag example/image:ci@sha256:abc example/image:ci',
            'docker pull plain/image:v1',
            'docker save example/image:ci plain/image:v1 -o images.tar',
        ])

    def test_failed_pull_never_tags_or_saves_archive(self):
        result, commands = self.run_helper(
            'pull_and_save_runtime_base_images "$IMAGES" images.tar', 'quota')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(commands, ['docker pull example/image:ci@sha256:abc'])

    def test_kind_verification_checks_every_node_without_registry_pulls(self):
        result, commands = self.run_helper(
            'verify_runtime_base_images_in_kind "$IMAGES" test')
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(commands, [
            'docker exec node1 crictl inspecti example/image:ci',
            'docker exec node2 crictl inspecti example/image:ci',
            'docker exec node1 crictl inspecti plain/image:v1',
            'docker exec node2 crictl inspecti plain/image:v1',
        ])

    def test_missing_loaded_image_fails_setup(self):
        result, _ = self.run_helper(
            'verify_runtime_base_images_in_kind "$IMAGES" test', 'missing')
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('example/image:ci is missing from node1', result.stderr)

    def test_missing_kind_nodes_fail_setup(self):
        result, commands = self.run_helper(
            'verify_runtime_base_images_in_kind "$IMAGES" test', 'empty')
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(commands, [])
        self.assertIn('No Kind nodes', result.stderr)

    def test_deploy_requires_archive_instead_of_fanout_pulls(self):
        action = (ROOT / '.github/actions/deploy/action.yml').read_text()
        self.assertIn('verify_runtime_base_images_in_kind', action)
        self.assertNotIn('load_runtime_base_images_into_kind', action)
        self.assertNotIn('pulling images directly', action)


if __name__ == '__main__':
    unittest.main()
