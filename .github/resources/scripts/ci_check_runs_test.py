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
"""Exercise the production check assessor with paginated API fixtures."""

import json
from pathlib import Path
import subprocess
import unittest

ROOT = Path(__file__).resolve().parents[3]
MODULE = ROOT / '.github/resources/scripts/ci_check_runs.js'


def check(**overrides):
    return {
        'id': 100,
        'name': 'Tests',
        'app': {
            'id': 15368,
            'slug': 'github-actions'
        },
        'check_suite': {
            'id': 700
        },
        'head_sha': 'head',
        'status': 'completed',
        'conclusion': 'success',
        **overrides,
    }


def external(**overrides):
    return check(**{
        'name': 'DCO',
        'app': {
            'id': 1861,
            'slug': 'dco'
        },
        **overrides
    })


def run(**overrides):
    return {
        'id': 10,
        'workflow_id': 7,
        'path': '.github/workflows/tests.yml',
        'check_suite_id': 700,
        'head_sha': 'head',
        'event': 'pull_request',
        'run_attempt': 1,
        'created_at': '2026-09-27T10:00:00Z',
        'run_started_at': '2026-09-27T10:00:00Z',
        'status': 'completed',
        'conclusion': 'success',
        **overrides,
    }


def job(**overrides):
    return {
        'id':
            1000,
        'run_id':
            10,
        'run_attempt':
            2,
        'check_run_url':
            'https://api.github.com/repos/owner/repo/check-runs/101',
        'name':
            'Tests',
        'status':
            'completed',
        'conclusion':
            'success',
        **overrides,
    }


def verify(checks=None, runs=None, jobs=None, errors=None, responses=None):
    fixture = {
        'check_runs': [check()] if checks is None else checks,
        'workflow_runs': [run()] if runs is None else runs,
        'jobs': [job()] if jobs is None else jobs,
        'errors': errors or {},
        'responses': responses or {},
    }
    script = f'''
const {{verifyCheckRuns}} = require({json.dumps(str(MODULE))});
const fixture = {json.dumps(fixture)};
const requests = [];
function route(field) {{
  return async options => {{
    requests.push({{field, options}});
    if (fixture.errors[field]) throw new Error(fixture.errors[field]);
    if (fixture.responses[field]) return fixture.responses[field][options.page - 1];
    const values = fixture[field];
    return {{data: {{total_count: values.length,
      [field]: values.slice((options.page - 1) * 100, options.page * 100)}}}};
  }};
}}
const github = {{rest: {{
  checks: {{listForRef: route('check_runs')}},
  actions: {{listWorkflowRunsForRepo: route('workflow_runs'),
    listJobsForWorkflowRunAttempt: route('jobs')}},
}}}};
verifyCheckRuns({{github, owner: 'owner', repo: 'repo', sha: 'head'}})
  .then(result => console.log(JSON.stringify({{...result, requests}})));
'''
    result = subprocess.run(['node'],
                            input=script,
                            capture_output=True,
                            text=True,
                            check=True,
                            cwd=ROOT)
    return json.loads(result.stdout)


