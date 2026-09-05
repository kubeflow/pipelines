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
"""Privilege guards for the CI health report workflow.

The workflow accepts workflow_dispatch and runs the collector from the
checked-out tree while holding write scopes. Without a ref gate, a dispatch
against an arbitrary branch would execute that branch's code with permission to
push commits, write issues, and mint an OIDC token.
"""

from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parents[3]
WORKFLOW = ROOT / '.github/workflows/ci-health-report.yml'


class CiHealthWorkflowTest(unittest.TestCase):

    def setUp(self):
        self.text = WORKFLOW.read_text(encoding='utf-8')

    def test_accepts_manual_dispatch(self):
        # The gates below only matter while dispatch is reachable.
        self.assertIn('workflow_dispatch:', self.text)

    def test_no_workflow_wide_write_permissions(self):
        # A workflow-level grant would apply to every job, including any added
        # later without its own block.
        self.assertRegex(self.text, re.compile(r'^permissions: \{\}$', re.M))

    def test_every_job_scopes_its_own_permissions(self):
        jobs_section = self.text.split('\njobs:\n', 1)[1]
        jobs = re.findall(r'^  (\w[\w-]*):$', jobs_section, flags=re.M)
        self.assertEqual(sorted(jobs), ['deploy', 'report'])
        for job in jobs:
            block = self.text.split(f'\n  {job}:\n', 1)[1]
            block = re.split(r'\n  \w[\w-]*:\n', block)[0]
            with self.subTest(job=job):
                self.assertRegex(block, re.compile(r'^    permissions:$', re.M))

    def test_jobs_are_gated_to_the_default_branch(self):
        for job in ('report', 'deploy'):
            block = self.text.split(f'\n  {job}:\n', 1)[1]
            block = re.split(r'\n  \w[\w-]*:\n', block)[0]
            with self.subTest(job=job):
                self.assertIn("github.ref == 'refs/heads/master'", block)

    def test_only_the_deploy_job_receives_oidc(self):
        report = self.text.split('\n  report:\n', 1)[1]
        report = re.split(r'\n  \w[\w-]*:\n', report)[0]
        deploy = self.text.split('\n  deploy:\n', 1)[1]

        self.assertNotIn('id-token:', report)
        self.assertIn('id-token: write', deploy)

    def test_checkout_pins_the_trusted_ref(self):
        report = self.text.split('\n  report:\n', 1)[1]
        report = re.split(r'\n  \w[\w-]*:\n', report)[0]
        checkout = report.split('actions/checkout', 1)[1]

        self.assertRegex(checkout, re.compile(r'^\s+ref: master$', re.M))


if __name__ == '__main__':
    unittest.main()
