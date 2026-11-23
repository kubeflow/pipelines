"""Regression tests for landed release CI evidence."""

import copy
import hashlib
import json
import unittest
from unittest import mock

from kfpr import landed_ci as ci

SHA = 'a' * 40
PARENT = 'b' * 40
MERGE = 'c' * 40
TREE = 'd' * 40
HEAD = f'ci/release-candidate/{SHA}'
BASE = f'ci/release-replay/{SHA}'


def side(ref, sha):
    return dict(ref=ref, sha=sha, repo=dict(full_name=ci.REPO, id=123))


def run(path='e2e-test.yml'):
    return dict(
        id=101,
        run_attempt=1,
        path=ci.PREFIX + path,
        event='pull_request',
        head_branch=HEAD,
        head_sha=SHA,
        created_at='2026-01-01T00:00:00Z',
        status='completed',
        conclusion='success',
        repository={'full_name': ci.REPO},
        head_repository={'full_name': ci.REPO},
        workflow_id=17,
        check_suite_id=23,
        pull_requests=[
            dict(number=9, base=side(BASE, PARENT), head=side(HEAD, SHA))
        ],
        referenced_workflows=[
            dict(
                path=ci.REPO + '/' + ci.PREFIX + 'image-builds.yml@' + MERGE,
                sha=MERGE,
                ref='refs/pull/9/merge',
            )
        ],
    )


def job(name, steps, conclusion='success'):
    return dict(
        id=12,
        run_id=101,
        run_attempt=1,
        head_sha=SHA,
        head_branch=HEAD,
        name=name,
        status='completed',
        conclusion=conclusion,
        steps=[dict(name=s, conclusion='success') for s in steps],
    )


