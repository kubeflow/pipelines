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
"""Keep profiling opt-in, isolated from test outcomes, and uniquely
archived."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]


def action():
    return yaml.safe_load(
        (ROOT / '.github/actions/test-and-report/action.yml').read_text())


class E2EProfileWorkflowTest(unittest.TestCase):

    def test_profile_directory_is_absolute_unique_and_name_is_safe(self):
        configure = next(step for step in action()['runs']['steps']
                         if step.get('id') == 'e2e-profile')
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary).resolve()
            outputs = []
            for index in range(2):
                output = root / f'output-{index}'
                report = 'TLS lane / db=mysql: [matrix] $() \n punctuation'
                completed = subprocess.run(
                    ['bash', '-e', '-c', configure['run']],
                    env=dict(
                        os.environ,
                        RUNNER_TEMP=str(root),
                        PROFILE_REPORT_NAME=report,
                        GITHUB_RUN_ID='1234',
                        GITHUB_RUN_ATTEMPT='2',
                        GITHUB_OUTPUT=str(output)),
                    capture_output=True,
                    text=True,
                    check=False)
                self.assertEqual(completed.returncode, 0, completed.stderr)
                fields = dict(
                    line.split('=', 1)
                    for line in output.read_text().splitlines())
                directory = Path(fields['path'])
                self.assertTrue(directory.is_absolute())
                self.assertEqual(directory.parent, root)
                self.assertTrue(directory.is_dir())
                self.assertRegex(fields['name'],
                                 r'^e2e-profile-[A-Za-z0-9_.=-]+$')
                self.assertIn('-1234-2-', fields['name'])
                self.assertLess(len(fields['name']), 255)
                outputs.append(fields)
            self.assertNotEqual(outputs[0]['path'], outputs[1]['path'])
            self.assertNotEqual(outputs[0]['name'], outputs[1]['name'])

    def test_default_disabled_and_environment_only_in_test_process(self):
        config = action()
        self.assertEqual(config['inputs']['profile_e2e']['default'], 'false')
        steps = config['runs']['steps']
        configure = next(
            step for step in steps if step.get('id') == 'e2e-profile')
        run = next(step for step in steps if step.get('id') == 'run-tests')
        self.assertIn("inputs.profile_e2e == 'true'", configure['if'])
        self.assertEqual(run['env']['KFP_E2E_PROFILE_DIR'],
                         '${{ steps.e2e-profile.outputs.path }}')
        self.assertNotIn('GITHUB_ENV', configure['run'])
        self.assertLess(steps.index(configure), steps.index(run))

    def test_passing_failed_and_cancelled_tests_keep_profiles_best_effort(self):
        steps = action()['runs']['steps']
        upload = next(step for step in steps
                      if step['name'] == 'Upload E2E lifecycle profiles')
        self.assertIn('always()', upload['if'])
        self.assertIn("steps.e2e-profile.outputs.path != ''", upload['if'])
        self.assertNotIn('run-tests.outcome', upload['if'])
        self.assertTrue(upload['continue-on-error'])
        self.assertEqual(upload['with']['path'],
                         '${{ steps.e2e-profile.outputs.path }}')
        self.assertEqual(upload['with']['name'],
                         '${{ steps.e2e-profile.outputs.name }}')
        self.assertEqual(upload['with']['retention-days'], 7)

    def test_only_automatic_e2e_lanes_opt_in(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/e2e-test.yml').read_text())
        count = 0
        for job in workflow['jobs'].values():
            for step in job.get('steps', []):
                if step.get('uses') == './.github/actions/test-and-report':
                    count += 1
                    self.assertEqual(
                        step['with']['profile_e2e'],
                        "${{ github.event_name != 'workflow_dispatch' }}")
        self.assertEqual(count, 4)
        api = yaml.safe_load(
            (ROOT / '.github/workflows/api-server-tests.yml').read_text())
        for job in api['jobs'].values():
            for step in job.get('steps', []):
                self.assertNotIn('profile_e2e', step.get('with', {}))
