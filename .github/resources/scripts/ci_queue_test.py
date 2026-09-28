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
"""Exercise the trusted release queue status publisher with API fixtures."""

import json
from pathlib import Path
import subprocess
import unittest

import yaml

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_passed.js'
EXPECTED_MODULE = ROOT / '.github/resources/scripts/ci_expected_workflows.js'
SHA = 'a' * 40


def exercise(options=None):
    script = r'''
const options = JSON.parse(process.argv[3]);
const sha = 'a'.repeat(40), base = 'b'.repeat(40), prHead = 'c'.repeat(40);
const priorSha = 'd'.repeat(40), priorHead = 'e'.repeat(40);
const paths = ['.github/workflows/frontend.yml', '.github/workflows/pre-commit.yml',
  '.github/workflows/build-tools-images-merge-group.yml'];
const inventory = paths.map(path => ({path, pull_request: {},
  merge_group: {types: ['checks_requested'], branches: ['release-2.18']},
  disabled_for_migration: false}));
inventory[2].pull_request = null;
inventory.push({path: '.github/workflows/build-tools-images.yml', pull_request: {},
  merge_group: null, disabled_for_migration: false});
if (options.missingTrigger) inventory[0].merge_group = null;
if (options.missingEquivalent) inventory.splice(2, 1);
const expectedModule = require.resolve(process.argv[2]);
require.cache[expectedModule] = {id: expectedModule, filename: expectedModule, loaded: true, exports: {
  applicable: () => true,
  loadBaseInventory: async () => {
    if (options.inventoryError) throw Error('Inventory unavailable');
    return {inventory: {workflows: inventory}};
  },
}};
const gate = require(process.argv[1]);
const statuses = new Map();
const statusHistory = new Map();
if (options.prefillStatuses) statusHistory.set(sha,
  Array.from({length: options.prefillStatuses}, () => ({
    context: 'ci-passed-release', state: 'success', description: 'Prior queue check',
  })));
if (options.initialStatus) {
  statuses.set(sha, options.initialStatus);
  statuses.set(priorSha, options.initialStatus);
}
let pr = {number: 7, node_id: 'PR_7', state: 'open', draft: false,
  head: {sha: prHead, repo: {full_name: 'contributor/pipelines',
    name: 'pipelines', owner: {login: 'contributor'}}},
  base: {sha: base, ref: 'release-2.18'},
  user: {login: 'human'}, author_association: 'MEMBER',
  labels: ['lgtm', 'approved'].map(name => ({name}))};
if (options.labels) pr.labels = options.labels.map(name => ({name}));
if (options.dependabot) pr.user.login = 'dependabot[bot]';
if (options.retargetAway) pr.base.ref = 'master';
if (options.sameRepo) pr.head.repo = {full_name: 'kubeflow/pipelines',
  name: 'pipelines', owner: {login: 'kubeflow'}};
let prior = {number: 8, node_id: 'PR_8', state: 'open', draft: false,
  head: {sha: options.priorHeadDrift ? 'f'.repeat(40) : priorHead,
    repo: {full_name: 'contributor/pipelines',
      name: 'pipelines', owner: {login: 'contributor'}}},
  base: {sha: base, ref: 'release-2.18'},
  user: {login: 'human'}, author_association: 'MEMBER',
  labels: ['lgtm', 'approved'].map(name => ({name}))};
if (options.priorLabels) prior.labels = options.priorLabels.map(name => ({name}));
let dequeued = false;
const entries = () => ({repository: {nameWithOwner: 'kubeflow/pipelines',
  mergeQueue: {configuration: {maximumEntriesToBuild: options.buildConcurrency || 1,
    maximumEntriesToMerge: options.groupSize || 1,
    mergingStrategy: options.strategy || 'ALLGREEN'},
    entries: {nodes: options.staleQueue ? [] : [
      ...(options.priorEntry ? [{id: 'ENTRY_8', position: 0,
        headCommit: options.priorHeadNull ? null : {oid: priorSha},
        pullRequest: {id: 'PR_8', number: 8, headRefOid: priorHead}}] : []),
      {id: 'ENTRY_7', position: options.priorEntry ? 1 : 0,
        headCommit: options.nullCurrentHead ? null : {oid: sha},
        pullRequest: {id: 'PR_7', number: 7,
          headRefOid: options.oldHead ? 'old' : prHead}},
      ...(options.duplicateQueue ? [{position: 2, headCommit: {oid: sha},
        pullRequest: {number: 7, headRefOid: prHead}}] : [])],
      pageInfo: {hasNextPage: false, endCursor: null}}}}});
const calls = [];
if (options.initialFences) statusHistory.set(sha,
  options.initialFences.map(([runId, attempt]) => ({
    context: 'ci-passed-release', state: 'pending',
    description: `queue-fence:1.1:${runId}:${attempt}`,
  })));
let currentAttempt = options.currentAttempt || 1;
let currentState = options.currentState || 'completed';
let currentConclusion = options.currentConclusion || 'success';
let listedAttempt = options.listedAttempt || 1;
const runData = (id, head, attempt, status, conclusion) => ({
  id, path: paths[(id - 1) % 10], event: 'merge_group', head_sha: head,
  head_branch: 'gh-readonly-queue/release-2.18/pr-7-abc',
  head_repository: {full_name: 'kubeflow/pipelines'},
  run_attempt: attempt, status, conclusion,
  created_at: '2026-09-26T00:00:00Z', run_started_at: '2026-09-26T00:01:00Z',
});
const runs = head => paths.filter(path => path !== options.missing).map((path, index) => ({
  ...runData(index + 1 + (head === priorSha ? 10 : 0), head,
    index === 0 && head === sha ? listedAttempt : 1,
    path === options.inProgress ? 'in_progress' : 'completed',
    path === options.failed ? 'failure' : 'success'),
  status: path === options.inProgress ? 'in_progress' : 'completed',
  conclusion: path === options.failed ? 'failure' : 'success',
}));
const github = {
  graphql: async (query, variables) => {
    if (query.includes('query ReleaseWorkflowTrees')) {
      calls.push(['workflow-trees', query]);
      if (options.workflowTreeError) throw Error('Workflow tree unavailable');
      const repository = {nameWithOwner: `${variables.owner}/${variables.repo}`};
      for (const [, alias, revision] of query.matchAll(/(r\d+): object\(expression: "([0-9a-f]{40}):\.github\/workflows"\)/g)) {
        const changed = revision === prHead && options.workflowChange ||
          revision === priorHead && options.priorWorkflowChange ||
          revision === '8'.repeat(40) && options.workflowChangeReverted;
        repository[alias] = options.workflowTreeMissing ? null :
          {oid: options.workflowOids?.[revision] ||
            (changed ? '2'.repeat(40) : '1'.repeat(40))};
      }
      return {repository};
    }
    if (options.queueError) throw Error('Queue unavailable');
    if (query.includes('mutation DequeueReleasePR')) {
      calls.push(['dequeue', variables.id]);
      if (options.dequeueDenied) throw Error('Dequeue denied');
      dequeued = true;
      return {dequeuePullRequest: {mergeQueueEntry: {id:
        variables.id === 'PR_8' ? 'ENTRY_8' : 'ENTRY_7'}}};
    }
    if (query.includes('query ReleaseQueuedPR')) {
      calls.push(['queue-entry', variables.id]);
      if (options.readbackError && dequeued) throw Error('Readback unavailable');
      return {node: {id: variables.id,
        mergeQueueEntry: !dequeued || options.readbackStillPresent ?
          {id: variables.id === 'PR_8' ? 'ENTRY_8' : 'ENTRY_7'} : null}};
    }
    return entries();
  },
  rest: {
    pulls: {get: async request => ({data: structuredClone(
      request.pull_number === 8 ? prior : pr)})},
    repos: {
      compareCommitsWithBasehead: async request => {
        calls.push(['compare', request.basehead]);
        if (options.compareError) throw Error('Compare unavailable');
        const target = request.basehead.endsWith(`:${priorHead}`) ? prior : pr;
        const comparedHead = request.basehead.split(':').at(-1);
        const prefix = options.prefixComparisons?.[comparedHead];
        const commits = prefix?.commits || (target === pr && options.commits) ||
          (options.workflowChangeReverted && target === pr ? [
          {sha: '8'.repeat(40), parents: [{sha: base}]},
          {sha: prHead, parents: [{sha: '8'.repeat(40)}]},
        ] : [{sha: target.head.sha,
          parents: options.parentMissing ? [] : [{sha: base}]}]);
        return {data: {
          base_commit: {sha: options.compareBaseDrift ? '0'.repeat(40) : target.base.sha},
          merge_base_commit: options.mergeBaseMissing ? null :
            {sha: prefix?.mergeBaseSha || base},
          commits, ahead_by: commits.length,
          total_commits: options.historyTruncated ? commits.length + 1 : commits.length,
        }};
      },
      getBranch: async () => ({data: {commit: {sha: base}}}),
      getCombinedStatusForRef: {},
      listCommitStatusesForRef: {},
      createCommitStatus: async request => {
        if (options.statusWriteError) throw Error('Status write unavailable');
        if ((statusHistory.get(request.sha) || []).filter(status =>
          status.context === request.context).length >= 1000) {
          throw Error('Commit status context limit reached');
        }
        calls.push([request.context, request.state, request.sha]);
        statuses.set(request.sha, request.state);
        statusHistory.set(request.sha, [{context: request.context,
          state: request.state, description: request.description},
        ...(statusHistory.get(request.sha) || [])]);
        if (request.state === 'success' && options.removeLabelAfterSuccess) {
          pr.labels = [{name: 'lgtm'}];
        }
        if (request.state === 'success' && options.removePriorLabelAfterSuccess) {
          prior.labels = [{name: 'lgtm'}];
        }
      },
    },
    actions: {
      listWorkflowRunsForRepo: {},
      getWorkflowRun: async request => {
        calls.push(['current-run', request.run_id]);
        const data = runData(request.run_id,
          request.run_id > 10 ? priorSha : sha,
          request.run_id === 1 ? currentAttempt : 1,
          request.run_id === 1 ? currentState : 'completed',
          request.run_id === 1 ? currentConclusion : 'success');
        if (options.omitCurrentAttempt && request.run_id === 1) {
          delete data.run_attempt;
        }
        return {data};
      },
      getWorkflowRunAttempt: async request => {
        calls.push(['exact-attempt', request.run_id, request.attempt_number]);
        return {data: runData(request.run_id,
        request.run_id > 10 ? priorSha : sha,
        request.attempt_number,
        request.attempt_number > currentAttempt ? 'in_progress' : 'completed',
        request.attempt_number > currentAttempt ? null :
          request.attempt_number === 2 && options.attemptTwoFailed ? 'failure' : 'success')};
      },
    },
  },
  paginate: async (method, request) => {
    if (method === github.rest.repos.listCommitStatusesForRef) {
      if (options.staleStatusRead) return (statusHistory.get(request.ref) || []).slice(1);
      return statusHistory.get(request.ref) || [];
    }
    if (options.runsError) throw Error('Runs unavailable');
    calls.push(['runs-for', request.event, request.head_sha]);
    return runs(request.head_sha);
  },
};
github.paginate.iterator = async function* (_, request) {
  const status = statuses.get(request.ref);
  yield {data: {statuses: status ? [{context: 'ci-passed-release', state: status}] : []}};
};
const context = {repo: {owner: 'kubeflow', repo: 'pipelines'}, runId: 99,
  runAttempt: 1,
  eventName: options.schedule ? 'schedule' : options.labelEvent || options.dequeueEvent ?
    'pull_request_target' : 'workflow_run',
  payload: {action: options.action || 'completed',
    pull_request: {number: options.labelEventEarlier ? 8 : 7,
      base: {ref: options.retargetAway ? 'master' : 'release-2.18'}},
    changes: options.retargetAway ? {base: {ref: {from: 'release-2.18'}}} : {},
    workflow_run: {event: 'merge_group', head_sha: sha,
      id: options.malformedAttempt ? undefined : 1,
      run_attempt: options.malformedAttempt ? undefined : (options.rerunAttempt || 1),
      head_branch: options.wrongBranch ? 'feature' : 'gh-readonly-queue/release-2.18/pr-7-abc',
      head_repository: {full_name: 'kubeflow/pipelines'}}}};
const core = {info: () => {}};
(async () => {
  let error;
  try {
    if (options.fenceOnly) {
      context.payload.action = 'in_progress';
      context.payload.workflow_run.run_attempt = 2;
      await gate.queueStatus(github, context, sha, 'pending',
        gate.queueInvalidationDescription(context));
      context.eventName = 'schedule';
      context.runId = 100;
      await gate.queueStatus(github, context, sha, 'pending',
        gate.queueInvalidationDescription(context));
      currentAttempt = 2;
      currentState = 'completed';
      currentConclusion = options.omitCurrentAttempt || options.attemptTwoFailed ?
        'success' : 'failure';
      const fence = await gate.queueRerunFence(github, context, sha);
      console.log(JSON.stringify({calls, status: statuses.get(sha),
        fenceState: fence.state, error}));
      return;
    } else if (options.dequeueEvent) {
      const candidates = await gate.queueEventCandidates({github, context});
      let retiredState;
      if (options.checkRetired) {
        context.eventName = 'schedule';
        context.runId = 100;
        await gate.queueStatus(github, context, sha, 'pending',
          gate.queueInvalidationDescription(context));
        retiredState = (await gate.queueRerunFence(github, context, sha)).state;
      }
      console.log(JSON.stringify({calls, status: statuses.get(sha) || null,
        priorStatus: statuses.get(priorSha) || null,
        candidates, dequeued, retiredState, error}));
      return;
    } else if (options.labelEvent) {
      const affected = await gate.queueHeads({github, context});
      for (const head of affected) {
        await gate.queueStatus(github, context, head, 'pending',
          gate.queueInvalidationDescription(context));
      }
      for (const head of affected) {
        await gate.reconcileQueue({github, context, core,
          recovery: {sha: head}, alreadyPending: true});
      }
    } else if (options.rerunSequence) {
      context.payload.action = 'in_progress';
      context.payload.workflow_run.run_attempt = 2;
      await gate.reconcileQueue({github, context, core, recovery: {sha}});
      const afterStart = statuses.get(sha);
      context.eventName = 'schedule';
      context.runId = 100;
      context.payload.action = 'completed';
      await gate.reconcileQueue({github, context, core, recovery: {sha}});
      const afterSchedule = statuses.get(sha);
      if (options.rerunSequence === 'recover' ||
          options.rerunSequence === 'third-attempt' ||
          options.rerunSequence === 'failed-then-success') {
        currentAttempt = 2;
        currentState = 'completed';
        currentConclusion = options.attemptTwoFailed ? 'failure' : 'success';
        listedAttempt = 2;
        if (options.rerunSequence === 'third-attempt') {
          context.eventName = 'workflow_run';
          context.runId = 101;
          context.payload.action = 'in_progress';
          context.payload.workflow_run.run_attempt = 3;
          await gate.reconcileQueue({github, context, core, recovery: {sha}});
          currentAttempt = 3;
          currentState = 'in_progress';
          currentConclusion = null;
          context.eventName = 'schedule';
          context.runId = 102;
          await gate.reconcileQueue({github, context, core, recovery: {sha}});
        }
        if (options.rerunSequence === 'failed-then-success') {
          context.eventName = 'workflow_run';
          context.runId = 101;
          context.payload.action = 'in_progress';
          context.payload.workflow_run.run_attempt = 3;
          await gate.reconcileQueue({github, context, core, recovery: {sha}});
          currentAttempt = 3;
          currentState = 'completed';
          currentConclusion = 'success';
          listedAttempt = 3;
          context.eventName = 'schedule';
          context.runId = 102;
          await gate.reconcileQueue({github, context, core, recovery: {sha}});
        }
        if (options.rerunSequence !== 'third-attempt') {
          context.runId = 103;
          await gate.reconcileQueue({github, context, core, recovery: {sha}});
        }
      }
      console.log(JSON.stringify({calls, status: statuses.get(sha),
        afterStart, afterSchedule, error}));
      return;
    } else {
      await gate.reconcileQueue({github, context, core, recovery: {sha}});
    }
  }
  catch (caught) {error = caught.message;}
  console.log(JSON.stringify({calls, status: statuses.get(sha) || null,
    priorStatus: statuses.get(priorSha) || null, error}));
})().catch(error => {console.error(error); process.exit(1);});
'''
    result = subprocess.run([
        'node', '-e', script,
        str(MODULE),
        str(EXPECTED_MODULE),
        json.dumps(options or {})
    ],
                            check=True,
                            capture_output=True,
                            text=True)
    return json.loads(result.stdout)


