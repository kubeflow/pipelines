"""Bound PR Gate infrastructure retries to current, approved PR evidence."""

import copy
import json
import subprocess
import unittest
from pathlib import Path

MODULE = Path(__file__).with_name('merge_queue_pr_gate_recovery.js')
HEAD = 'a' * 40
PR = {
    'number': 7,
    'head': {
        'sha': HEAD,
        'ref': 'feature',
        'repo': {'full_name': 'owner/repo', 'owner': {'login': 'owner'}},
    },
}
RUN = {
    'id': 100,
    'run_attempt': 1,
    'event': 'pull_request_target',
    'head_sha': HEAD,
    'head_branch': 'feature',
    'path': '.github/workflows/pr-gate.yml',
    'repository': {'full_name': 'owner/repo'},
    'head_repository': {'full_name': 'owner/repo'},
    'status': 'completed',
    'conclusion': 'failure',
    'created_at': '2026-11-17T00:00:00Z',
    'run_started_at': '2026-11-17T00:00:01Z',
    'updated_at': '2026-11-17T00:01:00Z',
}
JOB = {
    'name': 'check-pr-author',
    'status': 'completed',
    'conclusion': 'failure',
    'steps': [
        {'name': 'Checkout trusted membership helper', 'conclusion': 'success'},
        {
            'name': 'Look up PR author in Kubeflow internal-acls',
            'conclusion': 'failure',
        },
        {
            'name': 'Check linked issue admission for external contributors',
            'conclusion': 'skipped',
        },
        {
            'name': 'Approve pending workflow runs after admission',
            'conclusion': 'skipped',
        },
        {
            'name': 'Post Checkout trusted membership helper',
            'conclusion': 'success',
        },
    ],
}


def exercise(**overrides: object) -> dict[str, object]:
    fixture = {
        'pr': PR,
        'runs': [RUN],
        'jobs': [JOB],
        'statuses': [],
        'headPRs': [PR],
        'now': '2026-11-17T00:20:00Z',
        **overrides,
    }
    script = r"""
const fs = require('node:fs');
const f = JSON.parse(fs.readFileSync(0, 'utf8'));
const calls = [], reruns = [], writes = [];
let validation = 0, currentIndex = 0, runsIndex = 0;
const github = {rest:{
 actions:{listWorkflowRunsForRepo:'runs',listJobsForWorkflowRunAttempt:'jobs',
 getWorkflowRun:async o=>{calls.push(['current',o]);return {data:f.currents ? f.currents[Math.min(currentIndex++,f.currents.length-1)] : f.current||f.runs.find(r=>r.id===o.run_id)};},
 reRunWorkflowFailedJobs:async o=>{calls.push(['rerun',o]);if(f.rerunError)throw Error('rerun failed');reruns.push(o.run_id);}},
 repos:{listCommitStatusesForRef:'statuses',createCommitStatus:async o=>{calls.push(['reserve',o]);if(f.reserveError)throw Error('reservation failed');writes.push(o);f.statuses.unshift(o);}},
 pulls:{list:'prs'}},paginate:async(route,o)=>{calls.push([route,o]);if(f.apiError===route)throw Error('API unavailable');return {runs:route==='runs' && f.runLists ? f.runLists[Math.min(runsIndex++,f.runLists.length-1)] : f.runs,jobs:f.jobs.map(j=>({run_id:o.run_id,head_sha:f.pr.head.sha,...j})),statuses:f.statuses,prs:f.headPRs}[route];}};
(async()=>{let error,result=false;
 try{if(fs.existsSync(process.argv[1]))result=await require(process.argv[1]).recoverPRGate({github,
 context:{repo:{owner:'owner',repo:'repo'}},pr:f.pr,core:{info:()=>{}},now:()=>Date.parse(f.now),
 revalidate:async()=>f.validations ? f.validations[Math.min(validation++,f.validations.length-1)]:true});}
 catch(e){error=e.message;}
 console.log(JSON.stringify({result,error,calls,reruns,writes}));})();
"""
    result = subprocess.run(
        ['node', '-e', script, str(MODULE)],
        input=json.dumps(fixture),
        text=True,
        capture_output=True,
        check=True,
    )
    return json.loads(result.stdout)


