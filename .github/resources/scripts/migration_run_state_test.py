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

import importlib.util
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest import mock

MODULE_PATH = (
    Path(__file__).resolve().parents[3] /
    'test/kfp-kubernetes-native-migration-tests/run_state.py')
SPEC = importlib.util.spec_from_file_location('run_state', MODULE_PATH)
assert SPEC is not None and SPEC.loader is not None
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class MigrationRunStateTest(unittest.TestCase):

    def setUp(self):
        self.now = 100.0
        self.sleeps = []
        self.client = mock.Mock()
        self.run_id = 'migration-run-id'
        self.enterContext(
            mock.patch.object(
                MODULE.time, 'monotonic', side_effect=lambda: self.now))
        self.enterContext(
            mock.patch.object(MODULE.time, 'sleep', side_effect=self.sleep))

    def sleep(self, duration):
        self.sleeps.append(duration)
        self.now += duration

    def run_details(self, state):
        return SimpleNamespace(
            state=state,
            run_id=self.run_id,
            display_name='k8s-mode-test-run',
            experiment_id='migration-experiment-id',
        )

    def test_waits_for_reported_state_and_returns_complete_run(self):
        ready = self.run_details('RUNNING')
        self.client.get_run.side_effect = [
            self.run_details(state)
            for state in (None, '', 'RUNTIME_STATE_UNSPECIFIED')
        ] + [ready]

        result = MODULE.wait_for_run_state(self.client, self.run_id)

        self.assertIs(result, ready)
        self.assertEqual(result.run_id, self.run_id)
        self.assertEqual(result.display_name, 'k8s-mode-test-run')
        self.assertEqual(result.experiment_id, 'migration-experiment-id')
        self.assertEqual(self.sleeps, [1, 1, 1])
        self.assertEqual(self.client.get_run.call_args_list,
                         [mock.call(run_id=self.run_id)] * 4)

    def test_returns_initial_reported_state_without_sleeping(self):
        for state in ('PENDING', 'RUNNING', 'SUCCEEDED', 'SKIPPED', 'PAUSED'):
            with self.subTest(state=state):
                self.client.reset_mock()
                ready = self.run_details(state)
                self.client.get_run.return_value = ready

                result = MODULE.wait_for_run_state(self.client, self.run_id)

                self.assertIs(result, ready)
                self.client.get_run.assert_called_once_with(run_id=self.run_id)
                self.assertEqual(self.sleeps, [])

    def test_timeout_reports_run_last_state_budget_and_diagnostics(self):
        self.client.get_run.return_value = self.run_details(
            'RUNTIME_STATE_UNSPECIFIED')

        with self.assertRaises(TimeoutError) as caught:
            MODULE.wait_for_run_state(
                self.client, self.run_id, timeout=2.5, poll_interval=1)

        message = str(caught.exception)
        self.assertIn(self.run_id, message)
        self.assertIn('RUNTIME_STATE_UNSPECIFIED', message)
        self.assertIn('2.5', message)
        self.assertIn('log', message.lower())
        self.assertIn('persistence', message.lower())
        self.assertEqual(self.sleeps, [1, 1, 0.5])
        self.assertEqual(self.now, 102.5)

    def test_timeout_when_state_is_omitted(self):
        self.client.get_run.return_value = self.run_details(None)

        with self.assertRaises(TimeoutError) as caught:
            MODULE.wait_for_run_state(
                self.client, self.run_id, timeout=1, poll_interval=2)

        self.assertIn(self.run_id, str(caught.exception))
        self.assertIn('None', str(caught.exception))
        self.assertEqual(self.sleeps, [1])

    def test_terminal_failure_is_not_retried(self):
        for state in ('FAILED', 'CANCELED'):
            with self.subTest(state=state):
                self.client.reset_mock()
                self.client.get_run.return_value = self.run_details(state)

                with self.assertRaises(AssertionError) as caught:
                    MODULE.wait_for_run_state(self.client, self.run_id)

                self.assertIn(self.run_id, str(caught.exception))
                self.assertIn(state, str(caught.exception))
                self.client.get_run.assert_called_once_with(run_id=self.run_id)
                self.assertEqual(self.sleeps, [])

    def test_terminal_failure_after_unreported_state_is_not_success(self):
        for state in ('FAILED', 'CANCELED'):
            with self.subTest(state=state):
                self.sleeps.clear()
                self.client.reset_mock()
                self.client.get_run.side_effect = [
                    self.run_details(None),
                    self.run_details(state),
                ]

                with self.assertRaises(AssertionError) as caught:
                    MODULE.wait_for_run_state(self.client, self.run_id)

                self.assertIn(state, str(caught.exception))
                self.assertEqual(self.client.get_run.call_count, 2)
                self.assertEqual(self.sleeps, [1])

    def test_api_error_propagates_without_retry(self):
        error = RuntimeError('API lookup failed')
        self.client.get_run.side_effect = error

        with self.assertRaises(RuntimeError) as caught:
            MODULE.wait_for_run_state(self.client, self.run_id)

        self.assertIs(caught.exception, error)
        self.client.get_run.assert_called_once_with(run_id=self.run_id)
        self.assertEqual(self.sleeps, [])

    def test_nonpositive_poll_settings_fail_before_api_call(self):
        for option in ('timeout', 'poll_interval'):
            for value in (0, -1):
                with self.subTest(option=option, value=value):
                    with self.assertRaises(ValueError):
                        MODULE.wait_for_run_state(self.client, self.run_id,
                                                  **{option: value})

        self.client.get_run.assert_not_called()
        self.assertEqual(self.sleeps, [])


if __name__ == '__main__':
    unittest.main()
