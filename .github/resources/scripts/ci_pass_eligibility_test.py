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
const options = JSON.parse(process.argv[2]);
const root = process.argv[3];
const calls = [], outputs = {}, descriptions = [], statusContexts = [], targetUrls = [];
const workflowName = options.workflowName || 'frontend.yml';
let cycle = 0;
let pr = {
  number: 7, state: 'open', changed_files: 1,
  head: {sha: 'head', ref: 'feature', repo: {full_name: 'contributor/pipelines',
    name: 'pipelines', owner: {login: 'contributor'}}},
  base: {sha: 'b'.repeat(40), ref: 'master', repo: {full_name: 'kubeflow/pipelines'}},
  user: {login: 'dependabot[bot]'}, author_association: 'NONE', labels: [],
  ...options.pr,
};
if (pr.base.ref === 'release-2.18') pr.head.sha = 'a'.repeat(40);
const eventPR = structuredClone(pr);
if (options.oldHead) eventPR.head.sha = 'old-head';
const context = {repo: {owner: 'kubeflow', repo: 'pipelines'}, runId: 99,
  eventName: options.schedule ? 'schedule' : options.workflowRun ? 'workflow_run' : 'pull_request_target',
  payload: {pull_request: eventPR, action: options.action || 'opened',
    workflow_run: {event: 'pull_request', head_sha: pr.head.sha, head_branch: 'feature',
      head_repository: {owner: {login: pr.head.repo.full_name.split('/')[0]},
        full_name: pr.head.repo.full_name}}}};
let published = false;
const statuses = new Map();
const initialContext = options.initialStatusContext ||
  (pr.base.ref === 'release-2.18' ? 'ci-passed-release' : 'ci-passed');
