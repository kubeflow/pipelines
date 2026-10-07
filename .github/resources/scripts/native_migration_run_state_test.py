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
"""Exercise the migration wait with real helper code and no cluster."""

import importlib.util
from pathlib import Path
from types import SimpleNamespace
import unittest
from unittest import mock

ROOT = Path(__file__).resolve().parents[3]
SPEC = importlib.util.spec_from_file_location(
    'migration_run_state',
    ROOT / 'test/kfp-kubernetes-native-migration-tests/run_state.py')
run_state = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(run_state)


class NativeMigrationRunStateTest(unittest.TestCase):

    def test_waits_through_unreported_state_and_returns_complete_run(self):
        client = mock.Mock()
        ready = SimpleNamespace(
            run_id='created-run',
            state='RUNNING',
            display_name='migration run',
            experiment_id='experiment')
        client.get_run.side_effect = [
            SimpleNamespace(run_id='created-run'),
            SimpleNamespace(run_id='created-run', state=''),
            SimpleNamespace(
                run_id='created-run', state='RUNTIME_STATE_UNSPECIFIED'),
            ready,
        ]
        with mock.patch.object(run_state.time, 'monotonic', side_effect=[0, 1, 2, 3]), \
                mock.patch.object(run_state.time, 'sleep') as sleep:
            self.assertIs(
                run_state.wait_for_run_state(client, 'created-run'), ready)
        self.assertEqual(client.get_run.call_args_list,
                         [mock.call(run_id='created-run')] * 4)
        self.assertEqual(sleep.call_args_list, [mock.call(1)] * 3)
        self.assertEqual(ready.display_name, 'migration run')
        self.assertEqual(ready.experiment_id, 'experiment')

    def test_missing_state_still_fails_at_deadline(self):
        client = mock.Mock()
        client.get_run.return_value = SimpleNamespace(
            run_id='created-run', state=None)
        with mock.patch.object(run_state.time, 'monotonic', side_effect=[0, 0, 2]), \
                mock.patch.object(run_state.time, 'sleep') as sleep:
            with self.assertRaisesRegex(AssertionError,
                                        'created-run did not report state'):
                run_state.wait_for_run_state(
                    client, 'created-run', timeout=2, poll_interval=10)
        sleep.assert_called_once_with(2)
        self.assertEqual(client.get_run.call_count, 2)

    def test_different_run_and_lookup_errors_are_not_retried(self):
        client = mock.Mock()
        client.get_run.return_value = SimpleNamespace(
            run_id='different-run', state='RUNNING')
        with mock.patch.object(run_state.time, 'sleep') as sleep:
            with self.assertRaisesRegex(AssertionError, 'correct ID'):
                run_state.wait_for_run_state(client, 'created-run')
            sleep.assert_not_called()
            client.get_run.side_effect = RuntimeError('API unavailable')
            with self.assertRaisesRegex(RuntimeError, 'API unavailable'):
                run_state.wait_for_run_state(client, 'created-run')
            sleep.assert_not_called()


if __name__ == '__main__':
    unittest.main()
