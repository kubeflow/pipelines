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

from pathlib import Path
import unittest

ROOT = Path(__file__).resolve().parents[3]


class MetaWorkflowConcurrencyTest(unittest.TestCase):

    def _read_workflow(self, name: str) -> str:
        return (ROOT / '.github/workflows' / name).read_text(encoding='utf-8')

    def test_ci_check_filters_label_events_before_job_concurrency(self):
        workflow = self._read_workflow('ci-checks.yml')
        workflow_header, jobs = workflow.split('\njobs:', maxsplit=1)

        self.assertNotIn('\nconcurrency:', workflow_header)
        self.assertIn("github.event.label.name == 'ok-to-test'", jobs)
        self.assertIn("github.event.label.name == 'needs-ok-to-test'", jobs)
        self.assertIn('    concurrency:\n', jobs)
        self.assertIn("&& 'head-update' || github.run_id", jobs)
        self.assertIn(
            "cancel-in-progress: ${{ github.event.action == 'synchronize' }}",
            jobs,
        )

    def test_approval_runs_on_open_but_skips_unrelated_labels(self):
        workflow = self._read_workflow('gh-workflow-approve.yml')
        workflow_header, jobs = workflow.split('\njobs:', maxsplit=1)

        self.assertIn('      - opened\n', workflow_header)
        self.assertNotIn('\nconcurrency:', workflow_header)
        self.assertIn("github.event.label.name == 'ok-to-test'", jobs)
        self.assertIn('    concurrency:\n', jobs)
        self.assertIn("&& 'head-update' || github.run_id", jobs)
        self.assertIn(
            "cancel-in-progress: ${{ github.event.action == 'synchronize' }}",
            jobs,
        )


if __name__ == '__main__':
    unittest.main()
