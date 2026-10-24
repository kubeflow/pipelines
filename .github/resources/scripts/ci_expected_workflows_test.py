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
"""Execute the production workflow coverage verifier against API fixtures."""

import ast
import json
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_expected_workflows.js'


def node(body):
    script = f'const gate = require({json.dumps(str(MODULE))});\n' + body
    result = subprocess.run(
        ['node'],
        input=script,
        check=True,
        capture_output=True,
        text=True,
        cwd=ROOT,
    )
    return json.loads(result.stdout)


def verify(
    runs=None,
    files=None,
    changed_files=1,
    trigger=None,
    fresh_after=None,
    workflow_paths=None,
    registration_started_at=None,
    now=0,
    base_sha=None,
    live_base=True,
    base_arrived_at='2026-09-07T13:00:00Z',
    association_failure=False,
    associated_prs=None,
    fallback_pages=None,
    pr_records=None,
    fallback_failure=False,
    record_failure=False,
    base_ref='master',
    workflow_triggers=None,
):
    if runs is None:
        runs = [good_run()]
    if files is None:
        files = [{'filename': 'frontend/package.json'}]
    if trigger is None:
        trigger = {'branches': ['master'], 'paths': ['frontend/**']}
    if workflow_paths is None:
        workflow_paths = ['.github/workflows/frontend.yml']
    fixture = {
        'runs': runs,
        'files': files,
        'pullRequest': {
            'number': 7,
            'changed_files': changed_files,
            'head': {
                'sha': 'head',
                'ref': 'dependency',
                'repo': {'full_name': 'owner/repo'},
            },
            'base': {'ref': base_ref, 'sha': 'base'},
        },
        'inventory': {
            'version': 1,
            'workflows': [
                {
                    'path': path,
                    'header_sha256': 'header',
                    'pull_request': (workflow_triggers or {}).get(
                        path, trigger
                    ),
                }
                for path in workflow_paths
            ],
        },
        'workflowFiles': [
            {
                'path': path,
                'header_sha256': 'header',
            }
            for path in workflow_paths
        ],
        'baseSha': base_sha,
        'liveBase': live_base,
        'freshAfter': fresh_after,
        'registrationStartedAt': registration_started_at,
        'now': now,
    }
    if associated_prs is None:
        associated_prs = (
            [
                {
                    'number': 7,
                    'merged_at': base_arrived_at,
                }
            ]
            if base_sha
            else []
        )

    def record(pr):
        return {
            'number': 7,
            'merged': True,
            'merge_commit_sha': base_sha,
            'merged_at': base_arrived_at,
            'base': {'ref': 'master', 'repo': {'full_name': 'owner/repo'}},
            **pr,
        }

    associated_prs = [record(pr) for pr in associated_prs]
    fallback_pages = fallback_pages or [[]]
    if pr_records is None:
        pr_records = {str(pr['number']): record(pr) for pr in associated_prs}
    return node(f"""
const fixture = {json.dumps(fixture)};
const requests = [];
const associationCalls = [];
const associationFailure = {json.dumps(association_failure)};
const associatedPRs = {json.dumps(associated_prs)};
const fallbackPages = {json.dumps(fallback_pages)};
const prRecords = {json.dumps(pr_records)};
const fallbackCalls = [];
const recordCalls = [];
const github = {{
  rest: {{pulls: {{listFiles: 'files',
    list: async options => {{
      fallbackCalls.push(options);
      if ({json.dumps(fallback_failure)}) throw Error('PR list unavailable');
      return {{data: fallbackPages[options.page - 1] || []}};
    }},
    get: async options => {{
      recordCalls.push(options);
      if ({json.dumps(record_failure)}) throw Error('PR record unavailable');
      return {{data: prRecords[options.pull_number]}};
    }},
  }}, actions: {{listWorkflowRunsForRepo: 'runs'}},
    repos: {{listPullRequestsAssociatedWithCommit: 'associated-prs'}}}},
  paginate: async (route, options) => {{
    requests.push({{route, options}});
    if (route === 'associated-prs') {{
      associationCalls.push(options);
      if (associationFailure) throw Error('Associated PRs API unavailable');
      return associatedPRs;
    }}
    return fixture[route];
  }},
}};
gate.verifyExpectedWorkflows({{...fixture, github, owner: 'owner', repo: 'repo'}})
  .then(result => console.log(JSON.stringify({{...result, requests, associationCalls, fallbackCalls, recordCalls}})))
  .catch(error => console.log(JSON.stringify({{error: error.message}})));
""")


def good_run(**overrides):
    return {
        'id': 100,
        'path': '.github/workflows/frontend.yml',
        'run_attempt': 1,
        'event': 'pull_request',
        'head_sha': 'head',
        'head_branch': 'dependency',
        'head_repository': {'full_name': 'owner/repo'},
        'status': 'completed',
        'conclusion': 'success',
        'created_at': '2026-09-07T12:00:00Z',
        'run_started_at': '2026-09-07T12:00:00Z',
        'pull_requests': [],
        **overrides,
    }