class CheckRunsTest(unittest.TestCase):

    def test_success_and_individual_skipped_or_neutral(self):
        for conclusion in ['success', 'skipped', 'neutral']:
            with self.subTest(conclusion=conclusion):
                result = verify(checks=[check(conclusion=conclusion)])
                self.assertEqual(result['state'], 'success')
                self.assertEqual(result['reasons'], [])

    def test_pending_states_are_distinct_from_failure(self):
        for status in [
                'queued', 'in_progress', 'waiting', 'pending', 'requested'
        ]:
            with self.subTest(status=status):
                result = verify(
                    checks=[external(status=status, conclusion=None)], runs=[])
                self.assertEqual(result['state'], 'pending')

    def test_terminal_failure_beats_pending_in_either_order(self):
        for conclusion in [
                'failure', 'cancelled', 'timed_out', 'action_required', 'stale'
        ]:
            failed = external(conclusion=conclusion)
            pending = check(id=101, status='in_progress', conclusion=None)
            for checks in [[failed, pending], [pending, failed]]:
                with self.subTest(conclusion=conclusion, checks=checks):
                    result = verify(checks=checks)
                    self.assertEqual(result['state'], 'failure')
                    self.assertIn(conclusion, result['reasons'][0])

    def test_all_pages_and_all_workflow_events_are_requested(self):
        result = verify(
            checks=[external(id=i + 1, name=f'DCO {i}') for i in range(101)],
            runs=[
                run(id=i + 1, workflow_id=i + 1, check_suite_id=i + 1)
                for i in range(101)
            ])
        self.assertEqual(result['state'], 'success')
        for field in ['check_runs', 'workflow_runs']:
            requests = [
                r['options'] for r in result['requests'] if r['field'] == field
            ]
            self.assertEqual([r['page'] for r in requests], [1, 2])
            self.assertTrue(all(r['per_page'] == 100 for r in requests))
            self.assertTrue(all('event' not in r for r in requests))
        self.assertEqual(result['requests'][0]['options']['filter'], 'all')
        self.assertEqual(result['requests'][0]['options']['ref'], 'head')

    def test_newest_check_per_name_and_app(self):
        result = verify(
            checks=[external(conclusion='failure'),
                    external(id=101)], runs=[])
        self.assertEqual(result['state'], 'success')
        result = verify(
            checks=[
                external(),
                external(id=101, status='queued', conclusion=None)
            ],
            runs=[])
        self.assertEqual(result['state'], 'pending')
        other_app = external(id=101, app={'id': 2000, 'slug': 'another-app'})
        result = verify(
            checks=[external(conclusion='failure'), other_app], runs=[])
        self.assertEqual(result['state'], 'failure')

    def test_newest_workflow_replaces_superseded_checks_by_suite(self):
        result = verify(
            checks=[
                check(name='Old matrix', conclusion='cancelled'),
                check(id=101, name='New matrix', check_suite={'id': 701})
            ],
            runs=[run(conclusion='cancelled'),
                  run(id=11, check_suite_id=701)])
        self.assertEqual(result['state'], 'success')

    def test_different_workflow_ids_cannot_supersede_each_other(self):
        result = verify(
            checks=[
                check(conclusion='failure'),
                check(id=101, check_suite={'id': 701})
            ],
            runs=[
                run(conclusion='failure'),
                run(id=11, workflow_id=8, check_suite_id=701)
            ])
        self.assertEqual(result['state'], 'failure')

    def test_older_workflow_rerun_is_newer_than_later_run_id(self):
        result = verify(
            checks=[check(), check(id=101, check_suite={'id': 701})],
            runs=[
                run(run_attempt=2,
                    run_started_at='2026-09-27T12:00:00Z',
                    status='queued',
                    conclusion=None),
                run(id=11,
                    check_suite_id=701,
                    created_at='2026-09-27T11:00:00Z',
                    run_started_at='2026-09-27T11:00:00Z')
            ],
            jobs=[])
        self.assertEqual(result['state'], 'pending')
        self.assertEqual([
            r['options']['run_id']
            for r in result['requests']
            if r['field'] == 'jobs'
        ], [10])

    def test_queued_rerun_does_not_reuse_old_success_or_old_failure(self):
        for conclusion in ['success', 'failure']:
            with self.subTest(conclusion=conclusion):
                result = verify(
                    checks=[check(conclusion=conclusion)],
                    runs=[run(run_attempt=2, status='queued', conclusion=None)],
                    jobs=[])
                self.assertEqual(result['state'], 'pending')

    def test_failed_jobs_only_rerun_preserves_unchanged_successful_jobs(self):
        result = verify(
            checks=[
                check(id=99, name='Unchanged'),
                check(conclusion='failure'),
                check(id=101)
            ],
            runs=[run(run_attempt=2)])
        self.assertEqual(result['state'], 'success')
        request = [
            r['options'] for r in result['requests'] if r['field'] == 'jobs'
        ][0]
        self.assertEqual(request['attempt_number'], 2)

    def test_current_attempt_job_failure_beats_active_workflow(self):
        result = verify(
            checks=[check(), check(id=101, conclusion='failure')],
            runs=[run(run_attempt=2, status='in_progress', conclusion=None)],
            jobs=[job(conclusion='failure')])
        self.assertEqual(result['state'], 'failure')

    def test_completed_rerun_waits_for_missing_or_disagreeing_check_evidence(
            self):
        for checks in [[check()],
                       [check(id=101, status='in_progress', conclusion=None)]]:
            with self.subTest(checks=checks):
                result = verify(checks=checks, runs=[run(run_attempt=2)])
                self.assertEqual(result['state'], 'pending')
                self.assertTrue(
                    any('not synchronized' in r for r in result['reasons']))

    def test_failed_workflow_cannot_be_hidden_by_passed_checks(self):
        for conclusion in [
                'failure', 'cancelled', 'timed_out', 'startup_failure'
        ]:
            with self.subTest(conclusion=conclusion):
                result = verify(runs=[run(conclusion=conclusion)])
                self.assertEqual(result['state'], 'failure')

    def test_new_run_without_registered_checks_is_pending(self):
        result = verify(runs=[
            run(),
            run(id=11, check_suite_id=701, status='queued', conclusion=None)
        ])
        self.assertEqual(result['state'], 'pending')

    def test_workflow_registration_gap_cannot_reuse_external_success(self):
        for status in ['queued', 'in_progress']:
            with self.subTest(status=status):
                result = verify(
                    checks=[external()],
                    runs=[
                        run(status=status,
                            conclusion=None,
                            run_started_at=None
                            if status == 'queued' else '2026-09-27T10:00:00Z',
                            event='workflow_dispatch')
                    ])
                self.assertEqual(result['state'], 'pending')
                self.assertTrue(
                    any(status in reason for reason in result['reasons']))

    def test_failed_workflow_without_registered_checks_is_failure(self):
        result = verify(checks=[external()], runs=[run(conclusion='failure')])
        self.assertEqual(result['state'], 'failure')

    def test_ci_check_self_run_is_excluded_by_authoritative_path(self):
        for status, conclusion in [('queued', None), ('in_progress', None),
                                   ('completed', 'failure')]:
            with self.subTest(status=status):
                result = verify(
                    checks=[external()],
                    runs=[
                        run(path='.github/workflows/ci-checks.yml',
                            status=status,
                            conclusion=conclusion)
                    ])
                self.assertEqual(result['state'], 'success')
        result = verify(
            checks=[
                external(),
                check(
                    id=101,
                    name='recovery_candidates',
                    status='in_progress',
                    conclusion=None)
            ],
            runs=[
                run(path='.github/workflows/ci-checks.yml',
                    status='in_progress',
                    conclusion=None)
            ])
        self.assertEqual(result['state'], 'success')

    def test_self_display_name_or_excluded_job_cannot_exempt_other_workflow(
            self):
        for path in [
                '.github/workflows/tests.yml',
                '.github/workflows/other-ci-checks.yml',
                '.github/workflows/ci-checks.yml.extra'
        ]:
            with self.subTest(path=path):
                result = verify(
                    checks=[external(),
                            check(id=101, name='Prepare')],
                    runs=[
                        run(path=path,
                            name='CI Check',
                            status='queued',
                            conclusion=None)
                    ])
                self.assertEqual(result['state'], 'pending')

    def test_superseded_workflow_without_registered_checks_does_not_block(self):
        for status, conclusion in [('queued', None), ('completed', 'failure')]:
            with self.subTest(status=status):
                result = verify(
                    checks=[external()],
                    runs=[
                        run(status=status, conclusion=conclusion),
                        run(id=11, check_suite_id=701)
                    ])
                self.assertEqual(result['state'], 'success')

    def test_exclusions_and_empty_inventory_never_make_the_gate_green(self):
        names = [
            'check_ci_status', 'check_ci_status (17, sha)', 'Cleanup artifacts',
            'Upload results', 'Agent', 'Prepare'
        ]
        excluded = [
            check(id=i + 1, name=name, conclusion='failure')
            for i, name in enumerate(names)
        ]
        self.assertEqual(verify(checks=excluded, runs=[])['state'], 'pending')
        self.assertEqual(
            verify(checks=excluded + [external()], runs=[])['state'], 'success')
        self.assertEqual(verify(checks=[], runs=[])['state'], 'pending')
        for name in [
                'check_ci_status_bad', 'Prepare build', 'Agent (extra)',
                'Upload results extra'
        ]:
            with self.subTest(name=name):
                result = verify(
                    checks=[external(name=name, conclusion='failure')], runs=[])
                self.assertEqual(result['state'], 'failure')

    def test_unknown_or_inconsistent_state_fails_closed(self):
        for fields in [{
                'status': 'unknown'
        }, {
                'conclusion': None
        }, {
                'conclusion': 'unknown'
        }, {
                'status': 'in_progress',
                'conclusion': 'success'
        }]:
            with self.subTest(fields=fields):
                self.assertEqual(
                    verify(checks=[external(**fields)], runs=[])['state'],
                    'failure')

    def test_missing_check_identity_fails_closed(self):
        for field in ['head_sha', 'app', 'check_suite', 'name']:
            value = external()
            del value[field]
            with self.subTest(field=field):
                self.assertEqual(
                    verify(checks=[value], runs=[])['state'], 'failure')
        self.assertEqual(verify(checks=[check()], runs=[])['state'], 'failure')

    def test_missing_workflow_metadata_fails_closed(self):
        for field in [
                'head_sha', 'workflow_id', 'path', 'check_suite_id',
                'run_attempt', 'created_at', 'run_started_at', 'event'
        ]:
            value = run()
            del value[field]
            with self.subTest(field=field):
                self.assertEqual(verify(runs=[value])['state'], 'failure')

    def test_invalid_workflow_path_cannot_exempt_a_run(self):
        for path in [None, '', 7, ' .github/workflows/ci-checks.yml']:
            with self.subTest(path=path):
                result = verify(checks=[external()], runs=[run(path=path)])
                self.assertEqual(result['state'], 'failure')

    def test_only_unstarted_first_attempt_can_have_null_start_time(self):
        for status in ['queued', 'waiting', 'requested', 'pending']:
            with self.subTest(status=status):
                result = verify(
                    checks=[external()],
                    runs=[
                        run(status=status, conclusion=None, run_started_at=None)
                    ])
                self.assertEqual(result['state'], 'pending')
        for fields in [
                dict(status='in_progress', conclusion=None),
                dict(status='completed', conclusion='success'),
                dict(status='queued', conclusion=None, run_attempt=2)
        ]:
            with self.subTest(fields=fields):
                result = verify(
                    checks=[external()],
                    runs=[run(run_started_at=None, **fields)])
                self.assertEqual(result['state'], 'failure')

    def test_api_errors_fail_closed(self):
        for field in ['check_runs', 'workflow_runs', 'jobs']:
            with self.subTest(field=field):
                result = verify(
                    checks=[check(id=101)],
                    runs=[run(run_attempt=2)],
                    errors={field: 'unavailable'})
                self.assertEqual(result['state'], 'failure')
                self.assertTrue(
                    any('Cannot verify' in r for r in result['reasons']))

    def test_incomplete_or_truncated_pages_fail_closed(self):
        for response in [{
                'data': {
                    'total_count': 2,
                    'check_runs': [check()]
                }
        }, {
                'data': {
                    'total_count': 1000,
                    'check_runs': []
                }
        }, {
                'data': {
                    'total_count': 0,
                    'check_runs': [check()]
                }
        }, {
                'data': {
                    'check_runs': [check()]
                }
        }, {
                'data': {
                    'total_count': 1,
                    'check_runs': None
                }
        }, {
                'data': {
                    'total_count': 2,
                    'check_runs': [check(), check()]
                }
        }]:
            with self.subTest(response=response):
                result = verify(responses={'check_runs': [response]})
                self.assertEqual(result['state'], 'failure')

    def test_changed_total_between_pages_fails_closed(self):
        pages = [{
            'data': {
                'total_count':
                    101,
                'check_runs': [
                    external(id=i + 1, name=f'DCO {i}') for i in range(100)
                ]
            }
        }, {
            'data': {
                'total_count': 102,
                'check_runs': [external(id=101)]
            }
        }]
        self.assertEqual(
            verify(responses={'check_runs': pages})['state'], 'failure')

    def test_current_attempt_job_metadata_must_match(self):
        for fields in [{
                'run_id': 11
        }, {
                'run_attempt': 1
        }, {
                'name': None
        }, {
                'check_run_url':
                    'https://api.github.com/repos/other/repo/check-runs/101'
        }, {
                'check_run_url': 'invalid'
        }]:
            with self.subTest(fields=fields):
                result = verify(
                    checks=[check(id=101)],
                    runs=[run(run_attempt=2)],
                    jobs=[job(**fields)])
                self.assertEqual(result['state'], 'failure')

    def test_queued_job_without_published_check_is_pending(self):
        result = verify(
            runs=[run(run_attempt=2, status='queued', conclusion=None)],
            jobs=[job(status='queued', conclusion=None, check_run_url=None)])
        self.assertEqual(result['state'], 'pending')
        result = verify(
            runs=[run(run_attempt=2)], jobs=[job(check_run_url=None)])
        self.assertEqual(result['state'], 'failure')

    def test_current_attempt_jobs_are_paginated(self):
        checks = [check(id=i + 1, name=f'Job {i}') for i in range(101)]
        jobs = [
            job(id=i + 1,
                name=f'Job {i}',
                check_run_url=f'https://api.github.com/repos/owner/repo/check-runs/{i + 1}'
               ) for i in range(101)
        ]
        result = verify(checks=checks, jobs=jobs, runs=[run(run_attempt=2)])
        self.assertEqual(result['state'], 'success')
        self.assertEqual([
            r['options']['page']
            for r in result['requests']
            if r['field'] == 'jobs'
        ], [1, 2])


if __name__ == '__main__':
    unittest.main()
