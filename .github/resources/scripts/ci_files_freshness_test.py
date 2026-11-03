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
"""A cached PR snapshot must not pin a mutable pull-request file response."""

import json
import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_expected_workflows.js'


class FilesFreshnessTest(unittest.TestCase):
    def test_rechecks_paths_after_head_moves_away_and_back(self):
        # listFiles is addressed by PR number, not the snapshot SHA. Its first
        # response can belong to temporary head B even if the PR is back on A
        # at the next safety boundary and has the same changed-file count.
        script = r"""
const {verifyExpectedWorkflows} = require(process.argv[1]);
const workflows = ['frontend', 'sdk'].map(name => ({
  path: `.github/workflows/${name}.yml`, header_sha256: name,
  pull_request: {paths: [`${name}/**`]}
}));
const inventory = {version: 1, workflows};
const workflowFiles = workflows.map(({path, header_sha256}) => ({path, header_sha256}));
const pr = {number: 7, changed_files: 1,
  head: {sha: 'a'.repeat(40), ref: 'feature', repo: {full_name: 'fork/repo'}},
  base: {ref: 'master', sha: 'b'.repeat(40)}};
const run = {id: 1, run_attempt: 1, created_at: '2026-10-01T00:00:00Z',
  run_started_at: '2026-10-01T00:00:00Z', path: '.github/workflows/sdk.yml', event: 'pull_request',
  head_sha: pr.head.sha, head_branch: 'feature', head_repository: pr.head.repo,
  status: 'completed', conclusion: 'success'};
let fileReads = 0;
const github = {
  rest: {pulls: {listFiles: 'files'}, actions: {listWorkflowRunsForRepo: 'runs'}},
  paginate: async method => method === 'files' ?
    [{filename: ++fileReads === 1 ? 'sdk/file' : 'frontend/file'}] : [run]
};
const args = {github, owner: 'upstream', repo: 'repo', pullRequest: pr,
  inventory, workflowFiles, root: process.argv[2], cacheRunId: 123};
(async () => {
  const initial = await verifyExpectedWorkflows(args);
  const final = await verifyExpectedWorkflows(args);
  console.log(JSON.stringify({fileReads, initial, final}));
})().catch(error => {console.error(error); process.exit(1);});
"""
        with tempfile.TemporaryDirectory() as directory:
            environment = dict(
                os.environ,
                RUNNER_TEMP=directory,
                GITHUB_RUN_ID='123',
                GITHUB_RUN_ATTEMPT='1',
            )
            result = subprocess.run(
                ['node', '-e', script, str(MODULE), str(ROOT)],
                env=environment,
                capture_output=True,
                text=True,
                check=True,
            )
        actual = json.loads(result.stdout)
        self.assertTrue(actual['initial']['passed'])
        self.assertEqual(actual['fileReads'], 2)
        self.assertFalse(actual['final']['passed'])
        self.assertEqual(
            actual['final']['expected'], ['.github/workflows/frontend.yml']
        )
        self.assertIn(
            'expected workflow has not registered',
            actual['final']['reasons'][0],
        )


if __name__ == '__main__':
    unittest.main()
