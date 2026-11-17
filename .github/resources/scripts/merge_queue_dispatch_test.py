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
"""Exercise exact-candidate CI dispatch and bounded retry reservations."""

import json
import subprocess
import tempfile
import unittest
from pathlib import Path

from ci_merge_group_test import good_run
from merge_queue_admission_test import entry, pr, queue

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/merge_queue_dispatch.js'
HEAD = 'a' * 40
BASE = 'b' * 40
OTHER = 'c' * 40
NOW = '2026-11-13T12:00:00Z'
PATH = '.github/workflows/frontend.yml'


def lane(path=PATH, branch='master', **overrides):
    return {
        'path': path,
        'name': 'Frontend Tests',
        'header_sha256': 'header',
        'pull_request': {'branches': [branch], 'paths': ['frontend/**']},
        'merge_group': {'types': ['checks_requested'], 'branches': [branch]},
        'workflow_dispatch': {
            'inputs': {
                key: {'type': 'string', 'required': False}
                for key in [
                    'queue_head_sha',
                    'queue_base_ref',
                    'queue_base_sha',
                ]
            }
        },
        **overrides,
    }


def reservation(attempt=1, created_at='2026-11-13T11:59:30Z', **overrides):
    return {
        'context': 'ci-queue-dispatch/frontend.yml',
        'state': 'pending',
        'description': f'Queue dispatch attempt {attempt}; base {BASE}.',
        'created_at': created_at,
        **overrides,
    }