class LandedCITest(unittest.TestCase):
    def setUp(self):
        self.replay = dict(
            number=9,
            merged=False,
            state='open',
            changed_files=1,
            base=side(BASE, PARENT),
            head=side(HEAD, SHA),
            merge_commit_sha=MERGE,
        )
        self.responses = {
            'pulls/8': dict(
                number=8,
                merged=True,
                state='closed',
                base=side(ci.BRANCH, PARENT),
                merge_commit_sha=SHA,
            ),
            'pulls/9': self.replay,
            'git/ref/heads/' + ci.BRANCH: dict(
                ref='refs/heads/' + ci.BRANCH,
                object=dict(sha=SHA, type='commit'),
            ),
            'git/commits/' + SHA: dict(
                sha=SHA, tree=dict(sha=TREE), parents=[dict(sha=PARENT)]
            ),
            'git/commits/' + MERGE: dict(
                sha=MERGE,
                tree=dict(sha=TREE),
                parents=[dict(sha=PARENT), dict(sha=SHA)],
            ),
        }

    def test_binds_landed_sha_parent_and_replay_merge_tree(self):
        with mock.patch.object(
            ci, 'api', side_effect=self.responses.__getitem__
        ):
            self.assertEqual(ci.binding(8, SHA, 9), (PARENT, self.replay))
            with self.subTest('changed tree'):
                self.responses['git/commits/' + MERGE]['tree']['sha'] = 'e' * 40
                with self.assertRaisesRegex(RuntimeError, 'landed tree'):
                    ci.binding(8, SHA, 9)
            with self.subTest('wrong replay branch'):
                self.replay['head']['ref'] = ci.BRANCH
                with self.assertRaisesRegex(RuntimeError, 'replay head'):
                    ci.binding(8, SHA, 9)

    def test_release_branch_advance_or_alternate_repository_is_rejected(self):
        with mock.patch.object(
            ci, 'api', side_effect=self.responses.__getitem__
        ):
            self.responses['git/ref/heads/' + ci.BRANCH]['object']['sha'] = (
                MERGE
            )
            with self.assertRaisesRegex(RuntimeError, 'Release branch moved'):
                ci.binding(8, SHA, 9)
            self.responses['pulls/8']['base']['repo']['full_name'] = (
                'other/pipelines'
            )
            with self.assertRaisesRegex(RuntimeError, 'Release PR'):
                ci.binding(8, SHA, 9)

    def test_manual_e2e_cannot_replace_an_automatic_replay(self):
        manual = dict(run(), event='workflow_dispatch')
        with self.assertRaisesRegex(RuntimeError, 'Missing'):
            ci.select_run([manual], manual['path'], 'pull_request', HEAD, SHA)

    def test_older_green_does_not_override_newest_rerun_or_fork(self):
        old = run()
        newer = dict(
            old,
            id=103,
            conclusion=None,
            status='in_progress',
            created_at='2026-01-02T00:00:00Z',
        )
        with self.assertRaisesRegex(RuntimeError, 'in_progress'):
            ci.select_run([newer, old], old['path'], 'pull_request', HEAD, SHA)
        newer.update(head_repository={'full_name': 'fork/other'})
        self.assertEqual(
            ci.select_run([newer, old], old['path'], 'pull_request', HEAD, SHA),
            old,
        )

    def test_workflow_ref_and_snapshot_must_match_replay(self):
        ci.verify_replay_run(run(), self.replay)
        for key in ('ref', 'sha'):
            foreign = run()
            foreign['referenced_workflows'][0][key] = 'wrong'
            with self.assertRaisesRegex(RuntimeError, 'test merge'):
                ci.verify_replay_run(foreign, self.replay)
        wrong_pr = run()
        wrong_pr['pull_requests'][0]['base']['sha'] = SHA
        with self.assertRaisesRegex(RuntimeError, 'different replay'):
            ci.verify_replay_run(wrong_pr, self.replay)

    def test_postgres_requires_both_majors_and_the_real_upgrade_step(self):
        path = ci.PREFIX + 'postgresql-image-upgrade.yml'
        jobs = [
            job(name, steps)
            for name, steps in ci.TEST_STEPS[path.split('/')[-1]].items()
        ]
        ci.verify_jobs(path, run(), jobs)
        with self.assertRaisesRegex(RuntimeError, 'missing executed'):
            ci.verify_jobs(path, run(), jobs[:1])
        skipped = copy.deepcopy(jobs)
        skipped[1]['steps'][0]['conclusion'] = 'skipped'
        with self.assertRaisesRegex(RuntimeError, 'missing test steps'):
            ci.verify_jobs(path, run(), skipped)
        skipped[1]['run_attempt'] = 2
        with self.assertRaisesRegex(RuntimeError, 'Unexpected job'):
            ci.verify_jobs(path, run(), skipped)

    def test_build_only_or_skipped_suite_cannot_satisfy_replay(self):
        path = ci.PREFIX + 'e2e-test.yml'
        with self.assertRaisesRegex(RuntimeError, 'missing executed'):
            ci.verify_jobs(path, run(), [job('build / image', ['Build'])])
        with self.assertRaisesRegex(RuntimeError, 'did not pass'):
            ci.verify_jobs(
                path, run(), [job('End to End E2EFailure Tests', [], 'skipped')]
            )
        with self.assertRaisesRegex(RuntimeError, 'missing test steps'):
            ci.verify_jobs(
                path, run(), [job('End to End E2EFailure Tests', ['Build'])]
            )

    def test_exact_report_contains_current_run_attempt(self):
        manifests = 'kubeflow-pipelines-manifests.yml'
        current = run(manifests)
        current['run_attempt'] = 2
        actual = job('kubeflow-pipelines-manifests', ['Run tests'])
        actual['run_attempt'] = 2

        self.responses['actions/runs/101/attempts/1'] = dict(
            current, run_attempt=1
        )

        def page(_endpoint, field):
            if _endpoint == 'actions/runs/101/attempts/1/jobs':
                return [dict(actual, id=11, run_attempt=1)]
            return {
                'workflow_runs': [current],
                'workflows': [{'path': current['path'], 'id': 17}],
                'jobs': [actual],
            }[field]

        with (
            mock.patch.object(
                ci, 'api', side_effect=self.responses.__getitem__
            ),
            mock.patch.object(
                ci, 'expected_workflows', return_value=[current['path']]
            ),
            mock.patch.object(ci, 'paged', side_effect=page),
            mock.patch.object(ci.subprocess, 'run'),
            mock.patch.object(ci, 'verify_aggregate'),
        ):
            branch, evidence = ci.collect_evidence(8, SHA, 9)
        self.assertEqual(branch, ci.BRANCH)
        self.assertEqual(
            evidence,
            [
                dict(
                    path=current['path'],
                    workflow_id=17,
                    event='pull_request',
                    run_id=101,
                    run_attempt=2,
                    tested_sha=MERGE,
                )
            ],
        )

    def test_aggregate_rejects_generic_or_unreviewed_green(self):
        for status in (
            {'context': 'ci-passed', 'state': 'pending'},
            {
                'context': 'ci-passed',
                'state': 'success',
                'target_url': 'https://example.com/elsewhere',
            },
            {
                'context': 'ci-passed',
                'state': 'success',
                'creator': {'login': 'github-actions[bot]'},
                'target_url': 'https://github.com/kubeflow/pipelines/actions/runs/123',
                'description': 'Expected CI and all checks passed; base policy wrong.',
            },
        ):
            with (
                self.subTest(status=status),
                mock.patch.object(ci, 'api', return_value=[status]),
                self.assertRaisesRegex(RuntimeError, 'ci-passed'),
            ):
                ci.verify_aggregate(SHA, self.replay)

    def test_rerun_started_during_job_reads_is_rejected(self):
        current = run('kubeflow-pipelines-manifests.yml')
        later = dict(current, run_attempt=2, status='queued', conclusion=None)
        actual = job('kubeflow-pipelines-manifests', ['Run tests'])
        pages = [
            [current],
            [dict(path=current['path'], id=17)],
            [actual],
            [later],
        ]
        with (
            mock.patch.object(
                ci, 'api', side_effect=self.responses.__getitem__
            ),
            mock.patch.object(
                ci, 'expected_workflows', return_value=[current['path']]
            ),
            mock.patch.object(ci, 'paged', side_effect=pages),
            self.assertRaisesRegex(RuntimeError, 'queued'),
        ):
            ci.collect_evidence(8, SHA, 9)

    def test_changing_paginated_job_count_is_rejected(self):
        pages = [
            {'jobs': [dict(id=i + 1) for i in range(100)], 'total_count': 105},
            {'jobs': [dict(id=101)], 'total_count': 101},
        ]
        with (
            mock.patch.object(ci, 'api', side_effect=pages),
            self.assertRaisesRegex(RuntimeError, 'Changing'),
        ):
            ci.paged('actions/runs/1/jobs', 'jobs')

    def test_status_authorship_requires_replay_base_and_reconciliation_window(
        self,
    ):
        stamp = hashlib.sha256(
            json.dumps([BASE, PARENT], separators=(',', ':')).encode()
        ).hexdigest()
        status = dict(
            context='ci-passed',
            state='success',
            creator={'login': 'github-actions[bot]'},
            description=f'Expected CI and all checks passed; base policy {stamp}.',
            created_at='2026-01-01T00:00:03Z',
            target_url='https://github.com/kubeflow/pipelines/actions/runs/10',
        )
        owner = dict(
            id=10,
            run_attempt=1,
            workflow_id=7,
            path=ci.PREFIX + 'ci-checks.yml',
            event='schedule',
            head_branch='main',
            head_sha=MERGE,
            repository={'full_name': ci.REPO},
            head_repository={'full_name': ci.REPO},
            status='completed',
            conclusion='success',
        )
        self.responses.update(
            {
                f'commits/{SHA}/statuses?per_page=100': [status],
                'actions/runs/10': owner,
                'actions/workflows/ci-checks.yml': {
                    'path': owner['path'],
                    'id': 7,
                },
                f'compare/{MERGE}...main?per_page=1': {
                    'status': 'ahead',
                    'merge_base_commit': {'sha': MERGE},
                },
            }
        )
        jobs = [
            dict(
                name=f'check_ci_status (9, {SHA})',
                conclusion='success',
                steps=[
                    dict(
                        name='Reconcile status and informational label',
                        conclusion='success',
                        started_at='2026-01-01T00:00:01Z',
                        completed_at='2026-01-01T00:00:05Z',
                    )
                ],
            )
        ]
        with (
            mock.patch.object(
                ci, 'api', side_effect=self.responses.__getitem__
            ),
            mock.patch.object(ci, 'paged', return_value=jobs),
        ):
            ci.verify_aggregate(SHA, self.replay)
            status['created_at'] = '2026-01-02T00:00:03Z'
            with self.assertRaisesRegex(RuntimeError, 'reconciliation job'):
                ci.verify_aggregate(SHA, self.replay)

    def test_failed_jobs_only_rerun_retains_successful_postgres_major(self):
        current = dict(
            run('postgresql-image-upgrade.yml'),
            event='workflow_dispatch',
            run_attempt=2,
        )
        previous = dict(current, run_attempt=1, conclusion='failure')
        names = list(ci.TEST_STEPS['postgresql-image-upgrade.yml'])
        step = ['Upgrade using the rendered deployment and retained PVC']
        first = [
            dict(job(names[0], step), id=10),
            dict(job(names[1], [], 'failure'), id=11),
        ]
        retry = dict(job(names[1], step), run_attempt=2)
        with (
            mock.patch.object(ci, 'api', return_value=previous),
            mock.patch.object(ci, 'paged', side_effect=[first, [retry]]),
        ):
            jobs = ci.effective_jobs(current, self.replay)
            ci.verify_jobs(current['path'], current, jobs)
        self.assertEqual(
            [(j['name'], j['run_attempt']) for j in jobs],
            [(names[0], 1), (names[1], 2)],
        )

    def test_new_failure_or_active_retry_overrides_prior_success(self):
        current = dict(run('kubeflow-pipelines-manifests.yml'), run_attempt=2)
        previous = dict(current, run_attempt=1)
        earlier = job('kubeflow-pipelines-manifests', ['Run tests'])
        for status, result in [('completed', 'failure'), ('in_progress', None)]:
            latest = dict(
                earlier, id=13, run_attempt=2, status=status, conclusion=result
            )
            with (
                self.subTest(status=status),
                mock.patch.object(ci, 'api', return_value=previous),
                mock.patch.object(
                    ci, 'paged', side_effect=[[earlier], [latest]]
                ),
                self.assertRaises(RuntimeError),
            ):
                ci.verify_jobs(
                    current['path'],
                    current,
                    ci.effective_jobs(current, self.replay),
                )

    def test_prior_attempt_cannot_supply_a_job_from_another_test_merge(self):
        current = dict(run(), run_attempt=2)
        previous = copy.deepcopy(dict(current, run_attempt=1))
        previous['referenced_workflows'][0]['sha'] = SHA
        with (
            mock.patch.object(ci, 'api', return_value=previous),
            self.assertRaisesRegex(RuntimeError, 'test merge'),
        ):
            ci.effective_jobs(current, self.replay)

    def test_rerun_rejects_missing_or_cross_revision_jobs(self):
        current = dict(run(), run_attempt=2)
        previous = dict(current, run_attempt=1)
        earlier = job('example', ['Test'])
        for latest in (
            [],
            [dict(earlier, id=13, run_attempt=2, head_sha=PARENT)],
            [dict(earlier, id=13, run_attempt=1)],
            [dict(earlier, run_attempt=2)],
        ):
            with (
                self.subTest(latest=latest),
                mock.patch.object(ci, 'api', return_value=previous),
                mock.patch.object(ci, 'paged', side_effect=[[earlier], latest]),
                self.assertRaises(RuntimeError),
            ):
                ci.effective_jobs(current, self.replay)
