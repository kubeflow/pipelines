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
"""Keep broad frontend qualification observable without gating pull
requests."""

from pathlib import Path
import unittest

from ci_health_report import TARGET_WORKFLOWS
from generate_ci_workflow_inventory import UniqueKeyLoader
import yaml

ROOT = Path(__file__).resolve().parents[3]


def workflow(name):
    return yaml.load(
        (ROOT / '.github/workflows' / name).read_text(), Loader=UniqueKeyLoader)


class FrontendQualificationWorkflowTest(unittest.TestCase):

    def test_broad_qualification_is_weekly_manual_and_not_a_merge_gate(self):
        selector = workflow('ci-checks.yml')['on']['workflow_run']['workflows']
        for area in ('browser', 'performance', 'deployment'):
            with self.subTest(area=area):
                data = workflow(f'frontend-{area}-qualification.yml')
                self.assertEqual(
                    set(data['on']), {'schedule', 'workflow_dispatch'})
                schedules = data['on']['schedule']
                self.assertEqual(len(schedules), 1)
                self.assertEqual(schedules[0]['cron'].split()[2:],
                                 ['*', '*', '1'])
                self.assertNotIn(data['name'], selector)
                self.assertIn(f'frontend-{area}-qualification.yml',
                              TARGET_WORKFLOWS)
                # A failing observation stays failed, even though it is not a gate.
                for job in data['jobs'].values():
                    self.assertFalse(job.get('continue-on-error', False))
                    for step in job.get('steps', []):
                        self.assertFalse(step.get('continue-on-error', False))

    def test_mobile_deadline_retains_setup_and_diagnostics_allowance(self):
        job = workflow(
            'frontend-browser-qualification.yml')['jobs']['apple-browser']
        self.assertEqual(job['timeout-minutes'],
                         "${{ matrix.mode == 'desktop' && 35 || 90 }}")

    def test_relocated_browser_lanes_consume_shared_bundle_and_keep_evidence(
            self):
        data = workflow('frontend-browser-qualification.yml')
        job = data['jobs']['browser-tests']
        self.assertFalse(job['strategy']['fail-fast'])
        steps = job['steps']
        download = next(
            step for step in steps if step.get('uses') ==
            './.github/actions/download-artifact-with-retry')
        uploads = [
            step['with']['name']
            for step in data['jobs']['build']['steps']
            if step.get('uses', '').startswith('actions/upload-artifact@')
        ]
        self.assertIn(download['with']['name'], uploads)
        self.assertEqual(download['with']['required-files'],
                         'qualification-bundle.tar.gz')
        suite = next(step for step in steps if step.get('id') == 'suite')
        self.assertEqual(suite['run'], 'npm run test:bundle:built')
        upload = next(
            step for step in steps
            if step.get('uses', '').startswith('actions/upload-artifact@'))
        self.assertEqual(upload['if'], 'always()')
        self.assertEqual(upload['with']['if-no-files-found'], 'error')


if __name__ == '__main__':
    unittest.main()