def dispatch(branch='master', **options):
    workflows = options.pop('lanes', [lane(branch=branch)])
    fixture = {
        'branch': branch,
        'method': 'dispatchGroup',
        'group': {
            'head': HEAD,
            'branch': f'gh-readonly-queue/{branch}/pr-7-abcdef',
            'baseRef': branch,
        },
        'matchingRefs': [
            {
                'ref': f'refs/heads/gh-readonly-queue/{branch}/pr-7-abcdef',
                'object': {'type': 'commit', 'sha': HEAD},
            }
        ],
        'enabled': [branch],
        'runs': [],
        'statuses': [],
        'rules': [{'type': 'merge_queue'}],
        'refs': {},
        'queuePRs': [pr(branch=branch)],
        'queues': [queue([entry(branch=branch, headCommit={'oid': HEAD})])],
        'inventory': {'version': 1, 'workflows': workflows},
        'workflowFiles': [
            {'path': item['path'], 'header_sha256': item['header_sha256']}
            for item in workflows
        ],
        **options,
    }
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        (root / '.github').mkdir()
        (root / '.github/merge-queue-policy.json').write_text(
            json.dumps({'version': 1, 'enabled_branches': fixture['enabled']})
        )
        script = f"""
const gate = require({json.dumps(str(MODULE))});
const f = {json.dumps(fixture)};
const writes = [], dispatches = [], requests = [], inventoryRequests = [], operations = [];
const counts = new Map();
const summaries = [];
let candidateReplaced = false;
let activeDispatches = 0, peakDispatches = 0, completedDispatches = 0;
const delay = ms => new Promise(resolve => setTimeout(resolve, ms));
function graphqlPR(pr) {{
  return {{id:pr.node_id, number:pr.number, state:pr.state.toUpperCase(),
    isDraft:pr.draft, merged:pr.merged, headRefOid:pr.head.sha,
    baseRefName:pr.base.ref, baseRepository:{{nameWithOwner:pr.base.repo.full_name}},
    author:{{__typename:pr.user.type, login:pr.user.login}},
    labels:{{nodes:pr.labels,totalCount:pr.labels.length,pageInfo:{{hasNextPage:false}}}}}};
}}
const reserved = () => writes.some(w => w.context.startsWith('ci-queue-dispatch/'));
function next(key, values) {{
  const count = counts.get(key) || 0;
  counts.set(key, count + 1);
  return values[Math.min(count, values.length - 1)];
}}
const github = {{rest: {{
  git: {{listMatchingRefs:'matchingRefs', getRef: async options => {{
    requests.push({{route:'ref', options}});
    const changed = reserved() && f.refsAfterReservation;
    const refs = changed || f.refs;
    const sha = next('ref-' + options.ref, refs[options.ref] ||
      [['heads/master', 'heads/main', 'heads/release-2.18'].includes(options.ref) ? {json.dumps(BASE)} : {json.dumps(HEAD)}]);
    if (sha === null) throw Object.assign(new Error('Ref disappeared'), {{status:404}});
    return {{data:{{ref:'refs/' + options.ref, object:{{type:'commit',sha}}}}}};
  }}}},
  pulls: {{get: async options => {{
    requests.push({{route:'pr', options}});
    if (reserved() && f.prAfterReservation) return {{data:f.prAfterReservation}};
    return {{data:f.prs ? f.prs[options.pull_number] : next('pr',f.queuePRs)}};
  }}}},
  actions: {{listWorkflowRunsForRepo:'runs', createWorkflowDispatch:async options => {{
    dispatches.push(options); operations.push({{route:'dispatch',options}});
    activeDispatches++; peakDispatches = Math.max(peakDispatches, activeDispatches);
    try {{
      if (f.dispatchDelayMs) await delay(f.dispatchDelayMs);
      if (f.dispatchError) throw new Error('Dispatch response lost');
      return {{status:204}};
    }} finally {{ activeDispatches--; completedDispatches++; }}
  }}}},
  repos: {{get: async options => {{
    requests.push({{route:'repository',options}});
    if (f.repositoryError) throw new Error('Repository unavailable');
    return {{data:{{full_name:'owner/repo',default_branch:next('defaultBranch', f.defaultBranches || [f.defaultBranch || f.branch])}}}};
  }}, listCommitStatusesForRef:'statuses', createCommitStatus:async options => {{
    writes.push(options); operations.push({{route:'status',options}});
    if (f.statusWriteError || options.context === f.statusWriteErrorContext) throw new Error('Status write failed');
    if (f.replaceOnReservation === options.context) {{
      // Complete this reservation after the others have resolved, reproducing
      // a ref replacement before the batch can issue its dispatch requests.
      await new Promise(resolve => setImmediate(resolve));
      candidateReplaced = true;
    }}
    if (f.statusWriteDelayMs) await delay(f.statusWriteDelayMs);
    return {{data:{{...options,created_at:{json.dumps(NOW)}}}}};
  }}}},
}}, paginate:async (route,options) => {{
  requests.push({{route,options}});
  if (route.includes('/rules/branches/')) {{
    const rules = f.branchRules ? f.branchRules[options.branch] : f.rules;
    if (rules === null) throw new Error('Rule lookup unavailable');
    return rules;
  }}
  if (route === 'matchingRefs') return f.matchingRefs;
  if (route === 'runs') {{
    if (f.runsError) throw new Error('Runs unavailable');
    return f.runSnapshots ? next('runs',f.runSnapshots) : f.runs;
  }}
  if (route === 'statuses') {{
    if (f.statusesError) throw new Error('Statuses unavailable');
    return f.statuses;
  }}
  throw new Error('Unexpected API route: ' + route);
}}, graphql:async (query, variables) => {{
  if (query.includes('query DispatchSnapshot')) {{
    requests.push({{route:'dispatchSnapshot'}});
    if (f.repositoryError) throw new Error('Repository unavailable');
    if (f.queueError) throw new Error('Queue unavailable');
    const value = structuredClone(next('queue', f.queues));
    const repository = value.repository;
    const refs = reserved() && f.refsAfterReservation || f.refs;
    function refValue(ref) {{
      const oid = next('ref-' + ref, refs[ref] ||
        [['heads/master', 'heads/main', 'heads/release-2.18'].includes(ref) ? {json.dumps(BASE)} : {json.dumps(HEAD)}]);
      return oid === null ? null : {{target:{{__typename:'Commit',oid}}}};
    }}
    repository.defaultBranchRef = {{name:next('defaultBranch', f.defaultBranches || [f.defaultBranch || f.branch])}};
    repository.candidateRef = candidateReplaced ? {{target:{{__typename:'Commit',oid:{json.dumps(OTHER)}}}}} : refValue(variables.candidate.slice(5));
    repository.baseRef = refValue(variables.base.slice(5));
    for (const item of repository.mergeQueue.entries.nodes) {{
      const number = item.pullRequest.number;
      const pr = reserved() && f.prAfterReservation ? f.prAfterReservation :
        (f.prs ? f.prs[number] : next('pr',f.queuePRs));
      item.pullRequest = graphqlPR(pr);
    }}
    return value;
  }}
  if (query.includes('query QueuePolicies')) {{
    requests.push({{route:'policies'}});
    const repository = {{nameWithOwner:'owner/repo'}};
    for (const match of query.matchAll(/pr([0-9]+): pullRequest/g)) {{
      const number = match[1];
      const pr = reserved() && f.prAfterReservation ? f.prAfterReservation :
        (f.prs ? f.prs[number] : next('pr',f.queuePRs));
      repository['pr' + number] = {{id:pr.node_id, number:pr.number,
        state:pr.state.toUpperCase(), isDraft:pr.draft, merged:pr.merged,
        headRefOid:pr.head.sha, baseRefName:pr.base.ref,
        baseRepository:{{nameWithOwner:pr.base.repo.full_name}},
        author:{{__typename:pr.user.type, login:pr.user.login}},
        labels:{{nodes:pr.labels,totalCount:pr.labels.length,pageInfo:{{hasNextPage:false}}}}}};
    }}
    return {{repository}};
  }}
  requests.push({{route:'queue'}});
  if (f.queueError) throw new Error('Queue unavailable');
  return next('queue',f.queues);
}}}};
const loadInventory = async options => {{
  inventoryRequests.push(options);
  if (f.inventoryError) throw new Error('Inventory unavailable');
  return {{inventory:f.inventory,workflowFiles:f.workflowFiles}};
}};
const core = {{info:message=>{{
  if (message.startsWith('{{')) summaries.push(JSON.parse(message));
}},warning:()=>{{}},error:()=>{{}}}};
const workspaceRoot = {json.dumps(str(root))};
process.env.GITHUB_WORKSPACE = f.omitRoot ? workspaceRoot : workspaceRoot + '/unused';
gate[f.method]({{github, context:{{eventName:'schedule',payload:{{}},repo:{{owner:'owner',repo:'repo'}},runId:123}},
  group:f.group, ...(f.omitRoot ? {{}} : {{root:workspaceRoot}}), core, loadInventory,
  now:()=>Date.parse({json.dumps(NOW)})}})
.then(result=>console.log(JSON.stringify({{result,workspaceRoot,summaries,writes,dispatches,requests,inventoryRequests,operations,activeDispatches,peakDispatches,completedDispatches}})))
.catch(error=>console.log(JSON.stringify({{error:error.message,workspaceRoot,summaries,writes,dispatches,requests,inventoryRequests,operations,activeDispatches,peakDispatches,completedDispatches}})));
"""
        completed = subprocess.run(
            ['node'], input=script, text=True, capture_output=True, check=True
        )
        return json.loads(completed.stdout)


