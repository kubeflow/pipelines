# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import unittest

import yaml
from ci_pass_eligibility_test import ROOT, exercise


class ConsolidatedCheckTests(unittest.TestCase):
    def test_no_duplicate_external_checker_invocations(self):
        for filename in ('ci-checks.yml', 'ci-publish-status.yml'):
            workflow = yaml.safe_load(
                (ROOT / '.github/workflows' / filename).read_text()
            )
            for job in workflow['jobs'].values():
                for step in job.get('steps', []):
                    self.assertNotIn(
                        'wechuli/allcheckspassed@', step.get('uses', '')
                    )

    def test_workflow_binds_validation_to_successful_prerequisites(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/ci-publish-status.yml').read_text()
        )
        steps = workflow['jobs']['publish']['steps']
        finalizer = next(
            step
            for step in steps
            if step.get('name') == 'Reconcile status and informational label'
        )
        self.assertIn(
            "job.status == 'success' && steps.prepare.outcome == 'success'",
            finalizer['env']['CI_PREPARATION_SUCCEEDED'],
        )
        self.assertIn(
            "steps.prepare.outputs.ready == 'true'",
            finalizer['env']['CI_VALIDATION_READY'],
        )

    def test_interrupted_preparation_cannot_publish_green(self):
        result = exercise({'preparationInterrupted': True})
        self.assertNotIn(['status', 'success', 'head'], result['calls'])
        self.assertEqual(result['status'], 'failure')

    def test_success_retains_fresh_checks_before_and_after_publication(self):
        result = exercise({})
        self.assertEqual(result['status'], 'success')
        self.assertEqual(result['requestCounts']['checks'], 2)

    def test_failed_external_check_cannot_publish_green(self):
        result = exercise({'checkConclusion': 'failure'})
        self.assertNotIn(['status', 'success', 'head'], result['calls'])
        self.assertEqual(result['status'], 'failure')

    def test_external_failure_after_publication_revokes_green(self):
        result = exercise({'drift': 'external-failure'})
        self.assertIn(['status', 'success', 'head'], result['calls'])
        self.assertEqual(result['status'], 'failure')


if __name__ == '__main__':
    unittest.main()