class PRGateRecoveryTest(unittest.TestCase):
    def test_membership_failure_retries_once_after_cooldown(self) -> None:
        result = exercise()
        self.assertEqual(result['reruns'], [100])
        self.assertTrue(result['result'])
        self.assertLess(
            [x[0] for x in result['calls']].index('reserve'),
            [x[0] for x in result['calls']].index('rerun'),
        )
        again = exercise(statuses=result['writes'])
        self.assertEqual(again['reruns'], [])

    def test_second_attempt_and_active_runs_are_preserved(self) -> None:
        for updates in (
            {'run_attempt': 2},
            {'status': 'in_progress', 'conclusion': None},
            {'conclusion': 'success'},
            {'updated_at': '2026-11-17T00:19:00Z'},
            {'updated_at': '2026-11-18T00:00:00Z'},
            {'updated_at': None},
        ):
            run = {**RUN, **updates}
            self.assertEqual(exercise(runs=[run])['reruns'], [])

    def test_wrong_identity_is_not_retried(self) -> None:
        for updates in (
            {'path': '.github/workflows/untrusted.yml'},
            {'event': 'pull_request'},
            {'head_sha': 'b' * 40},
            {'head_branch': 'other'},
            {'repository': {'full_name': 'foreign/repo'}},
            {'head_repository': {'full_name': 'foreign/repo'}},
        ):
            self.assertEqual(exercise(runs=[{**RUN, **updates}])['reruns'], [])

    def test_admission_or_other_failure_is_not_infrastructure(self) -> None:
        for name in (
            'Check linked issue admission for external contributors',
            'Checkout trusted membership helper',
            'Approve pending workflow runs after admission',
        ):
            job = copy.deepcopy(JOB)
            for step in job['steps']:
                step['conclusion'] = (
                    'failure' if step['name'] == name else 'success'
                )
            self.assertEqual(exercise(jobs=[job])['reruns'], [])
        self.assertEqual(exercise(jobs=[])['reruns'], [])
        self.assertEqual(exercise(jobs=[JOB, JOB])['reruns'], [])

    def test_head_policy_attempt_changes_and_ambiguous_pr_preserve_runs(
        self,
    ) -> None:
        for validations in ([False], [True, False]):
            self.assertEqual(exercise(validations=validations)['reruns'], [])
        for updates in (
            {'run_attempt': 2},
            {'head_sha': 'b' * 40},
            {'conclusion': 'success'},
        ):
            self.assertEqual(exercise(current={**RUN, **updates})['reruns'], [])
        self.assertEqual(exercise(headPRs=[PR, PR])['reruns'], [])
        self.assertEqual(exercise(headPRs=[])['reruns'], [])

    def test_reservation_failure_prevents_rerun_and_response_loss_is_consumed(
        self,
    ):
        self.assertEqual(exercise(reserveError=True)['reruns'], [])
        result = exercise(rerunError=True)
        self.assertEqual(len(result['writes']), 1)
        self.assertEqual(exercise(statuses=result['writes'])['reruns'], [])

    def test_only_authoritative_run_is_retried_not_historical_failures(
        self,
    ) -> None:
        runs = [
            {**RUN, 'id': n, 'created_at': f'2026-11-17T00:00:0{n - 100}Z'}
            for n in range(100, 103)
        ]
        result = exercise(runs=runs)
        self.assertEqual(result['reruns'], [102])
        self.assertEqual(
            [o['run_id'] for route, o in result['calls'] if route == 'jobs'],
            [102],
        )
        # A consumed authoritative attempt must not fall back to older failures.
        self.assertEqual(
            exercise(runs=runs, statuses=result['writes'])['reruns'], []
        )
        runs[-1]['conclusion'] = 'success'
        self.assertEqual(exercise(runs=runs)['reruns'], [])

    def test_newer_attempt_of_older_run_supersedes_newer_created_failure(
        self,
    ) -> None:
        stale_failure = {**RUN, 'id': 101, 'created_at': '2026-11-17T00:00:05Z'}
        for updates in (
            {'status': 'in_progress', 'conclusion': None},
            {'status': 'completed', 'conclusion': 'success'},
            {'status': 'completed', 'conclusion': 'failure'},
        ):
            with self.subTest(updates=updates):
                rerun = {
                    **RUN,
                    'run_attempt': 2,
                    'run_started_at': '2026-11-17T00:00:10Z',
                    **updates,
                }
                result = exercise(runs=[rerun, stale_failure])
                self.assertEqual(result['reruns'], [])
                self.assertEqual(result['writes'], [])

    def test_authority_changes_at_write_boundaries_stop_recovery(self) -> None:
        for replacement in (
            {**RUN, 'id': 101, 'created_at': '2026-11-17T00:00:05Z'},
            {
                **RUN,
                'id': 99,
                'run_attempt': 2,
                'run_started_at': '2026-11-17T00:00:10Z',
                'status': 'in_progress',
                'conclusion': None,
            },
        ):
            for boundary in (1, 2):
                with self.subTest(replacement=replacement, boundary=boundary):
                    result = exercise(
                        runLists=[[RUN]] * boundary + [[RUN, replacement]]
                    )
                    self.assertEqual(result['reruns'], [])
                    self.assertEqual(len(result['writes']), boundary - 1)

    def test_attempt_time_ties_use_larger_run_id(self) -> None:
        active = {**RUN, 'id': 101, 'status': 'in_progress', 'conclusion': None}
        self.assertEqual(exercise(runs=[active, RUN])['reruns'], [])
        self.assertEqual(exercise(runs=[RUN, active])['reruns'], [])

    def test_retry_budget_caps_distinct_authoritative_runs_per_head(
        self,
    ) -> None:
        first = exercise()
        second_run = {**RUN, 'id': 101, 'created_at': '2026-11-17T00:00:02Z'}
        second = exercise(runs=[RUN, second_run], statuses=first['writes'])
        self.assertEqual(second['reruns'], [101])
        third_run = {**RUN, 'id': 102, 'created_at': '2026-11-17T00:00:03Z'}
        third = exercise(
            runs=[RUN, second_run, third_run],
            statuses=first['writes'] + second['writes'],
        )
        self.assertEqual(third['reruns'], [])

    def test_new_attempt_after_reservation_is_preserved(self) -> None:
        result = exercise(currents=[RUN, {**RUN, 'run_attempt': 2}])
        self.assertEqual(result['reruns'], [])
        self.assertEqual(len(result['writes']), 1)

    def test_foreign_job_and_incomplete_discovery_fail_closed(self) -> None:
        self.assertEqual(exercise(jobs=[{**JOB, 'run_id': 999}])['reruns'], [])
        self.assertEqual(
            exercise(jobs=[{**JOB, 'head_sha': 'b' * 40}])['reruns'], []
        )
        self.assertEqual(exercise(runs=[RUN] * 1000)['reruns'], [])
        self.assertEqual(exercise(statuses=[{}] * 1000)['reruns'], [])

    def test_valid_fork_identity_is_supported(self) -> None:
        pr = copy.deepcopy(PR)
        pr['head']['repo'] = {
            'full_name': 'fork/repo',
            'owner': {'login': 'fork'},
        }
        run = {**RUN, 'head_repository': {'full_name': 'fork/repo'}}
        self.assertEqual(
            exercise(pr=pr, runs=[run], headPRs=[pr])['reruns'], [100]
        )

    def test_api_failure_never_reruns(self) -> None:
        for route in ('runs', 'jobs', 'statuses', 'prs'):
            self.assertEqual(exercise(apiError=route)['reruns'], [])


if __name__ == '__main__':
    unittest.main()
