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

import datetime
import unittest
from unittest import mock

from click import testing

from kfp.cli import recurring_run
from kfp.server_api import V2RecurringRun


def _create_mock_recurring_run(
    recurring_run_id: str = 'test-id',
    status: str = 'ENABLED',
    name: str = 'test-job',
) -> V2RecurringRun:
    return V2RecurringRun(
        recurring_run_id=recurring_run_id,
        display_name=name,
        status=status,
        created_at=datetime.datetime(2026, 1, 1, 12, 0, 0),
    )


class TestRecurringRunWaitHelper(unittest.TestCase):
    def test_wait_immediate_success(self):
        mock_client = mock.MagicMock()
        mock_client.get_recurring_run.return_value = _create_mock_recurring_run(
            status='ENABLED'
        )

        result = recurring_run._wait_for_recurring_run_status(
            client_obj=mock_client,
            recurring_run_id='test-id',
            target_status='ENABLED',
            timeout=10,
            sleep_interval=0.01,
        )

        self.assertEqual(result.status, 'ENABLED')
        mock_client.get_recurring_run.assert_called_once_with(
            recurring_run_id='test-id',
            _request_timeout=mock.ANY,
        )

    @mock.patch('time.sleep')
    def test_wait_polling_success(self, mock_sleep):
        mock_client = mock.MagicMock()
        mock_client.get_recurring_run.side_effect = [
            _create_mock_recurring_run(status='DISABLED'),
            _create_mock_recurring_run(status='DISABLED'),
            _create_mock_recurring_run(status='ENABLED'),
        ]

        result = recurring_run._wait_for_recurring_run_status(
            client_obj=mock_client,
            recurring_run_id='test-id',
            target_status='ENABLED',
            timeout=10,
            sleep_interval=0.01,
        )

        self.assertEqual(result.status, 'ENABLED')
        self.assertEqual(mock_client.get_recurring_run.call_count, 3)
        self.assertEqual(mock_sleep.call_count, 2)

    @mock.patch('time.sleep')
    def test_wait_timeout(self, mock_sleep):
        mock_client = mock.MagicMock()
        mock_client.get_recurring_run.return_value = _create_mock_recurring_run(
            status='DISABLED'
        )

        with self.assertRaises(TimeoutError) as ctx:
            recurring_run._wait_for_recurring_run_status(
                client_obj=mock_client,
                recurring_run_id='test-id',
                target_status='ENABLED',
                timeout=0,
                sleep_interval=0.01,
            )

        self.assertIn(
            'Timed out after 0 seconds waiting for recurring run test-id',
            str(ctx.exception),
        )

    def test_wait_late_response_rejected_after_deadline(self):
        """Simulate clock advancing past deadline to verify late responses are
        rejected."""

        class FakeClock:
            def __init__(self):
                self.time = 0.0

            def monotonic(self) -> float:
                return self.time

            def sleep(self, seconds: float):
                self.time += seconds

        clock = FakeClock()
        mock_client = mock.MagicMock()

        def fake_get(*args, **kwargs):
            # First GET at 0.0s returns DISABLED at 0.6s
            if clock.time < 1.0:
                clock.time = 0.6
                return _create_mock_recurring_run(status='DISABLED')
            # Second GET returns ENABLED at 2.2s (after timeout=1 deadline)
            clock.time = 2.2
            return _create_mock_recurring_run(status='ENABLED')

        mock_client.get_recurring_run.side_effect = fake_get

        with (
            mock.patch('time.monotonic', side_effect=clock.monotonic),
            mock.patch('time.sleep', side_effect=clock.sleep) as mock_sleep,
        ):
            with self.assertRaises(TimeoutError) as ctx:
                recurring_run._wait_for_recurring_run_status(
                    client_obj=mock_client,
                    recurring_run_id='test-id',
                    target_status='ENABLED',
                    timeout=1,
                    sleep_interval=1.0,
                )

        self.assertIn(
            'Timed out after 1 seconds waiting for recurring run test-id',
            str(ctx.exception),
        )
        # Verify sleep was capped to remaining budget (0.4s), not full 1.0s
        mock_sleep.assert_called_once()
        slept = mock_sleep.call_args[0][0]
        self.assertAlmostEqual(slept, 0.4, places=2)

    def test_wait_response_arriving_after_deadline_raises(self):
        """Verify that a response arriving after the deadline is rejected."""

        class FakeClock:
            def __init__(self):
                self.time = 0.0

            def monotonic(self) -> float:
                return self.time

            def sleep(self, seconds: float):
                self.time += seconds

        clock = FakeClock()
        mock_client = mock.MagicMock()

        def fake_get(*args, **kwargs):
            # GET returns matching status after deadline has passed
            clock.time = 1.5
            return _create_mock_recurring_run(status='ENABLED')

        mock_client.get_recurring_run.side_effect = fake_get

        with (
            mock.patch('time.monotonic', side_effect=clock.monotonic),
            mock.patch('time.sleep', side_effect=clock.sleep),
        ):
            with self.assertRaises(TimeoutError) as ctx:
                recurring_run._wait_for_recurring_run_status(
                    client_obj=mock_client,
                    recurring_run_id='test-id',
                    target_status='ENABLED',
                    timeout=1,
                    sleep_interval=1.0,
                )

        self.assertIn("Current status: 'ENABLED'", str(ctx.exception))

    def test_wait_passes_integer_total_request_timeout(self):
        """Verify _request_timeout is passed as an integer total timeout
        budget."""
        mock_client = mock.MagicMock()
        mock_client.get_recurring_run.return_value = _create_mock_recurring_run(
            status='ENABLED'
        )

        recurring_run._wait_for_recurring_run_status(
            client_obj=mock_client,
            recurring_run_id='test-id',
            target_status='ENABLED',
            timeout=10,
            sleep_interval=0.01,
        )

        call_kwargs = mock_client.get_recurring_run.call_args[1]
        self.assertIn('_request_timeout', call_kwargs)
        timeout_arg = call_kwargs['_request_timeout']
        self.assertIsInstance(timeout_arg, int)
        self.assertGreaterEqual(timeout_arg, 1)
        self.assertLessEqual(timeout_arg, 10)

    def test_wait_internal_type_error_propagates(self):
        """Verify TypeError from get_recurring_run propagates and is not
        masked."""
        mock_client = mock.MagicMock()
        mock_client.get_recurring_run.side_effect = TypeError(
            'unexpected type error'
        )

        with self.assertRaises(TypeError):
            recurring_run._wait_for_recurring_run_status(
                client_obj=mock_client,
                recurring_run_id='test-id',
                target_status='ENABLED',
                timeout=10,
                sleep_interval=0.01,
            )

    def test_wait_transport_error_after_deadline_raises_timeout_error(self):
        """Verify transport failure after deadline expiration is converted to
        TimeoutError."""

        class FakeClock:
            def __init__(self):
                self.time = 0.0

            def monotonic(self) -> float:
                return self.time

        clock = FakeClock()
        mock_client = mock.MagicMock()

        def fake_get(*args, **kwargs):
            clock.time = 1.5
            raise ConnectionError('Transport read timeout')

        mock_client.get_recurring_run.side_effect = fake_get

        with mock.patch('time.monotonic', side_effect=clock.monotonic):
            with self.assertRaises(TimeoutError) as ctx:
                recurring_run._wait_for_recurring_run_status(
                    client_obj=mock_client,
                    recurring_run_id='test-id',
                    target_status='ENABLED',
                    timeout=1,
                    sleep_interval=0.1,
                )

        self.assertIn(
            'Timed out after 1 seconds waiting for recurring run test-id',
            str(ctx.exception),
        )
        self.assertIsInstance(ctx.exception.__cause__, ConnectionError)


