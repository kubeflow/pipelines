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
"""Exercise runner-aware E2E concurrency without dropping label coverage."""

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

import yaml
from api_multi_user_shards_test import render_references

ROOT = Path(__file__).resolve().parents[3]


class E2EWorkerLimitsTest(unittest.TestCase):
    def configure(self, cpus, label, dispatch=False):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text()
        )
        step = next(
            step
            for step in workflow['jobs']['end-to-end-scenario-tests']['steps']
            if step.get('id') == 'configure'
        )
        context = {
            'env': workflow['env'],
            'matrix': {'test_label': label},
            'github': {
                'event_name': 'workflow_dispatch'
                if dispatch
                else 'pull_request'
            },
        }
        with tempfile.TemporaryDirectory() as directory:
            output = Path(directory) / 'output'
            script = f'nproc() {{ echo {cpus}; }}\n' + render_references(
                step['run'], context
            )
            subprocess.run(
                ['bash', '-e', '-o', 'pipefail', '-c', script],
                env={
                    **os.environ,
                    'GITHUB_OUTPUT': str(output),
                    'DISPATCH_NUMBER_OF_NODES': '7',
                    'DISPATCH_TEST_LABEL': 'Smoke',
                    'DISPATCH_NAMESPACE': 'custom',
                },
                check=True,
                capture_output=True,
                text=True,
                timeout=10,
            )
            return dict(
                line.split('=', 1) for line in output.read_text().splitlines()
            )

    def test_automatic_runner_and_lane_limits(self):
        for cpus, label, expected in [
            (4, 'E2ECritical', 2),
            (4, 'E2ECriticalShardA', 2),
            (1, 'E2EEssential', 1),
            (0, 'E2EEssential', 1),
            (32, 'E2EEssential', 10),
            (32, 'E2EFailure', 2),
            (32, 'E2EParallelNested', 3),
            (4, 'E2EParallelNested', 2),
        ]:
            with self.subTest(cpus=cpus, label=label):
                actual = self.configure(cpus, label)
                self.assertEqual(actual['NUMBER_OF_NODES'], str(expected))
                expected_label = (
                    'E2ECritical && ' + label
                    if label == 'E2ECriticalShardA'
                    else label
                )
                self.assertEqual(actual['TEST_LABEL'], expected_label)

    def test_dispatch_override_is_preserved(self):
        self.assertEqual(
            self.configure(4, 'E2ECritical', dispatch=True),
            {
                'NUMBER_OF_NODES': '7',
                'TEST_LABEL': 'Smoke',
                'NAMESPACE': 'custom',
            },
        )


if __name__ == '__main__':
    unittest.main()
