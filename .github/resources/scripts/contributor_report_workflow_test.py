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
import re
import unittest

ROOT = Path(__file__).resolve().parents[3]


class ContributorReportWorkflowTest(unittest.TestCase):

    def test_script_checkout_uses_trusted_workflow_revision(self):
        workflow = (ROOT /
                    '.github/workflows/contributor-report.yml').read_text(
                        encoding='utf-8')
        steps = re.split(r'^      - ', workflow, flags=re.MULTILINE)[1:]
        checkouts = [
            step for step in steps
            if re.search(r'^\s*uses: actions/checkout@', step, re.MULTILINE)
        ]
        self.assertEqual(len(checkouts), 1)
        # PR base branches can lack the script; PR heads are untrusted.
        refs = re.findall(r'^\s+ref:\s*(.+)$', checkouts[0], re.MULTILINE)
        self.assertEqual(refs, ['${{ github.workflow_sha }}'])


if __name__ == '__main__':
    unittest.main()