class QueueRecoveryTest(unittest.TestCase):
    def test_dispatch_uses_workspace_when_caller_omits_root(self):
        for method in ['dispatchGroup', 'dispatchCandidates']:
            for omit_root in [True, False]:
                with self.subTest(method=method, omit_root=omit_root):
                    result = dispatch(method=method, omitRoot=omit_root)
                    self.assertNotIn('error', result)
                    self.assertEqual(len(result['dispatches']), 1, result)
                    self.assertTrue(result['inventoryRequests'])
                    for request in result['inventoryRequests']:
                        self.assertEqual(
                            request.get('root'), result['workspaceRoot']
                        )

    def test_default_rule_failure_does_not_block_release_dispatch(self):
        for rules in [[], None]:
            with self.subTest(rules=rules):
                result = dispatch(
                    method='dispatchCandidates',
                    branch='release-2.18',
                    defaultBranch='main',
                    enabled=['~DEFAULT_BRANCH', 'release-2.18'],
                    branchRules={
                        'main': rules,
                        'release-2.18': [{'type': 'merge_queue'}],
                    },
                )
                self.assertTrue(result['error'].startswith('main: '), result)
                self.assertNotIn('release-2.18:', result['error'])
                self.assertEqual(len(result['dispatches']), 1, result)
                self.assertEqual(
                    result['dispatches'][0]['inputs']['queue_base_ref'],
                    'release-2.18',
                )
                self.assertTrue(
                    any(w['context'] == 'ci-passed' for w in result['writes']),
                    result,
                )

    def test_default_selector_dispatches_resolved_candidate_for_either_name(
        self,
    ):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = dispatch(
                    method='dispatchCandidates',
                    branch=branch,
                    enabled=['~DEFAULT_BRANCH'],
                )
                self.assertNotIn('error', result)
                self.assertTrue(result['dispatches'])
                for call in result['dispatches']:
                    self.assertTrue(
                        call['ref'].startswith(f'gh-readonly-queue/{branch}/')
                    )
                    self.assertEqual(call['inputs']['queue_base_ref'], branch)

    def test_default_branch_change_before_initial_snapshot_never_dispatches(
        self,
    ):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                other = 'main' if branch == 'master' else 'master'
                result = dispatch(
                    branch=branch,
                    enabled=['~DEFAULT_BRANCH'],
                    defaultBranches=[branch, other],
                )
                self.assertEqual(result['dispatches'], [], result)
                self.assertEqual(result['writes'], [], result)

    def test_default_selector_metadata_failure_never_dispatches(self):
        result = dispatch(
            method='dispatchCandidates',
            enabled=['~DEFAULT_BRANCH'],
            repositoryError=True,
        )
        self.assertIn('Repository unavailable', result['error'])
        self.assertEqual(result['dispatches'], [])
        self.assertEqual(result['writes'], [])

    def test_scheduled_recovery_publishes_completed_ci_without_workflow_run(
        self,
    ):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = dispatch(
                    method='dispatchCandidates',
                    branch=branch,
                    runs=[
                        good_run(
                            event='workflow_dispatch',
                            head_branch=f'gh-readonly-queue/{branch}/pr-7-abcdef',
                        )
                    ],
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['dispatches'], [])
                self.assertEqual(
                    [
                        (write['sha'], write['state'])
                        for write in result['writes']
                    ],
                    [(HEAD, 'pending'), (HEAD, 'success')],
                    result,
                )

    def test_shared_candidate_head_is_dispatched_once(self):
        result = dispatch(
            method='dispatchCandidates',
            queues=[
                queue(
                    [
                        entry(headCommit={'oid': HEAD}),
                        entry(number=8, position=2, headCommit={'oid': HEAD}),
                    ]
                )
            ],
            prs={'7': pr(), '8': pr(number=8)},
        )
        self.assertNotIn('error', result)
        self.assertEqual(len(result['dispatches']), 1, result)
        self.assertEqual(
            [
                write['state']
                for write in result['writes']
                if write['context'] == 'ci-passed'
            ],
            ['pending'],
            result,
        )

    def test_rebuilt_candidate_uses_discovered_ref_and_new_sha(self):
        branch = 'gh-readonly-queue/master/pr-7-rebuilt'
        result = dispatch(
            method='dispatchCandidates',
            queues=[queue([entry(headCommit={'oid': OTHER})])],
            matchingRefs=[
                {
                    'ref': 'refs/heads/gh-readonly-queue/master/pr-7-abcdef',
                    'object': {'type': 'commit', 'sha': HEAD},
                },
                {
                    'ref': 'refs/heads/' + branch,
                    'object': {'type': 'commit', 'sha': OTHER},
                },
            ],
            refs={'heads/' + branch: [OTHER]},
            runs=[good_run(event='workflow_dispatch')],
        )
        self.assertNotIn('error', result)
        self.assertEqual(len(result['dispatches']), 1, result)
        self.assertEqual(result['dispatches'][0]['ref'], branch)
        self.assertEqual(
            result['dispatches'][0]['inputs']['queue_head_sha'], OTHER
        )
        self.assertTrue(
            all(write['sha'] == OTHER for write in result['writes'])
        )
        self.assertFalse(
            any(write['state'] == 'success' for write in result['writes'])
        )

    def test_disabled_policy_does_not_read_or_write_github(self):
        result = dispatch(method='dispatchCandidates', enabled=[])
        self.assertNotIn('error', result)
        self.assertEqual(result['requests'], [])
        self.assertEqual(result['inventoryRequests'], [])
        self.assertEqual(result['writes'], [])
        self.assertEqual(result['dispatches'], [])

    def test_missing_candidate_ref_waits_for_queue_rebuild(self):
        result = dispatch(method='dispatchCandidates', matchingRefs=[])
        self.assertNotIn('error', result)
        self.assertEqual(result['dispatches'], [])
        self.assertEqual(result['writes'], [])

    def test_ambiguous_candidate_refs_fail_closed(self):
        result = dispatch(
            method='dispatchCandidates',
            matchingRefs=[
                {
                    'ref': f'refs/heads/gh-readonly-queue/master/pr-7-{suffix}',
                    'object': {'type': 'commit', 'sha': HEAD},
                }
                for suffix in ['one', 'two']
            ],
        )
        self.assertIn('error', result)
        self.assertEqual(result['dispatches'], [])
        self.assertEqual(result['writes'], [])


