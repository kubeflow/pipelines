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
"""Exercise queue admission and revocation against deterministic GitHub
fixtures."""

import copy
import hashlib
import json
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/merge_queue_admission.js'
HEAD = 'a' * 40
BASE = 'b' * 40
CANDIDATE = 'c' * 40
OTHER = 'd' * 40


def attempts_url(runs):
    payload = sorted(
        [
            [r['id'], r['run_attempt'], r['status'], r['conclusion']]
            for r in runs
        ]
    )
    stamp = hashlib.sha256(
        json.dumps(payload, separators=(',', ':')).encode()
    ).hexdigest()
    return (
        f'https://github.com/owner/repo/actions/runs/42#queue-attempts={stamp}'
    )


def admitted_run(**overrides):
    return {
        'id': 100,
        'run_attempt': 1,
        'event': 'pull_request',
        'head_sha': HEAD,
        'status': 'completed',
        'conclusion': 'success',
        'created_at': '2026-11-13T12:00:00Z',
        'run_started_at': '2026-11-13T12:00:00Z',
        **overrides,
    }


def pr(number=7, branch='master', **overrides):
    return {
        'number': number,
        'node_id': f'PR_{number}',
        'state': 'open',
        'draft': False,
        'merged': False,
        'head': {'sha': HEAD},
        'base': {'ref': branch, 'repo': {'full_name': 'owner/repo'}},
        'labels': [{'name': 'approved'}, {'name': 'lgtm'}],
        'user': {'login': 'contributor', 'type': 'User'},
        **overrides,
    }


def entry(number=7, branch='master', position=1, **overrides):
    return {
        'id': f'ENTRY_{number}',
        'position': position,
        'headCommit': {'oid': CANDIDATE},
        'baseCommit': {'oid': BASE},
        'pullRequest': {
            'number': number,
            'headRefOid': HEAD,
            'baseRefName': branch,
        },
        **overrides,
    }


def queue(
    entries: list[dict] | None = None, *, base: str = BASE, **overrides: object
) -> dict:
    entries = [] if entries is None else entries
    return {
        'repository': {
            'nameWithOwner': 'owner/repo',
            'baseRef': {'target': {'oid': base}},
            'mergeQueue': {
                'entries': {
                    'nodes': entries,
                    'totalCount': len(entries),
                    'pageInfo': {'hasNextPage': False},
                    **overrides,
                }
            },
        }
    }


def run_case(
    method: str = 'reconcilePR', branch: str = 'master', **options: object
) -> dict:
    fixture = {
        'method': method,
        'branch': branch,
        'number': 7,
        'eventName': 'schedule',
        'payload': {},
        'enabled': [branch],
        'prs': {'7': [pr(branch=branch)]},
        'queues': [queue()],
        'rules': [[{'type': 'merge_queue'}]],
        'statuses': [
            [
                {
                    'id': 10,
                    'context': 'ci-passed',
                    'state': 'success',
                    'target_url': attempts_url([admitted_run()]),
                    'description': f'Queue admission {branch} CI passed; base policy '
                    + 'f' * 64
                    + '.',
                }
            ]
        ],
        'runs': [
            [
                {
                    'id': 100,
                    'run_attempt': 1,
                    'status': 'completed',
                    'conclusion': 'success',
                    'event': 'pull_request',
                    'head_sha': HEAD,
                    'created_at': '2026-11-13T12:00:00Z',
                    'run_started_at': '2026-11-13T12:00:00Z',
                }
            ]
        ],
        'listed': [{'number': 7}],
        **options,
    }
    with tempfile.TemporaryDirectory() as directory:
        root = Path(directory)
        (root / '.github').mkdir()
        (root / '.github/merge-queue-policy.json').write_text(
            json.dumps(
                {
                    'version': 1,
                    'enabled_branches': fixture['enabled'],
                }
            )
        )
        if fixture.get('repository_policy'):
            root = ROOT
        script = f"""
const f = {json.dumps(fixture)};
const writes = [], calls = [];
if (f.mockGateRecovery) {{
  const helper = require({json.dumps(str(MODULE.with_name('merge_queue_pr_gate_recovery.js')))});
  helper.recoverPRGate = async options => {{
    calls.push({{route:'gate-recovery'}});
    if (await options.revalidate() && await options.revalidate()) writes.push({{route:'gate-rerun'}});
  }};
}}
const gate = require({json.dumps(str(MODULE))});
let retryStatus;
let enqueued = false;
let compensatingRemoval = false;
const counts = {{}};
function next(name, values) {{
  const n = counts[name] || 0; counts[name] = n + 1;
  const value = values[Math.min(n, values.length - 1)];
  if (value === null) throw new Error('Fixture API failure: ' + name);
  return value;
}}
const github = {{rest: {{
  actions: {{listWorkflowRunsForRepo:'runs'}},
  pulls: {{list:'pulls', get: async options => {{
    calls.push({{route:'pr', options}});
    return {{data:next('pr-' + options.pull_number, f.prs[String(options.pull_number)])}};
  }}}},
  repos: {{get: async options => {{
    calls.push({{route:'repository',options}});
    if (f.repositoryError) throw new Error('Repository unavailable');
    return {{data:{{full_name:'owner/repo',default_branch:f.defaultBranch || f.branch}}}};
  }}, listCommitStatusesForRef:'statuses', createCommitStatus: async options => {{
    writes.push({{route:'status', options}});
    if (f.statusWriteError) throw new Error('Status write failed');
    const data = {{...options, id: 900, creator:{{type:'Bot',login:'github-actions[bot]'}}, created_at: '2026-11-13T13:00:00Z'}};
    if (options.context === 'queue-admission-retry') retryStatus = data;
    return {{data}};
  }}}},
}}, paginate: async (route, options) => {{
  calls.push({{route,options}});
  if (route === 'runs') return next('runs',f.runs);
  if (route === 'statuses') {{
    const statuses = next('statuses',f.statuses);
    return retryStatus ? [retryStatus, ...statuses] : statuses;
  }}
  if (route === 'pulls') return f.listed;
  if (route.includes('/rules/branches/')) return next('rules',f.rules);
  throw new Error('Unexpected paginate route');
}}, graphql: async (query, options) => {{
  if (query.includes('enqueuePullRequest')) {{
    writes.push({{route:'enqueue',options,query}});
    if (f.enqueueError) throw new Error('Enqueue rejected');
    enqueued = Number(options.id.split('_')[1]);
    return {{enqueuePullRequest:{{mergeQueueEntry:{{id:'ENTRY_7',enqueuedAt:f.enqueuedAt || '2026-11-13T13:00:00Z'}}}}}};
  }}
  if (query.includes('dequeuePullRequest')) {{
    writes.push({{route:'dequeue',options,query}}); enqueued = false; compensatingRemoval = true; return {{}};
  }}
  if (query.includes('query RetryRemoval')) {{
    calls.push({{route:'removal',options,query}});
    const result = structuredClone(next('removal', f.removals || []));
    if (compensatingRemoval && !f.staleCompensationEvent) {{
      result.repository.pullRequest.timelineItems.nodes = [{{
        __typename:'RemovedFromMergeQueueEvent',id:'COMPENSATING_REMOVAL',reason:'manually_removed',
        createdAt:'2026-11-13T13:00:00Z',beforeCommit:{{oid:{json.dumps(CANDIDATE)}}}}}];
    }}
    return result;
  }}
  if (query.includes('query DispatchSnapshot')) {{
    calls.push({{route:'dispatch-snapshot',options,query}});
    return next('dispatch-snapshot', f.dispatchSnapshots);
  }}
  if (query.includes('query QueuePolicies')) {{
    calls.push({{route:'policies',options,query}});
    if (f.policyResponse) return f.policyResponse;
    const repository = {{nameWithOwner:'owner/repo'}};
    for (const match of query.matchAll(/pr([0-9]+): pullRequest/g)) {{
      const number = match[1];
      const pr = next('pr-' + number, f.prs[number]);
      repository['pr' + number] = {{id:pr.node_id, number:pr.number,
        state:pr.state.toUpperCase(), isDraft:pr.draft, merged:pr.merged,
        headRefOid:pr.head.sha, baseRefName:pr.base.ref,
        baseRepository:{{nameWithOwner:pr.base.repo.full_name}},
        author:{{__typename:pr.user.type, login:pr.user.login}},
        labels:{{nodes:pr.labels,totalCount:pr.labels.length,pageInfo:{{hasNextPage:false}}}}}};
    }}
    return {{repository}};
  }}
  calls.push({{route:'queue',options}});
  const result = structuredClone(next('queues',f.queues));
  if (enqueued && !f.queueAdmissionMissing && result?.repository?.mergeQueue?.entries) {{
    const collection = result.repository.mergeQueue.entries;
    if (!collection.nodes.some(e => e.pullRequest.number === enqueued)) {{
      collection.nodes.push({{id:'ENTRY_' + enqueued,position:collection.nodes.length+1,headCommit:null,baseCommit:null,
        pullRequest:{{number:enqueued,headRefOid:{json.dumps(HEAD)},baseRefName:f.branch}}}});
      collection.totalCount = collection.nodes.length;
    }}
  }}
  return result;
}}}};
const conflictCheck = async ({{pr, entries, base, statuses, cache}}) => {{
  calls.push({{route:'dependency-check',number:pr.number,entries,base,
    status:statuses[0]?.context,cache:cache instanceof Map}});
  return (f.conflicted || []).includes(pr.number);
}};
const args = {{github, conflictCheck, context:{{repo:{{owner:'owner',repo:'repo'}},runId:42,eventName:f.eventName,payload:f.payload,ref:f.ref}},
  queueOnly:f.queueOnly || false, root:{json.dumps(str(root))}, number:f.number, branch:f.branch,
  group:{{baseRef:f.branch,branch:'gh-readonly-queue/' + f.branch + '/pr-7-test',head:f.candidate || {json.dumps(CANDIDATE)}}},
  core:{{info:()=>{{}}}}, now:()=>Date.parse(f.now || '2026-11-13T13:00:00Z')}};
(async () => {{
  try {{
    let result;
    if (f.method === 'readQueue') result = await gate.readQueue(github,args.context,f.branch);
    else if (f.method === 'policyReason') result = gate.policyReason(f.prs['7'][0],'owner/repo',f.branch);
    else result = await gate[f.method](args);
    console.log(JSON.stringify({{result,writes,calls}}));
  }} catch(error) {{console.log(JSON.stringify({{error:error.message,writes,calls}}));}}
}})();
"""
        result = subprocess.run(
            ['node'], input=script, text=True, capture_output=True, check=True
        )
        return json.loads(result.stdout)


