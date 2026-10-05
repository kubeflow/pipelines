#!/usr/bin/env python3
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Require download changes to schedule integration consumers."""

import json
from pathlib import Path
import subprocess
import unittest

from generate_ci_workflow_inventory import UniqueKeyLoader
import yaml

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_expected_workflows.js'
UPGRADE = '.github/workflows/upgrade-test.yml'
ACTIONS = {
    './.github/actions/deploy',
    './.github/actions/download-artifact-with-retry',
}


class ArtifactDownloadTriggersTest(unittest.TestCase):

    def test_changes_require_all_pr_consumers_and_preserve_upgrade_pause(self):
        consumers = set()
        for path in (ROOT / '.github/workflows').iterdir():
            if path.suffix not in ('.yml', '.yaml'):
                continue
            workflow = yaml.load(path.read_text(), Loader=UniqueKeyLoader)
            if 'pull_request' not in workflow.get('on', {}):
                continue
            if any(
                    step.get('uses') in ACTIONS
                    for job in workflow['jobs'].values()
                    for step in job.get('steps', [])):
                consumers.add(path.relative_to(ROOT).as_posix())
        self.assertIn('.github/workflows/e2e-test-frontend.yml', consumers)
        self.assertIn('.github/workflows/arm64-presubmit.yml', consumers)
        self.assertIn('.github/workflows/build-tools-images.yml', consumers)
        self.assertIn(UPGRADE, consumers)

        for branch in ('master', 'release-2.17'):
            for changed_path in (
                    '.github/actions/download-artifact-with-retry/action.yml',
                    '.github/resources/scripts/artifact-files.sh',
                    '.github/resources/scripts/ci-image-artifacts.sh'):
                with self.subTest(branch=branch, changed_path=changed_path):
                    script = f'''
const gate = require({json.dumps(str(MODULE))});
const loaded = gate.loadLocalInventory(process.cwd());
const github = {{
  rest: {{pulls: {{listFiles: 'files'}}, actions: {{listWorkflowRunsForRepo: 'runs'}}}},
  paginate: async route => route === 'files'
    ? [{{filename: {json.dumps(changed_path)}}}] : [],
}};
gate.verifyExpectedWorkflows({{...loaded, github, owner: 'owner', repo: 'repo',
  pullRequest: {{number: 7, changed_files: 1,
    base: {{ref: {json.dumps(branch)}}},
    head: {{sha: 'head', ref: 'artifact-fix', repo: {{full_name: 'owner/repo'}}}},
  }},
}}).then(result => console.log(JSON.stringify(result)));
'''
                    result = subprocess.run(['node'],
                                            input=script,
                                            check=True,
                                            capture_output=True,
                                            text=True,
                                            cwd=ROOT)
                    coverage = json.loads(result.stdout)
                    expected = consumers - {UPGRADE}
                    self.assertTrue(
                        expected.issubset(coverage['expected']),
                        expected - set(coverage['expected']))
                    self.assertTrue(expected.issubset(coverage['missing']))
                    self.assertNotIn(UPGRADE, coverage['expected'])
                    self.assertEqual(
                        [item['path'] for item in coverage['disabled']],
                        [UPGRADE])
                    self.assertFalse(coverage['passed'])


if __name__ == '__main__':
    unittest.main()
