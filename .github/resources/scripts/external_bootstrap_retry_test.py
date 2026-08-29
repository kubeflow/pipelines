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

import http.server
import os
import pathlib
import shutil
import subprocess
import tempfile
import threading
import unittest

SCRIPTS_DIR = pathlib.Path(__file__).resolve().parent
REPO_ROOT = SCRIPTS_DIR.parents[2]
SETUP_BUILDX_SCRIPT = SCRIPTS_DIR / 'setup-buildx-with-retry.sh'
CURL_CONFIG = REPO_ROOT / '.github/resources/curl-retry/.curlrc'
CREATE_CLUSTER_ACTION = REPO_ROOT / '.github/actions/create-cluster/action.yml'
CREATE_MANIFEST_WORKFLOW = REPO_ROOT / '.github/workflows/create-manifest.yml'
CI_SCRIPTS_WORKFLOW = REPO_ROOT / '.github/workflows/ci-scripts-tests.yml'
BUILDKIT_IMAGE = 'moby/buildkit:buildx-stable-1'
BUILDKIT_MIRROR_IMAGE = 'mirror.gcr.io/moby/buildkit:buildx-stable-1'
BUILDKIT_CONFIG = REPO_ROOT / '.github/resources/buildkitd.toml'


class _RetryHandler(http.server.BaseHTTPRequestHandler):

    request_count = 0

    def do_GET(self):
        type(self).request_count += 1
        if type(self).request_count < 3:
            self.send_response(503)
            self.end_headers()
            return

        self.send_response(200)
        self.end_headers()
        self.wfile.write(b'ok')

    def log_message(self, format, *args):
        pass


