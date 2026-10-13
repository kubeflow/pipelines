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
"""Regression coverage for dependency-only retries."""

from pathlib import Path
import subprocess
import unittest
from unittest import mock

import download_policy_dependencies as dependencies


class DownloadPolicyDependenciesTest(unittest.TestCase):

    def setUp(self):
        self.run = mock.patch.object(dependencies.subprocess, 'run').start()
        self.sleep = mock.patch.object(dependencies.time, 'sleep').start()
        self.addCleanup(mock.patch.stopall)
        self.source = Path('policy-source')

    def test_success_uses_pinned_checkout_and_bounded_download_only(self):
        dependencies.download(self.source)
        self.run.assert_called_once_with(['go', 'mod', 'download'],
                                         cwd=self.source.resolve(),
                                         check=True,
                                         timeout=120)
        self.sleep.assert_not_called()

    def test_transient_failure_recovers_with_backoff(self):
        self.run.side_effect = [subprocess.CalledProcessError(1, 'go'), None]
        dependencies.download(self.source)
        self.assertEqual(self.run.call_count, 2)
        self.sleep.assert_called_once_with(5)

    def test_repeated_failures_exhaust_budget(self):
        error = subprocess.CalledProcessError(1, 'go')
        self.run.side_effect = error
        with self.assertRaises(subprocess.CalledProcessError):
            dependencies.download(self.source)
        self.assertEqual(self.run.call_count, 3)
        self.assertEqual(self.sleep.call_args_list,
                         [mock.call(5), mock.call(10)])

    def test_timeouts_are_bounded_and_fail_after_three_attempts(self):
        self.run.side_effect = subprocess.TimeoutExpired('go', 120)
        with self.assertRaises(subprocess.TimeoutExpired):
            dependencies.download(self.source)
        self.assertEqual(self.run.call_count, 3)
        self.assertTrue(
            all(c.kwargs['timeout'] == 120 for c in self.run.call_args_list))

    def test_missing_executable_is_not_retried(self):
        self.run.side_effect = FileNotFoundError('go')
        with self.assertRaises(FileNotFoundError):
            dependencies.download(self.source)
        self.run.assert_called_once()
        self.sleep.assert_not_called()

    def test_cancellation_is_not_retried(self):
        self.run.side_effect = KeyboardInterrupt()
        with self.assertRaises(KeyboardInterrupt):
            dependencies.download(self.source)
        self.run.assert_called_once()
        self.sleep.assert_not_called()

    def test_workflow_prefetches_before_single_conformance_execution(self):
        workflow = (Path(__file__).resolve().parents[2] /
                    '.github/workflows/upgrade-readiness.yml').read_text()
        download = (
            'python3 tools/upgrade-readiness/'
            'download_policy_dependencies.py --backend-source policy-source')
        compare = ('python3 tools/upgrade-readiness/conformance.py '
                   '--backend-source policy-source')
        self.assertEqual(workflow.count(download), 1)
        self.assertEqual(workflow.count(compare), 1)
        self.assertLess(workflow.index(download), workflow.index(compare))
        self.assertNotIn('continue-on-error', workflow)


if __name__ == '__main__':
    unittest.main()