def dispatch_snapshot(branch='master', entries=None, prs=None, **overrides):
    entries = [entry(branch=branch)] if entries is None else entries
    prs = {7: pr(branch=branch)} if prs is None else prs
    response = queue(entries)
    repo = response['repository']
    repo.update(
        {
            'defaultBranchRef': {'name': branch},
            'candidateRef': {
                'target': {'__typename': 'Commit', 'oid': CANDIDATE}
            },
            'baseRef': {'target': {'__typename': 'Commit', 'oid': BASE}},
            **overrides,
        }
    )
    for member in entries:
        p = prs[member['pullRequest']['number']]
        member['pullRequest'] = {
            'id': p['node_id'],
            'number': p['number'],
            'state': p['state'].upper(),
            'isDraft': p['draft'],
            'merged': p['merged'],
            'headRefOid': p['head']['sha'],
            'baseRefName': p['base']['ref'],
            'baseRepository': {'nameWithOwner': p['base']['repo']['full_name']},
            'author': {
                '__typename': p['user']['type'],
                'login': p['user']['login'],
            },
            'labels': {
                'nodes': p['labels'],
                'totalCount': len(p['labels']),
                'pageInfo': {'hasNextPage': False},
            },
        }
    return response


class RecoveryIntegrationTest(unittest.TestCase):
    def test_failed_aggregate_uses_production_recovery_callback(self) -> None:
        result = run_case(
            mockGateRecovery=True,
            statuses=[[{'context': 'ci-passed', 'state': 'failure'}]],
        )
        self.assertNotIn('error', result)
        self.assertEqual([w['route'] for w in result['writes']], ['gate-rerun'])
        self.assertEqual(
            len([c for c in result['calls'] if c['route'] == 'gate-recovery']),
            1,
        )

    def test_recovery_rechecks_hold_head_success_and_queue_enforcement(
        self,
    ) -> None:
        failure = [{'context': 'ci-passed', 'state': 'failure'}]
        for changes in (
            {'prs': {'7': [pr(), pr(labels=[{'name': 'do-not-merge/hold'}])]}},
            {'prs': {'7': [pr(), pr(head={'sha': OTHER})]}},
            {
                'statuses': [
                    failure,
                    [{'context': 'ci-passed', 'state': 'success'}],
                ]
            },
            {'rules': [[{'type': 'merge_queue'}], []]},
        ):
            with self.subTest(changes=changes):
                options = {
                    'mockGateRecovery': True,
                    'statuses': [failure],
                    **changes,
                }
                result = run_case(**options)
                if 'rules' in changes:
                    self.assertIn(
                        'enforced merge_queue rule', result.get('error', '')
                    )
                else:
                    self.assertNotIn('error', result)
                self.assertEqual(result['writes'], [])

    def test_successful_aggregate_does_not_run_gate_recovery(self) -> None:
        result = run_case(mockGateRecovery=True)
        self.assertFalse(
            any(c['route'] == 'gate-recovery' for c in result['calls'])
        )


