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
"""Exercise the scheduled approval workflow against mocked GitHub responses."""

import json
from pathlib import Path
import subprocess
import textwrap
import unittest

WORKFLOW = (
    Path(__file__).resolve().parents[2] / 'workflows' /
    'gh-workflow-approve.yml')


def recovery_script():
    workflow = WORKFLOW.read_text(encoding='utf-8')
    block = workflow.split('\n  scheduled-recovery:\n', 1)[1]
    return textwrap.dedent(block.split('          script: |\n', 1)[1])


def exercise(options=None):
    harness = r"""
const script = process.argv[1];
const options = JSON.parse(process.argv[2]);
const AsyncFunction = Object.getPrototypeOf(async function() {}).constructor;
const execute = new AsyncFunction('github', 'context', 'core', script);
const calls = [];
const pr = {
  number: 7, state: 'open', labels: [{name: 'ok-to-test'}],
  head: {sha: 'head', ref: 'feature', repo: {
    id: 11, owner: {login: 'contributor'}}},
};
if (options.noLabel) pr.labels = [];
if (options.needsLabel) pr.labels.push({name: 'needs-ok-to-test'});
const duplicate = {...pr, number: 8};
const openPRs = options.ambiguous ? [pr, duplicate] : [pr];
const run = {
  id: 42, event: 'pull_request', status: 'completed',
  conclusion: 'action_required', head_sha: 'head',
  head_branch: 'feature', head_repository: {id: options.wrongRepo ? 99 : 11},
  pull_requests: options.wrongAssociation ? [{number: 8}] : [],
};
const methods = {listPRs: {}, listRuns: {}};
const github = {
  rest: {
    pulls: {
      list: methods.listPRs,
      get: async () => {
        calls.push(['get-pr']);
        return {data: {...pr, state: options.closed ? 'closed' : 'open',
          head: options.stale ? {...pr.head, sha: 'new-head'} : pr.head,
          labels: options.revoked ? [] : pr.labels}};
      },
    },
    actions: {
      listWorkflowRunsForRepo: methods.listRuns,
      approveWorkflowRun: async request => {
        calls.push(['approve', request.run_id]);
        if (options.apiFailure) throw Error('approval forbidden');
      },
      getWorkflowRun: async () => ({data: {conclusion:
        options.alreadyResolved ? null : 'action_required'}}),
    },
  },
  paginate: async (method, params) => {
    if (method === methods.listPRs) {
      calls.push(['list-prs', params.head || 'all']);
      if (params.head && options.lateAmbiguity) return [pr, duplicate];
      if (params.head && options.lateReplacement) return [duplicate];
      return openPRs;
    }
    if (method === methods.listRuns) {
      calls.push(['list-runs', params.event, params.status, params.head_sha]);
      return [run];
    }
    throw Error('Unexpected pagination request');
  },
};
const context = {repo: {owner: 'kubeflow', repo: 'pipelines'}};
const core = {info: () => {}};
(async () => {
  let error;
  try { await execute(github, context, core); } catch (caught) { error = caught.message; }
  console.log(JSON.stringify({calls, error}));
})().catch(error => { console.error(error); process.exit(1); });
"""
    result = subprocess.run(
        ['node', '-e', harness,
         recovery_script(),
         json.dumps(options or {})],
        check=True,
        capture_output=True,
        text=True)
    return json.loads(result.stdout)


class ApprovalRecoveryTest(unittest.TestCase):

    def test_schedule_is_distinct_from_pull_request_job(self):
        workflow = WORKFLOW.read_text(encoding='utf-8')
        self.assertIn("- cron: '2,17,32,47 * * * *'", workflow)
        self.assertIn("github.event_name == 'pull_request_target' &&", workflow)
        self.assertIn(
            "  scheduled-recovery:\n    if: github.event_name == 'schedule'",
            workflow)

    def test_eligible_unique_head_approves_pending_run(self):
        result = exercise()
        self.assertEqual(result['calls'].count(['approve', 42]), 1)
        self.assertIn(['list-runs', 'pull_request', 'action_required', 'head'],
                      result['calls'])
        self.assertIsNone(result.get('error'))

    def test_ambiguous_head_skips_even_if_other_pr_is_unadmitted(self):
        result = exercise({'ambiguous': True})
        self.assertFalse(any(call[0] == 'approve' for call in result['calls']))
        self.assertFalse(
            any(call[0] == 'list-runs' for call in result['calls']))

    def test_new_ambiguity_before_approval_skips(self):
        result = exercise({'lateAmbiguity': True})
        self.assertFalse(any(call[0] == 'approve' for call in result['calls']))

    def test_different_pr_replacing_original_head_skips(self):
        result = exercise({'lateReplacement': True})
        self.assertFalse(any(call[0] == 'approve' for call in result['calls']))

    def test_closed_revoked_or_stale_pr_skips(self):
        for options in ({'closed': True}, {'revoked': True}, {'stale': True}):
            with self.subTest(options=options):
                result = exercise(options)
                self.assertFalse(
                    any(call[0] == 'approve' for call in result['calls']))

    def test_unadmitted_pr_skips(self):
        for options in ({'noLabel': True}, {'needsLabel': True}):
            with self.subTest(options=options):
                result = exercise(options)
                self.assertFalse(
                    any(call[0] == 'list-runs' for call in result['calls']))

    def test_run_identity_and_explicit_association_must_match(self):
        for options in ({'wrongRepo': True}, {'wrongAssociation': True}):
            with self.subTest(options=options):
                result = exercise(options)
                self.assertFalse(
                    any(call[0] == 'approve' for call in result['calls']))

    def test_approval_error_remains_visible_if_run_is_pending(self):
        result = exercise({'apiFailure': True})
        self.assertEqual(result['error'], 'approval forbidden')

    def test_concurrent_approval_is_idempotent(self):
        result = exercise({'apiFailure': True, 'alreadyResolved': True})
        self.assertIsNone(result.get('error'))


if __name__ == '__main__':
    unittest.main()