class TestRecurringRunCommands(unittest.TestCase):
    def setUp(self):
        self.runner = testing.CliRunner()
        self.mock_client = mock.MagicMock()

    def test_enable_no_wait(self):
        self.mock_client.get_recurring_run.return_value = (
            _create_mock_recurring_run(status='DISABLED')
        )

        result = self.runner.invoke(
            cli=recurring_run.recurring_run,
            args=['enable', 'test-id'],
            obj={'client': self.mock_client, 'output': 'table'},
            catch_exceptions=False,
        )

        self.assertEqual(result.exit_code, 0)
        self.mock_client.enable_recurring_run.assert_called_once_with(
            recurring_run_id='test-id'
        )
        self.mock_client.get_recurring_run.assert_called_once_with(
            recurring_run_id='test-id'
        )
        self.assertIn('test-id', result.output)
        self.assertIn('DISABLED', result.output)

    @mock.patch('time.sleep')
    def test_enable_with_wait(self, mock_sleep):
        self.mock_client.get_recurring_run.side_effect = [
            _create_mock_recurring_run(status='DISABLED'),
            _create_mock_recurring_run(status='ENABLED'),
        ]

        result = self.runner.invoke(
            cli=recurring_run.recurring_run,
            args=['enable', 'test-id', '--wait'],
            obj={'client': self.mock_client, 'output': 'table'},
            catch_exceptions=False,
        )

        self.assertEqual(result.exit_code, 0)
        self.mock_client.enable_recurring_run.assert_called_once_with(
            recurring_run_id='test-id'
        )
        self.assertEqual(self.mock_client.get_recurring_run.call_count, 2)
        self.assertIn('ENABLED', result.output)

    @mock.patch('time.sleep')
    def test_enable_with_wait_timeout(self, mock_sleep):
        self.mock_client.get_recurring_run.return_value = (
            _create_mock_recurring_run(status='DISABLED')
        )

        result = self.runner.invoke(
            cli=recurring_run.recurring_run,
            args=[
                'enable',
                'test-id',
                '--wait',
                '--timeout',
                '0',
            ],
            obj={'client': self.mock_client, 'output': 'table'},
        )

        self.assertNotEqual(result.exit_code, 0)
        self.assertIsInstance(result.exception, TimeoutError)

    def test_disable_no_wait(self):
        self.mock_client.get_recurring_run.return_value = (
            _create_mock_recurring_run(status='ENABLED')
        )

        result = self.runner.invoke(
            cli=recurring_run.recurring_run,
            args=['disable', 'test-id'],
            obj={'client': self.mock_client, 'output': 'table'},
            catch_exceptions=False,
        )

        self.assertEqual(result.exit_code, 0)
        self.mock_client.disable_recurring_run.assert_called_once_with(
            recurring_run_id='test-id'
        )
        self.mock_client.get_recurring_run.assert_called_once_with(
            recurring_run_id='test-id'
        )
        self.assertIn('test-id', result.output)
        self.assertIn('ENABLED', result.output)

    @mock.patch('time.sleep')
    def test_disable_with_wait(self, mock_sleep):
        self.mock_client.get_recurring_run.side_effect = [
            _create_mock_recurring_run(status='ENABLED'),
            _create_mock_recurring_run(status='DISABLED'),
        ]

        result = self.runner.invoke(
            cli=recurring_run.recurring_run,
            args=['disable', 'test-id', '--wait'],
            obj={'client': self.mock_client, 'output': 'table'},
            catch_exceptions=False,
        )

        self.assertEqual(result.exit_code, 0)
        self.mock_client.disable_recurring_run.assert_called_once_with(
            recurring_run_id='test-id'
        )
        self.assertEqual(self.mock_client.get_recurring_run.call_count, 2)
        self.assertIn('DISABLED', result.output)

    @mock.patch('time.sleep')
    def test_disable_with_wait_timeout(self, mock_sleep):
        self.mock_client.get_recurring_run.return_value = (
            _create_mock_recurring_run(status='ENABLED')
        )

        result = self.runner.invoke(
            cli=recurring_run.recurring_run,
            args=[
                'disable',
                'test-id',
                '--wait',
                '--timeout',
                '0',
            ],
            obj={'client': self.mock_client, 'output': 'table'},
        )

        self.assertNotEqual(result.exit_code, 0)
        self.assertIsInstance(result.exception, TimeoutError)


if __name__ == '__main__':
    unittest.main()