class QueueAdmissionTest(unittest.TestCase):
    def test_release_admission_matches_dependabot_exception_and_respects_holds(
        self,
    ):
        branch = 'release-2.18'
        result = run_case(branch=branch)
        self.assertNotIn('error', result)
        self.assertTrue(
            any(w['route'] == 'enqueue' for w in result['writes']), result
        )
        bot = pr(
            branch=branch,
            labels=[],
            user={'login': 'dependabot[bot]', 'type': 'Bot'},
        )
        result = run_case(branch=branch, prs={'7': [bot]})
        self.assertTrue(
            any(w['route'] == 'enqueue' for w in result['writes']), result
        )
        for changes in [
            {'labels': [{'name': 'do-not-merge/hold'}]},
            {'user': {'login': 'another-bot', 'type': 'Bot'}},
            {'user': {'login': 'dependabot[bot]', 'type': 'User'}},
        ]:
            result = run_case(branch=branch, prs={'7': [{**bot, **changes}]})
            self.assertEqual(result['writes'], [], result)

    def test_explicit_release_snapshot_survives_default_rename(self):
        snapshot = dispatch_snapshot('release-2.18')
        snapshot['repository']['defaultBranchRef'] = {'name': 'main'}
        result = run_case(
            'inspectDispatchSnapshot',
            branch='release-2.18',
            enabled=['~DEFAULT_BRANCH', 'release-2.18'],
            dispatchSnapshots=[snapshot],
        )
        self.assertNotIn('error', result)
        self.assertIsNotNone(result['result'])
        result = run_case(
            'inspectDispatchSnapshot',
            branch='release-2.18',
            enabled=['~DEFAULT_BRANCH'],
            dispatchSnapshots=[snapshot],
        )
        self.assertIsNone(result['result'])
        self.assertEqual(result['calls'], [])

    def test_healthy_queue_sweep_reads_members_once(self):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                members = [
                    entry(n, branch=branch, position=i + 1)
                    for i, n in enumerate(range(7, 14))
                ]
                result = run_case(
                    'sweep',
                    branch=branch,
                    enabled=['~DEFAULT_BRANCH'],
                    eventName='workflow_run',
                    payload={'workflow_run': {'event': 'workflow_dispatch'}},
                    queues=[queue(members)],
                    prs={str(n): [pr(n, branch=branch)] for n in range(7, 14)},
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['writes'], [])
                routes = [call['route'] for call in result['calls']]
                self.assertEqual(routes.count('queue'), 1, result)
                self.assertEqual(routes.count('policies'), 1, result)
                self.assertEqual(routes.count('pr'), 0, result)
                self.assertEqual(routes.count('repository'), 2, result)

    def test_sweep_rechecks_invalid_member_before_dequeue(self):
        result = run_case(
            'sweep',
            eventName='workflow_run',
            payload={'workflow_run': {'event': 'workflow_dispatch'}},
            queues=[queue([entry()])],
            prs={'7': [pr(labels=[]), pr()]},
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['writes'], [])
        self.assertEqual([c['route'] for c in result['calls']].count('pr'), 1)

    def test_sweep_incomplete_batched_policy_revokes_candidates(self):
        result = run_case(
            'sweep',
            eventName='workflow_run',
            payload={'workflow_run': {'event': 'workflow_dispatch'}},
            queues=[queue([entry()])],
            policyResponse={'repository': {}},
        )
        self.assertIn('Incomplete queue policy', result['error'])
        self.assertEqual([w['route'] for w in result['writes']], ['status'])
        self.assertEqual(result['writes'][0]['options']['state'], 'failure')

    def test_dispatch_snapshot_matches_existing_policy_with_two_requests(self):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = run_case(
                    'inspectDispatchSnapshot',
                    branch=branch,
                    enabled=['~DEFAULT_BRANCH'],
                    dispatchSnapshots=[dispatch_snapshot(branch)],
                )
                old = run_case(
                    'inspectQueue',
                    branch=branch,
                    queues=[queue([entry(branch=branch)])],
                )
                self.assertNotIn('error', result)
                self.assertEqual(
                    result['result'],
                    {'head': CANDIDATE, 'base': BASE, 'policy': old['result']},
                )
                self.assertEqual(len(result['calls']), 2, result)
                self.assertEqual(
                    result['calls'][0]['route'], 'dispatch-snapshot'
                )
                self.assertIn('/rules/branches/', result['calls'][1]['route'])

    def test_dispatch_snapshot_default_rename_and_disabled_policy_fail_closed(
        self,
    ):
        result = run_case(
            'inspectDispatchSnapshot',
            enabled=['~DEFAULT_BRANCH'],
            dispatchSnapshots=[
                dispatch_snapshot(defaultBranchRef={'name': 'main'})
            ],
        )
        self.assertIsNone(result['result'])
        self.assertEqual(len(result['calls']), 1)
        disabled = run_case('inspectDispatchSnapshot', enabled=[])
        self.assertIsNone(disabled['result'])
        self.assertEqual(disabled['calls'], [])
        for name in ['', 'release-2.18']:
            bad = run_case(
                'inspectDispatchSnapshot',
                enabled=['~DEFAULT_BRANCH'],
                dispatchSnapshots=[
                    dispatch_snapshot(defaultBranchRef={'name': name})
                ],
            )
            self.assertIn('Cannot resolve', bad['error'])

    def test_dispatch_snapshot_rejects_incomplete_data_and_holds(self):
        fixtures = []
        for key in ['candidateRef', 'baseRef']:
            fixtures.extend(
                [
                    dispatch_snapshot(**{key: None}),
                    dispatch_snapshot(
                        **{key: {'target': {'__typename': 'Tag', 'oid': BASE}}}
                    ),
                ]
            )
        missing = dispatch_snapshot()
        missing['repository']['mergeQueue']['entries']['pageInfo'][
            'hasNextPage'
        ] = True
        fixtures.append(missing)
        missing = dispatch_snapshot()
        missing['repository']['mergeQueue']['entries']['nodes'][0][
            'pullRequest'
        ]['labels']['pageInfo']['hasNextPage'] = True
        fixtures.append(missing)
        fixtures.append(dispatch_snapshot(prs={7: pr(labels=[])}))
        missing = dispatch_snapshot()
        missing['errors'] = [{'message': 'partial response'}]
        fixtures.append(missing)
        for fixture in fixtures:
            with self.subTest(fixture=fixture):
                result = run_case(
                    'inspectDispatchSnapshot', dispatchSnapshots=[fixture]
                )
                self.assertIn('error', result)
                self.assertEqual(result['writes'], [])
        result = run_case(
            'inspectDispatchSnapshot',
            dispatchSnapshots=[dispatch_snapshot()],
            rules=[[]],
        )
        self.assertIn('enforced merge_queue rule', result['error'])

    def test_dispatch_snapshot_member_head_change_changes_fingerprint(self):
        before = run_case(
            'inspectDispatchSnapshot', dispatchSnapshots=[dispatch_snapshot()]
        )
        after = run_case(
            'inspectDispatchSnapshot',
            dispatchSnapshots=[
                dispatch_snapshot(prs={7: pr(head={'sha': OTHER})})
            ],
        )
        self.assertNotIn('error', after)
        self.assertNotEqual(
            before['result']['policy'], after['result']['policy']
        )

    def test_dispatch_snapshot_reports_changed_refs_for_caller_comparison(self):
        for key, returned in [('candidateRef', 'head'), ('baseRef', 'base')]:
            result = run_case(
                'inspectDispatchSnapshot',
                dispatchSnapshots=[
                    dispatch_snapshot(
                        **{
                            key: {
                                'target': {'__typename': 'Commit', 'oid': OTHER}
                            }
                        }
                    )
                ],
            )
            self.assertNotIn('error', result)
            self.assertEqual(result['result'][returned], OTHER)

    def test_disabled_policy_does_not_read_or_write_github(self):
        for method in ['sweep', 'reconcilePR', 'inspectQueue']:
            result = run_case(method, enabled=[])
            self.assertEqual(result['writes'], [], result)
            self.assertEqual(result['calls'], [], result)

    def test_default_selector_sweep_enqueues_only_resolved_branch(self):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = run_case(
                    'sweep', branch=branch, enabled=['~DEFAULT_BRANCH']
                )
                self.assertNotIn('error', result)
                self.assertEqual(
                    [w['route'] for w in result['writes']],
                    ['enqueue', 'status'],
                )
                queue_reads = [
                    c for c in result['calls'] if c['route'] == 'queue'
                ]
                self.assertTrue(queue_reads)
                self.assertTrue(
                    all(c['options']['branch'] == branch for c in queue_reads)
                )
                reads = [
                    c
                    for c in result['calls']
                    if '/rules/branches/' in c['route']
                ]
                self.assertTrue(
                    all(c['options']['branch'] == branch for c in reads)
                )

    def test_default_selector_sweep_fails_before_mutation_when_metadata_unavailable(
        self,
    ):
        result = run_case(
            'sweep', enabled=['~DEFAULT_BRANCH'], repositoryError=True
        )
        self.assertIn('Repository unavailable', result['error'])
        self.assertEqual(result['writes'], [])
        self.assertEqual([c['route'] for c in result['calls']], ['repository'])

    def test_approved_pr_enqueues_exact_head_for_either_branch(self):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = run_case(branch=branch)
                self.assertNotIn('error', result)
                self.assertEqual(
                    [write['route'] for write in result['writes']],
                    ['enqueue', 'status'],
                )
                enqueue = result['writes'][0]
                self.assertEqual(
                    enqueue['options'], {'id': 'PR_7', 'head': HEAD}
                )
                self.assertIn('expectedHeadOid:$head', enqueue['query'])
                self.assertIn('jump:false', enqueue['query'])

    def test_approval_and_all_hold_labels_are_enforced(self):
        for labels in [
            [],
            ['approved'],
            ['lgtm'],
            *[
                ['approved', 'lgtm', hold]
                for hold in [
                    'do-not-merge',
                    'do-not-merge/hold',
                    'do-not-merge/custom',
                    'needs-ok-to-test',
                    'needs-rebase',
                ]
            ],
        ]:
            with self.subTest(labels=labels):
                result = run_case(
                    prs={
                        '7': [pr(labels=[{'name': label} for label in labels])]
                    }
                )
                self.assertEqual(result['writes'], [])

    def test_only_dependabot_bot_has_review_exception_and_holds_still_apply(
        self,
    ):
        for user, expected in [
            ({'login': 'dependabot[bot]', 'type': 'Bot'}, True),
            ({'login': 'dependabot[bot]', 'type': 'User'}, False),
            ({'login': 'other-bot', 'type': 'Bot'}, False),
        ]:
            result = run_case(prs={'7': [pr(labels=[], user=user)]})
            self.assertEqual(bool(result['writes']), expected)
        result = run_case(
            prs={
                '7': [
                    pr(
                        labels=[{'name': 'do-not-merge/hold'}],
                        user={'login': 'dependabot[bot]', 'type': 'Bot'},
                    )
                ]
            }
        )
        self.assertEqual(result['writes'], [])

    def test_missing_enforcement_and_api_errors_never_enqueue(self):
        for options in [
            {'rules': [[]]},
            {'rules': [None]},
            {'queues': [None]},
            {'statuses': [None]},
            {'prs': {'7': [None]}},
            {'enqueueError': True},
        ]:
            with self.subTest(options=options):
                result = run_case(**options)
                self.assertIn('error', result)
                self.assertFalse(
                    any(
                        write['route'] == 'status'
                        and write['options']['state'] == 'success'
                        for write in result['writes']
                    )
                )
                if not options.get('enqueueError'):
                    self.assertFalse(
                        any(
                            write['route'] == 'enqueue'
                            for write in result['writes']
                        )
                    )

    def test_current_queue_policy_stamp_is_required_on_head(self):
        for statuses in [
            [],
            [{'id': 10, 'context': 'ci-passed', 'state': 'pending'}],
            [
                {
                    'id': 10,
                    'context': 'ci-passed',
                    'state': 'success',
                    'target_url': attempts_url([admitted_run()]),
                    'description': 'Expected CI and all checks passed',
                }
            ],
            [
                {'id': 10, 'context': 'ci-passed', 'state': 'failure'},
                {
                    'id': 10,
                    'context': 'ci-passed',
                    'state': 'success',
                    'target_url': attempts_url([admitted_run()]),
                    'description': 'Queue admission master CI passed; base policy '
                    + 'f' * 64
                    + '.',
                },
            ],
        ]:
            with self.subTest(statuses=statuses):
                result = run_case(statuses=[statuses])
                self.assertEqual(result['writes'], [])

    def test_same_head_retarget_cannot_reuse_other_branch_admission(self):
        for branch, previous in [('master', 'main'), ('main', 'master')]:
            with self.subTest(branch=branch):
                result = run_case(
                    branch=branch,
                    statuses=[
                        [
                            {
                                'id': 10,
                                'context': 'ci-passed',
                                'state': 'success',
                                'target_url': attempts_url([admitted_run()]),
                                'description': f'Queue admission {previous} CI passed; base policy '
                                + 'f' * 64
                                + '.',
                            }
                        ]
                    ],
                )
                self.assertEqual(result['writes'], [], result)

    def test_queue_admission_rechecks_probe_input_before_enqueue(self) -> None:
        queued = queue([entry(8)])
        for first, current in [
            (queue(), queued),
            (queued, queue()),
            (queued, queue([entry(8)], base=OTHER)),
            (queued, queue([entry(8, headCommit={'oid': OTHER})])),
            (queued, queue([entry(9)])),
        ]:
            with self.subTest(first=first, current=current):
                result = run_case(queues=[first, current])
                self.assertNotIn('error', result, result)
                self.assertEqual(result['writes'], [], result)
        result = run_case(
            prs={'7': [pr(), pr(head={'sha': OTHER})]}, queues=[queued]
        )
        self.assertNotIn('error', result, result)
        self.assertEqual(result['writes'], [], result)

    def test_sweep_checks_new_unbuilt_entry_and_admits_independent_pr(
        self,
    ) -> None:
        newly_admitted = entry(7, headCommit=None, baseCommit=None)
        queued = queue([newly_admitted])
        result = run_case(
            'sweep',
            listed=[{'number': n} for n in [7, 8, 9]],
            prs={str(n): [pr(n)] for n in [7, 8, 9]},
            queues=[queue(), queue(), queue(), queue(), queued],
            conflicted=[8],
        )
        self.assertNotIn('error', result, result)
        self.assertEqual(
            [
                w['options']['id']
                for w in result['writes']
                if w['route'] == 'enqueue'
            ],
            ['PR_7', 'PR_9'],
        )
        checks = [
            c for c in result['calls'] if c['route'] == 'dependency-check'
        ]
        self.assertEqual([c['number'] for c in checks], [8, 9])
        self.assertTrue(all(c['entries'] == [newly_admitted] for c in checks))
        self.assertTrue(all(c['cache'] for c in checks))

    def test_post_enqueue_predecessor_race_is_compensated_without_consumption(
        self,
    ) -> None:
        result = run_case(queues=[queue(), queue(), queue([entry(8)])])
        self.assertNotIn('error', result, result)
        self.assertEqual(
            [w['route'] for w in result['writes']], ['enqueue', 'dequeue']
        )

    def test_post_enqueue_append_does_not_change_predecessors(self) -> None:
        result = run_case(
            queues=[queue(), queue(), queue([entry(), entry(8, position=2)])]
        )
        self.assertNotIn('error', result, result)
        self.assertEqual(
            [w['route'] for w in result['writes']], ['enqueue', 'status']
        )

    def test_pre_enqueue_head_or_label_changes_do_not_enqueue(self):
        for changed in [pr(head={'sha': OTHER}), pr(labels=[]), pr(draft=True)]:
            result = run_case(prs={'7': [pr(), changed]})
            self.assertEqual(result['writes'], [], result)

    def test_post_enqueue_changes_invalidate_candidates_and_dequeue(self):
        for changed in [
            pr(head={'sha': OTHER}),
            pr(labels=[]),
            pr(branch='main'),
            pr(state='closed'),
        ]:
            result = run_case(
                prs={'7': [pr(), pr(), changed]},
                queues=[queue(), queue(), queue([entry()])],
            )
            self.assertNotIn('error', result)
            self.assertEqual(
                [write['route'] for write in result['writes']],
                ['enqueue', 'status', 'dequeue'],
            )
            self.assertEqual(result['writes'][1]['options']['sha'], CANDIDATE)
            self.assertEqual(result['writes'][1]['options']['state'], 'failure')

    def test_post_enqueue_api_or_enforcement_failure_removes_admission(self):
        for options in [
            {'prs': {'7': [pr(), pr(), None]}},
            {
                'rules': [
                    [{'type': 'merge_queue'}],
                    [{'type': 'merge_queue'}],
                    [],
                ]
            },
            {
                'statuses': [
                    [
                        {
                            'id': 10,
                            'context': 'ci-passed',
                            'state': 'success',
                            'target_url': attempts_url([admitted_run()]),
                            'description': 'Queue admission master CI passed; base policy '
                            + 'f' * 64
                            + '.',
                        }
                    ],
                    [
                        {
                            'id': 10,
                            'context': 'ci-passed',
                            'state': 'success',
                            'target_url': attempts_url([admitted_run()]),
                            'description': 'Queue admission master CI passed; base policy '
                            + 'f' * 64
                            + '.',
                        }
                    ],
                    None,
                ]
            },
        ]:
            with self.subTest(options=options):
                result = run_case(
                    queues=[queue(), queue(), queue([entry()])], **options
                )
                self.assertIn('error', result)
                self.assertTrue(
                    any(
                        write['route'] == 'dequeue'
                        for write in result['writes']
                    ),
                    result,
                )
                self.assertTrue(
                    any(
                        write['route'] == 'status'
                        and write['options']['sha'] == CANDIDATE
                        for write in result['writes']
                    ),
                    result,
                )

    def test_admission_consumes_current_ci_evidence_without_another_token(self):
        result = run_case()
        self.assertNotIn('error', result)
        receipt = result['writes'][-1]['options']
        self.assertEqual(receipt['sha'], HEAD)
        self.assertEqual(receipt['context'], 'queue-admission-attempt')
        self.assertRegex(
            receipt['description'],
            r'^Admission master; CI attempts [a-f0-9]{64}\.$',
        )
        passed = {
            'id': 10,
            'context': 'ci-passed',
            'state': 'success',
            'target_url': attempts_url([admitted_run()]),
            'description': 'Queue admission master CI passed; base policy '
            + 'f' * 64
            + '.',
        }
        # GitHub removed a failed candidate: a sweep must not create an endless
        # stream of new queue SHAs using the same original green PR evidence.
        consumed = run_case(statuses=[[receipt, passed]])
        self.assertEqual(consumed['writes'], [], consumed)
        restamped = run_case(statuses=[[dict(passed, id=11), receipt]])
        self.assertEqual(restamped['writes'], [], restamped)
        run = {
            'id': 100,
            'run_attempt': 1,
            'status': 'completed',
            'conclusion': 'success',
            'event': 'pull_request',
            'head_sha': HEAD,
            'created_at': '2026-11-13T12:00:00Z',
            'run_started_at': '2026-11-13T12:00:00Z',
        }
        for changed_run in [dict(run, run_attempt=2), dict(run, id=101)]:
            rerun = run_case(
                statuses=[
                    [
                        dict(
                            passed,
                            id=11,
                            target_url=attempts_url([changed_run]),
                        ),
                        receipt,
                    ]
                ],
                runs=[[changed_run]],
            )
            self.assertNotIn('error', rerun)
            self.assertEqual(rerun['writes'][0]['route'], 'enqueue', rerun)
            self.assertNotEqual(
                rerun['writes'][-1]['options']['description'],
                receipt['description'],
            )
        changed = run_case(runs=[[run], [dict(run, run_attempt=2)]])
        self.assertEqual(changed['writes'], [], changed)
        changed_after = run_case(
            runs=[[run], [run], [dict(run, run_attempt=2)]],
            queues=[queue(), queue(), queue([entry()])],
        )
        self.assertEqual(
            [w['route'] for w in changed_after['writes']],
            ['enqueue', 'status', 'dequeue'],
            changed_after,
        )
        for invalid in [
            dict(run, run_attempt=0),
            dict(run, head_sha='b' * 40),
            dict(run, event='workflow_dispatch'),
            dict(run, created_at=None),
        ]:
            result = run_case(runs=[[invalid]])
            self.assertIn('error', result)
            self.assertEqual(result['writes'], [])

    def test_stale_green_does_not_admit_changed_workflow_attempts(self):
        for run in [
            admitted_run(run_attempt=2, status='in_progress', conclusion=None),
            admitted_run(run_attempt=2, conclusion='failure'),
            admitted_run(run_attempt=2),
            admitted_run(id=101),
        ]:
            with self.subTest(run=run):
                result = run_case(runs=[[run]])
                self.assertNotIn('error', result)
                self.assertEqual(result['writes'], [], result)

    def test_admission_rejects_unbound_success(self):
        for url in [
            '',
            'https://github.com/owner/repo/actions/runs/42',
            attempts_url([admitted_run()]).replace('owner/repo', 'other/repo'),
        ]:
            result = run_case(
                statuses=[
                    [
                        {
                            'id': 10,
                            'context': 'ci-passed',
                            'state': 'success',
                            'description': 'Queue admission master CI passed; base policy '
                            + 'f' * 64
                            + '.',
                            'target_url': url,
                        }
                    ]
                ]
            )
            self.assertEqual(result['writes'], [], result)

    def test_dequeue_uses_required_id_input_with_pr_node_id(self):
        for branch in ['master', 'main']:
            with self.subTest(branch=branch):
                result = run_case(
                    branch=branch,
                    prs={'7': [pr(branch=branch, labels=[])]},
                    queues=[queue([entry(branch=branch)])],
                )
                self.assertNotIn('error', result)
                dequeues = [
                    write
                    for write in result['writes']
                    if write['route'] == 'dequeue'
                ]
                self.assertEqual(len(dequeues), 1, result)
                self.assertEqual(dequeues[0]['options'], {'id': 'PR_7'})
                # DequeuePullRequestInput requires id, unlike enqueue's
                # pullRequestId. The live API rejects the latter on dequeue.
                self.assertRegex(
                    dequeues[0]['query'],
                    r'dequeuePullRequest\s*\(\s*input\s*:\s*'
                    r'\{\s*id\s*:\s*\$id\s*\}',
                )

    def test_queued_pr_revocation_covers_labels_head_retarget_and_closed(self):
        for changed in [
            pr(labels=[]),
            pr(head={'sha': OTHER}),
            pr(branch='main'),
            pr(state='closed'),
            pr(draft=True),
        ]:
            result = run_case(prs={'7': [changed]}, queues=[queue([entry()])])
            self.assertEqual(
                [write['route'] for write in result['writes']],
                ['status', 'dequeue'],
                result,
            )
        result = run_case(prs={'7': [None]}, queues=[queue([entry()])])
        self.assertIn('error', result)
        self.assertEqual(result['writes'][0]['options']['state'], 'failure')

    def test_rejected_enqueue_with_confirmed_absence_preserves_other_candidates(
        self,
    ):
        result = run_case(queues=[queue([entry(8)])], enqueueError=True)
        self.assertIn('Enqueue rejected', result['error'])
        self.assertEqual(
            [write['route'] for write in result['writes']], ['enqueue']
        )
        ambiguous = run_case(
            queues=[queue([entry(8)]), queue([entry(8)]), None],
            enqueueError=True,
        )
        self.assertIn('Enqueue rejected', ambiguous['error'])
        self.assertEqual(
            [write['route'] for write in ambiguous['writes']],
            ['enqueue', 'status', 'dequeue'],
        )

    def test_existing_held_entry_is_dequeued_even_when_status_write_fails(self):
        result = run_case(
            prs={'7': [pr(labels=[])]},
            queues=[queue([entry()])],
            statusWriteError=True,
        )
        self.assertIn('Status write failed', result['error'])
        self.assertTrue(
            any(write['route'] == 'dequeue' for write in result['writes']),
            result,
        )
        self.assertFalse(
            any(write['route'] == 'enqueue' for write in result['writes']),
            result,
        )

    def test_missing_or_malformed_merge_state_and_author_never_authorize(self):
        for changed in [
            {'merged': None},
            {'merged': 'false'},
            {'user': None},
            {'user': {}},
            {'user': {'login': '', 'type': 'User'}},
            {'user': {'login': 'contributor', 'type': 'Organization'}},
        ]:
            with self.subTest(changed=changed):
                result = run_case(prs={'7': [pr(**changed)]})
                self.assertEqual(result['writes'], [], result)
                queued = run_case(
                    prs={'7': [pr(**changed)]}, queues=[queue([entry()])]
                )
                self.assertEqual(
                    [write['route'] for write in queued['writes']],
                    ['status', 'dequeue'],
                )

    def test_queued_eligible_pr_is_not_enqueued_again(self):
        result = run_case(queues=[queue([entry()])])
        self.assertEqual(result['writes'], [], result)

    def test_membership_accepts_zero_or_one_based_contiguous_positions(self):
        for first in [0, 1]:
            result = run_case(
                'readQueue',
                queues=[
                    queue([entry(position=first), entry(8, position=first + 1)])
                ],
            )
            self.assertNotIn('error', result)
            self.assertEqual(len(result['result']), 2)

    def test_membership_rejects_truncation_duplicates_wrong_branch_and_malformed_identity(
        self,
    ):
        malformed = [
            queue([entry()], totalCount=2),
            queue([entry()], pageInfo={'hasNextPage': True}),
            queue([entry(), entry()]),
            queue([entry(branch='main')]),
            queue([entry(position=2)]),
            queue([entry(headCommit={'oid': 'bad'})]),
            {
                'repository': {
                    'nameWithOwner': 'wrong/repo',
                    'mergeQueue': {'entries': {}},
                }
            },
        ]
        for membership in malformed:
            with self.subTest(membership=membership):
                result = run_case('readQueue', queues=[membership])
                self.assertIn('error', result)
                self.assertEqual(result['writes'], [])

    def test_candidate_policy_reads_are_batched_and_complete(self):
        entries = [entry(n, position=n - 6) for n in range(7, 107)]
        result = run_case(
            'inspectQueue',
            queues=[queue(entries)],
            prs={str(n): [pr(n)] for n in range(7, 107)},
        )
        self.assertNotIn('error', result)
        self.assertEqual(sum(c['route'] == 'queue' for c in result['calls']), 1)
        self.assertEqual(
            sum(c['route'] == 'policies' for c in result['calls']), 1
        )
        self.assertEqual(sum(c['route'] == 'pr' for c in result['calls']), 0)
        for response in [
            {'repository': {'nameWithOwner': 'owner/repo'}},
            {'repository': {'nameWithOwner': 'other/repo'}},
            {'errors': [], 'repository': {'nameWithOwner': 'owner/repo'}},
        ]:
            invalid = run_case(
                'inspectQueue',
                queues=[queue([entry()])],
                policyResponse=response,
            )
            self.assertIn('error', invalid)

    def test_batched_policy_normalizes_graphql_dependabot_identity(self):
        result = run_case(
            'inspectQueue',
            queues=[queue([entry()])],
            prs={
                '7': [
                    pr(labels=[], user={'login': 'dependabot', 'type': 'Bot'})
                ]
            },
        )
        self.assertNotIn('error', result)
        rejected = run_case(
            'inspectQueue',
            queues=[queue([entry()])],
            prs={'7': [pr(labels=[], user={'login': 'other', 'type': 'Bot'})]},
        )
        self.assertIn('error', rejected)

    def test_candidate_requires_exact_membership_and_every_queued_pr_eligible(
        self,
    ):
        members = queue(
            [entry(), entry(8, position=2, headCommit={'oid': OTHER})]
        )
        result = run_case(
            'inspectQueue', queues=[members], prs={'7': [pr()], '8': [pr(8)]}
        )
        self.assertNotIn('error', result)
        self.assertIsInstance(result['result'], str)
        for later in [
            pr(8, labels=[]),
            pr(8, head={'sha': OTHER}),
            pr(8, state='closed'),
        ]:
            result = run_case(
                'inspectQueue',
                queues=[members],
                prs={'7': [pr()], '8': [later]},
            )
            self.assertIn('error', result)
        result = run_case('inspectQueue', queues=[members], candidate='e' * 40)
        self.assertIn('error', result)

    def test_candidate_fingerprint_ignores_eligible_suffix_and_unrelated_labels(
        self,
    ):
        original = run_case('inspectQueue', queues=[queue([entry()])])
        extended = run_case(
            'inspectQueue',
            queues=[
                queue(
                    [entry(), entry(8, position=2, headCommit={'oid': OTHER})]
                )
            ],
            prs={
                '7': [
                    pr(
                        labels=[
                            {'name': name}
                            for name in ['approved', 'lgtm', 'triage/accepted']
                        ]
                    )
                ],
                '8': [pr(8)],
            },
        )
        self.assertNotIn('error', extended)
        self.assertEqual(original['result'], extended['result'])
        held_suffix = run_case(
            'inspectQueue',
            queues=[
                queue(
                    [entry(), entry(8, position=2, headCommit={'oid': OTHER})]
                )
            ],
            prs={'7': [pr()], '8': [pr(8, labels=[])]},
        )
        self.assertIn('error', held_suffix)

    def test_surviving_writer_admits_ready_pr_without_unrelated_recovery(
        self,
    ) -> None:
        ready = pr(
            labels=[
                {'name': name} for name in ['lgtm', 'approved', 'ci-passed']
            ]
        )
        listed = [ready, pr(99, labels=[])]
        for options in [
            {
                'eventName': 'pull_request_target',
                'payload': {'pull_request': {'number': 8}},
            },
            {'queueOnly': True},
            {
                'eventName': 'workflow_dispatch',
                'ref': 'refs/heads/master',
                'payload': {
                    'repository': {'default_branch': 'master'},
                    'inputs': {'pr_number': '8', 'head_sha': HEAD},
                },
            },
        ]:
            with self.subTest(options=options):
                result = run_case(
                    'sweep',
                    listed=listed,
                    prs={'7': [ready], '8': [pr(8, labels=[])]},
                    **options,
                )
                self.assertNotIn('error', result, result)
                self.assertEqual(
                    len(
                        [w for w in result['writes'] if w['route'] == 'enqueue']
                    ),
                    1,
                    result,
                )
                self.assertEqual(
                    len([c for c in result['calls'] if c['route'] == 'pulls']),
                    1,
                )
                self.assertNotIn(
                    99,
                    [
                        c['options']['pull_number']
                        for c in result['calls']
                        if c['route'] == 'pr'
                    ],
                )

    def test_ready_label_cannot_override_hold_or_stale_evidence(self) -> None:
        ready = pr(
            labels=[
                {'name': name} for name in ['lgtm', 'approved', 'ci-passed']
            ]
        )
        held = copy.deepcopy(ready)
        held['labels'].append({'name': 'do-not-merge/hold'})
        for options in [{'prs': {'7': [held]}}, {'statuses': [[]]}]:
            result = run_case(
                'sweep', queueOnly=True, listed=[ready], **options
            )
            self.assertFalse(
                [w for w in result['writes'] if w['route'] == 'enqueue'], result
            )
        result = run_case('sweep', queueOnly=True, listed=[held])
        self.assertFalse([c for c in result['calls'] if c['route'] == 'pr'])

    def test_wakeup_requires_trusted_branch_and_matching_head(self) -> None:
        payload = {
            'repository': {'default_branch': 'master'},
            'inputs': {'pr_number': '7', 'head_sha': HEAD},
        }
        result = run_case(
            'sweep',
            eventName='workflow_dispatch',
            ref='refs/heads/master',
            payload=payload,
            listed=[],
        )
        self.assertNotIn('error', result, result)
        self.assertTrue(
            [w for w in result['writes'] if w['route'] == 'enqueue']
        )
        for ref in ['refs/heads/feature', None]:
            result = run_case(
                'sweep', eventName='workflow_dispatch', ref=ref, payload=payload
            )
            self.assertIn('Invalid trusted', result['error'])
            self.assertEqual(result['writes'], [])
        payload['inputs']['head_sha'] = OTHER
        result = run_case(
            'sweep',
            eventName='workflow_dispatch',
            ref='refs/heads/master',
            payload=payload,
            listed=[],
        )
        self.assertEqual(result['writes'], [])

    def test_pr_event_reconciles_only_affected_pr_and_all_queue_members(self):
        result = run_case(
            'sweep',
            eventName='pull_request_target',
            payload={'pull_request': {'number': 8}},
            queues=[queue([entry()])],
            listed=[{'number': 99}],
            prs={'7': [pr(labels=[])], '8': [pr(8, labels=[])]},
        )
        self.assertNotIn('error', result)
        self.assertEqual(
            len([call for call in result['calls'] if call['route'] == 'pulls']),
            1,
        )
        self.assertEqual(
            {
                call['options']['pull_number']
                for call in result['calls']
                if call['route'] == 'pr'
            },
            {7, 8},
        )
        self.assertTrue(
            any(write['route'] == 'dequeue' for write in result['writes'])
        )

    def test_workflow_event_uses_exact_head_discovery_plus_queue_members(self):
        run = {
            'event': 'pull_request',
            'head_sha': HEAD,
            'head_branch': 'feature',
            'head_repository': {
                'full_name': 'contributor/repo',
                'owner': {'login': 'contributor'},
            },
        }
        listed = [
            {
                'number': 8,
                'head': {
                    'sha': HEAD,
                    'repo': {'full_name': 'contributor/repo'},
                },
            },
            {
                'number': 9,
                'head': {
                    'sha': OTHER,
                    'repo': {'full_name': 'contributor/repo'},
                },
            },
            {
                'number': 10,
                'head': {'sha': HEAD, 'repo': {'full_name': 'other/repo'}},
            },
        ]
        result = run_case(
            'sweep',
            eventName='workflow_run',
            payload={'workflow_run': run},
            queues=[queue([entry()])],
            listed=listed,
            prs={'7': [pr(labels=[])], '8': [pr(8, labels=[])]},
        )
        self.assertNotIn('error', result)
        self.assertEqual(
            [
                call['options']
                for call in result['calls']
                if call['route'] == 'pulls'
            ],
            [
                {
                    'owner': 'owner',
                    'repo': 'repo',
                    'state': 'open',
                    'head': 'contributor:feature',
                    'per_page': 100,
                },
                {
                    'owner': 'owner',
                    'repo': 'repo',
                    'state': 'open',
                    'base': 'master',
                    'per_page': 100,
                },
            ],
        )
        self.assertEqual(
            {
                call['options']['pull_number']
                for call in result['calls']
                if call['route'] == 'pr'
            },
            {7, 8},
        )

    def test_schedule_sweeps_all_open_prs_and_unsupported_events_fail_closed(
        self,
    ):
        result = run_case(
            'sweep',
            listed=[{'number': 7}, {'number': 8}],
            prs={'7': [pr(labels=[])], '8': [pr(8, labels=[])]},
        )
        self.assertNotIn('error', result)
        self.assertEqual(
            [
                call['options']
                for call in result['calls']
                if call['route'] == 'pulls'
            ],
            [
                {
                    'owner': 'owner',
                    'repo': 'repo',
                    'base': 'master',
                    'state': 'open',
                    'per_page': 100,
                }
            ],
        )
        for event in ['push', 'issue_comment', '']:
            unsupported = run_case('sweep', eventName=event)
            self.assertIn('Unsupported', unsupported['error'])
            self.assertEqual(unsupported['calls'], [])
            self.assertEqual(unsupported['writes'], [])

    def test_sweep_isolates_default_rule_failure_from_release_admission(self):
        for rules in [[], None]:
            with self.subTest(rules=rules):
                result = run_case(
                    'sweep',
                    branch='release-2.18',
                    defaultBranch='main',
                    enabled=['~DEFAULT_BRANCH', 'release-2.18'],
                    rules=[rules, [{'type': 'merge_queue'}]],
                )
                self.assertTrue(result['error'].startswith('main: '), result)
                self.assertNotIn('release-2.18:', result['error'])
                self.assertEqual(
                    [w['route'] for w in result['writes']],
                    ['enqueue', 'status'],
                )
                self.assertEqual(
                    [
                        c['options']['base']
                        for c in result['calls']
                        if c['route'] == 'pulls'
                    ],
                    ['release-2.18'],
                )

    def test_sweep_includes_closed_queued_prs_and_continues_after_one_failure(
        self,
    ):
        result = run_case(
            'sweep',
            queues=[queue([entry()])],
            listed=[{'number': 8}],
            prs={'7': [pr(state='closed')], '8': [None]},
        )
        self.assertIn('error', result)
        self.assertTrue(
            any(write['route'] == 'dequeue' for write in result['writes'])
        )
        result = run_case(
            'sweep', listed=[{'number': index} for index in range(1, 258)]
        )
        self.assertIn('error', result)
        self.assertEqual(result['writes'], [])


