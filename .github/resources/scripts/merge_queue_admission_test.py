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
"""Exercise queue admission and revocation against deterministic GitHub fixtures."""

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


def queue(entries=None, **overrides):
    entries = [] if entries is None else entries
    return {
        'repository': {
            'nameWithOwner': 'owner/repo',
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


def run_case(method='reconcilePR', branch='master', **options):
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
const gate = require({json.dumps(str(MODULE))});
const f = {json.dumps(fixture)};
const writes = [], calls = [];
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
  repos: {{listCommitStatusesForRef:'statuses', createCommitStatus: async options => {{
    writes.push({{route:'status', options}});
    if (f.statusWriteError) throw new Error('Status write failed');
    return {{data: options}};
  }}}},
}}, paginate: async (route, options) => {{
  calls.push({{route,options}});
  if (route === 'runs') return next('runs',f.runs);
  if (route === 'statuses') return next('statuses',f.statuses);
  if (route === 'pulls') return f.listed;
  if (route.includes('/rules/branches/')) return next('rules',f.rules);
  throw new Error('Unexpected paginate route');
}}, graphql: async (query, options) => {{
  if (query.includes('enqueuePullRequest')) {{
    writes.push({{route:'enqueue',options,query}});
    if (f.enqueueError) throw new Error('Enqueue rejected');
    return {{enqueuePullRequest:{{mergeQueueEntry:{{id:'ENTRY_7'}}}}}};
  }}
  if (query.includes('dequeuePullRequest')) {{
    writes.push({{route:'dequeue',options,query}}); return {{}};
  }}
  calls.push({{route:'queue',options}});
  return next('queues',f.queues);
}}}};
const args = {{github, context:{{repo:{{owner:'owner',repo:'repo'}},eventName:f.eventName,payload:f.payload}},
  root:{json.dumps(str(root))}, number:f.number, branch:f.branch,
  group:{{baseRef:f.branch,head:f.candidate || {json.dumps(CANDIDATE)}}},
  core:{{info:()=>{{}}}}}};
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


class QueueAdmissionTest(unittest.TestCase):
    def test_checked_in_disabled_policy_does_not_read_or_write_github(self):
        for method in ['sweep', 'reconcilePR', 'inspectQueue']:
            result = run_case(method, repository_policy=True)
            self.assertEqual(result['writes'], [], result)
            self.assertEqual(result['calls'], [], result)

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
                    'description': 'Expected CI and all checks passed',
                }
            ],
            [
                {'id': 10, 'context': 'ci-passed', 'state': 'failure'},
                {
                    'id': 10,
                    'context': 'ci-passed',
                    'state': 'success',
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
                                'description': f'Queue admission {previous} CI passed; base policy '
                                + 'f' * 64
                                + '.',
                            }
                        ]
                    ],
                )
                self.assertEqual(result['writes'], [], result)

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
                queues=[queue(), queue([entry()])],
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
                result = run_case(queues=[queue(), queue([entry()])], **options)
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
            'event': 'pull_request',
            'head_sha': HEAD,
            'created_at': '2026-11-13T12:00:00Z',
            'run_started_at': '2026-11-13T12:00:00Z',
        }
        for changed_run in [dict(run, run_attempt=2), dict(run, id=101)]:
            rerun = run_case(
                statuses=[[dict(passed, id=11), receipt]], runs=[[changed_run]]
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
            queues=[queue(), queue([entry()])],
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
            queues=[queue([entry(8)]), None], enqueueError=True
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
        self.assertFalse(
            [call for call in result['calls'] if call['route'] == 'pulls']
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
                }
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


if __name__ == '__main__':
    unittest.main()