class ExpectedWorkflowsTest(unittest.TestCase):
    def test_completion_selector_covers_master_and_release_workflows(self):
        inventory = json.loads(
            (ROOT / '.github/resources/ci-workflow-inventory.json').read_text()
        )
        expected = [
            workflow['name']
            for workflow in inventory['workflows']
            if workflow['pull_request'] is not None
        ]
        # The default-branch publisher also handles release-branch reruns.
        legacy = 'KFP API Integration V2 (Legacy)'
        if legacy not in expected:
            expected.append(legacy)
        workflow = (ROOT / '.github/workflows/ci-checks.yml').read_text()
        selector = workflow.split('  workflow_run:\n', 1)[1].split(
            '    workflows:\n', 1
        )[1]
        names = []
        for line in selector.splitlines():
            if line.startswith('      - '):
                names.append(ast.literal_eval(line[len('      - ') :]))
            elif line.strip() and not line.lstrip().startswith('#'):
                break
        self.assertCountEqual(names, expected)

    def test_inventory_matches_every_workflow_trigger_header(self):
        result = node("""
const loaded = gate.loadLocalInventory(process.cwd());
console.log(JSON.stringify(loaded.inventory.workflows.length));
""")
        self.assertGreater(result, 30)

    def test_ordered_exclusions_and_zero_directory_glob(self):
        result = node("""
const patterns = ['**/*.py', '!sdk/**', 'sdk/keep.py'];
console.log(JSON.stringify(['root.py', 'a/b.py', 'sdk/no.py', 'sdk/keep.py']
  .map(path => gate.matchesPatterns(path, patterns))));
""")
        self.assertEqual(result, [True, True, False, True])

    def test_unknown_syntax_and_trigger_keys_fail_closed(self):
        for trigger in (
            {'paths': ['frontend/[ab].js']},
            {'types': ['labeled']},
            {'paths': ['frontend/**'], 'paths-ignore': ['docs/**']},
        ):
            with self.subTest(trigger=trigger):
                self.assertIn('error', verify(trigger=trigger))

    def test_missing_workflow_cannot_be_green(self):
        result = verify(runs=[])
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'pending')
        self.assertEqual(result['missing'], ['.github/workflows/frontend.yml'])
        self.assertIn('has not registered', result['reasons'][0])

    def test_missing_workflow_registration_grace_expires(self):
        for now, expected in [
            (15 * 60 * 1000 - 1, 'pending'),
            (15 * 60 * 1000, 'failure'),
        ]:
            with self.subTest(now=now):
                result = verify(
                    runs=[],
                    registration_started_at='1970-01-01T00:00:00Z',
                    now=now,
                )
                self.assertEqual(result['state'], expected)
                self.assertFalse(result['passed'])
                self.assertEqual(
                    result['missing'], ['.github/workflows/frontend.yml']
                )
                if expected == 'failure':
                    self.assertIn('after 15 minutes', result['reasons'][0])
        result = verify(
            registration_started_at='1970-01-01T00:00:00Z', now=15 * 60 * 1000
        )
        self.assertEqual(result['state'], 'success')
        self.assertTrue(result['passed'])
        self.assertEqual(result['missing'], [])

    def test_invalid_registration_timestamps_fail_closed(self):
        for timestamp in ['', 'invalid', 0, [], {}]:
            with self.subTest(timestamp=timestamp):
                self.assertIn(
                    'error', verify(registration_started_at=timestamp)
                )
        for now in [None, 'invalid']:
            with self.subTest(now=now):
                self.assertIn('error', verify(now=now))

    def test_active_workflows_are_pending(self):
        for status in [
            'queued',
            'requested',
            'waiting',
            'in_progress',
            'pending',
        ]:
            with self.subTest(status=status):
                result = verify(runs=[good_run(status=status, conclusion=None)])
                self.assertEqual(result['state'], 'pending')
                self.assertFalse(result['passed'])
                self.assertEqual(result['missing'], [])

    def test_malformed_or_unsuccessful_runs_fail_closed(self):
        for status, conclusion in [
            ('completed', 'cancelled'),
            ('completed', 'failure'),
            ('completed', 'timed_out'),
            ('completed', 'action_required'),
            ('completed', 'stale'),
            ('completed', 'skipped'),
            ('completed', 'neutral'),
            ('completed', None),
            ('completed', 'unknown'),
            ('completed', ['success']),
            ('in_progress', 'success'),
            ('unknown', None),
            (None, None),
        ]:
            with self.subTest(status=status, conclusion=conclusion):
                result = verify(
                    runs=[good_run(status=status, conclusion=conclusion)]
                )
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])

    def test_matching_run_requires_identity_and_attempt_timestamps(self):
        for field in ['id', 'run_attempt', 'created_at', 'run_started_at']:
            with self.subTest(field=field, missing=True):
                run = good_run()
                del run[field]
                result = verify(runs=[run])
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])
        invalid_values = {
            'id': [None, 0, -1, 1.5, '100', True, 9007199254740992],
            'run_attempt': [None, 0, -1, 1.5, '1', True, 9007199254740992],
            'created_at': [None, 0, True, '', 'invalid', [], {}],
            'run_started_at': [None, 0, True, '', 'invalid', [], {}],
        }
        for field, values in invalid_values.items():
            for value in values:
                with self.subTest(field=field, value=value):
                    result = verify(runs=[good_run(**{field: value})])
                    self.assertEqual(result['state'], 'failure')
                    self.assertFalse(result['passed'])

    def test_malformed_competing_run_cannot_fall_back_to_success(self):
        for field in ['id', 'run_attempt', 'created_at', 'run_started_at']:
            malformed = good_run(id=99, conclusion='failure')
            del malformed[field]
            for runs in [[malformed, good_run()], [good_run(), malformed]]:
                with self.subTest(field=field, runs=runs):
                    result = verify(runs=runs)
                    self.assertEqual(result['state'], 'failure')
                    self.assertFalse(result['passed'])

    def test_only_unstarted_first_attempt_can_have_null_start_time(self):
        result = verify(
            runs=[
                good_run(status='queued', conclusion=None, run_started_at=None)
            ]
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        for overrides in [{'run_attempt': 2}, {'status': 'in_progress'}]:
            with self.subTest(overrides=overrides):
                result = verify(
                    runs=[
                        good_run(
                            **{
                                'status': 'queued',
                                'conclusion': None,
                                'run_started_at': None,
                                **overrides,
                            }
                        )
                    ]
                )
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])

    def test_failure_takes_precedence_over_pending_workflow(self):
        paths = [
            '.github/workflows/frontend.yml',
            '.github/workflows/backend.yml',
        ]
        result = verify(
            workflow_paths=paths,
            runs=[
                good_run(status='in_progress', conclusion=None),
                good_run(path=paths[1], conclusion='failure'),
            ],
        )
        self.assertEqual(result['state'], 'failure')
        self.assertFalse(result['passed'])
        self.assertIn('completed/failure', result['reasons'][0])
        self.assertIn('in_progress', result['reasons'][1])

    def test_only_latest_execution_counts(self):
        for status, conclusion in [
            ('queued', None),
            ('in_progress', None),
            ('completed', 'cancelled'),
            ('completed', 'action_required'),
            ('completed', 'failure'),
            ('completed', 'skipped'),
        ]:
            with self.subTest(status=status, conclusion=conclusion):
                result = verify(
                    runs=[
                        good_run(),
                        good_run(id=101, status=status, conclusion=conclusion),
                    ]
                )
                self.assertFalse(result['passed'])
        result = verify(runs=[good_run(conclusion='failure'), good_run(id=101)])
        self.assertTrue(result['passed'])
        self.assertEqual(result['state'], 'success')

    def test_current_attempt_cannot_reuse_earlier_success(self):
        result = verify(
            runs=[
                good_run(run_attempt=2, status='in_progress', conclusion=None)
            ]
        )
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'pending')
        result = verify(
            runs=[
                good_run(id=101),
                good_run(
                    id=100,
                    run_attempt=2,
                    status='completed',
                    conclusion='cancelled',
                    run_started_at='2026-09-07T14:00:00Z',
                ),
            ]
        )
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'failure')

    def test_retargeted_base_needs_new_execution_not_old_run_rerun(self):
        cutoff = '2026-09-07T13:00:00Z'
        result = verify(
            runs=[
                good_run(run_attempt=2, run_started_at='2026-09-07T14:00:00Z')
            ],
            fresh_after=cutoff,
        )
        self.assertFalse(result['passed'])
        result = verify(
            runs=[good_run(created_at='2026-09-07T14:00:00Z')],
            fresh_after=cutoff,
        )
        self.assertTrue(result['passed'])

    def test_wrong_head_repo_event_or_base_cannot_supply_evidence(self):
        for overrides in (
            {'head_sha': 'old'},
            {'path': '.github/workflows/unrelated.yml'},
            {'event': 'push'},
            {'head_branch': 'another'},
            {'head_repository': {'full_name': 'attacker/repo'}},
            {'pull_requests': [{'number': 7, 'base': {'ref': 'release-2.17'}}]},
        ):
            with self.subTest(overrides=overrides):
                self.assertFalse(verify(runs=[good_run(**overrides)])['passed'])

    def test_active_workflow_cannot_hide_stale_base_evidence(self):
        for options in [
            {'fresh_after': '2026-09-07T13:00:00Z'},
            {
                'runs': [
                    good_run(
                        status='in_progress',
                        conclusion=None,
                        pull_requests=[
                            {'number': 7, 'base': {'ref': 'release'}}
                        ],
                    )
                ]
            },
        ]:
            with self.subTest(options=options):
                result = verify(
                    **{
                        'runs': [
                            good_run(status='in_progress', conclusion=None)
                        ],
                        **options,
                    }
                )
                self.assertEqual(result['state'], 'failure')
                self.assertFalse(result['passed'])

    def test_run_created_before_base_landed_is_stale(self):
        # The run succeeded against an earlier base; master has since advanced
        # to base_sha. GitHub may have mutated the run's association to report
        # the current base, but the run was CREATED before the new base landed,
        # so its merge-ref CI could not have tested it. The immutable creation
        # time, not the mutated association, decides freshness.
        result = verify(
            base_sha='c' * 40,
            runs=[
                good_run(
                    created_at='2026-09-07T12:00:00Z',
                    pull_requests=[
                        {
                            'number': 7,
                            'base': {'ref': 'master', 'sha': 'c' * 40},
                        }
                    ],
                )
            ],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertIn('awaiting a fresh CI run', result['reasons'][0])
        self.assertEqual(len(result['associationCalls']), 1)

    def test_delayed_fast_forward_integration_is_stale(self):
        # A commit prepared well before it is pushed does not become reachable
        # on master until the push. A run created in the window between the
        # commit's own timestamp and the push tested the previous base, so the
        # merge record's arrival time, not the commit's creation time, must
        # decide freshness: the run is stale.
        result = verify(
            base_sha='c' * 40,
            base_arrived_at='2026-09-07T13:00:00Z',
            runs=[good_run(created_at='2026-09-07T12:00:00Z')],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertIn('awaiting a fresh CI run', result['reasons'][0])

    def test_run_created_in_same_second_as_base_fails_closed(self):
        # created_at and the base arrival time are both second-granularity, so
        # an equal instant cannot prove the run was created after the base
        # arrived: it may have been created sub-second before and tested an
        # older base. The tie must resolve fail-closed to pending, never fresh,
        # so the merge gate cannot pass stale evidence as fresh.
        result = verify(
            base_sha='c' * 40,
            base_arrived_at='2026-09-07T13:00:00Z',
            runs=[good_run(created_at='2026-09-07T13:00:00Z')],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertIn('awaiting a fresh CI run', result['reasons'][0])

    def test_rerunning_old_execution_is_not_fresh(self):
        # Rerunning an old successful execution preserves its original
        # GITHUB_SHA/GITHUB_REF and original creation time, while its
        # association base may be mutated to the current base. The preserved
        # creation time, not the mutated association, decides freshness.
        result = verify(
            base_sha='c' * 40,
            runs=[
                good_run(
                    created_at='2026-09-07T12:00:00Z',
                    run_attempt=2,
                    pull_requests=[
                        {
                            'number': 7,
                            'base': {'ref': 'master', 'sha': 'c' * 40},
                        }
                    ],
                )
            ],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])

    def test_empty_association_fails_closed_when_run_predates_base(self):
        # Forked PRs can carry an empty pull_requests array. The freshness check
        # must still key off the run's immutable creation time rather than the
        # current PR base, and fail closed (pending, never fresh) when the run
        # was created before the base landed.
        result = verify(
            base_sha='c' * 40,
            runs=[
                good_run(created_at='2026-09-07T12:00:00Z', pull_requests=[])
            ],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertEqual(len(result['associationCalls']), 1)

    def test_source_branch_behind_base_with_fresh_merge_ci_is_fresh(self):
        # A pull_request run's head_sha is the SOURCE commit, but CI checks out
        # the synthetic merge commit. A source branch is normally behind its
        # base (compare(base, head) is "diverged"), yet merge-ref CI that ran
        # after the base landed DID test the base. This must pass, not pend:
        # source-branch currency is a separate policy question from whether the
        # run tested the current base.
        result = verify(
            base_sha='c' * 40,
            base_arrived_at='2026-09-07T11:00:00Z',
            runs=[good_run(created_at='2026-09-07T12:00:00Z')],
        )
        self.assertTrue(result['passed'])
        self.assertEqual(result['state'], 'success')
        self.assertEqual(len(result['associationCalls']), 1)

    def test_run_created_after_base_landed_is_fresh(self):
        # Positive control: a run created after the base landed tested a merge
        # commit containing the base, so it is accepted as fresh evidence.
        result = verify(
            base_sha='c' * 40,
            base_arrived_at='2026-09-07T11:00:00Z',
            runs=[good_run(created_at='2026-09-07T12:00:00Z')],
        )
        self.assertTrue(result['passed'])
        self.assertEqual(result['state'], 'success')
        self.assertEqual(len(result['associationCalls']), 1)

    def test_release_branch_frozen_base_skips_live_freshness_check(self):
        # Release branches keep their frozen base.sha behavior: the live-tip
        # freshness check must not apply, so a run created long before the
        # (frozen) base is still accepted and the base commit is never looked up.
        result = verify(
            base_sha='c' * 40,
            live_base=False,
            runs=[good_run(created_at='2026-09-07T12:00:00Z')],
        )
        self.assertTrue(result['passed'])
        self.assertEqual(result['state'], 'success')
        self.assertEqual(result['associationCalls'], [])

    def test_base_arrival_lookup_failure_is_distinct(self):
        # A transient lookup failure must not read as stale CI: it emits a
        # distinct pending reason that retries, not the stale-CI message.
        result = verify(
            base_sha='c' * 40,
            association_failure=True,
            runs=[good_run(created_at='2026-09-07T12:00:00Z')],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertIn('lookup failed', result['reasons'][0])
        self.assertNotIn('awaiting a fresh CI run', result['reasons'][0])

    def fallback_result(self, record=None, **options):
        valid = {
            'number': 14815,
            'merged': True,
            'merge_commit_sha': 'c' * 40,
            'merged_at': '2026-09-07T11:00:00Z',
            'base': {'ref': 'master', 'repo': {'full_name': 'owner/repo'}},
        }
        return verify(
            base_sha='c' * 40,
            associated_prs=[],
            fallback_pages=[[{'number': 14815, 'merge_commit_sha': 'c' * 40}]],
            pr_records={'14815': {**valid, **(record or {})}},
            **options,
        )

    def test_empty_association_recovers_verified_merge_record(self):
        result = self.fallback_result()
        self.assertTrue(result['passed'])
        self.assertEqual(len(result['fallbackCalls']), 1)
        self.assertEqual(
            result['fallbackCalls'][0],
            {
                'owner': 'owner',
                'repo': 'repo',
                'state': 'closed',
                'base': 'master',
                'sort': 'updated',
                'direction': 'desc',
                'per_page': 100,
                'page': 1,
            },
        )
        self.assertEqual(
            result['recordCalls'],
            [
                {
                    'owner': 'owner',
                    'repo': 'repo',
                    'pull_number': 14815,
                }
            ],
        )

    def test_recovered_record_still_rejects_stale_and_rerun_ci(self):
        for attempt in [1, 2]:
            with self.subTest(attempt=attempt):
                result = self.fallback_result(
                    runs=[
                        good_run(
                            created_at='2026-09-07T10:00:00Z',
                            run_attempt=attempt,
                            run_started_at='2026-09-07T14:00:00Z',
                        )
                    ]
                )
                self.assertFalse(result['passed'])
                self.assertIn('awaiting a fresh CI run', result['reasons'][0])

    def test_fallback_requires_authoritative_exact_merge_identity(self):
        for override in [
            {'merge_commit_sha': 'd' * 40},
            {'merged': False},
            {'number': 14816},
            {
                'base': {
                    'ref': 'release-2.18',
                    'repo': {'full_name': 'owner/repo'},
                }
            },
            {'base': {'ref': 'master', 'repo': {'full_name': 'fork/repo'}}},
        ]:
            with self.subTest(override=override):
                result = self.fallback_result(record=override)
                self.assertFalse(result['passed'])
                self.assertIn('no merged PR', result['reasons'][0])

    def test_fallback_requires_valid_merge_timestamp(self):
        for value in [None, 0, '', 'not-a-date']:
            with self.subTest(value=value):
                result = self.fallback_result(record={'merged_at': value})
                self.assertFalse(result['passed'])
                self.assertIn('malformed merged-PR', result['reasons'][0])

    def test_fallback_api_errors_do_not_authorize_success(self):
        for failure in ['fallback_failure', 'record_failure']:
            with self.subTest(failure=failure):
                result = self.fallback_result(**{failure: True})
                self.assertFalse(result['passed'])
                self.assertIn('lookup failed', result['reasons'][0])

    def test_fallback_search_is_bounded_and_ignores_unrelated_records(self):
        unrelated = [
            {'number': number + 1, 'merge_commit_sha': 'd' * 40}
            for number in range(100)
        ]
        for pages in [[unrelated, []], [unrelated] * 6]:
            with self.subTest(pages=len(pages)):
                result = verify(
                    base_sha='c' * 40, associated_prs=[], fallback_pages=pages
                )
                self.assertFalse(result['passed'])
                self.assertEqual(
                    len(result['fallbackCalls']), min(len(pages), 5)
                )
                self.assertEqual(result['recordCalls'], [])

    def test_fallback_finds_candidate_on_later_page(self):
        result = verify(
            base_sha='c' * 40,
            associated_prs=[],
            fallback_pages=[
                [{'number': 1, 'merge_commit_sha': 'd' * 40}] * 100,
                [{'number': 8, 'merge_commit_sha': 'c' * 40}],
            ],
            pr_records={
                '8': {
                    'number': 8,
                    'merged': True,
                    'merge_commit_sha': 'c' * 40,
                    'merged_at': '2026-09-07T11:00:00Z',
                    'base': {
                        'ref': 'master',
                        'repo': {'full_name': 'owner/repo'},
                    },
                }
            },
        )
        self.assertTrue(result['passed'])
        self.assertEqual(len(result['fallbackCalls']), 2)

    def test_unrelated_association_does_not_hide_verified_fallback(self):
        result = verify(
            base_sha='c' * 40,
            associated_prs=[{'number': 9, 'merge_commit_sha': 'd' * 40}],
            fallback_pages=[[{'number': 8, 'merge_commit_sha': 'c' * 40}]],
            pr_records={
                '8': {
                    'number': 8,
                    'merged': True,
                    'merge_commit_sha': 'c' * 40,
                    'merged_at': '2026-09-07T11:00:00Z',
                    'base': {
                        'ref': 'master',
                        'repo': {'full_name': 'owner/repo'},
                    },
                }
            },
        )
        self.assertTrue(result['passed'])
        self.assertEqual(len(result['fallbackCalls']), 1)
        self.assertEqual(
            [call['pull_number'] for call in result['recordCalls']], [8]
        )

    def test_primary_association_is_verified_without_fallback(self):
        result = verify(
            base_sha='c' * 40, base_arrived_at='2026-09-07T11:00:00Z'
        )
        self.assertTrue(result['passed'])
        self.assertEqual(len(result['recordCalls']), 1)
        self.assertEqual(result['fallbackCalls'], [])

    def test_missing_base_merge_record_requires_maintainer(self):
        # A direct push leaves no merged PR for the base revision, so its
        # arrival time cannot be established. Fresh CI cannot fix that; the
        # reason must name the base SHA and request maintainer investigation
        # instead of the stale-CI message.
        result = verify(
            base_sha='c' * 40,
            associated_prs=[],
            runs=[good_run(created_at='2026-09-07T12:00:00Z')],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertIn('no merged PR for base ' + 'c' * 40, result['reasons'][0])
        self.assertNotIn('awaiting a fresh CI run', result['reasons'][0])

    def test_invalid_base_merge_record_requires_maintainer(self):
        # A merged PR whose merged_at is unparseable is a malformed record, not
        # a missing one. It must get its own wording (and its own test): the
        # reason names the base SHA, says the record is malformed, and still
        # requests maintainer investigation, never the stale-CI message.
        result = verify(
            base_sha='c' * 40,
            associated_prs=[{'number': 7, 'merged_at': 'not-a-date'}],
            runs=[good_run(created_at='2026-09-07T12:00:00Z')],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertIn(
            'malformed merged-PR record for base ' + 'c' * 40,
            result['reasons'][0],
        )
        self.assertNotIn('awaiting a fresh CI run', result['reasons'][0])

    def test_missing_base_reason_fits_140_char_status(self):
        # ci_passed.js slices the published status description to 140 chars.
        # The missing/invalid reasons are emitted without the workflow-path
        # prefix (the condition is repo-wide), so the full message must fit and
        # the trailing "Maintainer investigation required." must survive. The
        # longest real inventory path is used so an accidental re-introduction
        # of the prefix would push the reason past the limit.
        inventory = json.loads(
            (ROOT / '.github/resources/ci-workflow-inventory.json').read_text()
        )
        longest = max(workflow['path'] for workflow in inventory['workflows'])
        for associated_prs, fragment in [
            ([], 'no merged PR for base '),
            (
                [{'number': 7, 'merged_at': 'not-a-date'}],
                'malformed merged-PR record for base ',
            ),
        ]:
            with self.subTest(fragment=fragment):
                result = verify(
                    base_sha='c' * 40,
                    associated_prs=associated_prs,
                    workflow_paths=[longest],
                    runs=[
                        good_run(
                            path=longest, created_at='2026-09-07T12:00:00Z'
                        )
                    ],
                )
                reason = result['reasons'][0]
                self.assertLessEqual(len(reason), 140, reason)
                self.assertIn(fragment + 'c' * 40, reason)
                self.assertTrue(
                    reason.endswith('Maintainer investigation required.'),
                    reason,
                )

    def test_missing_base_reason_survives_absent_workflows(self):
        # When no expected workflow has registered, the loop never reaches its
        # per-workflow provenance branch, so the repository-wide reason must
        # still be emitted up front (with its recovery docs link) and must
        # precede the per-workflow 'has not registered' pending message.
        for associated_prs, fragment in [
            ([], 'no merged PR for base '),
            (
                [{'number': 7, 'merged_at': 'not-a-date'}],
                'malformed merged-PR record for base ',
            ),
        ]:
            with self.subTest(fragment=fragment):
                result = verify(
                    base_sha='c' * 40, associated_prs=associated_prs, runs=[]
                )
                self.assertEqual(result['state'], 'pending')
                self.assertFalse(result['passed'])
                self.assertEqual(
                    result['missing'], ['.github/workflows/frontend.yml']
                )
                self.assertIn(fragment + 'c' * 40, result['reasons'][0])
                self.assertTrue(
                    result['reasons'][0].endswith(
                        'Maintainer investigation required.'
                    ),
                    result['reasons'][0],
                )
                self.assertIn('has not registered', result['reasons'][1])
                self.assertEqual(
                    result['documentation'],
                    'https://github.com/kubeflow/pipelines/blob/master/docs/'
                    'agents/ci-passed.md#missing-base-merge-record',
                )

    def test_missing_base_reason_precedes_in_progress_workflow(self):
        # When the base has no merge record and a workflow is still in progress,
        # the repository-wide reason must be emitted first so the 140-char
        # status slice (ci_passed.js publishes reasons joined with '; ') keeps
        # the base SHA and the maintainer instruction intact.
        for associated_prs, fragment in [
            ([], 'no merged PR for base '),
            (
                [{'number': 7, 'merged_at': 'not-a-date'}],
                'malformed merged-PR record for base ',
            ),
        ]:
            with self.subTest(fragment=fragment):
                result = verify(
                    base_sha='c' * 40,
                    associated_prs=associated_prs,
                    runs=[good_run(status='in_progress', conclusion=None)],
                )
                self.assertEqual(result['state'], 'pending')
                self.assertFalse(result['passed'])
                self.assertIn(fragment + 'c' * 40, result['reasons'][0])
                self.assertIn('in_progress', result['reasons'][1])
                # The full maintainer sentence (base SHA + instruction) must
                # survive the 140-char status slice published by ci_passed.js.
                bounded = '; '.join(result['reasons'])[:140]
                self.assertIn(
                    fragment
                    + 'c' * 40
                    + '. Maintainer investigation required.',
                    bounded,
                )

    def test_latest_stale_success_cannot_outrank_fresh_failure(self):
        # The "latest-attempt ordering" attack (#14705): an old successful
        # run created before the base landed sorts latest by attempt time (a
        # rerun), so the ordering selects it over a competing failed execution.
        # The freshness check must downgrade that stale success to pending so it
        # never aggregates to success.
        result = verify(
            base_sha='c' * 40,
            runs=[
                # Stale success: LATER attempt time, association already
                # mutated to the current base, but created before it landed.
                good_run(
                    created_at='2026-09-07T12:00:00Z',
                    run_started_at='2026-09-07T14:00:00Z',
                    pull_requests=[
                        {
                            'number': 7,
                            'base': {'ref': 'master', 'sha': 'c' * 40},
                        }
                    ],
                ),
                # Competing failure: EARLIER attempt time, genuinely failed.
                good_run(
                    id=101,
                    conclusion='failure',
                    created_at='2026-09-07T12:00:00Z',
                    run_started_at='2026-09-07T12:00:00Z',
                ),
            ],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertFalse(result['passed'])
        self.assertIn('awaiting a fresh CI run', result['reasons'][0])
        self.assertEqual(len(result['associationCalls']), 1)

    def test_failure_dominates_stale_pending_across_workflows(self):
        # Cross-workflow ordering lock: a stale success in one workflow only
        # degrades to pending, while a fresh failure in another is a hard
        # failure. The aggregate must be failure (failure > pending > success).
        paths = [
            '.github/workflows/frontend.yml',
            '.github/workflows/backend.yml',
        ]
        result = verify(
            base_sha='c' * 40,
            workflow_paths=paths,
            runs=[
                good_run(created_at='2026-09-07T12:00:00Z'),
                good_run(
                    path=paths[1],
                    conclusion='failure',
                    created_at='2026-09-07T12:00:00Z',
                ),
            ],
        )
        self.assertEqual(result['state'], 'failure')
        self.assertFalse(result['passed'])
        self.assertIn('completed/failure', result['reasons'][0])
        self.assertIn('awaiting a fresh CI run', result['reasons'][1])

    def test_renamed_source_still_requires_its_workflow(self):
        result = verify(
            files=[
                {
                    'filename': 'docs/package.json',
                    'previous_filename': 'frontend/package.json',
                }
            ]
        )
        self.assertTrue(result['passed'])

    def test_file_and_run_history_truncation_fail_closed(self):
        self.assertIn('error', verify(changed_files=2))
        self.assertIn('error', verify(runs=[good_run()] * 1000))

    def test_api_reads_are_paginated_and_head_scoped(self):
        result = verify()
        self.assertTrue(result['passed'])
        self.assertEqual(result['requests'][1]['options']['head_sha'], 'head')
        self.assertEqual(
            result['requests'][1]['options']['event'], 'pull_request'
        )
        self.assertEqual(result['requests'][0]['options']['per_page'], 100)

    def test_all_expected_lanes_share_one_paginated_run_snapshot(self):
        result = node(f"""
const inventory = gate.loadLocalInventory(process.cwd());
const example = {json.dumps(good_run())};
let reads = 0;
const github = {{
  rest: {{pulls: {{listFiles: 'files'}}, actions: {{listWorkflowRunsForRepo: 'runs'}}}},
  paginate: async route => {{
    if (route === 'files') return [{{filename: 'frontend/package.json'}}];
    reads++;
    return inventory.inventory.workflows.map(workflow => ({{...example, path: workflow.path}}));
  }},
}};
gate.verifyExpectedWorkflows({{github, owner: 'owner', repo: 'repo', ...inventory,
  pullRequest: {{number: 7, changed_files: 1, base: {{ref: 'master'}},
    head: {{sha: 'head', ref: 'dependency', repo: {{full_name: 'owner/repo'}}}}}},
}}).then(result => console.log(JSON.stringify({{...result, reads}})));
""")
        self.assertTrue(result['passed'])
        self.assertGreater(len(result['expected']), 1)
        self.assertEqual(result['reads'], 1)

    def test_no_expected_workflow_is_not_vacuous_success(self):
        result = verify(files=[{'filename': 'README.md'}])
        self.assertFalse(result['passed'])
        self.assertEqual(result['state'], 'failure')

    def test_legacy_deletion_requires_successful_replacement(self):
        legacy = '.github/workflows/legacy-v2-api-integration-tests.yml'
        replacement = '.github/workflows/api-server-tests.yml'
        for replacement_run in [
            None,
            good_run(path=replacement, conclusion='failure'),
            good_run(path=replacement, status='in_progress', conclusion=None),
            good_run(path=replacement, head_sha='old'),
            good_run(path=replacement),
        ]:
            with self.subTest(run=replacement_run):
                result = verify(
                    workflow_paths=[legacy, replacement],
                    trigger={'paths': ['.github/**']},
                    files=[{'filename': legacy, 'status': 'removed'}],
                    runs=[good_run(path=legacy)]
                    + ([replacement_run] if replacement_run else []),
                )
                self.assertEqual(result['expected'], [replacement])
                self.assertEqual(
                    result['passed'],
                    replacement_run == good_run(path=replacement),
                )
                self.assertEqual(result['disabled'][0]['path'], legacy)

    def test_legacy_replacement_rejects_stale_success(self):
        legacy = '.github/workflows/legacy-v2-api-integration-tests.yml'
        replacement = '.github/workflows/api-server-tests.yml'
        for freshness in [
            {'fresh_after': '2026-09-07T13:00:00Z'},
            {'base_sha': 'a' * 40},
        ]:
            with self.subTest(freshness=freshness):
                result = verify(
                    workflow_paths=[legacy, replacement],
                    trigger={'paths': ['backend/**']},
                    files=[
                        {'filename': 'backend/test/v2/api/cache_api_test.go'}
                    ],
                    runs=[good_run(path=replacement)],
                    **freshness,
                )
                self.assertEqual(result['expected'], [replacement])
                self.assertFalse(result['passed'])

    def test_legacy_replacement_must_exist_and_apply_in_trusted_base(self):
        legacy = '.github/workflows/legacy-v2-api-integration-tests.yml'
        replacement = '.github/workflows/api-server-tests.yml'
        for paths, triggers in [
            ([legacy], {}),
            ([legacy, replacement], {replacement: None}),
            ([legacy, replacement], {replacement: {'paths': ['frontend/**']}}),
        ]:
            with self.subTest(paths=paths, triggers=triggers):
                result = verify(
                    workflow_paths=paths,
                    workflow_triggers=triggers,
                    trigger={'paths': ['backend/**']},
                    files=[
                        {'filename': 'backend/test/v2/api/cache_api_test.go'}
                    ],
                    runs=[
                        good_run(path=legacy),
                        good_run(id=101, path=replacement),
                    ],
                )
                self.assertEqual(result['state'], 'failure')
                self.assertIn('replacement', ' '.join(result['reasons']))

    def test_release_branches_still_require_legacy(self):
        legacy = '.github/workflows/legacy-v2-api-integration-tests.yml'
        replacement = '.github/workflows/api-server-tests.yml'
        result = verify(
            base_ref='release-2.18',
            workflow_paths=[legacy, replacement],
            trigger={'paths': ['backend/**']},
            files=[{'filename': 'backend/test/v2/api/cache_api_test.go'}],
            runs=[good_run(path=replacement)],
        )
        self.assertEqual(result['state'], 'pending')
        self.assertEqual(result['missing'], [legacy])
        self.assertCountEqual(result['expected'], [legacy, replacement])
        self.assertEqual(result['disabled'], [])

    def test_real_api_trigger_covers_legacy_workflow_deletion(self):
        result = node("""
const {inventory} = gate.loadLocalInventory(process.cwd());
const replacement = inventory.workflows.find(workflow =>
  workflow.path === '.github/workflows/api-server-tests.yml');
console.log(JSON.stringify(gate.applicable(replacement.pull_request, 'master',
  ['.github/workflows/legacy-v2-api-integration-tests.yml'])));
""")
        self.assertTrue(result)

    def test_retired_base_still_requires_successful_replacement(self):
        # Model the final trusted base, not an exemption supplied by a PR head.
        legacy = '.github/workflows/legacy-v2-api-integration-tests.yml'
        replacement = '.github/workflows/api-server-tests.yml'
        for conclusion in [None, 'failure', 'success']:
            with self.subTest(conclusion=conclusion):
                runs = [good_run(path=legacy)]
                if conclusion is not None:
                    runs.append(
                        good_run(
                            id=101, path=replacement, conclusion=conclusion
                        )
                    )
                result = verify(
                    workflow_paths=[replacement],
                    trigger={'branches': ['master'], 'paths': ['backend/**']},
                    files=[
                        {'filename': 'backend/test/v2/api/cache_api_test.go'}
                    ],
                    runs=runs,
                )
                self.assertEqual(result['expected'], [replacement])
                self.assertEqual(result['passed'], conclusion == 'success')
                self.assertEqual(
                    result['missing'],
                    [replacement] if conclusion is None else [],
                )

    def test_real_inventory_uses_current_frontend_coverage(self):
        result = node("""
const {inventory} = gate.loadLocalInventory(process.cwd());
console.log(JSON.stringify(inventory.workflows.filter(workflow =>
  workflow.pull_request !== null && gate.applicable(workflow.pull_request,
    'master', ['frontend/package.json']))
  .map(workflow => workflow.path)));
""")
        self.assertIn('.github/workflows/e2e-test-frontend.yml', result)
        self.assertIn('.github/workflows/pre-commit.yml', result)

    def test_added_or_modified_workflow_invalidates_inventory(self):
        result = node("""
const inventory = {version: 1, workflows: []};
try {
  gate.validateInventory(inventory, [{path: '.github/workflows/new.yml', header_sha256: 'new'}]);
  console.log(JSON.stringify(false));
} catch (error) { console.log(JSON.stringify(true)); }
""")
        self.assertTrue(result)

    def test_dependency_bumps_do_not_change_trigger_header(self):
        result = node("""
const before = 'name: CI\\non:\\n  pull_request:\\npermissions:\\n  contents: read\\njobs:\\n  test:\\n    uses: actions/example@v1\\n';
const after = before.replace('@v1', '@v2');
console.log(JSON.stringify(gate.triggerHeader(before) === gate.triggerHeader(after)));
""")
        self.assertTrue(result)

    def test_trigger_changes_change_header_and_key_order_is_supported(self):
        result = node("""
const before = 'on:\\n  pull_request:\\n    paths: [frontend/**]\\npermissions: {}\\nname: Frontend\\njobs: {}\\n';
const after = before.replace('frontend/**', 'backend/**');
const header = gate.triggerHeader(before);
console.log(JSON.stringify([header.includes('name: Frontend'),
  header !== gate.triggerHeader(after), !header.includes('permissions')]));
""")
        self.assertEqual(result, [True, True, True])


if __name__ == '__main__':
    unittest.main()
