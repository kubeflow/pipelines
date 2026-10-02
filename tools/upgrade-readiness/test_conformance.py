# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
from pathlib import Path
import unittest
from unittest import mock

import conformance
import schedule_policy


class ConformanceTest(unittest.TestCase):

    def test_workflow_uses_the_same_immutable_reference(self):
        workflow = (Path(__file__).resolve().parents[2] /
                    '.github/workflows/upgrade-readiness.yml').read_text()
        self.assertIn('ref: ' + schedule_policy.POLICY_SOURCE, workflow)

    def test_fixtures_cover_sar_and_pre_sar_decisions(self):
        cases = {case['name']: case for case in conformance.cases()}
        for name in ('default-exemption', 'allowlist-denied',
                     'literal-allowlist-star'):
            self.assertEqual(cases[name]['expected_sar_requests'], 0)
        for name in ('custom-scoped-grant', 'custom-denied', 'audit-denied',
                     'audit-allowlist-denied', 'wrong-named-grant',
                     'trimmed-allowlist', 'group-only-grant'):
            self.assertEqual(cases[name]['expected_sar_requests'], 1)
        self.assertEqual(cases['audit-denied']['prediction'],
                         'operational_impact')
        self.assertEqual(cases['custom-denied']['prediction'],
                         'policy_rejection')

    def test_wrong_policy_revision_cannot_run_backend(self):
        with mock.patch.object(
                conformance.subprocess, 'check_output',
                return_value='a' * 40), mock.patch.object(
                    conformance.subprocess, 'run') as run:
            with self.assertRaisesRegex(ValueError, 'pinned policy source'):
                conformance.run(Path('.'))
            run.assert_not_called()

    def test_dirty_policy_checkout_cannot_run_backend(self):
        with mock.patch.object(
                conformance.subprocess,
                'check_output',
                side_effect=[
                    schedule_policy.POLICY_SOURCE, ' M backend/source.go'
                ]), mock.patch.object(conformance.subprocess, 'run') as run:
            with self.assertRaisesRegex(ValueError, 'clean backend checkout'):
                conformance.run(Path('.'))
            run.assert_not_called()


if __name__ == '__main__':
    unittest.main()
