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

import hashlib
import json
import os
from pathlib import Path
import subprocess
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_passed.js'


def exercise(options=None):
    script = r"""
const gate = require(process.argv[1]);
const options = JSON.parse(process.argv[2]);
const root = process.argv[3];
const calls = [], outputs = {}, descriptions = [], targetUrls = [];
const workflowName = options.workflowName || 'frontend.yml';
let cycle = 0;
let pr = {
  number: 7, state: 'open', changed_files: 1,
  head: {sha: 'head', ref: 'feature', repo: {full_name: 'contributor/pipelines'}},
  base: {sha: 'b'.repeat(40), ref: 'master', repo: {full_name: 'kubeflow/pipelines'}},
  user: {login: 'outsider'}, author_association: 'CONTRIBUTOR', labels: [],
  ...options.pr,
};
let baseTip = options.baseTip || 'b'.repeat(40);
const eventPR = structuredClone(pr);
if (options.oldHead) eventPR.head.sha = 'old-head';
const context = {repo: {owner: 'kubeflow', repo: 'pipelines'}, runId: 99,
  eventName: options.schedule ? 'schedule' : options.workflowRun ? 'workflow_run' : 'pull_request_target',
  payload: {pull_request: eventPR, action: options.action || 'opened',
    workflow_run: {event: 'pull_request', head_sha: 'head', head_branch: 'feature',
      head_repository: {owner: {login: 'contributor'}, full_name: 'contributor/pipelines'}}}};
let published = false;
let status = options.initialStatus;
let description = options.initialDescription;
const core = {info: () => {}, setOutput: (key, value) => {outputs[key] = value;}};
const methods = {files: {}, timeline: {}, pulls: {}, statuses: {}};
methods.runs = async () => ({data: {total_count: 0, workflow_runs: []}});
methods.associated = {};
const statusHistory = options.statusHistory || [];
if (options.initialStatus) statusHistory.push({context: 'ci-passed',
  created_at: options.registrationStartedAt || new Date().toISOString()});
const github = {graphql: async (query, variables) => {
  calls.push(['base-workflows', variables]);
  if (options.inventoryFailure) throw Error('Workflow tree unavailable');
  const content = `name: Frontend
on:
  pull_request:
    paths: [frontend/**]
jobs:
  test:
    runs-on: ubuntu-latest
`;
  const definitions = [{name: workflowName, content}];
  if (options.upgradePolicy) definitions.push({name: 'upgrade-test.yml', content:
    content + (options.upgradePolicy === 'paused' ? '    if: false\n' : '')});
  return {repository: {nameWithOwner: 'kubeflow/pipelines', object: {__typename: 'Tree',
    entries: definitions.map(({name, content}) => ({name, type: 'blob', mode: 33188,
      object: {__typename: 'Blob', text: content, byteSize: Buffer.byteLength(content),
        isBinary: false, isTruncated: false}}))}}};
}, rest: {
  pulls: {listFiles: methods.files, list: methods.pulls,
    get: async () => {
      if (options.pullFailure) throw Error('Pull read unavailable');
      return {data: structuredClone(pr)};
    }},
  actions: {listWorkflowRunsForRepo: methods.runs},
  checks: {listForRef: async () => {
    if (options.checkApiFailure) throw Error('Checks API unavailable');
    const status = published && options.drift === 'external-rerun' ? 'in_progress' :
      options.checkStatus || 'completed';
    const conclusion = status !== 'completed' ? null :
      published && options.drift === 'external-failure' ? 'failure' :
      options.pollPassed === false && !(options.recoverLast && cycle === options.cycles - 1) ? 'failure' : 'success';
    return {data: {total_count: 1, check_runs: [{id: 700, name: 'DCO',
      head_sha: 'head', app: {id: 1861, slug: 'dco'}, status, conclusion,
      started_at: '2026-09-07T11:00:00Z', completed_at: status === 'completed' ? '2026-09-07T12:00:00Z' : null,
      check_suite: {id: 900}}]}};
  }},
  issues: {listEventsForTimeline: methods.timeline,
    addLabels: async request => {calls.push(['add-label', request.labels]);},
    removeLabel: async request => {
      calls.push(['remove-label', request.name]);
      if (options.removeLabelFailure) throw Object.assign(Error('Label write unavailable'), {status: 403});
    }},
  repos: {getCombinedStatusForRef: {}, listCommitStatusesForRef: methods.statuses,
    listPullRequestsAssociatedWithCommit: methods.associated,
    createCommitStatus: async request => {
    statusHistory.push({context: 'ci-passed', created_at: options.registrationStartedAt || new Date().toISOString()});
    status = request.state;
    description = request.description;
    descriptions.push(description);
    targetUrls.push(request.target_url);
    calls.push(['status', request.state, request.sha]);
    if (request.state === 'success') {
      published = true;
      if (options.drift === 'hold') pr.labels = [{name: 'needs-ok-to-test'}];
      if (options.drift === 'base') pr.base.ref = 'release';
      if (options.drift === 'base-sha') pr.base.sha = 'new-base';
      if (options.drift === 'head') pr.head.sha = 'new-head';
      if (options.drift === 'closed') pr.state = 'closed';
    }
  }},
  git: {getRef: async () => ({data: {object: {sha: baseTip}}})},
}, paginate: async (method, params) => {
  if (method === methods.statuses) return statusHistory;
  if (method === methods.files) return [{filename: 'frontend/src/mlmd/Api.ts'}];
  if (method === methods.pulls) return options.ambiguous ? [pr, {...pr, number: 8}] : [pr];
  if (method === methods.timeline) {
    if (options.apiFailure) throw Error('API unavailable');
    return options.retarget ? [{event: 'base_ref_changed', created_at: '2026-09-07T12:00:00Z'}] : [];
  }
  if (method === methods.associated) {
    if (options.associationFailure) throw Error('Associated PRs API unavailable');
    return [{number: 7, merged_at: options.baseArrivedAt || '2026-09-07T10:00:00Z'}];
  }
  if (method === methods.runs) {
    if (options.missing) return [];
    const conclusion = published && options.drift === 'rerun' ? 'cancelled' : (options.conclusions?.[cycle] || options.conclusion || 'success');
    return [{path: `.github/workflows/${workflowName}`, id: 42, run_attempt: 1, event: 'pull_request', head_sha: 'head', head_branch: 'feature',
      head_repository: {full_name: 'contributor/pipelines'},
      status: options.runStatus || 'completed', conclusion: options.runStatus && options.runStatus !== 'completed' ? null : conclusion,
      created_at: options.fresh ? '2026-09-07T12:01:00Z' : '2026-09-07T11:00:00Z',
      run_started_at: '2026-09-07T12:02:00Z', pull_requests: [], ...options.runPatch}];
  }
  throw Error('Unexpected API request');
}};
github.paginate.iterator = async function* () {
  if (options.statusReadFailure) throw Error('Status read unavailable');
  yield {data: {statuses: status ? [{context: 'ci-passed', state: status, description}] : []}};
};
(async () => {
  let error;
  const recoveryBefore = options.verifyRecovery ? await gate.recoveryCandidates({github, context}) : null;
  for (cycle = 0; cycle < (options.cycles || 1); cycle++) {
  context.runId = 99 + cycle;
  try {await gate.prepare({github, context, core, root, recovery: {number: 7, head: eventPR.head.sha}});} catch (e) {error = e.message;}
  if (options.revokeBeforeFinal) pr.labels = [{name: 'needs-ok-to-test'}];
  if (options.recoverBeforeFinal) options.conclusion = 'success';
  if (options.mergeBeforeFinal) {pr.merged = true; pr.state = 'closed';}
  try {
    await gate.finalize({github, context, core, root, number: outputs.pr_number,
      head: outputs.head_sha, before: outputs.snapshot,
      pollSkipped: outputs.ready === 'false' && !error,
      pollPassed: outputs.ready === 'true' && !options.checkerFailure && (options.pollPassed !== false || (options.recoverLast && cycle === options.cycles - 1)) && !error});
  } catch (e) {error = e.message;}
  }
  const recoveryAfter = options.verifyRecovery ? await gate.recoveryCandidates({github, context}) : null;
  console.log(JSON.stringify({calls, outputs, error, status, descriptions, targetUrls, recoveryBefore, recoveryAfter}));
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

    def test_verified_ci_is_independent_of_author_association(self):
        authors = [('outsider', 'NONE'), ('outsider', 'CONTRIBUTOR'),
                   ('outsider', 'MEMBER'), ('outsider', 'OWNER'),
                   ('outsider', 'COLLABORATOR'), ('dependabot[bot]', 'NONE'),
                   ('renovate[bot]', 'NONE')]
        for author, association in authors:
            with self.subTest(author=author, association=association):
                result = exercise({
                    'pr': {
                        'user': {
                            'login': author
                        },
                        'author_association': association,
                    }
                })
                self.assertEqual(result['outputs']['ready'], 'true')
                self.assert_last_status(result, 'success')

    def test_complete_ci_publishes_pending_then_success(self):
        result = exercise()
        self.assertEqual(result['calls'][0], ['status', 'pending', 'head'])
        self.assert_last_status(result, 'success')

    def test_verified_ci_does_not_require_approval_label(self):
        for schedule in [False, True]:
            for labels in [[], ['ok-to-test']]:
                with self.subTest(schedule=schedule, labels=labels):
                    result = exercise({
                        'schedule': schedule,
                        'pr': {
                            'labels': [{
                                'name': name
                            } for name in labels]
                        },
                    })
                    self.assertEqual(result['outputs']['ready'], 'true')
                    self.assert_last_status(result, 'success')
                    self.assertEqual(
                        json.loads(result['outputs']['snapshot']),
                        [7, 'open', 'head', 'master', 'b' * 40, False])

    def test_membership_file_is_not_required_for_ci_publication(self):
        with mock.patch.dict(
                os.environ,
            {'KUBEFLOW_MEMBERS_FILE': '/nonexistent/kubeflow-members.json'}):
            self.assert_last_status(exercise(), 'success')

    def test_explicit_hold_blocks_authors_even_with_approval_label(self):
        for author in ['outsider', 'dependabot[bot]']:
            for approved in [False, True]:
                with self.subTest(author=author, approved=approved):
                    labels = ['needs-ok-to-test']
                    if approved:
                        labels.append('ok-to-test')
                    result = exercise({
                        'pr': {
                            'user': {
                                'login': author
                            },
                            'author_association': 'MEMBER',
                            'labels': [{
                                'name': name
                            } for name in labels],
                        }
                    })
                    self.assert_last_status(result, 'failure')
                    self.assertNotIn('ready', result['outputs'])

    def test_failed_poll_blocks_otherwise_complete_workflows(self):
        result = exercise({'pollPassed': False})
        self.assert_last_status(result, 'failure')
        self.assertEqual(result['descriptions'][-1],
                         'Check DCO (app 1861): failure')

    def test_missing_workflows_never_reach_poller(self):
        result = exercise({'missing': True})
        self.assertEqual(result['outputs']['ready'], 'false')
        self.assert_last_status(result, 'pending')

    def test_recovery_revisits_legacy_and_changed_base_success(self):
        script = r"""