class ExternalBootstrapRetryTest(unittest.TestCase):

    def setUp(self):
        self.temp_dir = tempfile.TemporaryDirectory()
        self.temp_path = pathlib.Path(self.temp_dir.name)
        self.bin_dir = self.temp_path / 'bin'
        self.bin_dir.mkdir()
        self.command_log = self.temp_path / 'docker.log'
        self.sleep_log = self.temp_path / 'sleep.log'
        self.github_output = self.temp_path / 'github-output'
        self._write_fake_commands()

    def tearDown(self):
        self.temp_dir.cleanup()

    def _write_fake_commands(self):
        docker = self.bin_dir / 'docker'
        docker.write_text('#!/usr/bin/env bash\n'
                          'printf \'%s\\n\' "$*" >> "$COMMAND_LOG"\n'
                          '[[ "$1 $2" == "pull $MIRROR_IMAGE" ]] && '
                          '  [[ "$FAIL_MIRROR_PULL" == "true" ]] && exit 1\n'
                          '[[ "$1 $2" == "pull $CANONICAL_IMAGE" ]] && '
                          '  [[ "$FAIL_CANONICAL_PULL" == "true" ]] && exit 1\n'
                          'exit 0\n')
        docker.chmod(0o755)

        sleep = self.bin_dir / 'sleep'
        sleep.write_text('#!/usr/bin/env bash\n'
                         'printf \'%s\\n\' "$1" >> "$SLEEP_LOG"\n')
        sleep.chmod(0o755)

    def _run_setup_buildx(self,
                          fail_mirror_pull=False,
                          fail_canonical_pull=False):
        environment = os.environ.copy()
        environment.update({
            'CANONICAL_IMAGE': BUILDKIT_IMAGE,
            'COMMAND_LOG': str(self.command_log),
            'FAIL_CANONICAL_PULL': str(fail_canonical_pull).lower(),
            'FAIL_MIRROR_PULL': str(fail_mirror_pull).lower(),
            'GITHUB_OUTPUT': str(self.github_output),
            'MIRROR_IMAGE': BUILDKIT_MIRROR_IMAGE,
            'PATH': f'{self.bin_dir}:{environment["PATH"]}',
            'SLEEP_LOG': str(self.sleep_log),
        })
        return subprocess.run(
            ['bash', str(SETUP_BUILDX_SCRIPT), 'test-builder'],
            capture_output=True,
            check=False,
            env=environment,
            text=True,
        )

    def test_buildkit_pull_uses_mirror_before_bootstrap(self):
        result = self._run_setup_buildx()

        self.assertEqual(result.returncode, 0, result.stderr)
        commands = self.command_log.read_text().splitlines()
        self.assertEqual(
            commands[:2],
            [
                f'pull {BUILDKIT_MIRROR_IMAGE}',
                f'tag {BUILDKIT_MIRROR_IMAGE} {BUILDKIT_IMAGE}',
            ],
        )
        self.assertIn(
            f'buildx create --name test-builder --driver docker-container '
            f'--driver-opt image={BUILDKIT_IMAGE} '
            f'--buildkitd-config {BUILDKIT_CONFIG} --use', commands)
        self.assertFalse(self.sleep_log.exists())
        self.assertEqual(self.github_output.read_text(),
                         'builder_name=test-builder\n')

    def test_buildkit_pull_exhaustion_stops_before_bootstrap(self):
        result = self._run_setup_buildx(
            fail_mirror_pull=True, fail_canonical_pull=True)

        self.assertNotEqual(result.returncode, 0)
        commands = self.command_log.read_text().splitlines()
        self.assertEqual(
            commands,
            [
                f'pull {BUILDKIT_MIRROR_IMAGE}',
                f'pull {BUILDKIT_IMAGE}',
            ] * 5,
        )
        self.assertEqual(self.sleep_log.read_text().splitlines(), ['20'] * 4)
        self.assertNotIn('buildx create', '\n'.join(commands))

    def test_buildkit_pull_falls_back_to_docker_hub_in_same_attempt(self):
        result = self._run_setup_buildx(fail_mirror_pull=True)

        self.assertEqual(result.returncode, 0, result.stderr)
        commands = self.command_log.read_text().splitlines()
        self.assertEqual(
            commands[:2],
            [
                f'pull {BUILDKIT_MIRROR_IMAGE}',
                f'pull {BUILDKIT_IMAGE}',
            ],
        )
        self.assertFalse(self.sleep_log.exists())
        self.assertNotIn(f'tag {BUILDKIT_MIRROR_IMAGE}', '\n'.join(commands))

    @unittest.skipUnless(shutil.which('curl'), 'curl is required')
    def test_kind_curl_config_retries_transient_http_failures(self):
        _RetryHandler.request_count = 0
        server = http.server.HTTPServer(('127.0.0.1', 0), _RetryHandler)
        server_thread = threading.Thread(target=server.serve_forever)
        server_thread.start()
        try:
            result = subprocess.run(
                [
                    'curl',
                    '--retry-delay',
                    '0',
                    '--fail',
                    '--silent',
                    '--show-error',
                    f'http://127.0.0.1:{server.server_port}/tool',
                ],
                capture_output=True,
                check=False,
                env={
                    **os.environ,
                    'CURL_HOME': str(CURL_CONFIG.parent),
                },
                text=True,
                timeout=10,
            )
        finally:
            server.shutdown()
            server.server_close()
            server_thread.join()

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(result.stdout, 'ok')
        self.assertEqual(_RetryHandler.request_count, 3)

    def test_kind_action_scopes_curl_retry_config_to_both_attempts(self):
        action = CREATE_CLUSTER_ACTION.read_text()
        curl_home = (
            'CURL_HOME: ${{ github.workspace }}/.github/resources/curl-retry')
        self.assertEqual(action.count(curl_home), 2)

    def test_manifest_creation_uses_daemon_backed_buildx_driver(self):
        workflow = CREATE_MANIFEST_WORKFLOW.read_text()
        setup_step = workflow.split(
            '- name: Set up Docker Buildx', maxsplit=1)[1].split(
                '- name: Create and Push Manifest', maxsplit=1)[0]

        self.assertIn('uses: docker/setup-buildx-action@', setup_step)
        self.assertIn('driver: docker', setup_step)

    def test_manifest_workflow_changes_run_ci_script_tests(self):
        workflow = CI_SCRIPTS_WORKFLOW.read_text()

        self.assertIn("'.github/workflows/create-manifest.yml'", workflow)

    def test_buildkit_and_kind_use_docker_hub_mirror(self):
        buildkit_config = BUILDKIT_CONFIG.read_text()
        action = CREATE_CLUSTER_ACTION.read_text()

        self.assertIn('[registry."docker.io"]', buildkit_config)
        self.assertIn('mirrors = ["mirror.gcr.io"]', buildkit_config)
        self.assertIn(
            'bash ./.github/resources/scripts/'
            'configure-docker-hub-mirror.sh',
            action,
        )
        self.assertLess(
            action.index('Configure Docker Hub mirror'),
            action.index('Restore Kind node image cache'),
        )


if __name__ == '__main__':
    unittest.main()
