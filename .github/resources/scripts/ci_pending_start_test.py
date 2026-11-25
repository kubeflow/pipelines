"""Pending workflow starts can skip PR details, but never live run
validation."""

import unittest

from ci_pass_eligibility_test import exercise


class PendingStartTests(unittest.TestCase):
    def start(self, **changes):
        return exercise(
            dict(
                workflowRun=True,
                action='in_progress',
                runStatus='in_progress',
                initialStatus='pending',
                **changes,
            )
        )

    def test_pending_active_start_uses_three_reads_and_no_writes(self):
        for action in ('requested', 'in_progress'):
            with self.subTest(action=action):
                result = exercise(
                    dict(
                        workflowRun=True,
                        action=action,
                        runStatus='in_progress',
                        initialStatus='pending',
                    )
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['calls'], [])
                self.assertEqual(result['outputs'], {})
                self.assertEqual(result['status'], 'pending')
                self.assertEqual(result['requestCounts']['pr'], 0)
                self.assertEqual(sum(result['requestCounts'].values()), 3)

    def test_completed_start_survivor_reconciles_displaced_completion(self):
        for conclusion in ('success', 'failure'):
            with self.subTest(conclusion=conclusion):
                result = exercise(
                    dict(
                        workflowRun=True,
                        action='in_progress',
                        initialStatus='pending',
                        conclusion=conclusion,
                        startRun={
                            'status': 'completed',
                            'conclusion': conclusion,
                        },
                    )
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['status'], conclusion)
                self.assertEqual(result['requestCounts']['currentRun'], 1)
                self.assertGreater(result['requestCounts']['pr'], 0)
                self.assertGreater(result['requestCounts']['files'], 0)

    def test_green_published_during_run_read_is_revoked(self):
        result = self.start(statusDuringStartRead='success')
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'pending')
        self.assertIn(['status', 'pending', 'head'], result['calls'])
        self.assertEqual(result['requestCounts']['pr'], 1)
        self.assertEqual(result['requestCounts']['currentRun'], 1)

    def test_non_pending_probe_cannot_authorize_later_publication(self):
        result = self.start(
            statusDuringStartRead='failure', statusDuringPRRead='success'
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'pending')
        self.assertIn(['status', 'pending', 'head'], result['calls'])
        self.assertEqual(result['requestCounts']['statuses'], 2)

    def test_later_completed_start_recovers_after_read_only_noop(self):
        result = exercise(
            dict(
                workflowRun=True,
                action='in_progress',
                initialStatus='pending',
                cycles=2,
                startStatesByCycle=['in_progress', 'completed'],
            )
        )
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'success')
        self.assertEqual(result['requestCounts']['currentRun'], 2)
        self.assertGreater(result['requestCounts']['files'], 0)

    def test_head_change_during_read_cannot_mutate_new_head(self):
        for status in ('pending', 'success'):
            with self.subTest(status=status):
                result = self.start(
                    headDuringStartRead='new-head', statusDuringStartRead=status
                )
                self.assertNotIn('error', result)
                self.assertEqual(result['calls'], [])
                self.assertEqual(result['outputs'], {})

    def test_stale_green_label_keeps_fresh_pr_cleanup(self):
        result = self.start(pr={'labels': [{'name': 'ci-passed'}]})
        self.assertNotIn('error', result)
        self.assertEqual(result['requestCounts']['pr'], 1)
        self.assertEqual(result['requestCounts']['currentRun'], 1)
        self.assertIn(['remove-label', 'ci-passed'], result['calls'])

    def test_unknown_list_labels_cannot_take_read_only_shortcut(self):
        result = self.start(missingListedLabels=True)
        self.assertNotIn('error', result)
        self.assertEqual(result['requestCounts']['pr'], 1)
        self.assertEqual(result['requestCounts']['currentRun'], 1)

    def test_probe_failure_falls_back_to_existing_provisional_publication(self):
        result = self.start(statusReadFailureAt=1)
        self.assertNotIn('error', result)
        self.assertEqual(result['status'], 'pending')
        self.assertEqual(result['requestCounts']['pr'], 1)
        self.assertEqual(result['requestCounts']['currentRun'], 1)
        self.assertEqual(result['requestCounts']['statuses'], 2)

    def test_invalid_run_still_fails_closed_on_original_head(self):
        result = self.start(startRun={'head_sha': 'wrong-head'})
        self.assertIn('error', result)
        self.assertEqual(result['status'], 'failure')
        self.assertIn(['status', 'failure', 'head'], result['calls'])


if __name__ == '__main__':
    unittest.main()
