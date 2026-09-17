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
"""Execute the production publisher with GitHub API fixtures."""

import json
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_passed.js'


def exercise(options=None):
    script = r"""
const gate = require(process.argv[1]);
const {loadLocalInventory} = require(process.argv[1].replace('ci_passed.js', 'ci_expected_workflows.js'));
const options = JSON.parse(process.argv[2]);
const root = process.argv[3];
const calls = [], outputs = {};
let pr = {
  number: 7, state: 'open', changed_files: 1,
  head: {sha: 'head', ref: 'feature', repo: {full_name: 'contributor/pipelines'}},
  base: {sha: 'base', ref: 'master'},
  user: {login: 'dependabot[bot]'}, author_association: 'NONE', labels: [],
  ...options.pr,
};
const eventPR = structuredClone(pr);
if (options.oldHead) eventPR.head.sha = 'old-head';
const context = {repo: {owner: 'kubeflow', repo: 'pipelines'}, runId: 99,
  eventName: options.schedule ? 'schedule' : options.workflowRun ? 'workflow_run' : 'pull_request_target',
  payload: {pull_request: eventPR, action: options.action || 'opened',
    workflow_run: {event: 'pull_request', head_sha: 'head', head_branch: 'feature',
      head_repository: {owner: {login: 'contributor'}, full_name: 'contributor/pipelines'}}}};
let published = false;
let status = options.initialStatus;
const core = {info: () => {}, setOutput: (key, value) => {outputs[key] = value;}};
const methods = {files: {}, runs: {}, timeline: {}, pulls: {}};
const github = {rest: {
  pulls: {listFiles: methods.files, list: methods.pulls,
    get: async () => ({data: structuredClone(pr)})},
  actions: {listWorkflowRunsForRepo: methods.runs},
  issues: {listEventsForTimeline: methods.timeline,
    addLabels: async request => {calls.push(['add-label', request.labels]);},
    removeLabel: async request => {calls.push(['remove-label', request.name]);}},
  repos: {getCombinedStatusForRef: {}, createCommitStatus: async request => {
    status = request.state;
    calls.push(['status', request.state, request.sha]);
    if (request.state === 'success') {
      published = true;
      if (options.drift === 'eligibility') pr.labels = [{name: 'needs-ok-to-test'}];
      if (options.drift === 'base') pr.base.ref = 'release';
      if (options.drift === 'base-sha') pr.base.sha = 'new-base';
      if (options.drift === 'head') pr.head.sha = 'new-head';
      if (options.drift === 'closed') pr.state = 'closed';
    }
  }},
}, paginate: async (method, params) => {
  if (method === methods.files) return [{filename: 'frontend/src/mlmd/Api.ts'}];
  if (method === methods.pulls) return options.ambiguous ? [pr, {...pr, number: 8}] : [pr];
  if (method === methods.timeline) {
    if (options.apiFailure) throw Error('API unavailable');
    return options.retarget ? [{event: 'base_ref_changed', created_at: '2026-09-07T12:00:00Z'}] : [];
  }
  if (method === methods.runs) {
    if (options.missing) return [];
    const conclusion = published && options.drift === 'rerun' ? 'cancelled' : (options.conclusion || 'success');
    return loadLocalInventory(root).inventory.workflows.map(workflow => ({path: workflow.path, id: 42, event: 'pull_request', head_sha: 'head', head_branch: 'feature',
      head_repository: {full_name: 'contributor/pipelines'},
      status: options.runStatus || 'completed', conclusion,
      created_at: options.fresh ? '2026-09-07T12:01:00Z' : '2026-09-07T11:00:00Z',
      run_started_at: '2026-09-07T12:02:00Z', pull_requests: []}));
  }
  throw Error('Unexpected API request');
}};
github.paginate.iterator = async function* () {
  if (options.statusReadFailure) throw Error('Status read unavailable');
  yield {data: {statuses: status ? [{context: 'ci-passed', state: status}] : []}};
};
(async () => {
  let error;
  for (let cycle = 0; cycle < (options.cycles || 1); cycle++) {
  try {await gate.prepare({github, context, core, root, recovery: {number: 7, head: eventPR.head.sha}});} catch (e) {error = e.message;}
  if (options.revokeBeforeFinal) pr.labels = [{name: 'needs-ok-to-test'}];
  try {
    await gate.finalize({github, context, core, root, number: outputs.pr_number,
      head: outputs.head_sha, before: outputs.snapshot,
      pollPassed: outputs.ready === 'true' && (options.pollPassed !== false || (options.recoverLast && cycle === options.cycles - 1)) && !error});
  } catch (e) {error = e.message;}
  }
  console.log(JSON.stringify({calls, outputs, error, status}));
})().catch(e => {console.error(e); process.exit(1);});
"""
    result = subprocess.run([
        'node', '-e', script,
        str(MODULE),
        json.dumps(options or {}),
        str(ROOT)
    ],
                            check=True,
                            capture_output=True,
                            text=True)
    return json.loads(result.stdout)


