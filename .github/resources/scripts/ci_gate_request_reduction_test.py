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
"""Guard the cheaper CI paths without weakening success verification."""

import json
import unittest

from ci_pass_eligibility_test import exercise


class CIGateRequestReductionTest(unittest.TestCase):
    def assert_never_green(self, result):
        self.assertNotIn(['status', 'success', 'head'], result['calls'], result)
        self.assertNotIn(['add-label', ['ci-passed']], result['calls'], result)
        self.assertNotEqual(result['status'], 'success', result)

    def assert_no_evidence_reads(self, result):
        for name in (
            'inventory',
            'timeline',
            'files',
            'runs',
            'checks',
            'base',
            'associated',
            'history',
        ):
            self.assertEqual(result['requestCounts'][name], 0, (name, result))

    def test_active_starts_revoke_green_without_full_evidence_reads(self):
        for action in ('requested', 'in_progress'):
            for status in ('queued', 'in_progress', 'waiting'):
                with self.subTest(action=action, status=status):
                    result = exercise(
                        {
                            'workflowRun': True,
                            'action': action,
                            'runStatus': status,
                            'initialStatus': 'success',
                        }
                    )
                    self.assertNotIn('error', result)
                    self.assertEqual(result['status'], 'pending', result)
                    self.assertEqual(
                        [c for c in result['calls'] if c[0] == 'status'],
                        [['status', 'pending', 'head']],
                    )
                    self.assertNotIn('pr_number', result['outputs'])
                    self.assert_no_evidence_reads(result)
                    for name in ('pr', 'pulls', 'currentRun', 'statuses'):
                        self.assertEqual(
                            result['requestCounts'][name], 1, (name, result)
                        )

    def test_active_starts_preserve_unchanged_non_green_status(self):
        for status in ('pending', 'failure'):
            with self.subTest(status=status):
                result = exercise(
                    {
                        'workflowRun': True,
                        'action': 'in_progress',
                        'runStatus': 'in_progress',
                        'initialStatus': status,
                        'initialDescription': 'Existing CI result',
                        'cycles': 3,
                    }
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['status'], status, result)
                self.assertEqual(
                    [
                        c
                        for c in result['calls']
                        if c[0] in ('status', 'add-label', 'remove-label')
                    ],
                    [],
                )
                self.assert_no_evidence_reads(result)

    def test_active_starts_remove_a_stale_green_label_only_once(self):
        result = exercise(
            {
                'workflowRun': True,
                'action': 'in_progress',
                'runStatus': 'in_progress',
                'initialStatus': 'success',
                'pr': {'labels': [{'name': 'ci-passed'}]},
                'cycles': 3,
            }
        )
        self.assert_never_green(result)
        self.assertEqual(
            [c for c in result['calls'] if c[0] == 'remove-label'],
            [['remove-label', 'ci-passed']],
        )
        self.assert_no_evidence_reads(result)

    def test_same_sha_rerun_start_revokes_previous_success(self):
        for current_attempt in (2, 3):
            with self.subTest(current_attempt=current_attempt):
                result = exercise(
                    {
                        'workflowRun': True,
                        'action': 'in_progress',
                        'runStatus': 'in_progress',
                        'runAttempt': 2,
                        'startRun': {'run_attempt': current_attempt},
                        'initialStatus': 'success',
                    }
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['status'], 'pending', result)
                self.assert_no_evidence_reads(result)

    def test_invalid_current_run_identity_fails_closed(self):
        overrides = [
            {'id': 43},
            {'head_sha': 'another-head'},
            {'head_branch': 'another-branch'},
            {'head_repository': {'full_name': 'attacker/pipelines'}},
            {'repository': {'full_name': 'attacker/pipelines'}},
            {'event': 'push'},
            {'run_attempt': 0},
            {'run_attempt': 1},
            {'run_attempt': None},
            {'status': 'unknown'},
            {'status': 'in_progress', 'conclusion': 'success'},
        ]
        for override in overrides:
            with self.subTest(override=override):
                result = exercise(
                    {
                        'workflowRun': True,
                        'action': 'in_progress',
                        'runStatus': 'in_progress',
                        'initialStatus': 'success',
                        'runAttempt': 2,
                        'startRun': override,
                    }
                )
                self.assertIn('error', result)
                self.assertEqual(result['status'], 'failure', result)
                self.assert_never_green(result)
                self.assert_no_evidence_reads(result)

    def test_unreadable_current_run_cannot_leave_green(self):
        result = exercise(
            {
                'workflowRun': True,
                'action': 'requested',
                'runStatus': 'queued',
                'initialStatus': 'success',
                'startReadFailure': True,
            }
        )
        self.assertIn('error', result)
        self.assertEqual(result['status'], 'failure', result)
        self.assert_no_evidence_reads(result)

    def test_delayed_start_for_completed_run_rechecks_all_evidence(self):
        result = exercise(
            {
                'workflowRun': True,
                'action': 'in_progress',
                'startRun': {'status': 'completed', 'conclusion': 'success'},
            }
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'success', result)
        self.assertEqual(result['requestCounts']['currentRun'], 1)
        self.assertEqual(result['requestCounts']['files'], 3)
        self.assertEqual(result['requestCounts']['checks'], 2)

    def test_completed_start_cannot_hide_another_active_execution(self):
        result = exercise(
            {
                'workflowRun': True,
                'action': 'in_progress',
                'initialStatus': 'success',
                'startRun': {'status': 'completed', 'conclusion': 'success'},
                'runPatch': {
                    'id': 43,
                    'status': 'in_progress',
                    'conclusion': None,
                },
            }
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'pending', result)
        self.assert_never_green(result)
        self.assertEqual(result['requestCounts']['files'], 1)
        self.assertEqual(result['requestCounts']['checks'], 1)

    def test_completed_start_still_checks_retarget_history_and_holds(self):
        for extra in (
            {'retarget': True},
            {'pr': {'labels': [{'name': 'needs-ok-to-test'}]}},
        ):
            with self.subTest(extra=extra):
                result = exercise(
                    {
                        'workflowRun': True,
                        'action': 'requested',
                        'initialStatus': 'success',
                        'startRun': {
                            'status': 'completed',
                            'conclusion': 'success',
                        },
                        **extra,
                    }
                )
                self.assertEqual(result['status'], 'failure', result)
                self.assert_never_green(result)

    def test_negative_assessment_reads_workflow_evidence_once(self):
        for options, expected in (
            ({'runStatus': 'in_progress'}, 'pending'),
            ({'conclusion': 'failure'}, 'failure'),
        ):
            with self.subTest(options=options):
                result = exercise(options)
                self.assertNotIn('error', result)
                self.assertEqual(result['status'], expected, result)
                self.assertEqual(result['outputs']['ready'], 'false')
                self.assertIn('prepared_result', result['outputs'])
                prepared = json.loads(result['outputs']['prepared_result'])
                self.assertEqual(prepared['state'], expected)
                self.assertTrue(prepared['reasons'])
                for name in ('files', 'timeline', 'checks'):
                    self.assertEqual(
                        result['requestCounts'][name], 1, (name, result)
                    )
                self.assertEqual(result['requestCounts']['runs'], 2)
                self.assert_never_green(result)

    def test_missing_workflow_still_obeys_registration_deadline(self):
        result = exercise(
            {
                'missing': True,
                'registrationStartedAt': '2026-09-07T11:00:00Z',
            }
        )
        self.assertEqual(result['status'], 'failure', result)
        self.assertIn('after 15 minutes', result['descriptions'][-1])
        self.assertEqual(result['requestCounts']['timeline'], 1)
        self.assertEqual(result['requestCounts']['files'], 2)
        self.assertEqual(result['requestCounts']['runs'], 3)
        self.assertEqual(result['requestCounts']['checks'], 1)

    def test_external_failure_keeps_precedence_over_cached_pending_result(self):
        result = exercise({'runStatus': 'in_progress', 'pollPassed': False})
        self.assertEqual(result['status'], 'failure', result)
        self.assertIn('DCO', result['descriptions'][-1])
        self.assertEqual(result['requestCounts']['files'], 1)
        self.assertEqual(result['requestCounts']['checks'], 1)
        self.assert_never_green(result)

    def test_workflow_failure_keeps_precedence_over_external_pending(self):
        result = exercise(
            {
                'conclusion': 'failure',
                'checkStatus': 'in_progress',
            }
        )
        self.assertEqual(result['status'], 'failure', result)
        self.assertIn('completed/failure', result['descriptions'][-1])
        self.assertEqual(result['requestCounts']['files'], 1)
        self.assertEqual(result['requestCounts']['checks'], 1)

    def test_external_read_failure_cannot_promote_cached_pending_result(self):
        result = exercise(
            {
                'runStatus': 'in_progress',
                'checkApiFailure': True,
            }
        )
        self.assertEqual(result['status'], 'failure', result)
        self.assertEqual(result['requestCounts']['files'], 1)
        self.assertEqual(result['requestCounts']['checks'], 1)
        self.assert_never_green(result)

    def test_malformed_or_positive_prepared_result_is_rejected(self):
        invalid = [
            '{',
            json.dumps({'state': 'success', 'reasons': ['Looks green']}),
            json.dumps({'state': 'failure', 'reasons': 'not an array'}),
            json.dumps({'state': 'pending', 'reasons': [1]}),
            json.dumps({'state': 'failure'}),
        ]
        for value in invalid:
            with self.subTest(value=value):
                result = exercise(
                    {
                        'runStatus': 'in_progress',
                        'preparedResultOverride': value,
                    }
                )
                self.assertIn('error', result)
                self.assertEqual(result['status'], 'failure', result)
                self.assert_never_green(result)

    def test_negative_result_rechecks_pr_identity_before_publication(self):
        for drift in ('head', 'base', 'base-sha', 'hold', 'closed'):
            with self.subTest(drift=drift):
                result = exercise(
                    {
                        'runStatus': 'in_progress',
                        'beforeFinalDrift': drift,
                    }
                )
                self.assertEqual(result['status'], 'failure', result)
                self.assertEqual(result['requestCounts']['files'], 1)
                self.assert_never_green(result)

    def test_merge_before_negative_finalize_preserves_status(self):
        result = exercise(
            {
                'runStatus': 'in_progress',
                'initialStatus': 'success',
                'mergeBeforeFinal': True,
            }
        )
        self.assertNotIn('error', result)
        self.assertEqual(
            [c for c in result['calls'] if c[0] == 'status'],
            [['status', 'pending', 'head']],
        )
        self.assertEqual(result['requestCounts']['checks'], 0)

    def test_recovery_during_negative_assessment_waits_for_next_event(self):
        result = exercise(
            {
                'conclusion': 'failure',
                'recoverBeforeFinal': True,
            }
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'failure', result)
        self.assertEqual(result['requestCounts']['files'], 1)
        self.assert_never_green(result)
        recovered = exercise({'initialStatus': result['status']})
        self.assertEqual(recovered['status'], 'success', recovered)

    def test_stale_or_merged_start_event_does_not_publish(self):
        cases = (
            {'eventRun': {'head_sha': 'old-head'}},
            {'pr': {'merged': True, 'state': 'closed'}},
        )
        for extra in cases:
            with self.subTest(extra=extra):
                result = exercise(
                    {
                        'workflowRun': True,
                        'action': 'in_progress',
                        'runStatus': 'in_progress',
                        'initialStatus': 'success',
                        **extra,
                    }
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['calls'], [], result)
                self.assertEqual(result['status'], 'success', result)
                self.assertEqual(result['requestCounts']['currentRun'], 0)
                self.assert_no_evidence_reads(result)

    def test_malformed_start_event_fails_before_current_run_lookup(self):
        for override in (
            {'id': 0},
            {'id': None},
            {'run_attempt': 0},
            {'run_attempt': None},
        ):
            with self.subTest(override=override):
                result = exercise(
                    {
                        'workflowRun': True,
                        'action': 'in_progress',
                        'runStatus': 'in_progress',
                        'initialStatus': 'success',
                        'eventRun': override,
                    }
                )
                self.assertIn('error', result)
                self.assertEqual(result['status'], 'failure', result)
                self.assertEqual(result['requestCounts']['currentRun'], 0)
                self.assert_no_evidence_reads(result)
                self.assert_never_green(result)

    def test_success_keeps_all_pre_and_post_publication_checks(self):
        result = exercise()
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'success', result)
        self.assertEqual(result['outputs']['ready'], 'true')
        self.assertNotIn('prepared_result', result['outputs'])
        for name in ('files', 'timeline'):
            self.assertEqual(result['requestCounts'][name], 3, (name, result))
        self.assertEqual(result['requestCounts']['runs'], 5)
        self.assertEqual(result['requestCounts']['checks'], 2)

    def test_success_is_revoked_if_checks_change_during_publication(self):
        for drift, expected in (
            ('rerun', 'failure'),
            ('external-failure', 'failure'),
            ('external-rerun', 'pending'),
        ):
            with self.subTest(drift=drift):
                result = exercise({'drift': drift})
                self.assertIn(['status', 'success', 'head'], result['calls'])
                self.assertEqual(result['status'], expected, result)
                self.assertEqual(result['requestCounts']['files'], 3)
                self.assertEqual(result['requestCounts']['checks'], 2)


if __name__ == '__main__':
    unittest.main()
