# Copyright 2024 The Kubeflow Authors
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
"""Tests for kfp.cli.recurring_run."""

import unittest
from unittest import mock

from click import testing
from kfp.cli import cli


class TestRecurringRunCommands(unittest.TestCase):

    def setUp(self):
        self.runner = testing.CliRunner()
        self.client_mock = mock.MagicMock()

    def invoke(self, args, catch_exceptions=False, input_data=None):
        with mock.patch(
                'kfp.cli.cli.client.Client', return_value=self.client_mock):
            return self.runner.invoke(
                cli.cli,
                args,
                catch_exceptions=catch_exceptions,
                obj={
                    'client': self.client_mock,
                    'output': 'json'
                },
                input=input_data,
            )

    def test_recurring_run_create_with_interval(self):
        self.client_mock.create_recurring_run.return_value = mock.MagicMock(
            recurring_run_id='rr-123')
        self.client_mock.create_experiment.return_value = mock.MagicMock(
            experiment_id='exp-123')
        result = self.invoke([
            'recurring-run', 'create', '--job-name', 'my-job',
            '--interval-second', '3600', '--experiment-name', 'my-exp'
        ])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.create_recurring_run.assert_called_once()
        call_kwargs = self.client_mock.create_recurring_run.call_args.kwargs
        self.assertEqual(call_kwargs['job_name'], 'my-job')
        self.assertEqual(call_kwargs['interval_second'], '3600')
        self.assertEqual(call_kwargs['experiment_id'], 'exp-123')

    def test_recurring_run_create_with_cron(self):
        self.client_mock.create_recurring_run.return_value = mock.MagicMock(
            recurring_run_id='rr-123')
        self.client_mock.create_experiment.return_value = mock.MagicMock(
            experiment_id='exp-123')
        result = self.invoke([
            'recurring-run', 'create', '--job-name', 'my-job',
            '--cron-expression', '0 * * * *', '--experiment-id', 'exp-456'
        ])
        self.assertEqual(result.exit_code, 0)
        call_kwargs = self.client_mock.create_recurring_run.call_args.kwargs
        self.assertEqual(call_kwargs['cron_expression'], '0 * * * *')
        self.assertEqual(call_kwargs['experiment_id'], 'exp-456')

    def test_recurring_run_create_requires_interval_or_cron(self):
        result = self.invoke([
            'recurring-run', 'create', '--job-name', 'my-job',
            '--experiment-name', 'my-exp'
        ],
                             catch_exceptions=True)
        self.assertEqual(result.exit_code, 1)
        self.assertIn(
            'Either of --interval-second or --cron-expression options is required',
            str(result.exception))

    def test_recurring_run_create_not_both_interval_and_cron(self):
        result = self.invoke([
            'recurring-run', 'create', '--job-name', 'my-job',
            '--interval-second', '3600', '--cron-expression', '0 * * * *',
            '--experiment-name', 'my-exp'
        ],
                             catch_exceptions=True)
        self.assertEqual(result.exit_code, 1)
        self.assertIn(
            'Either of --interval-second or --cron-expression options is required',
            str(result.exception))

    def test_recurring_run_create_requires_experiment(self):
        result = self.invoke([
            'recurring-run', 'create', '--job-name', 'my-job',
            '--interval-second', '3600'
        ],
                             catch_exceptions=True)
        self.assertEqual(result.exit_code, 1)
        self.assertIn('Either --experiment-id or --experiment-name is required',
                      str(result.exception))

    def test_recurring_run_create_with_params(self):
        self.client_mock.create_recurring_run.return_value = mock.MagicMock(
            recurring_run_id='rr-123')
        self.client_mock.create_experiment.return_value = mock.MagicMock(
            experiment_id='exp-123')
        result = self.invoke([
            'recurring-run', 'create', '--job-name', 'my-job',
            '--interval-second', '3600', '--experiment-name', 'my-exp',
            'param1=value1', 'param2=value2'
        ])
        self.assertEqual(result.exit_code, 0)
        call_kwargs = self.client_mock.create_recurring_run.call_args.kwargs
        self.assertEqual(call_kwargs['params'], {
            'param1': 'value1',
            'param2': 'value2'
        })

    def test_recurring_run_create_with_all_options(self):
        self.client_mock.create_recurring_run.return_value = mock.MagicMock(
            recurring_run_id='rr-123')
        self.client_mock.create_experiment.return_value = mock.MagicMock(
            experiment_id='exp-123')
        result = self.invoke([
            'recurring-run', 'create', '--job-name', 'my-job',
            '--interval-second', '3600', '--experiment-name', 'my-exp',
            '--description', 'test desc', '--enabled', '--catchup',
            '--start-time', '2024-01-01T00:00:00Z', '--end-time',
            '2024-12-31T23:59:59Z', '--max-concurrency', '5', '--pipeline-id',
            'pipe-123', '--version-id', 'ver-123', 'key=value'
        ])
        self.assertEqual(result.exit_code, 0)
        call_kwargs = self.client_mock.create_recurring_run.call_args.kwargs
        self.assertEqual(call_kwargs['description'], 'test desc')
        self.assertEqual(call_kwargs['enabled'], True)
        self.assertEqual(call_kwargs['no_catchup'], False)
        self.assertEqual(call_kwargs['start_time'], '2024-01-01T00:00:00Z')
        self.assertEqual(call_kwargs['end_time'], '2024-12-31T23:59:59Z')
        self.assertEqual(call_kwargs['max_concurrency'], 5)
        self.assertEqual(call_kwargs['pipeline_id'], 'pipe-123')
        self.assertEqual(call_kwargs['version_id'], 'ver-123')

    def test_recurring_run_list(self):
        self.client_mock.list_recurring_runs.return_value = mock.MagicMock(
            recurring_runs=[])
        result = self.invoke(['recurring-run', 'list'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.list_recurring_runs.assert_called_once_with(
            experiment_id=None,
            page_token='',
            page_size=100,
            sort_by='created_at desc',
            filter=None)

    def test_recurring_run_list_with_options(self):
        self.client_mock.list_recurring_runs.return_value = mock.MagicMock(
            recurring_runs=[])
        result = self.invoke([
            'recurring-run', 'list', '--experiment-id', 'exp-123',
            '--page-token', 'token123', '--max-size', '50', '--sort-by', 'name',
            '--filter', 'name=test'
        ])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.list_recurring_runs.assert_called_once_with(
            experiment_id='exp-123',
            page_token='token123',
            page_size=50,
            sort_by='name',
            filter='name=test')

    def test_recurring_run_get(self):
        self.client_mock.get_recurring_run.return_value = mock.MagicMock(
            recurring_run_id='rr-123')
        result = self.invoke(['recurring-run', 'get', 'rr-123'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.get_recurring_run.assert_called_once_with('rr-123')

    def test_recurring_run_delete(self):
        self.client_mock.delete_recurring_run.return_value = None
        result = self.invoke(['recurring-run', 'delete', 'rr-123'],
                             input_data='y\n')
        self.assertEqual(result.exit_code, 0)
        self.client_mock.delete_recurring_run.assert_called_once_with('rr-123')

    def test_recurring_run_enable(self):
        self.client_mock.enable_recurring_run.return_value = None
        rr_mock = mock.MagicMock(recurring_run_id='rr-123')
        self.client_mock.get_recurring_run.return_value = rr_mock
        result = self.invoke(['recurring-run', 'enable', 'rr-123'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.enable_recurring_run.assert_called_once_with(
            recurring_run_id='rr-123')
        self.client_mock.get_recurring_run.assert_called_once_with(
            recurring_run_id='rr-123')

    def test_recurring_run_disable(self):
        self.client_mock.disable_recurring_run.return_value = None
        rr_mock = mock.MagicMock(recurring_run_id='rr-123')
        self.client_mock.get_recurring_run.return_value = rr_mock
        result = self.invoke(['recurring-run', 'disable', 'rr-123'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.disable_recurring_run.assert_called_once_with(
            recurring_run_id='rr-123')
        self.client_mock.get_recurring_run.assert_called_once_with(
            recurring_run_id='rr-123')


if __name__ == '__main__':
    unittest.main()
