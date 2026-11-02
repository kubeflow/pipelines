#!/usr/bin/env python3
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
"""Cache only immutable gate evidence, including across github-script
processes."""

import hashlib
import json
import os
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_expected_workflows.js'
PARSER = Path('.github/resources/scripts/generate_ci_workflow_inventory.py')
SCRIPT = r"""
const {loadBaseInventory, verifyExpectedWorkflows} = require(process.argv[1]);
const options = JSON.parse(process.argv[2]);
const calls = {inventory: 0, files: 0, runs: 0};
const content = 'name: Test\non:\n  pull_request:\n    paths: [old/**]\njobs:\n  test:\n    runs-on: ubuntu-latest\n';
const pr = {number: options.number || 7, changed_files: options.incomplete ? 2 : 1,
  head: {sha: options.head || 'a'.repeat(40), ref: 'feature', repo: {full_name: 'fork/pipelines'}},
  base: {sha: options.base || 'b'.repeat(40), ref: 'master', repo: {full_name: 'kubeflow/pipelines'}}};
const github = {rest: {pulls: {listFiles: 'files'}, actions: {listWorkflowRunsForRepo: 'runs'}},
  graphql: async (query, variables) => {
    calls.inventory++;
    if (variables.expression !== `${options.liveBaseSha || pr.base.sha}:.github/workflows`) {
      throw Error('Inventory did not use resolved base revision');
    }
    return {repository: {nameWithOwner: 'kubeflow/pipelines', object: {__typename: 'Tree', entries: [
      {name: 'test.yml', type: 'blob', mode: 33188, object: {__typename: 'Blob', text: content,
        byteSize: Buffer.byteLength(content), isBinary: false, isTruncated: false}}
    ]}}};
  },
  paginate: async method => {
    calls[method]++;
    if (method === 'files') return [{filename: 'new/file', previous_filename: 'old/file'}];
    return [{path: '.github/workflows/test.yml', id: 1, run_attempt: 1, event: 'pull_request',
      head_sha: pr.head.sha, head_branch: pr.head.ref, head_repository: pr.head.repo,
      status: 'completed', conclusion: options.failed ? 'failure' : 'success',
      created_at: '2026-10-01T00:00:00Z', run_started_at: '2026-10-01T00:00:00Z'}];
  }};
(async () => {
  const args = {github, owner: 'kubeflow', repo: 'pipelines', pullRequest: pr,
    baseSha: options.liveBaseSha, liveBase: false, root: options.root, cacheRunId: Number(process.env.GITHUB_RUN_ID)};
  let result;
  for (let i = 0; i < 2; i++) {
    const inventory = await loadBaseInventory(args);
    result = await verifyExpectedWorkflows({...args, ...inventory, freshAfter: options.freshAfter || null});
  }
  console.log(JSON.stringify({calls, passed: result.passed}));
})().catch(error => {console.log(JSON.stringify({calls, error: error.message}));});
"""


class EvidenceCacheTest(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.root = Path(self.directory.name) / 'trusted'
        (self.root / PARSER).parent.mkdir(parents=True)
        shutil.copyfile(ROOT / PARSER, self.root / PARSER)
        self.env = dict(
            os.environ,
            RUNNER_TEMP=self.directory.name,
            GITHUB_RUN_ID='314159',
            GITHUB_RUN_ATTEMPT='1',
        )

    def exercise(self, **options):
        result = subprocess.run(
            [
                'node',
                '-e',
                SCRIPT,
                str(MODULE),
                json.dumps(dict(root=str(self.root), **options)),
            ],
            env=self.env,
            check=True,
            text=True,
            capture_output=True,
        )
        return json.loads(result.stdout)

    def test_reuses_inventory_but_refetches_rename_paths_across_processes(self):
        first = self.exercise()
        self.assertTrue(first['passed'])
        self.assertEqual(first['calls'], dict(inventory=1, files=2, runs=2))
        second = self.exercise()
        self.assertTrue(second['passed'])
        self.assertEqual(second['calls'], dict(inventory=0, files=2, runs=2))

    def test_mutable_runs_and_retarget_cutoff_are_not_cached(self):
        self.exercise()
        for options in [
            dict(failed=True),
            dict(freshAfter='2026-10-02T00:00:00Z'),
        ]:
            result = self.exercise(**options)
            self.assertFalse(result['passed'])
            self.assertEqual(
                result['calls'], dict(inventory=0, files=2, runs=2)
            )

    def test_inventory_identity_changes_miss(self):
        self.exercise()
        self.assertEqual(
            self.exercise(head='c' * 40)['calls'],
            dict(inventory=0, files=2, runs=2),
        )
        self.assertEqual(
            self.exercise(number=8)['calls'], dict(inventory=0, files=2, runs=2)
        )
        self.assertEqual(
            self.exercise(base='d' * 40)['calls'],
            dict(inventory=1, files=2, runs=2),
        )
        for key in ['GITHUB_RUN_ID', 'GITHUB_RUN_ATTEMPT']:
            self.env[key] = str(int(self.env[key]) + 1)
            self.assertEqual(
                self.exercise()['calls'], dict(inventory=1, files=2, runs=2)
            )

    def test_resolved_live_base_sha_is_part_of_cache_identity(self):
        first = self.exercise(liveBaseSha='e' * 40)
        self.assertEqual(first['calls'], dict(inventory=1, files=2, runs=2))
        self.assertEqual(
            self.exercise(liveBaseSha='e' * 40)['calls'],
            dict(inventory=0, files=2, runs=2),
        )
        self.assertEqual(
            self.exercise(liveBaseSha='f' * 40)['calls'],
            dict(inventory=1, files=2, runs=2),
        )

    def test_parser_revision_change_misses(self):
        self.exercise()
        with (self.root / PARSER).open('a') as parser:
            parser.write('\n# Different trusted parser revision\n')
        self.assertEqual(
            self.exercise()['calls'], dict(inventory=1, files=2, runs=2)
        )

    def test_corrupt_cache_refetches(self):
        self.exercise()
        for entry in (Path(self.directory.name) / 'kfp-ci-evidence').glob(
            '*.json'
        ):
            entry.write_text('{"value": {"wrong": true}, "digest": "invalid"}')
        result = self.exercise()
        self.assertTrue(result['passed'])
        self.assertEqual(result['calls'], dict(inventory=1, files=2, runs=2))

    def test_well_formed_envelope_with_invalid_evidence_refetches(self):
        self.exercise()
        value = '{"wrong":true}'
        envelope = dict(
            value=json.loads(value),
            digest=hashlib.sha256(value.encode()).hexdigest(),
        )
        for entry in (Path(self.directory.name) / 'kfp-ci-evidence').glob(
            '*.json'
        ):
            entry.write_text(json.dumps(envelope))
        result = self.exercise()
        self.assertTrue(result['passed'])
        self.assertEqual(result['calls'], dict(inventory=1, files=2, runs=2))

    def test_no_runner_temp_disables_cache(self):
        del self.env['RUNNER_TEMP']
        self.assertEqual(
            self.exercise()['calls'], dict(inventory=2, files=2, runs=2)
        )

    def test_incomplete_files_always_fail_closed(self):
        for _ in range(2):
            result = self.exercise(incomplete=True)
            self.assertIn('Incomplete PR file list', result['error'])
            self.assertEqual(result['calls']['files'], 1)
            self.assertEqual(result['calls']['runs'], 0)


if __name__ == '__main__':
    unittest.main()
