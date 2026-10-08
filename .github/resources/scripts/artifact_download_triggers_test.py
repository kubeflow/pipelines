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
ACTIONS = {
    './.github/actions/deploy',
    './.github/actions/download-artifact-with-retry',
}


class ArtifactDownloadTriggersTest(unittest.TestCase):

    def test_changes_trigger_all_pr_consumers(self):
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

        for branch in ('master', 'release-2.17'):
            for changed_path in (
                    '.github/actions/download-artifact-with-retry/action.yml',
                    '.github/resources/scripts/artifact-files.sh',
                    '.github/resources/scripts/ci-image-artifacts.sh'):
                with self.subTest(branch=branch, changed_path=changed_path):
                    script = f'''
const gate = require({json.dumps(str(MODULE))});
const {{inventory}} = gate.loadLocalInventory(process.cwd());
console.log(JSON.stringify(inventory.workflows.filter(workflow =>
  workflow.pull_request !== null && gate.applicable(workflow.pull_request,
    {json.dumps(branch)}, [{json.dumps(changed_path)}]))
  .map(workflow => workflow.path)));
'''
                    result = subprocess.run(['node'],
                                            input=script,
                                            check=True,
                                            capture_output=True,
                                            text=True,
                                            cwd=ROOT)
                    triggered = set(json.loads(result.stdout))
                    # Master-only qualification must not be required on release PRs.
                    branch_script = f"""
const gate = require({json.dumps(str(MODULE))});
const {{inventory}} = gate.loadLocalInventory(process.cwd());
console.log(JSON.stringify(inventory.workflows.filter(workflow =>
  workflow.pull_request !== null && gate.applicable(
    Object.fromEntries(Object.entries(workflow.pull_request).filter(
      ([key]) => key !== 'paths' && key !== 'paths-ignore')),
    {json.dumps(branch)}, []))
  .map(workflow => workflow.path)));
"""
                    eligible = subprocess.run(['node'],
                                              input=branch_script,
                                              check=True,
                                              capture_output=True,
                                              text=True,
                                              cwd=ROOT)
                    required = consumers & set(json.loads(eligible.stdout))
                    self.assertTrue(
                        required.issubset(triggered), required - triggered)


if __name__ == '__main__':
    unittest.main()