class CIPassedTest(unittest.TestCase):

    def assert_last_status(self, result, state):
        statuses = [call for call in result['calls'] if call[0] == 'status']
        self.assertTrue(statuses, result)
        self.assertEqual(statuses[-1], ['status', state, 'head'], result)

    def test_eligibility_truth_table(self):
        script = """
const {eligible} = require(process.argv[1]);
const result = [];
for (const author of ['dependabot[bot]', 'renovate[bot]', 'human']) {
  for (const association of ['NONE', 'CONTRIBUTOR', 'MEMBER', 'OWNER', 'COLLABORATOR']) {
    for (const ok of [false, true]) for (const needs of [false, true]) {
      const labels = [ok && 'ok-to-test', needs && 'needs-ok-to-test'].filter(Boolean).map(name => ({name}));
      result.push([author, association, ok, needs, eligible({user: {login: author}, author_association: association, labels})]);
    }
  }
}
console.log(JSON.stringify(result));
"""
        result = subprocess.run(
            ['node', '-e', script, str(MODULE)],
            check=True,
            capture_output=True,
            text=True)
        for author, association, ok, needs, actual in json.loads(result.stdout):
            expected = not needs and (ok or author == 'dependabot[bot]' or
                                      association
                                      in {'MEMBER', 'OWNER', 'COLLABORATOR'})
            self.assertEqual(actual, expected, (author, association, ok, needs))

    def test_complete_ci_publishes_pending_then_success(self):
        result = exercise()
        self.assertEqual(result['calls'][0], ['status', 'pending', 'head'])
        self.assert_last_status(result, 'success')

    def test_failed_poll_blocks_otherwise_complete_workflows(self):
        self.assert_last_status(exercise({'pollPassed': False}), 'failure')

    def test_missing_workflows_never_reach_poller(self):
        result = exercise({'missing': True})
        self.assertEqual(result['outputs']['ready'], 'false')
        self.assert_last_status(result, 'failure')

    def test_recovery_discovery_skips_green_and_ineligible_prs(self):
        script = r"""
const {recoveryCandidates} = require(process.argv[1]);
const requests = [];
const prs = ['success', 'failure', 'pending', 'missing', 'untrusted', 'revoked'].map((state, i) => ({
  number: i + 1, head: {sha: state}, user: {login: state === 'untrusted' ? 'human' : 'dependabot[bot]'},
  labels: state === 'revoked' ? [{name: 'needs-ok-to-test'}] : [], author_association: 'NONE',
}));
const github = {paginate: async () => prs, rest: {pulls: {list: {}}, repos: {
  getCombinedStatusForRef: async ({ref}) => {
    requests.push(ref);
    return {data: {statuses: ref === 'missing' ? [] : [{context: 'ci-passed', state: ref}]}};
  },
}}};
github.paginate.iterator = async function* (method, params) {
  yield {data: {statuses: [{context: 'other', state: 'success'}]}};
  yield await github.rest.repos.getCombinedStatusForRef(params);
};
recoveryCandidates({github, context: {repo: {owner: 'o', repo: 'r'}}}).then(result => {
  console.log(JSON.stringify({result, requests}));
}).catch(e => {console.error(e); process.exit(1);});
"""
        result = subprocess.run(
            ['node', '-e', script, str(MODULE)],
            check=True,
            capture_output=True,
            text=True)
        actual = json.loads(result.stdout)
        self.assertEqual(actual['result'], [{
            'number': 2,
            'head': 'failure'
        }, {
            'number': 3,
            'head': 'pending'
        }, {
            'number': 4,
            'head': 'missing'
        }])
        self.assertEqual(actual['requests'],
                         ['success', 'failure', 'pending', 'missing'])

    def test_external_check_finishing_after_final_workflow_recovers(self):
        self.assert_last_status(
            exercise({
                'workflowRun': True,
                'pollPassed': False
            }), 'failure')
        self.assert_last_status(exercise({'schedule': True}), 'success')

    def test_scheduled_recovery_preserves_safety_checks(self):
        for options in [{
                'pollPassed': False
        }, {
                'retarget': True
        }, {
                'missing': True
        }, {
                'pr': {
                    'labels': [{
                        'name': 'needs-ok-to-test'
                    }]
                }
        }, {
                'drift': 'rerun'
        }, {
                'apiFailure': True
        }]:
            with self.subTest(options=options):
                self.assert_last_status(
                    exercise({
                        'schedule': True,
                        **options
                    }), 'failure')
        self.assertEqual(
            exercise({
                'schedule': True,
                'oldHead': True
            })['calls'], [])
        self.assertEqual(
            exercise({
                'schedule': True,
                'pr': {
                    'state': 'closed'
                }
            })['calls'], [])

    def test_status_read_failure_still_invalidates_but_cannot_succeed(self):
        result = exercise({
            'schedule': True,
            'initialStatus': 'success',
            'statusReadFailure': True
        })
        self.assert_last_status(result, 'failure')
        self.assertNotIn(['status', 'success', 'head'], result['calls'])
        self.assertIn('Status read unavailable', result['error'])

    def test_repeated_failing_sweeps_do_not_exhaust_status_history(self):
        result = exercise({
            'schedule': True,
            'pollPassed': False,
            'initialStatus': 'failure',
            'cycles': 6
        })
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'], [])
        self.assertEqual(result['status'], 'failure')
        result = exercise({
            'schedule': True,
            'pollPassed': False,
            'initialStatus': 'failure',
            'cycles': 6,
            'recoverLast': True
        })
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'],
                         [['status', 'success', 'head']])

    def test_queued_recovery_invalidates_success_when_checks_change(self):
        result = exercise({
            'schedule': True,
            'pollPassed': False,
            'initialStatus': 'success'
        })
        self.assertEqual(
            [c for c in result['calls'] if c[0] == 'status'],
            [['status', 'pending', 'head'], ['status', 'failure', 'head']])

    def test_existing_pr_hold_is_never_removed(self):
        for passed in [True, False]:
            result = exercise({
                'schedule': True,
                'pollPassed': passed,
                'pr': {
                    'labels': [{
                        'name': 'do-not-merge/hold'
                    }]
                }
            })
            labels = [call for call in result['calls'] if call[0] != 'status']
            self.assertTrue(labels)
            for call in labels:
                self.assertIn(call, [['add-label', ['ci-passed']],
                                     ['remove-label', 'ci-passed']])

    def test_rerun_lifecycle(self):
        for status in ['queued', 'in_progress', 'waiting']:
            with self.subTest(status=status):
                self.assert_last_status(
                    exercise({
                        'workflowRun': True,
                        'runStatus': status
                    }), 'failure')
        for conclusion in [
                'cancelled', 'failure', 'timed_out', 'action_required', 'stale',
                'skipped', 'neutral'
        ]:
            with self.subTest(conclusion=conclusion):
                self.assert_last_status(
                    exercise({
                        'workflowRun': True,
                        'conclusion': conclusion
                    }), 'failure')
        self.assert_last_status(exercise({'workflowRun': True}), 'success')

    def test_retarget_survives_label_reopen_and_stale_completion(self):
        for action in ['edited', 'labeled', 'reopened']:
            with self.subTest(action=action):
                self.assert_last_status(
                    exercise({
                        'retarget': True,
                        'action': action
                    }), 'failure')
        self.assert_last_status(
            exercise({
                'retarget': True,
                'workflowRun': True
            }), 'failure')

    def test_rerunning_old_workflow_after_retarget_is_not_fresh_evidence(self):
        # The fixture attempt starts after the retarget but was created before it.
        self.assert_last_status(exercise({'retarget': True}), 'failure')

    def test_new_execution_after_retarget_recovers(self):
        self.assert_last_status(
            exercise({
                'retarget': True,
                'fresh': True
            }), 'success')

    def test_ineligible_and_closed_prs_fail(self):
        for pr in [{
                'labels': [{
                    'name': 'needs-ok-to-test'
                }]
        }, {
                'state': 'closed'
        }]:
            with self.subTest(pr=pr):
                self.assert_last_status(exercise({'pr': pr}), 'failure')

    def test_revocation_before_publication_fails(self):
        self.assert_last_status(
            exercise({'revokeBeforeFinal': True}), 'failure')

    def test_publication_reconciles_full_state_and_ci(self):
        for drift in [
                'eligibility', 'base', 'base-sha', 'head', 'closed', 'rerun'
        ]:
            with self.subTest(drift=drift):
                self.assert_last_status(exercise({'drift': drift}), 'failure')

    def test_old_head_event_does_not_publish_to_new_head(self):
        self.assertEqual(exercise({'oldHead': True})['calls'], [])

    def test_ambiguous_workflow_mapping_does_not_guess(self):
        result = exercise({'workflowRun': True, 'ambiguous': True})
        self.assertIn('exactly one', result['error'])
        self.assert_last_status(result, 'failure')

    def test_api_failure_leaves_gate_failed(self):
        result = exercise({'apiFailure': True})
        self.assertIn('API unavailable', result['error'])
        self.assert_last_status(result, 'failure')


if __name__ == '__main__':
    unittest.main()