const {recoveryCandidates} = require(process.argv[1]);
const requests = [];
const prs = ['success', 'failure', 'pending', 'missing', 'untrusted', 'revoked', 'stale-success', 'legacy-success', 'retarget-success'].map((state, i) => ({
  number: i + 1, head: {sha: state}, base: {ref: 'master', sha: 'b'.repeat(40)}, user: {login: state === 'untrusted' ? 'human' : 'dependabot[bot]'},
  labels: state === 'revoked' ? [{name: 'needs-ok-to-test'}] : [], author_association: 'NONE',
}));
const github = {paginate: async () => prs, rest: {pulls: {list: {}}, git: {
  getRef: async () => ({data: {object: {sha: 'b'.repeat(40)}}}),
}, repos: {
  getCombinedStatusForRef: async ({ref}) => {
    requests.push(ref);
    const state = ref.endsWith('success') ? 'success' : ref;
    const description = ref === 'legacy-success' ? 'Expected CI and all checks passed for this head.' :
      ref === 'stale-success' ? 'Expected CI and all checks passed; base policy f5b15f0f51bf0e3cbf5297bbe7629a426d5320f3955fbeb78d2de0c60e0e19a8.' : ref === 'retarget-success' ? 'Expected CI and all checks passed; base policy 3088c340a17fbd230c03d711b568e0962de9b3aa3565c6ab227bd0638dae905e.' : 'Expected CI and all checks passed; base policy bf60a45a7f48d31d1cf806d3517d6cc8d24985621eaf55baea8599faee6e508c.';
    return {data: {statuses: ref === 'missing' ? [] : [{context: 'ci-passed', state, description}]}};
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
        }, {
            'number': 5,
            'head': 'untrusted'
        }, {
            'number': 7,
            'head': 'stale-success'
        }, {
            'number': 8,
            'head': 'legacy-success'
        }, {
            'number': 9,
            'head': 'retarget-success'
        }])
        self.assertEqual(actual['requests'], [
            'success', 'failure', 'pending', 'missing', 'untrusted',
            'stale-success', 'legacy-success', 'retarget-success'
        ])

    def test_recovery_revokes_green_when_base_tip_advances(self):
        script = r"""
const {recoveryCandidates} = require(process.argv[1]);
const crypto = require('node:crypto');
const B1 = 'a'.repeat(40);
const B2 = 'c'.repeat(40);
const publishedStamp = crypto.createHash('sha256')
  .update(JSON.stringify(['master', B1])).digest('hex');
async function run(liveTip) {
  const pr = {number: 42, head: {sha: 'H'}, base: {ref: 'master', sha: B1},
    labels: [], author_association: 'NONE'};
  const github = {rest: {pulls: {list: {}},
    git: {getRef: async () => ({data: {object: {sha: liveTip}}})},
    repos: {getCombinedStatusForRef: async ({ref}) => ({
      data: {statuses: [{context: 'ci-passed', state: 'success',
        description: 'Expected CI and all checks passed; base policy ' + publishedStamp + '.'}]}
    })}}};
  github.paginate = async () => [pr];
  github.paginate.iterator = async function* (method, params) {
    yield {data: {statuses: [{context: 'other', state: 'success'}]}};
    yield await github.rest.repos.getCombinedStatusForRef(params);
  };
  return recoveryCandidates({github, context: {repo: {owner: 'o', repo: 'r'}}});
}
(async () => {
  const advancedStamp = crypto.createHash('sha256')
    .update(JSON.stringify(['master', B2])).digest('hex');
  const noAdvance = await run(B1);
  const advanced = await run(B2);
  console.log(JSON.stringify({noAdvance, advanced, publishedStamp, advancedStamp}));
})().catch(e => {console.error(e); process.exit(1);});
"""
        result = subprocess.run(
            ['node', '-e', script, str(MODULE)],
            check=True,
            capture_output=True,
            text=True)
        actual = json.loads(result.stdout)
        self.assertNotEqual(actual['publishedStamp'], actual['advancedStamp'])
        self.assertEqual(actual['noAdvance'], [])
        self.assertEqual(actual['advanced'], [{'number': 42, 'head': 'H'}])

    def test_base_advance_revokes_stale_green_across_full_reconciliation(self):
        B1 = 'a' * 40
        B2 = 'c' * 40
        published_stamp = hashlib.sha256(
            json.dumps(['master', B1],
                       separators=(',', ':')).encode()).hexdigest()
        initial = ('Expected CI and all checks passed; base policy ' +
                   published_stamp + '.')
        # The PR's frozen base is B1 but master has advanced to B2. Its stored
        # ci-passed success is stamped B1, and the run was created before B2
        # became reachable on master. Reconciliation must revoke the green
        # (pending) using the run's immutable creation time against B2's push
        # arrival time, and the next sweep must STILL select the PR. Cover both
        # the run-scoped association (already mutated to B2) and the
        # empty-pull_requests fallback.
        for run_pull_requests in ([], [{
                'number': 7,
                'base': {
                    'ref': 'master',
                    'sha': B2,
                },
        }]):
            with self.subTest(run_pull_requests=run_pull_requests):
                result = exercise({
                    'schedule': True,
                    'verifyRecovery': True,
                    'baseTip': B2,
                    'baseArrivedAt': '2026-09-07T12:00:00Z',
                    'pr': {
                        'base': {
                            'sha': B1,
                            'ref': 'master',
                            'repo': {
                                'full_name': 'kubeflow/pipelines',
                            },
                        },
                    },
                    'initialStatus': 'success',
                    'initialDescription': initial,
                    'runPatch': {
                        'pull_requests': run_pull_requests,
                    },
                })
                # Discovery: the stale success (stamped B1) is selected because
                # the live tip B2 no longer matches the stored stamp.
                self.assertEqual(result['recoveryBefore'], [{
                    'number': 7,
                    'head': 'head'
                }], result)
                # Reconciliation revokes the green instead of re-stamping it.
                self.assertNotIn(['status', 'success', 'head'], result['calls'])
                self.assert_last_status(result, 'pending')
                self.assertNotIn(['add-label', ['ci-passed']], result['calls'])
                self.assertIn(['remove-label', 'ci-passed'], result['calls'])
                # Next sweep still selects the PR: it is not green.
                self.assertEqual(result['recoveryAfter'], [{
                    'number': 7,
                    'head': 'head'
                }], result)

    def test_success_records_the_validated_base_policy(self):
        result = exercise()
        self.assert_last_status(result, 'success')
        self.assertEqual(
            result['descriptions'][-1],
            'Expected CI and all checks passed; base policy bf60a45a7f48d31d1cf806d3517d6cc8d24985621eaf55baea8599faee6e508c.'
        )

    def test_green_recovery_requires_newly_enabled_upgrade_workflow(self):
        for policy, expected in [('paused', 'success'), ('enabled', 'pending')]:
            with self.subTest(policy=policy):
                result = exercise({
                    'schedule': True,
                    'initialStatus': 'success',
                    'upgradePolicy': policy,
                })
                self.assert_last_status(result, expected)
                if policy == 'enabled':
                    self.assertEqual(result['outputs']['ready'], 'false')
                    self.assertNotIn(['status', 'success', 'head'],
                                     result['calls'])

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
                'missing': True,
                'registrationStartedAt': '2026-09-07T11:00:00Z'
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
        description = 'Check DCO (app 1861): failure'
        result = exercise({
            'schedule': True,
            'pollPassed': False,
            'initialStatus': 'failure',
            'initialDescription': description,
            'cycles': 6
        })
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'], [])
        self.assertEqual(result['status'], 'failure')
        result = exercise({
            'schedule': True,
            'pollPassed': False,
            'initialStatus': 'failure',
            'initialDescription': description,
            'cycles': 6,
            'recoverLast': True
        })
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'],
                         [['status', 'success', 'head']])

    def test_recovery_refreshes_legacy_failure_with_current_workflow_reason(
            self):
        legacy_reason = 'Cannot verify CI evidence; inspect CI Check and retry.'
        for options, expected in [
            ({
                'conclusion': 'skipped'
            },
             '.github/workflows/frontend.yml: latest run is completed/skipped'),
            ({
                'missing': True,
                'registrationStartedAt': '2026-09-07T11:00:00Z'
            },
             '.github/workflows/frontend.yml: expected workflow has not registered after 15 minutes; inspect its trigger and approval state'
            ),
        ]:
            with self.subTest(options=options):
                result = exercise({
                    'schedule': True,
                    'initialStatus': 'failure',
                    'initialDescription': legacy_reason,
                    'cycles': 6,
                    **options,
                })
                self.assertEqual(
                    [c for c in result['calls'] if c[0] == 'status'],
                    [['status', 'failure', 'head']])
                self.assertEqual(result['descriptions'], [expected])
                self.assertEqual(
                    result['targetUrls'],
                    ['https://github.com/kubeflow/pipelines/actions/runs/99'])

    def test_repeated_label_failure_preserves_workflow_reason_without_churn(
            self):
        result = exercise({
            'schedule': True,
            'initialStatus': 'failure',
            'initialDescription': 'Legacy failure',
            'conclusion': 'skipped',
            'removeLabelFailure': True,
            'cycles': 6,
        })
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'],
                         [['status', 'failure', 'head']])
        self.assertEqual(
            result['descriptions'],
            ['.github/workflows/frontend.yml: latest run is completed/skipped'])
        self.assertEqual(result['status'], 'failure')
        self.assertIn('Label write unavailable', result['error'])

    def test_recovery_publishes_each_changed_reason_once_then_success(self):
        reason = '.github/workflows/frontend.yml: latest run is completed/'
        result = exercise({
            'schedule': True,
            'initialStatus': 'failure',
            'initialDescription': reason + 'skipped',
            'conclusions': ['skipped', 'failure', 'failure', 'success'],
            'cycles': 4,
        })
        self.assertEqual(
            [c for c in result['calls'] if c[0] == 'status'],
            [['status', 'failure', 'head'], ['status', 'success', 'head']])
        self.assertEqual(result['descriptions'][0], reason + 'failure')
        self.assertEqual(result['targetUrls'], [
            'https://github.com/kubeflow/pipelines/actions/runs/100',
            'https://github.com/kubeflow/pipelines/actions/runs/102',
        ])

    def test_repeated_long_failure_descriptions_compare_published_length(self):
        workflow_name = 'frontend-' + 'x' * 120 + '.yml'
        reason = (f'.github/workflows/{workflow_name}: '
                  'latest run is completed/skipped')
        result = exercise({
            'schedule': True,
            'initialStatus': 'failure',
            'initialDescription': 'Legacy failure',
            'workflowName': workflow_name,
            'conclusions': ['skipped', 'failure', 'failure'],
            'cycles': 3,
        })
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'],
                         [['status', 'failure', 'head']])
        self.assertEqual(result['descriptions'], [reason[:140]])
        self.assertEqual(len(result['descriptions'][0]), 140)

    def test_evidence_recovery_stays_pending_when_poller_was_skipped(self):
        result = exercise({'conclusion': 'skipped', 'recoverBeforeFinal': True})
        self.assertEqual(result['outputs']['ready'], 'false')
        self.assert_last_status(result, 'pending')
        self.assertEqual(
            result['descriptions'][-1],
            'CI changed since initial assessment; awaiting check validation.')

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
        for hold in ['do-not-merge/hold', 'needs-ok-to-test']:
            for passed in [True, False]:
                with self.subTest(hold=hold, passed=passed):
                    result = exercise({
                        'schedule': True,
                        'pollPassed': passed,
                        'pr': {
                            'labels': [{
                                'name': hold
                            }]
                        },
                    })
                    labels = [
                        call for call in result['calls']
                        if call[0] in ('add-label', 'remove-label')
                    ]
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
                    }), 'pending')
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

    def test_rerun_clears_previous_failure_to_pending_without_status_churn(
            self):
        for schedule in [False, True]:
            result = exercise({
                'schedule': schedule,
                'workflowRun': not schedule,
                'initialStatus': 'failure',
                'initialDescription': 'Earlier failed attempt',
                'runStatus': 'in_progress',
                'cycles': 6,
            })
            self.assertEqual([c for c in result['calls'] if c[0] == 'status'],
                             [['status', 'pending', 'head']])
            self.assertNotIn(['add-label', ['ci-passed']], result['calls'])

    def test_external_pending_is_pending_and_recovers(self):
        result = exercise({'checkStatus': 'in_progress'})
        self.assert_last_status(result, 'pending')
        self.assertNotIn(['add-label', ['ci-passed']], result['calls'])
        self.assert_last_status(exercise({'schedule': True}), 'success')

    def test_actual_failure_wins_over_other_pending_evidence(self):
        self.assert_last_status(
            exercise({
                'runStatus': 'in_progress',
                'pollPassed': False
            }), 'failure')
        self.assert_last_status(
            exercise({
                'conclusion': 'failure',
                'checkStatus': 'in_progress'
            }), 'failure')

    def test_pinned_checker_failure_still_prevents_success(self):
        result = exercise({'checkerFailure': True})
        self.assert_last_status(result, 'failure')
        self.assertIn('Cannot verify all checks passed',
                      result['descriptions'][-1])

    def test_external_rerun_during_publication_revokes_success_to_pending(self):
        result = exercise({'drift': 'external-rerun'})
        self.assertIn(['status', 'success', 'head'], result['calls'])
        self.assert_last_status(result, 'pending')
        self.assertEqual(result['calls'][-1], ['remove-label', 'ci-passed'])

    def test_malformed_expected_workflow_metadata_cannot_publish_success(self):
        for field, value in [('id', None), ('run_attempt', 0),
                             ('created_at', 'invalid'),
                             ('run_started_at', None)]:
            with self.subTest(field=field):
                result = exercise({'runPatch': {field: value}})
                self.assert_last_status(result, 'failure')
                self.assertNotIn(['status', 'success', 'head'], result['calls'])
                self.assertNotIn(['add-label', ['ci-passed']], result['calls'])

    def test_external_api_failure_cannot_publish_success(self):
        result = exercise({'checkApiFailure': True})
        self.assert_last_status(result, 'failure')
        self.assertNotIn(['status', 'success', 'head'], result['calls'])

    def test_registration_deadline_uses_earliest_status_not_latest(self):
        result = exercise({
            'missing':
                True,
            'statusHistory': [{
                'context': 'ci-passed',
                'created_at': '2026-09-07T11:00:00Z'
            },],
        })
        self.assert_last_status(result, 'failure')
        self.assertIn('after 15 minutes', result['descriptions'][-1])

    def test_invalid_registration_timestamp_fails_closed(self):
        result = exercise({'missing': True, 'registrationStartedAt': 'invalid'})
        self.assert_last_status(result, 'failure')
        self.assertIn('registration timestamp', result['error'])

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

    def test_held_and_closed_prs_fail(self):
        for pr in [{
                'labels': [{
                    'name': 'needs-ok-to-test'
                }]
        }, {
                'state': 'closed'
        }]:
            with self.subTest(pr=pr):
                self.assert_last_status(exercise({'pr': pr}), 'failure')

    def test_closed_unmerged_pr_invalidates_prior_success_without_polling(self):
        result = exercise({
            'action': 'closed',
            'initialStatus': 'success',
            'pr': {
                'state': 'closed',
                'merged': False
            },
        })
        self.assertNotIn('error', result)
        self.assert_last_status(result, 'failure')
        self.assertNotIn('ready', result['outputs'])
        self.assertIn(['remove-label', 'ci-passed'], result['calls'])
        self.assertNotIn(['add-label', ['ci-passed']], result['calls'])
        self.assertNotIn(['status', 'success', 'head'], result['calls'])

    def test_merged_pr_preserves_published_status(self):
        result = exercise({
            'action': 'closed',
            'initialStatus': 'success',
            'pr': {
                'state': 'closed',
                'merged': True
            },
        })
        self.assertNotIn('error', result)
        self.assertNotIn('ready', result['outputs'])
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'], [])
        self.assertNotIn(['remove-label', 'ci-passed'], result['calls'])
        self.assertNotIn(['add-label', ['ci-passed']], result['calls'])

    def test_merge_during_reconciliation_preserves_status(self):
        # PR open when prepare runs (publishes provisional pending), merged by
        # the time finalize re-reads it. finalize must publish nothing.
        result = exercise({
            'initialStatus': 'success',
            'mergeBeforeFinal': True,
        })
        self.assertNotIn('error', result)
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'],
                         [['status', 'pending', 'head']])
        self.assertNotIn(['add-label', ['ci-passed']], result['calls'])

    def test_merged_pr_resolve_failure_does_not_publish_failure(self):
        # A transient pull read failure on a merged PR's closed event must not
        # republish a failure onto the merged head.
        result = exercise({
            'action': 'closed',
            'pr': {
                'state': 'closed',
                'merged': True
            },
            'pullFailure': True,
        })
        self.assertEqual([c for c in result['calls'] if c[0] == 'status'], [])

    def test_open_pr_resolve_failure_publishes_failure(self):
        result = exercise({'pullFailure': True})
        self.assertIn(['status', 'failure', 'head'], result['calls'])

    def test_revocation_before_publication_fails(self):
        self.assert_last_status(
            exercise({'revokeBeforeFinal': True}), 'failure')

    def test_publication_reconciles_full_state_and_ci(self):
        for drift in [
                'hold', 'base', 'base-sha', 'head', 'closed', 'rerun',
                'external-failure'
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

    def test_base_workflow_lookup_failure_cannot_publish_success(self):
        result = exercise({'inventoryFailure': True})
        self.assertIn('Workflow tree unavailable', result['error'])
        self.assert_last_status(result, 'failure')
        self.assertNotIn(['status', 'success', 'head'], result['calls'])

    def test_each_publication_boundary_reads_the_immutable_base(self):
        result = exercise()
        self.assert_last_status(result, 'success')
        reads = [c[1] for c in result['calls'] if c[0] == 'base-workflows']
        self.assertEqual(len(reads), 3)
        for read in reads:
            self.assertEqual(
                read, {
                    'owner': 'kubeflow',
                    'repo': 'pipelines',
                    'expression': 'b' * 40 + ':.github/workflows',
                })


if __name__ == '__main__':
    unittest.main()