class QueueCITest(unittest.TestCase):

    def test_queue_matrix_uses_one_sha_per_writer_and_invalidates_first(self):
        workflow = yaml.safe_load(
            (ROOT / '.github/workflows/ci-checks.yml').read_text())
        jobs = workflow['jobs']
        invalidate = jobs['invalidate_queue_status']
        validate = jobs['check_queue_status']
        self.assertIn('invalidate_queue_status', validate['needs'])
        self.assertEqual(invalidate['concurrency']['group'],
                         validate['concurrency']['group'])
        self.assertIn('matrix.candidate.sha', validate['concurrency']['group'])
        self.assertEqual(invalidate['strategy']['matrix'],
                         validate['strategy']['matrix'])
        self.assertIn('queueInvalidationDescription(context)',
                      invalidate['steps'][-1]['with']['script'])
        discovery = jobs['queue_candidates']
        self.assertNotIn('concurrency', discovery)
        self.assertIn('statuses', discovery['permissions'])
        self.assertEqual(discovery['permissions']['pull-requests'], 'write')
        self.assertIn("github.event.changes.base.ref.from == 'release-2.18'",
                      discovery['if'])
        self.assertIn('queueEventCandidates({github, context})',
                      discovery['steps'][-1]['with']['script'])
        self.assertIn('queueInvalidationDescription(context)',
                      discovery['steps'][-1]['with']['script'])

    def test_exact_group_sha_passes_only_after_all_release_workflows(self):
        result = exercise()
        self.assertEqual(result['status'], 'success', result)
        statuses = [
            call for call in result['calls'] if call[0] == 'ci-passed-release'
        ]
        self.assertEqual(statuses, [['ci-passed-release', 'pending', SHA],
                                    ['ci-passed-release', 'success', SHA]])
        self.assertEqual(
            len([call for call in result['calls'] if call[0] == 'current-run']),
            3)

    def test_green_queue_budget_counts_both_rerun_fence_passes(self):
        for fences, attempt in [([[1, 1], [2, 1], [3, 1]], 1),
                                ([[1, 1], [1, 2], [2, 1], [3, 1]], 2)]:
            with self.subTest(fences=fences):
                result = exercise({
                    'initialFences': fences,
                    'currentAttempt': attempt,
                    'listedAttempt': attempt,
                })
                self.assertEqual(result['status'], 'success', result)
                lookups = [
                    call for call in result['calls']
                    if call[0] in ('current-run', 'exact-attempt')
                ]
                self.assertEqual(len(lookups), 3 + 4 * len(fences))

    def test_missing_or_running_workflow_keeps_group_pending(self):
        for options in [{
                'missing': '.github/workflows/frontend.yml'
        }, {
                'inProgress': '.github/workflows/pre-commit.yml'
        }]:
            with self.subTest(options=options):
                self.assertEqual(exercise(options)['status'], 'pending')

    def test_failed_workflow_or_missing_trigger_blocks_group(self):
        for options in [{
                'failed': '.github/workflows/frontend.yml'
        }, {
                'missingTrigger': True
        }, {
                'missingEquivalent': True
        }, {
                'groupSize': 2
        }, {
                'buildConcurrency': 2
        }, {
                'strategy': 'HEADGREEN'
        }, {
                'duplicateQueue': True
        }, {
                'oldHead': True
        }]:
            with self.subTest(options=options):
                self.assertEqual(exercise(options)['status'], 'failure')
        self.assertEqual(exercise({'staleQueue': True})['status'], 'pending')

    def test_merge_group_trigger_requires_checks_requested_for_release(self):
        script = f'''const gate = require({json.dumps(str(MODULE))});
const tests = [
  {{types: ['checks_requested'], branches: ['release-2.18']}},
  {{branches: ['master']}},
  {{types: ['completed'], branches: ['release-2.18']}},
];
console.log(JSON.stringify(tests.map(trigger => {{
  try {{ return gate.queueTriggerApplies(trigger); }}
  catch (error) {{ return error.message; }}
}})));
'''
        result = subprocess.run(['node'],
                                input=script,
                                check=True,
                                capture_output=True,
                                text=True)
        self.assertEqual(
            json.loads(result.stdout),
            [True, False, 'Unsupported release merge_group event types'])

    def test_unapproved_or_held_pr_blocks_group(self):
        for labels in [['lgtm'], ['approved'],
                       ['lgtm', 'approved', 'do-not-merge/hold']]:
            with self.subTest(labels=labels):
                self.assertEqual(
                    exercise({'labels': labels})['status'], 'failure')
        self.assertEqual(
            exercise({
                'labels': [],
                'dependabot': True
            })['status'], 'success')

    def test_fork_workflow_edits_and_truncated_history_block_group(self):
        for option in [
                'workflowChange', 'workflowChangeReverted',
                'workflowTreeMissing', 'compareBaseDrift', 'mergeBaseMissing',
                'historyTruncated', 'parentMissing'
        ]:
            with self.subTest(option=option):
                result = exercise({option: True})
                self.assertEqual(result['status'], 'failure', result)
                self.assertFalse(
                    any(call[0] == 'current-run' for call in result['calls']))
        prior = exercise({
            'priorEntry': True,
            'priorHeadNull': True,
            'priorWorkflowChange': True
        })
        self.assertEqual(prior['status'], 'failure', prior)
        same_repo = exercise({'sameRepo': True, 'workflowChange': True})
        self.assertEqual(same_repo['status'], 'success', same_repo)
        self.assertFalse(
            any(call[0] == 'test-merge' for call in same_repo['calls']))

    def test_fork_merge_imports_trusted_release_workflow_change(self):
        base, old_base, feature, imported, head = ('b' * 40, '9' * 40, '8' * 40,
                                                   '7' * 40, 'c' * 40)
        commits = [{
            'sha': feature,
            'parents': [{
                'sha': old_base
            }]
        }, {
            'sha': imported,
            'parents': [{
                'sha': feature
            }, {
                'sha': base
            }]
        }, {
            'sha': head,
            'parents': [{
                'sha': imported
            }]
        }]
        result = exercise({
            'commits': commits,
            'prefixComparisons': {
                imported: {
                    'mergeBaseSha': base,
                    'commits': commits[:2],
                }
            },
            'workflowOids': {
                old_base: '1' * 40,
                feature: '1' * 40,
                base: '2' * 40,
                imported: '2' * 40,
                head: '2' * 40,
            }
        })
        self.assertEqual(result['status'], 'success', result)
        self.assertIn(['compare', f'{base}...contributor:{imported}'],
                      result['calls'])
        queries = [
            call[1] for call in result['calls'] if call[0] == 'workflow-trees'
        ]
        self.assertTrue(
            any(f'{old_base}:.github/workflows' in query for query in queries))

    def test_fork_merge_cannot_launder_side_branch_workflow_edit(self):
        base, feature, edited, reverted, head = ('b' * 40, '8' * 40, '6' * 40,
                                                 '7' * 40, 'c' * 40)
        result = exercise({
            'commits': [{
                'sha': feature,
                'parents': [{
                    'sha': base
                }]
            }, {
                'sha': edited,
                'parents': [{
                    'sha': base
                }]
            }, {
                'sha': reverted,
                'parents': [{
                    'sha': edited
                }]
            }, {
                'sha': head,
                'parents': [{
                    'sha': feature
                }, {
                    'sha': reverted
                }]
            }],
            'workflowOids': {
                edited: '2' * 40,
            }
        })
        self.assertEqual(result['status'], 'failure', result)
        self.assertFalse(
            any(call[0] == 'current-run' for call in result['calls']))
        queries = [
            call[1] for call in result['calls'] if call[0] == 'workflow-trees'
        ]
        self.assertTrue(
            any(f'{edited}:.github/workflows' in query for query in queries))

    def test_fork_merge_cannot_reimport_stale_base_workflows(self):
        base, stale, feature, imported, head = ('b' * 40, '5' * 40, '8' * 40,
                                                '7' * 40, 'c' * 40)
        commits = [{
            'sha': feature,
            'parents': [{
                'sha': base
            }]
        }, {
            'sha': imported,
            'parents': [{
                'sha': feature
            }, {
                'sha': stale
            }]
        }, {
            'sha': head,
            'parents': [{
                'sha': imported
            }, {
                'sha': base
            }]
        }]
        result = exercise({
            'commits': commits,
            'prefixComparisons': {
                imported: {
                    'mergeBaseSha': base,
                    'commits': commits[:2],
                }
            },
            'workflowOids': {
                stale: '2' * 40,
                imported: '2' * 40,
            }
        })
        self.assertEqual(result['status'], 'failure', result)
        self.assertIn(['compare', f'{base}...contributor:{imported}'],
                      result['calls'])
        self.assertFalse(
            any(call[0] == 'current-run' for call in result['calls']))

    def test_cumulative_group_checks_all_earlier_queue_entries(self):
        self.assertEqual(
            exercise({
                'priorEntry': True,
                'priorHeadNull': True
            })['status'], 'success')
        for options in [{'priorLabels': ['lgtm']}, {'priorHeadDrift': True}]:
            with self.subTest(options=options):
                self.assertEqual(
                    exercise({
                        'priorEntry': True,
                        'priorHeadNull': True,
                        **options
                    })['status'], 'failure')

    def test_two_built_heads_fail_before_direct_run_reads(self):
        result = exercise({'priorEntry': True})
        self.assertEqual(result['status'], 'failure', result)
        self.assertFalse(
            any(call[0] == 'current-run' for call in result['calls']))

    def test_earlier_pr_label_event_invalidates_all_later_groups(self):
        result = exercise({
            'priorEntry': True,
            'priorLabels': ['lgtm'],
            'labelEvent': True,
            'labelEventEarlier': True,
            'initialStatus': 'success'
        })
        self.assertEqual(result['status'], 'failure')
        self.assertEqual(result['priorStatus'], 'failure')
        self.assertEqual(result['calls'][:2], [
            ['ci-passed-release', 'pending', 'd' * 40],
            ['ci-passed-release', 'pending', SHA],
        ])
        multiple_built = exercise({
            'priorEntry': True,
            'labelEvent': True,
            'labelEventEarlier': True
        })
        self.assertFalse(
            any(call[0] == 'ci-passed-release' and call[1] == 'success'
                for call in multiple_built['calls']))

    def test_rerun_start_revokes_success_before_run_api_updates(self):
        for action in ['requested', 'in_progress']:
            with self.subTest(action=action):
                result = exercise({
                    'initialStatus': 'success',
                    'action': action,
                    'runsError': True
                })
                self.assertEqual(result['status'], 'pending')
                self.assertNotIn('error', result)
                self.assertFalse(
                    any(call[0] == 'runs-for' for call in result['calls']))

    def test_rerun_fence_blocks_stale_run_listing_until_exact_attempt_completes(
            self):
        stale = exercise({'initialStatus': 'success', 'rerunSequence': 'stale'})
        self.assertEqual(stale['afterStart'], 'pending', stale)
        self.assertEqual(stale['afterSchedule'], 'pending', stale)
        self.assertEqual(stale['status'], 'pending', stale)
        recovered = exercise({
            'initialStatus': 'success',
            'rerunSequence': 'recover'
        })
        self.assertEqual(recovered['afterSchedule'], 'pending', recovered)
        self.assertEqual(recovered['status'], 'success', recovered)
        latest_failed = exercise({'fenceOnly': True})
        self.assertEqual(latest_failed['fenceState'], 'failure', latest_failed)
        malformed_current = exercise({
            'fenceOnly': True,
            'omitCurrentAttempt': True
        })
        self.assertEqual(malformed_current['fenceState'], 'pending',
                         malformed_current)
        contradictory = exercise({'fenceOnly': True, 'attemptTwoFailed': True})
        self.assertEqual(contradictory['fenceState'], 'pending', contradictory)

    def test_rerun_fence_handles_later_attempt_and_missing_status_visibility(
            self):
        third = exercise({
            'rerunSequence': 'third-attempt',
            'attemptTwoFailed': True
        })
        self.assertEqual(third['status'], 'pending', third)
        recovered = exercise({
            'rerunSequence': 'failed-then-success',
            'attemptTwoFailed': True
        })
        self.assertEqual(recovered['status'], 'success', recovered)
        self.assertEqual(
            exercise({'staleStatusRead': True})['status'], 'pending')
        malformed = exercise({
            'action': 'in_progress',
            'malformedAttempt': True
        })
        self.assertEqual(malformed['status'], 'pending', malformed)
        self.assertNotIn('error', malformed)

    def test_status_history_reserves_revocation_capacity(self):
        near_limit = exercise({'prefillStatuses': 998})
        self.assertEqual(near_limit['status'], 'pending', near_limit)
        self.assertNotIn('error', near_limit)
        self.assertFalse(
            any(call[1] == 'success'
                for call in near_limit['calls']
                if call[0] == 'ci-passed-release'))

    def test_dequeue_invalid_release_pr_before_null_successor_blocks_discovery(
            self):
        earlier = exercise({
            'dequeueEvent': True,
            'labelEventEarlier': True,
            'priorEntry': True,
            'nullCurrentHead': True,
            'priorLabels': ['lgtm'],
            'initialStatus': 'success'
        })
        self.assertEqual(earlier['priorStatus'], 'pending', earlier)
        self.assertEqual(earlier['status'], 'success', earlier)
        self.assertEqual(earlier['candidates'], [], earlier)
        self.assertIn(['dequeue', 'PR_8'], earlier['calls'])
        self.assertTrue(earlier['dequeued'])

        stale_successor = exercise({
            'dequeueEvent': True,
            'labelEventEarlier': True,
            'priorEntry': True,
            'priorLabels': ['lgtm'],
            'checkRetired': True
        })
        self.assertEqual(stale_successor['status'], 'pending', stale_successor)
        self.assertEqual(stale_successor['retiredState'], 'pending',
                         stale_successor)

        unbuilt = exercise({
            'dequeueEvent': True,
            'priorEntry': True,
            'nullCurrentHead': True,
            'labels': ['lgtm']
        })
        self.assertEqual(unbuilt['priorStatus'], None, unbuilt)
        self.assertEqual(unbuilt['candidates'], [], unbuilt)
        self.assertIn(['dequeue', 'PR_7'], unbuilt['calls'])

    def test_dequeue_denial_and_readback_failure_preserve_pending(self):
        options = {
            'dequeueEvent': True,
            'priorEntry': True,
            'nullCurrentHead': True,
            'labelEventEarlier': True,
            'priorLabels': ['lgtm'],
            'initialStatus': 'success'
        }
        for failure in [{
                'dequeueDenied': True
        }, {
                'readbackError': True
        }, {
                'readbackStillPresent': True
        }]:
            with self.subTest(failure=failure):
                result = exercise({**options, **failure})
                self.assertEqual(result['priorStatus'], 'pending', result)
                self.assertIn('error', result)
                self.assertIn(['dequeue', 'PR_8'], result['calls'])
        write_failure = exercise({**options, 'statusWriteError': True})
        self.assertIn(['dequeue', 'PR_8'], write_failure['calls'])
        self.assertIn('error', write_failure)

    def test_eligible_release_pr_does_not_dequeue(self):
        result = exercise({'dequeueEvent': True})
        self.assertEqual(result['candidates'], [{'sha': SHA}], result)
        self.assertFalse(any(call[0] == 'dequeue' for call in result['calls']))

    def test_eligible_release_pr_refreshes_built_head_with_unbuilt_successor(
            self):
        result = exercise({
            'dequeueEvent': True,
            'labelEventEarlier': True,
            'priorEntry': True,
            'nullCurrentHead': True
        })
        self.assertEqual(result['candidates'], [{'sha': 'd' * 40}], result)
        self.assertFalse(any(call[0] == 'dequeue' for call in result['calls']))

        unbuilt = exercise({'dequeueEvent': True, 'nullCurrentHead': True})
        self.assertEqual(unbuilt['candidates'], [], unbuilt)
        self.assertFalse(any(call[0] == 'dequeue' for call in unbuilt['calls']))

    def test_retarget_away_retires_and_dequeues_release_entry(self):
        result = exercise({'dequeueEvent': True, 'retargetAway': True})
        self.assertEqual(result['status'], 'pending', result)
        self.assertEqual(result['candidates'], [], result)
        self.assertIn(['dequeue', 'PR_7'], result['calls'])

    def test_late_workflow_and_label_event_recover(self):
        self.assertEqual(
            exercise({
                'missing': '.github/workflows/frontend.yml',
                'schedule': True
            })['status'], 'pending')
        self.assertEqual(exercise({'labelEvent': True})['status'], 'success')

    def test_drift_after_success_revokes_group(self):
        result = exercise({'removeLabelAfterSuccess': True})
        self.assertEqual(result['status'], 'failure')
        self.assertEqual(result['calls'][-1],
                         ['ci-passed-release', 'failure', SHA])
        self.assertEqual(
            exercise({
                'priorEntry': True,
                'priorHeadNull': True,
                'removePriorLabelAfterSuccess': True
            })['status'], 'failure')

    def test_api_errors_fail_closed_and_stale_branch_does_not_publish(self):
        for options in [{
                'inventoryError': True
        }, {
                'runsError': True
        }, {
                'queueError': True
        }, {
                'statusWriteError': True
        }]:
            with self.subTest(options=options):
                result = exercise(options)
                self.assertNotEqual(result['status'], 'success', result)
                self.assertIn('error', result)
        self.assertEqual(exercise({'wrongBranch': True})['calls'], [])


if __name__ == '__main__':
    unittest.main()