class QueueDispatchTest(unittest.TestCase):
    def test_full_fanout_batches_member_policy_at_every_boundary(self):
        members = [
            entry(number=number, position=index + 1, headCommit={'oid': HEAD})
            for index, number in enumerate(range(7, 27))
        ]
        result = dispatch(
            lanes=[
                lane(path=f'.github/workflows/lane-{index}.yml')
                for index in range(32)
            ],
            queues=[queue(members)],
            prs={str(number): pr(number=number) for number in range(7, 27)},
        )
        self.assertNotIn('error', result)
        self.assertEqual(len(result['dispatches']), 32)
        routes = [request['route'] for request in result['requests']]
        # Initial validation plus two fresh snapshots for each four-lane batch.
        # Each fresh boundary is one GraphQL snapshot plus active-rules REST;
        # native/dispatched workflow and status discovery add three reads.
        self.assertEqual(len(result['requests']), 37)
        self.assertEqual(routes.count('dispatchSnapshot'), 17)
        self.assertEqual(routes.count('queue'), 0)
        self.assertEqual(routes.count('policies'), 0)
        self.assertEqual(routes.count('pr'), 0)
        self.assert_candidate_counts(result, 17, 32, 32)

    def assert_candidate_counts(
        self, result, validations, reservations, dispatches
    ):
        summaries = [
            row
            for row in result['summaries']
            if row.get('event') == 'queue_candidate_dispatch'
        ]
        self.assertEqual(len(summaries), 1, summaries)
        summary = summaries[0]
        self.assertEqual(summary['candidate'], HEAD)
        self.assertEqual(summary['validations'], validations)
        self.assertEqual(summary['reservations'], reservations)
        self.assertEqual(summary['dispatches'], dispatches)

    def test_replaced_candidate_during_batch_reservation_gets_zero_dispatches(
        self,
    ):
        result = dispatch(
            lanes=[
                lane(path=f'.github/workflows/lane-{i}.yml') for i in range(8)
            ],
            replaceOnReservation='ci-queue-dispatch/lane-3.yml',
        )
        self.assertNotIn('error', result)
        self.assertEqual(len(result['writes']), 4)
        self.assertEqual(
            [call['workflow_id'] for call in result['dispatches']], []
        )
        self.assert_candidate_counts(result, 3, 4, 0)

    def test_missing_workflows_use_bounded_parallel_dispatch(self):
        result = dispatch(
            lanes=[
                lane(path=f'.github/workflows/lane-{i}.yml') for i in range(9)
            ],
            dispatchDelayMs=10,
        )
        self.assertNotIn('error', result)
        self.assertEqual(len(result['dispatches']), 9)
        self.assertEqual(result['peakDispatches'], 4)
        self.assertEqual(result['activeDispatches'], 0)
        self.assertEqual(result['completedDispatches'], 9)

    def test_failed_reservation_drains_siblings_before_reporting_error(self):
        result = dispatch(
            lanes=[
                lane(path=f'.github/workflows/lane-{i}.yml') for i in range(9)
            ],
            statusWriteErrorContext='ci-queue-dispatch/lane-0.yml',
            statusWriteDelayMs=20,
            dispatchDelayMs=10,
        )
        self.assertIn('Status write failed', result['error'])
        self.assertEqual(len(result['writes']), 4)
        self.assertEqual(len(result['dispatches']), 3)
        self.assert_candidate_counts(result, 3, 4, 3)
        self.assertEqual(result['completedDispatches'], 3)
        self.assertEqual(result['activeDispatches'], 0)
        self.assertFalse(
            any(
                'lane-4' in call['workflow_id'] for call in result['dispatches']
            )
        )

    def test_invalid_receipt_preserves_valid_sibling_dispatch(self):
        result = dispatch(
            lanes=[lane(), lane(path='.github/workflows/other.yml')],
            statuses=[reservation(state='error')],
        )
        self.assertIn('Invalid queue dispatch reservation', result['error'])
        self.assertEqual(
            [call['workflow_id'] for call in result['dispatches']],
            ['other.yml'],
        )
        self.assert_candidate_counts(result, 3, 1, 1)

    def test_hold_after_parallel_reservations_stops_later_batches(self):
        result = dispatch(
            lanes=[
                lane(path=f'.github/workflows/lane-{i}.yml') for i in range(9)
            ],
            prAfterReservation=pr(labels=[{'name': 'do-not-merge/hold'}]),
        )
        self.assertIn('error', result)
        self.assertEqual(len(result['writes']), 4)
        self.assertEqual(result['dispatches'], [])

    def assert_never_publishes_success(self, result):
        self.assertFalse(
            any(status['state'] == 'success' for status in result['writes']),
            result,
        )

    def test_missing_lane_dispatches_exact_candidate_for_either_branch(self):
        for branch in ['master', 'main', 'release-2.18']:
            with self.subTest(branch=branch):
                result = dispatch(branch=branch)
                self.assertNotIn('error', result)
                self.assertEqual(len(result['dispatches']), 1, result)
                call = result['dispatches'][0]
                self.assertEqual(call['owner'], 'owner')
                self.assertEqual(call['repo'], 'repo')
                self.assertEqual(call['workflow_id'], 'frontend.yml')
                self.assertEqual(
                    call['ref'], f'gh-readonly-queue/{branch}/pr-7-abcdef'
                )
                self.assertEqual(
                    call['inputs'],
                    {
                        'queue_head_sha': HEAD,
                        'queue_base_ref': branch,
                        'queue_base_sha': BASE,
                    },
                )
                self.assert_never_publishes_success(result)
                reservations = [
                    write
                    for write in result['writes']
                    if write['context'] == 'ci-queue-dispatch/frontend.yml'
                ]
                self.assertEqual(len(reservations), 1, result)
                self.assertEqual(
                    reservations[0]['description'],
                    f'Queue dispatch attempt 1; base {BASE}.',
                )
                self.assertEqual(reservations[0]['sha'], HEAD)
                self.assertEqual(reservations[0]['state'], 'pending')
                self.assertTrue(result['inventoryRequests'])
                for request in result['inventoryRequests']:
                    self.assertEqual(
                        request['pullRequest']['base']['sha'], BASE
                    )
                    self.assertEqual(
                        request['pullRequest']['base']['ref'], branch
                    )

    def test_disabled_policy_never_dispatches(self):
        result = dispatch(enabled=[])
        self.assertEqual(result['dispatches'], [])
        self.assertEqual(result['writes'], [])

    def test_unenforced_queue_never_dispatches(self):
        result = dispatch(rules=[])
        self.assertIn('error', result)
        self.assertEqual(result['dispatches'], [])
        self.assert_never_publishes_success(result)

    def test_existing_native_or_dispatched_attempt_is_not_duplicated(self):
        for event in ['merge_group', 'workflow_dispatch']:
            for status, conclusion in [
                ('queued', None),
                ('in_progress', None),
                ('completed', 'success'),
                ('completed', 'failure'),
                ('completed', 'cancelled'),
            ]:
                with self.subTest(
                    event=event, status=status, conclusion=conclusion
                ):
                    run = good_run(
                        event=event, status=status, conclusion=conclusion
                    )
                    result = dispatch(runs=[run])
                    self.assertNotIn('error', result)
                    self.assertEqual(result['dispatches'], [], result)
                    self.assert_never_publishes_success(result)

    def test_unrelated_runs_cannot_suppress_required_dispatch(self):
        for changed in [
            {'event': 'push'},
            {'event': 'pull_request'},
            {'head_sha': OTHER},
            {'head_branch': 'master'},
            {'head_repository': {'full_name': 'attacker/repo'}},
            {'path': '.github/workflows/unrelated.yml'},
        ]:
            with self.subTest(changed=changed):
                result = dispatch(runs=[good_run(**changed)])
                self.assertNotIn('error', result)
                self.assertEqual(len(result['dispatches']), 1, result)

    def test_only_missing_inventory_lane_is_dispatched(self):
        result = dispatch(
            lanes=[lane(), lane('.github/workflows/backend.yml')],
            runs=[good_run()],
        )
        self.assertNotIn('error', result)
        self.assertEqual(
            [call['workflow_id'] for call in result['dispatches']],
            ['backend.yml'],
        )
        self.assert_never_publishes_success(result)

    def test_recent_reservation_covers_delayed_run_registration(self):
        result = dispatch(statuses=[reservation()])
        self.assertNotIn('error', result)
        self.assertEqual(result['dispatches'], [], result)

    def test_expired_reservation_retries_with_next_attempt(self):
        result = dispatch(
            statuses=[reservation(created_at='2026-11-13T11:57:00Z')]
        )
        self.assertNotIn('error', result)
        self.assertEqual(len(result['dispatches']), 1, result)
        self.assertTrue(
            any(
                write['description']
                == f'Queue dispatch attempt 2; base {BASE}.'
                for write in result['writes']
            ),
            result,
        )

    def test_retry_limit_cannot_create_unbounded_dispatches(self):
        result = dispatch(
            statuses=[reservation(attempt=3, created_at='2026-11-13T11:57:00Z')]
        )
        self.assertEqual(result['dispatches'], [], result)
        self.assert_never_publishes_success(result)

    def test_lost_dispatch_response_preserves_reservation_for_retry(self):
        result = dispatch(dispatchError=True)
        self.assertIn('error', result)
        self.assertEqual(len(result['dispatches']), 1, result)
        self.assert_candidate_counts(result, 3, 1, 1)
        self.assertTrue(
            any(
                write['context'] == 'ci-queue-dispatch/frontend.yml'
                and write['state'] == 'pending'
                for write in result['writes']
            ),
            result,
        )
        self.assert_never_publishes_success(result)

    def test_reservation_failure_prevents_dispatch(self):
        result = dispatch(statusWriteError=True)
        self.assertIn('error', result)
        self.assertEqual(result['dispatches'], [])
        self.assert_never_publishes_success(result)

    def test_candidate_rebuild_before_dispatch_does_not_reuse_old_sha(self):
        for ref in [
            'heads/gh-readonly-queue/master/pr-7-abcdef',
            'heads/master',
        ]:
            with self.subTest(ref=ref):
                result = dispatch(refsAfterReservation={ref: [OTHER]})
                self.assertEqual(result['dispatches'], [], result)
                self.assert_never_publishes_success(result)

    def test_hold_added_after_reservation_prevents_dispatch(self):
        result = dispatch(
            prAfterReservation=pr(labels=[{'name': 'do-not-merge/hold'}])
        )
        self.assertIn('error', result)
        self.assertEqual(result['dispatches'], [], result)
        self.assert_never_publishes_success(result)

    def test_missing_candidate_or_incomplete_membership_fails_closed(self):
        for changed in [queue([]), queue([], hasNextPage=True)]:
            with self.subTest(changed=changed):
                result = dispatch(queues=[changed])
                self.assertIn('error', result)
                self.assertEqual(result['dispatches'], [], result)
                self.assert_never_publishes_success(result)

    def test_unavailable_evidence_fails_closed(self):
        for option in [
            'runsError',
            'statusesError',
            'queueError',
            'inventoryError',
        ]:
            with self.subTest(option=option):
                result = dispatch(**{option: True})
                self.assertIn('error', result)
                self.assertEqual(result['dispatches'], [], result)
                self.assert_never_publishes_success(result)

    def test_invalid_existing_run_metadata_is_not_treated_as_evidence(self):
        result = dispatch(runs=[good_run(run_attempt=0)])
        self.assertIn('error', result)
        self.assertEqual(result['dispatches'], [], result)
        self.assert_never_publishes_success(result)


if __name__ == '__main__':
    unittest.main()