if (options.initialStatus) statuses.set(initialContext, options.initialStatus);
const statusDescriptions = new Map();
if (options.initialDescription) statusDescriptions.set(initialContext, options.initialDescription);
const core = {info: () => {}, setOutput: (key, value) => {outputs[key] = value;}};
const methods = {files: {}, runs: {}, timeline: {}, pulls: {}};
const github = {graphql: async (query, variables) => {
  if (query.includes('query ReleaseWorkflowTrees')) {
    calls.push(['workflow-trees', query]);
    if (options.workflowTreeError) throw Error('Workflow tree unavailable');
    const repository = {nameWithOwner: pr.head.repo.full_name};
    for (const [, alias, sha] of query.matchAll(/(r\d+): object\(expression: "([0-9a-f]{40}):\.github\/workflows"\)/g)) {
      const changed = options.workflowChange && sha === pr.head.sha ||
        options.workflowChangeReverted && sha === '8'.repeat(40) ||
        options.mergeSecondParentChanged && sha === '6'.repeat(40);
      repository[alias] = options.workflowTreeMissing ? null :
        {oid: options.workflowOids?.[sha] ||
          (changed ? '2'.repeat(40) : '1'.repeat(40))};
    }
    return {repository};
  }
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
    get: async () => ({data: structuredClone(pr)})},
  actions: {listWorkflowRunsForRepo: methods.runs},
  issues: {listEventsForTimeline: methods.timeline,
    addLabels: async request => {calls.push(['add-label', request.labels]);},
    removeLabel: async request => {
      calls.push(['remove-label', request.name]);
      if (options.removeLabelFailure) throw Object.assign(Error('Label write unavailable'), {status: 403});
    }},
  repos: {getCombinedStatusForRef: {},
    compareCommitsWithBasehead: async request => {
      calls.push(['compare', request.basehead]);
      if (options.compareError) throw Error('Compare unavailable');
      const comparedHead = request.basehead.split(':').at(-1);
      const prefix = options.prefixComparisons?.[comparedHead];
      const mergeBase = prefix?.mergeBaseSha || options.mergeBaseSha || pr.base.sha;
      const commits = prefix?.commits || options.commits || (options.workflowChangeReverted ? [
        {sha: '8'.repeat(40), parents: [{sha: mergeBase}]},
        {sha: pr.head.sha, parents: [{sha: '8'.repeat(40)}]},
      ] : options.mergeSecondParentChanged ? [
        {sha: '6'.repeat(40), parents: [{sha: mergeBase}]},
        {sha: pr.head.sha, parents: [{sha: mergeBase}, {sha: '6'.repeat(40)}]},
      ] : [{sha: pr.head.sha,
        parents: options.parentMissing ? [] : [{sha: mergeBase}]}]);
      return {data: {
        base_commit: {sha: options.compareBaseDrift ? '0'.repeat(40) : pr.base.sha},
        merge_base_commit: options.mergeBaseMissing ? null : {sha: mergeBase},
        commits, ahead_by: commits.length,
        total_commits: options.historyTruncated ? commits.length + 1 : commits.length,
      }};
    },
    createCommitStatus: async request => {
    if (options.legacyWriteError && request.context === 'ci-passed' &&
        pr.base.ref === 'release-2.18') throw Error('Legacy status write unavailable');
    statuses.set(request.context, request.state);
    descriptions.push(request.description);
    statusDescriptions.set(request.context, request.description);
    targetUrls.push(request.target_url);
    statusContexts.push(request.context);
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
    const conclusion = published && options.drift === 'rerun' ? 'cancelled' : (options.conclusions?.[cycle] || options.conclusion || 'success');
    return [{path: `.github/workflows/${workflowName}`, id: 42, event: 'pull_request', head_sha: pr.head.sha, head_branch: 'feature',
      head_repository: {full_name: pr.head.repo.full_name},
      status: options.runStatus || 'completed', conclusion,
      created_at: options.fresh ? '2026-09-07T12:01:00Z' : '2026-09-07T11:00:00Z',
      run_started_at: '2026-09-07T12:02:00Z', pull_requests: []}];
  }
  throw Error('Unexpected API request');
}};
github.paginate.iterator = async function* () {
  if (options.statusReadFailure) throw Error('Status read unavailable');
  yield {data: {statuses: [...statuses].map(([name, state]) => ({context: name, state, description: statusDescriptions.get(name)}))}};
};
(async () => {
  let error;
  for (cycle = 0; cycle < (options.cycles || 1); cycle++) {
  context.runId = 99 + cycle;
  try {await gate.prepare({github, context, core, root, recovery: {number: 7, head: eventPR.head.sha}});} catch (e) {error = e.message;}
  if (options.revokeBeforeFinal) pr.labels = [{name: 'needs-ok-to-test'}];
  if (options.recoverBeforeFinal) options.conclusion = 'success';
  try {
    await gate.finalize({github, context, core, root, number: outputs.pr_number,
      head: outputs.head_sha, before: outputs.snapshot,
      pollPassed: outputs.ready === 'true' && (options.pollPassed !== false || (options.recoverLast && cycle === options.cycles - 1)) && !error});
  } catch (e) {error = e.message;}
  }
  console.log(JSON.stringify({calls, outputs, error,
    status: statuses.get(pr.base.ref === 'release-2.18' ? 'ci-passed-release' : 'ci-passed'),
    descriptions, statusContexts, targetUrls}));
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
        self.assertEqual(
            statuses[-1],
            ['status', state, result['outputs'].get('head_sha', 'head')],
            result)

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

    def test_release_admission_requires_tide_labels_and_rejects_holds(self):
        release = {
            'base': {
                'sha': 'b' * 40,
                'ref': 'release-2.18',
                'repo': {
                    'full_name': 'kubeflow/pipelines'
                }
            },
            'user': {
                'login': 'human'
            },
            'author_association': 'MEMBER'
        }
        for names, expected in [(['lgtm', 'approved'], 'success'),
                                (['lgtm'], 'failure'),
                                (['approved'], 'failure'),
                                (['lgtm', 'approved',
                                  'do-not-merge/hold'], 'failure'),
                                (['lgtm', 'approved',
                                  'needs-rebase'], 'failure')]:
            with self.subTest(names=names):
                pr = {**release, 'labels': [{'name': name} for name in names]}
                self.assert_last_status(exercise({'pr': pr}), expected)
        self.assert_last_status(
            exercise({
                'pr': {
                    **release, 'draft': True,
                    'labels': [{
                        'name': 'lgtm'
                    }, {
                        'name': 'approved'
                    }]
                }
            }), 'failure')
        # The existing Tide Dependabot query admits this exact account without
        # human approval labels; author eligibility and holds still apply.
        self.assert_last_status(
            exercise({
                'pr': {
                    **release, 'user': {
                        'login': 'dependabot[bot]'
                    },
                    'labels': []
                }
            }), 'success')

    def test_retarget_to_release_revokes_old_master_status(self):
        release = {
            'pr': {
                'base': {
                    'sha': 'b' * 40,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                }
            }
        }
        result = exercise({
            **release, 'initialStatus': 'success',
            'initialStatusContext': 'ci-passed',
            'action': 'edited'
        })
        self.assert_last_status(result, 'success')
        self.assertEqual(result['statusContexts'], [
            'ci-passed-release', 'ci-passed', 'ci-passed-release', 'ci-passed'
        ])
        self.assertIn(['compare', 'b' * 40 + '...contributor:' + 'a' * 40],
                      result['calls'])

    def test_fork_release_pr_cannot_change_workflows(self):
        release = {
            'sha': 'b' * 40,
            'ref': 'release-2.18',
            'repo': {
                'full_name': 'kubeflow/pipelines'
            }
        }
        for option in [
                'workflowChange', 'workflowChangeReverted',
                'mergeSecondParentChanged', 'workflowTreeMissing',
                'compareBaseDrift', 'mergeBaseMissing', 'historyTruncated',
                'parentMissing'
        ]:
            with self.subTest(option=option):
                changed = {'pr': {'base': release}, option: True}
                result = exercise(changed)
                self.assert_last_status(result, 'failure')
                self.assertFalse(
                    any(call[0] == 'status' and call[1] == 'success'
                        for call in result['calls']))
                self.assertEqual(result['statusContexts'], [
                    'ci-passed-release', 'ci-passed', 'ci-passed-release',
                    'ci-passed'
                ])
        good = exercise({'pr': {'base': release, 'merge_commit_sha': None}})
        self.assert_last_status(good, 'success')
        self.assertEqual(
            len([call for call in good['calls'] if call[0] == 'workflow-trees'
                ]), 3)
        guard_calls = [
            call for call in good['calls']
            if call[0] in ('compare', 'workflow-trees')
        ]
        self.assertEqual(guard_calls[0],
                         ['compare', 'b' * 40 + '...contributor:' + 'a' * 40])
        self.assertIn('b' * 40 + ':.github/workflows', guard_calls[1][1])
        self.assertIn('a' * 40 + ':.github/workflows', guard_calls[1][1])

        advanced = exercise({
            'mergeBaseSha': 'b' * 40,
            'pr': {
                'base': {
                    'sha': 'd' * 40,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                }
            }
        })
        self.assert_last_status(advanced, 'success')
        self.assertIn(['compare', 'd' * 40 + '...contributor:' + 'a' * 40],
                      advanced['calls'])

    def test_fork_release_merge_imports_trusted_workflow_change(self):
        base, old_base, feature, imported, head = ('b' * 40, '9' * 40, '8' * 40,
                                                   '7' * 40, 'a' * 40)
        # The release base changed its workflows after the fork branched.
        # Merging that base into the fork preserves the trusted base tree.
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
            'pr': {
                'base': {
                    'sha': base,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                }
            },
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
        self.assert_last_status(result, 'success')
        self.assertIn(['compare', f'{base}...contributor:{imported}'],
                      result['calls'])
        queries = [
            call[1] for call in result['calls'] if call[0] == 'workflow-trees'
        ]
        self.assertTrue(
            any(f'{old_base}:.github/workflows' in query for query in queries))

    def test_fork_release_merge_cannot_launder_side_branch_workflow_edit(self):
        base, feature, edited, reverted, head = ('b' * 40, '8' * 40, '6' * 40,
                                                 '7' * 40, 'a' * 40)
        # The side branch returns to the original tree before merging, but
        # its intermediate workflow edit must still block admission.
        result = exercise({
            'pr': {
                'base': {
                    'sha': base,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                }
            },
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
        self.assert_last_status(result, 'failure')
        self.assertFalse(
            any(call[0] == 'status' and call[1] == 'success'
                for call in result['calls']))
        queries = [
            call[1] for call in result['calls'] if call[0] == 'workflow-trees'
        ]
        self.assertTrue(
            any(f'{edited}:.github/workflows' in query for query in queries))

    def test_fork_release_merge_cannot_reimport_stale_base_workflows(self):
        base, stale, feature, imported, head = ('b' * 40, '5' * 40, '8' * 40,
                                                '7' * 40, 'a' * 40)
        # A stale release ancestor is a merge parent, but the current base is
        # already in this fork's history. The stale parent cannot authorize a
        # workflow rollback followed by a clean final tree.
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
            'pr': {
                'base': {
                    'sha': base,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                }
            },
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
        self.assert_last_status(result, 'failure')
        self.assertIn(['compare', f'{base}...contributor:{imported}'],
                      result['calls'])
        self.assertFalse(
            any(call[0] == 'status' and call[1] == 'success'
                for call in result['calls']))

    def test_fork_release_merge_cannot_hide_workflow_conflict_resolution(self):
        base, feature, side, resolved, head = ('b' * 40, '8' * 40, '6' * 40,
                                               '7' * 40, 'a' * 40)
        # Neither parent changed workflows. The merge resolution did, then a
        # later merge restored the final tree to the trusted release base.
        result = exercise({
            'pr': {
                'base': {
                    'sha': base,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                }
            },
            'commits': [{
                'sha': feature,
                'parents': [{
                    'sha': base
                }]
            }, {
                'sha': side,
                'parents': [{
                    'sha': base
                }]
            }, {
                'sha': resolved,
                'parents': [{
                    'sha': feature
                }, {
                    'sha': side
                }]
            }, {
                'sha': head,
                'parents': [{
                    'sha': resolved
                }, {
                    'sha': base
                }]
            }],
            'workflowOids': {
                resolved: '2' * 40,
            }
        })
        self.assert_last_status(result, 'failure')
        self.assertFalse(
            any(call[0] == 'status' and call[1] == 'success'
                for call in result['calls']))

    def test_same_repository_release_workflow_change_uses_writer_trust(self):
        result = exercise({
            'workflowChange': True,
            'pr': {
                'base': {
                    'sha': 'b' * 40,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                },
                'head': {
                    'sha': 'head',
                    'ref': 'feature',
                    'repo': {
                        'full_name': 'kubeflow/pipelines',
                        'name': 'pipelines',
                        'owner': {
                            'login': 'kubeflow'
                        }
                    }
                },
            }
        })
        self.assert_last_status(result, 'success')
        self.assertFalse(any(call[0] == 'compare' for call in result['calls']))

    def test_release_invalidation_survives_legacy_status_write_failure(self):
        result = exercise({
            'legacyWriteError': True,
            'initialStatus': 'success',
            'initialStatusContext': 'ci-passed-release',
            'pr': {
                'base': {
                    'sha': 'b' * 40,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    }
                }
            }
        })
        self.assertEqual(result['status'], 'failure', result)
        self.assertEqual(result['statusContexts'],
                         ['ci-passed-release', 'ci-passed-release'])

    def test_complete_ci_publishes_pending_then_success(self):
        result = exercise()
        self.assertEqual(result['calls'][0], ['status', 'pending', 'head'])
        self.assert_last_status(result, 'success')
        self.assertEqual(result['statusContexts'], ['ci-passed', 'ci-passed'])

    def test_failed_poll_blocks_otherwise_complete_workflows(self):
        result = exercise({'pollPassed': False})
        self.assert_last_status(result, 'failure')
        self.assertEqual(
            result['descriptions'][-1],
            'CI did not pass; complete current-head CI and retry.')

    def test_missing_workflows_never_reach_poller(self):
        result = exercise({'missing': True})
        self.assertEqual(result['outputs']['ready'], 'false')
        self.assert_last_status(result, 'failure')

    def test_recovery_revisits_legacy_and_changed_base_success(self):
        script = r"""
const {recoveryCandidates} = require(process.argv[1]);
const requests = [];
const prs = ['success', 'failure', 'pending', 'missing', 'untrusted', 'revoked', 'stale-success', 'legacy-success', 'retarget-success'].map((state, i) => ({
  number: i + 1, head: {sha: state}, base: {ref: 'master', sha: 'b'.repeat(40)}, user: {login: state === 'untrusted' ? 'human' : 'dependabot[bot]'},
  labels: state === 'revoked' ? [{name: 'needs-ok-to-test'}] : [], author_association: 'NONE',
}));
const github = {paginate: async () => prs, rest: {pulls: {list: {}}, repos: {
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
            'success', 'failure', 'pending', 'missing', 'stale-success',
            'legacy-success', 'retarget-success'
        ])

    def test_release_recovery_requires_both_status_contexts(self):
        script = r"""
const {recoveryCandidates} = require(process.argv[1]);
const crypto = require('node:crypto');
const pr = {number: 7, state: 'open', draft: false,
  head: {sha: 'a'.repeat(40)}, base: {sha: 'b'.repeat(40), ref: 'release-2.18'},
  user: {login: 'human'}, author_association: 'MEMBER',
  labels: ['lgtm', 'approved'].map(name => ({name}))};
const stamp = crypto.createHash('sha256').update(JSON.stringify([
  'release-2.18', pr.base.sha, 'release-workflow-guard-v4',
])).digest('hex');
const description = `Expected CI and all checks passed; base policy ${stamp}.`;
const statuses = ['ci-passed-release', 'ci-passed'].map(context =>
  ({context, state: 'success', description}));
const github = {paginate: async () => [pr], rest: {pulls: {list: {}},
  repos: {getCombinedStatusForRef: {}}}};
github.paginate.iterator = async function* () {yield {data: {statuses: github.visible}};};
(async () => {
  const results = [];
  for (const visible of [statuses, statuses.slice(0, 1), statuses.slice(1)]) {
    github.visible = visible;
    results.push(await recoveryCandidates({github,
      context: {repo: {owner: 'kubeflow', repo: 'pipelines'}}}));
  }
  console.log(JSON.stringify(results));
})().catch(error => {console.error(error); process.exit(1);});
"""
        result = subprocess.run(
            ['node', '-e', script, str(MODULE)],
            check=True,
            capture_output=True,
            text=True)
        candidate = [{'number': 7, 'head': 'a' * 40}]
        self.assertEqual(json.loads(result.stdout), [[], candidate, candidate])

    def test_success_records_the_validated_base_policy(self):
        result = exercise()
        self.assert_last_status(result, 'success')
        self.assertEqual(
            result['descriptions'][-1],
            'Expected CI and all checks passed; base policy bf60a45a7f48d31d1cf806d3517d6cc8d24985621eaf55baea8599faee6e508c.'
        )

    def test_green_recovery_requires_newly_enabled_upgrade_workflow(self):
        for policy, expected in [('paused', 'success'), ('enabled', 'failure')]:
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
        description = 'CI did not pass; complete current-head CI and retry.'
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
                'missing': True
            },
             '.github/workflows/frontend.yml: expected workflow has not registered'
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

    def test_release_failure_reason_refreshes_both_contexts_without_churn(self):
        result = exercise({
            'schedule': True,
            'conclusion': 'skipped',
            'cycles': 6,
            'pr': {
                'base': {
                    'sha': 'b' * 40,
                    'ref': 'release-2.18',
                    'repo': {
                        'full_name': 'kubeflow/pipelines'
                    },
                },
            },
        })
        reason = '.github/workflows/frontend.yml: latest run is completed/skipped'
        self.assertEqual(result['statusContexts'],
                         ['ci-passed-release', 'ci-passed'] * 2)
        self.assertEqual(result['descriptions'][-2:], [reason, reason])
        self.assertEqual(result['status'], 'failure')

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

    def test_evidence_recovery_cannot_succeed_when_poller_was_skipped(self):
        result = exercise({'conclusion': 'skipped', 'recoverBeforeFinal': True})
        self.assertEqual(result['outputs']['ready'], 'false')
        self.assert_last_status(result, 'failure')
        self.assertEqual(
            result['descriptions'][-1],
            'CI did not pass; complete current-head CI and retry.')

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

    def test_closed_pr_invalidates_prior_success_without_polling(self):
        for merged in [False, True]:
            with self.subTest(merged=merged):
                result = exercise({
                    'action': 'closed',
                    'initialStatus': 'success',
                    'pr': {
                        'state': 'closed',
                        'merged': merged
                    },
                })
                self.assertNotIn('error', result)
                self.assert_last_status(result, 'failure')
                self.assertNotIn('ready', result['outputs'])
                self.assertIn(['remove-label', 'ci-passed'], result['calls'])
                self.assertNotIn(['add-label', ['ci-passed']], result['calls'])
                self.assertNotIn(['status', 'success', 'head'], result['calls'])

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
