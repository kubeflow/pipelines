# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Keep multi-claim DRA coverage on a Kubernetes release with stable status."""

import hashlib
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml
from artifact_workflow_test_support import evaluate

ROOT = Path(__file__).resolve().parents[3]
BUILD_SCRIPT = ROOT / '.github/resources/scripts/build-kind-node-image.sh'


class DRAClusterTest(unittest.TestCase):
    def test_dra_lane_has_multi_claim_controller_fix(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text()
        )
        lanes = [
            lane
            for job in workflow['jobs'].values()
            for lane in job.get('strategy', {})
            .get('matrix', {})
            .get('include', [])
            if lane.get('test_label') == 'dra-check'
        ]
        self.assertEqual(len(lanes), 1)
        version = tuple(
            map(int, lanes[0]['k8s_version'].removeprefix('v').split('.'))
        )
        self.assertEqual(version[:2], (1, 34), 'Retain DRA GA-minimum coverage')
        # Kubernetes #142025 first shipped in 1.34.13. Earlier controllers can
        # remove an existing claim status when applying another template claim.
        self.assertGreaterEqual(version, (1, 34, 13))

    def test_only_dra_builds_release_images(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text()
        )
        for job in workflow['jobs'].values():
            for step in job.get('steps', []):
                if step.get('uses') == './.github/actions/create-cluster':
                    expression = step['with'].get('build_node_image', 'false')
                    lanes = job.get('strategy', {}).get('matrix', {}).get(
                        'include'
                    ) or [{}]
                    for lane in lanes:
                        self.assertEqual(
                            evaluate(expression, {'matrix': lane}),
                            lane.get('test_label') == 'dra-check',
                        )

    def test_cache_hit_skips_both_image_acquisition_paths(self):
        action = yaml.safe_load(
            (ROOT / '.github/actions/create-cluster/action.yml').read_text()
        )
        self.assertEqual(
            action['inputs']['build_node_image']['default'], 'false'
        )
        steps = {step['name']: step for step in action['runs']['steps']}
        for build in ('false', 'true'):
            for hit in ('false', 'true'):
                with self.subTest(build=build, hit=hit):
                    context = {
                        'inputs': {'build_node_image': build},
                        'steps': {
                            'kind-node-cache': {'outputs': {'cache-hit': hit}}
                        },
                    }
                    for name, expected in (
                        ('Load Kind node image from cache', hit == 'true'),
                        (
                            'Pull and cache Kind node image',
                            hit != 'true' and build != 'true',
                        ),
                        (
                            'Install Kind for release image build',
                            hit != 'true' and build == 'true',
                        ),
                        (
                            'Build and cache Kind node image',
                            hit != 'true' and build == 'true',
                        ),
                    ):
                        self.assertEqual(
                            evaluate(steps[name]['if'], context), expected
                        )

    def test_release_archive_is_verified_before_build(self):
        for bad_checksum, download_failure, build_exit in (
            (False, False, 0),
            (True, False, 0),
            (False, True, 0),
            (False, False, 7),
        ):
            with (
                self.subTest(
                    checksum=bad_checksum,
                    download=download_failure,
                    build=build_exit,
                ),
                tempfile.TemporaryDirectory() as directory,
            ):
                root = Path(directory)
                binary = root / 'bin'
                binary.mkdir()
                command_log = root / 'kind.log'
                checksum = hashlib.sha512(b'archive').hexdigest()
                if bad_checksum:
                    checksum = '0' * 128
                commands = {
                    'uname': '#!/bin/sh\necho x86_64\n',
                    'curl': '#!/bin/sh\n'
                    + ('exit 22\n' if download_failure else '')
                    + 'while [ "$1" != "-o" ]; do shift; done\n'
                    + 'case "$2" in\n'
                    + f'  *.sha512) printf "%s" "{checksum}" > "$2" ;;\n'
                    + '  *) printf archive > "$2" ;;\nesac\n',
                    'kind': '#!/bin/sh\nprintf "%s\\n" "$@" > "$COMMAND_LOG"\n'
                    + f'exit {build_exit}\n',
                }
                for name, contents in commands.items():
                    command = binary / name
                    command.write_text(contents)
                    command.chmod(0o755)
                result = subprocess.run(
                    ['bash', str(BUILD_SCRIPT), 'v1.34.13'],
                    env={
                        **os.environ,
                        'PATH': f'{binary}:{os.environ["PATH"]}',
                        'COMMAND_LOG': str(command_log),
                    },
                    capture_output=True,
                    text=True,
                    check=False,
                )
                if bad_checksum or download_failure:
                    self.assertNotEqual(result.returncode, 0)
                    self.assertFalse(command_log.exists())
                else:
                    self.assertEqual(
                        result.returncode, build_exit, result.stderr
                    )
                    args = command_log.read_text().splitlines()
                    self.assertEqual(
                        args[:4], ['build', 'node-image', '--type', 'file']
                    )
                    self.assertTrue(
                        args[4].endswith(
                            '/kubernetes-server-linux-amd64.tar.gz'
                        )
                    )
                    self.assertEqual(
                        args[5:], ['--image', 'kindest/node:v1.34.13']
                    )
                    self.assertFalse(
                        Path(args[4]).exists(), 'Remove the downloaded archive'
                    )


if __name__ == '__main__':
    unittest.main()