class AutomaticRetryTest(unittest.TestCase):
    def fixture(self, **changes: object) -> dict:
        stamp = attempts_url([admitted_run()]).split('=')[-1]
        passed = {
            'id': 10,
            'context': 'ci-passed',
            'state': 'success',
            'target_url': attempts_url([admitted_run()]),
            'description': 'Queue admission master CI passed; base policy '
            + 'f' * 64
            + '.',
        }
        consumed = {
            'id': 11,
            'context': 'queue-admission-attempt',
            'state': 'pending',
            'created_at': '2026-11-13T12:10:00Z',
            'description': f'Admission master; CI attempts {stamp}.',
        }
        removal = {
            'repository': {
                'nameWithOwner': 'owner/repo',
                'pullRequest': {
                    'number': 7,
                    'headRefOid': HEAD,
                    'baseRefName': 'master',
                    'timelineItems': {
                        'nodes': [
                            {
                                '__typename': 'RemovedFromMergeQueueEvent',
                                'id': 'REMOVAL_1',
                                'reason': 'failed_checks',
                                'createdAt': '2026-11-13T12:30:00Z',
                                'beforeCommit': {'oid': CANDIDATE},
                            }
                        ]
                    },
                },
            }
        }
        return {
            'statuses': [[passed, consumed]],
            'removals': [removal],
            **changes,
        }

    def test_dependency_conflict_preserves_retry_budget(self) -> None:
        earlier = entry(8, headCommit=None)
        result = run_case(
            **self.fixture(queues=[queue([earlier])], conflicted=[7])
        )
        self.assertNotIn('error', result, result)
        self.assertEqual(result['writes'], [], result)
        checks = [
            c for c in result['calls'] if c['route'] == 'dependency-check'
        ]
        self.assertEqual(len(checks), 1, checks)
        self.assertEqual(checks[0]['entries'], [earlier])
        self.assertEqual(checks[0]['base'], BASE)
        self.assertEqual(checks[0]['status'], 'ci-passed')

    def test_queue_change_after_clean_probe_does_not_reserve_retry(
        self,
    ) -> None:
        result = run_case(**self.fixture(queues=[queue([entry(8)]), queue()]))
        self.assertNotIn('error', result, result)
        self.assertEqual(result['writes'], [], result)
        retried = run_case(**self.fixture(queues=[queue()]))
        self.assertNotIn('error', retried, retried)
        self.assertIn('enqueue', [w['route'] for w in retried['writes']])

    def test_post_enqueue_race_releases_retry_only_after_confirmed_dequeue(
        self,
    ) -> None:
        result = run_case(
            **self.fixture(queues=[queue(), queue(), queue([entry(8)])])
        )
        self.assertNotIn('error', result, result)
        self.assertEqual(
            [w['route'] for w in result['writes']],
            ['status', 'enqueue', 'dequeue', 'status'],
        )
        compensation = result['writes'][-1]['options']
        self.assertEqual(compensation['context'], 'queue-admission-retry')
        self.assertEqual(compensation['state'], 'error')
        self.assertIn(
            '#queue-retry-compensation=11:', compensation['target_url']
        )
        compensation.update(
            id=901, creator={'type': 'Bot', 'login': 'github-actions[bot]'}
        )
        fixture = self.fixture()
        fixture['statuses'][0].insert(0, compensation)
        fixture['removals'][0]['repository']['pullRequest']['timelineItems'][
            'nodes'
        ][0]['id'] = 'COMPENSATING_REMOVAL'
        retried = run_case(**fixture)
        self.assertNotIn('error', retried, retried)
        self.assertEqual(
            [w['route'] for w in retried['writes']],
            ['status', 'enqueue', 'status'],
        )
        # A later manual removal or enqueue cannot revive this old receipt.
        for later in (
            {
                '__typename': 'RemovedFromMergeQueueEvent',
                'id': 'MANUAL_REMOVAL',
                'reason': 'manually_removed',
            },
            {'__typename': 'AddedToMergeQueueEvent', 'id': 'MANUAL_ENQUEUE'},
        ):
            for before_enqueue in (False, True):
                with self.subTest(later=later, before_enqueue=before_enqueue):
                    moved = self.fixture()
                    moved['statuses'][0].insert(0, compensation)
                    old = copy.deepcopy(fixture['removals'][0])
                    new = copy.deepcopy(old)
                    new['repository']['pullRequest']['timelineItems'][
                        'nodes'
                    ] = [later]
                    moved['removals'] = (
                        [old, old, new] if before_enqueue else [new]
                    )
                    rejected = run_case(**moved)
                    self.assertNotIn('error', rejected, rejected)
                    self.assertNotIn(
                        'enqueue', [w['route'] for w in rejected['writes']]
                    )
        for change in (
            {'creator': {'type': 'User', 'login': 'someone'}},
            {'target_url': compensation['target_url'].replace('=11:', '=12:')},
        ):
            invalid = self.fixture()
            invalid['statuses'][0].insert(0, dict(compensation, **change))
            rejected = run_case(**invalid)
            self.assertEqual(rejected['writes'], [])

    def test_failed_compensation_does_not_release_retry(self) -> None:
        result = run_case(
            **self.fixture(queues=[queue(), queue(), queue([entry(8)]), None])
        )
        self.assertIn('error', result)
        self.assertFalse(
            any(
                w['route'] == 'status' and w['options'].get('state') == 'error'
                for w in result['writes']
            )
        )

    def test_stale_compensating_event_cannot_release_retry(self) -> None:
        result = run_case(
            **self.fixture(
                queues=[queue(), queue(), queue([entry(8)])],
                staleCompensationEvent=True,
            )
        )
        self.assertIn(
            'Compensating removal event not confirmed', result.get('error', '')
        )
        self.assertFalse(
            any(
                w['route'] == 'status' and w['options'].get('state') == 'error'
                for w in result['writes']
            )
        )

    def test_disappeared_acknowledged_admission_consumes_budget_across_sweeps(
        self,
    ) -> None:
        first = run_case(queueAdmissionMissing=True)
        self.assertNotIn('error', first, first)
        self.assertEqual(
            [w['route'] for w in first['writes']], ['enqueue', 'status']
        )
        consumed = first['writes'][-1]['options']
        consumed.update(
            id=901,
            created_at='2026-11-13T13:00:03Z',
            creator={'type': 'Bot', 'login': 'github-actions[bot]'},
        )
        second = self.fixture()
        second['statuses'][0][-1] = consumed
        event = second['removals'][0]['repository']['pullRequest'][
            'timelineItems'
        ]['nodes'][0]
        event['createdAt'] = '2026-11-13T13:00:01Z'
        immediate = run_case(**second, now='2026-11-13T13:00:04Z')
        self.assertEqual(immediate['writes'], [], immediate)
        delayed = run_case(**second, now='2026-11-13T13:16:00Z')
        self.assertEqual(
            [w['route'] for w in delayed['writes']],
            ['status', 'enqueue', 'status'],
        )
        event['createdAt'] = '2026-11-13T12:59:59Z'
        older = run_case(**second, now='2026-11-13T13:16:00Z')
        self.assertEqual(older['writes'], [], older)
        event['createdAt'] = '2026-11-13T13:00:01Z'
        for change in (
            {'creator': {'type': 'User', 'login': 'someone'}},
            {
                'target_url': consumed['target_url'].replace(
                    'owner/repo', 'other/repo'
                )
            },
            {
                'target_url': consumed['target_url'].replace(
                    '13%3A00%3A00', '13%3A01%3A00'
                )
            },
        ):
            invalid = copy.deepcopy(second)
            invalid['statuses'][0][-1].update(change)
            rejected = run_case(**invalid, now='2026-11-13T13:16:00Z')
            self.assertEqual(rejected['writes'], [], rejected)

    def test_acknowledged_enqueue_read_failure_records_consumption(
        self,
    ) -> None:
        result = run_case(prs={'7': [pr(), pr(), None]})
        self.assertIn('error', result)
        records = [
            w
            for w in result['writes']
            if w['route'] == 'status'
            and w['options'].get('context') == 'queue-admission-attempt'
        ]
        self.assertEqual(len(records), 1, result)
        self.assertIn('dequeue', [w['route'] for w in result['writes']])

    def test_retry_is_reserved_before_enqueue(self) -> None:
        result = run_case(**self.fixture())
        self.assertNotIn('error', result)
        self.assertEqual(
            [w['route'] for w in result['writes']],
            ['status', 'enqueue', 'status'],
        )
        self.assertEqual(
            result['writes'][0]['options']['context'], 'queue-admission-retry'
        )

    def test_consumed_retry_and_restamped_green_cannot_retry_again(
        self,
    ) -> None:
        fixture = self.fixture()
        first = run_case(**fixture)
        receipt = dict(first['writes'][0]['options'], id=900)
        for passed in (
            fixture['statuses'][0][0],
            dict(fixture['statuses'][0][0], id=20),
        ):
            result = run_case(
                **self.fixture(
                    statuses=[[receipt, passed, fixture['statuses'][0][1]]]
                )
            )
            self.assertEqual(result['writes'], [], result)
            self.assertFalse(
                any(c['route'] == 'removal' for c in result['calls'])
            )

    def test_retry_waits_fifteen_minutes_after_removal(self) -> None:
        result = run_case(**self.fixture(now='2026-11-13T12:44:59Z'))
        self.assertEqual(result['writes'], [], result)
        result = run_case(**self.fixture(now='2026-11-13T12:45:00Z'))
        self.assertEqual(
            [w['route'] for w in result['writes']],
            ['status', 'enqueue', 'status'],
        )

    def test_manual_unknown_and_active_queue_events_do_not_retry(self) -> None:
        for reason in ('manually_removed', 'merged', 'unknown', None):
            fixture = self.fixture()
            fixture['removals'][0]['repository']['pullRequest'][
                'timelineItems'
            ]['nodes'][0]['reason'] = reason
            result = run_case(**fixture)
            self.assertEqual(result['writes'], [], result)
        fixture = self.fixture()
        fixture['removals'][0]['repository']['pullRequest']['timelineItems'][
            'nodes'
        ] = [{'__typename': 'AddedToMergeQueueEvent'}]
        self.assertEqual(run_case(**fixture)['writes'], [])

    def test_old_future_missing_and_malformed_removal_evidence_cannot_retry(
        self,
    ):
        for change in (
            {'createdAt': '2026-11-13T12:09:00Z'},
            {'createdAt': '2026-11-13T14:00:00Z'},
            {'createdAt': None},
            {'id': None},
            {'beforeCommit': {'oid': 'bad'}},
        ):
            fixture = self.fixture()
            fixture['removals'][0]['repository']['pullRequest'][
                'timelineItems'
            ]['nodes'][0].update(change)
            self.assertEqual(run_case(**fixture)['writes'], [])
        for change in (
            {'headRefOid': OTHER},
            {'baseRefName': 'main'},
            {'number': 8},
        ):
            fixture = self.fixture()
            fixture['removals'][0]['repository']['pullRequest'].update(change)
            result = run_case(**fixture)
            self.assertIn('error', result)
            self.assertEqual(result['writes'], [])

    def test_hold_or_changed_head_prevents_retry(self) -> None:
        for changed in (
            pr(
                labels=[
                    {'name': 'approved'},
                    {'name': 'lgtm'},
                    {'name': 'hold'},
                ]
            ),
            pr(head={'sha': OTHER}),
        ):
            fixture = self.fixture(prs={'7': [pr(), changed]})
            result = run_case(**fixture)
            self.assertEqual(result['writes'], [], result)

    def test_hold_after_reservation_consumes_budget_without_enqueue(
        self,
    ) -> None:
        held = pr(
            labels=[{'name': 'approved'}, {'name': 'lgtm'}, {'name': 'hold'}]
        )
        result = run_case(**self.fixture(prs={'7': [pr(), pr(), held]}))
        self.assertEqual([w['route'] for w in result['writes']], ['status'])
        self.assertEqual(
            result['writes'][0]['options']['context'], 'queue-admission-retry'
        )

    def test_new_removal_after_reservation_prevents_enqueue(self) -> None:
        fixture = self.fixture()
        changed = copy.deepcopy(fixture['removals'][0])
        changed['repository']['pullRequest']['timelineItems']['nodes'][0][
            'reason'
        ] = 'manually_removed'
        fixture['removals'] = [
            fixture['removals'][0],
            fixture['removals'][0],
            changed,
        ]
        result = run_case(**fixture)
        self.assertEqual([w['route'] for w in result['writes']], ['status'])

    def test_failed_retry_reservation_never_enqueues(self) -> None:
        result = run_case(**self.fixture(statusWriteError=True))
        self.assertIn('error', result)
        self.assertFalse(any(w['route'] == 'enqueue' for w in result['writes']))

    def test_ambiguous_enqueue_does_not_restore_retry_budget(self) -> None:
        fixture = self.fixture(enqueueError=True)
        result = run_case(**fixture)
        self.assertIn('error', result)
        self.assertEqual(
            result['writes'][0]['options']['context'], 'queue-admission-retry'
        )
        receipt = dict(result['writes'][0]['options'], id=900)
        result = run_case(
            **self.fixture(statuses=[[receipt, *fixture['statuses'][0]]])
        )
        self.assertEqual(result['writes'], [])

    def test_post_enqueue_head_change_dequeues_retry(self) -> None:
        result = run_case(
            **self.fixture(
                prs={'7': [pr(), pr(), pr(), pr(head={'sha': OTHER})]},
                queues=[queue(), queue(), queue([entry()])],
            )
        )
        self.assertIn('dequeue', [w['route'] for w in result['writes']])

    def test_partial_history_response_cannot_retry(self) -> None:
        fixture = self.fixture()
        fixture['removals'][0]['errors'] = [{'message': 'Partial history'}]
        result = run_case(**fixture)
        self.assertIn('error', result)
        self.assertEqual(result['writes'], [])

    def test_history_api_failure_cannot_retry(self) -> None:
        result = run_case(**self.fixture(removals=[None]))
        self.assertIn('error', result)
        self.assertEqual(result['writes'], [])

    def test_new_verified_workflow_attempt_resets_retry_budget(self) -> None:
        fixture = self.fixture()
        result = run_case(**fixture)
        retry = dict(result['writes'][0]['options'], id=900)
        rerun = admitted_run(run_attempt=2)
        passed = dict(
            fixture['statuses'][0][0], target_url=attempts_url([rerun])
        )
        result = run_case(
            **self.fixture(
                statuses=[[retry, passed, fixture['statuses'][0][1]]],
                runs=[[rerun]],
            )
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['writes'][0]['route'], 'enqueue')


if __name__ == '__main__':
    unittest.main()
