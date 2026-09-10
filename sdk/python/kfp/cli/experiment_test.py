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
"""Tests for kfp.cli.experiment."""

import functools
import unittest
from unittest import mock

from click import testing
from kfp.cli import cli


class TestExperimentCommands(unittest.TestCase):

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

    def test_experiment_create(self):
        self.client_mock.create_experiment.return_value = mock.MagicMock(
            experiment_id='exp-123')
        result = self.invoke(
            ['experiment', 'create', '--description', 'desc', 'my-exp'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.create_experiment.assert_called_once_with(
            'my-exp', description='desc')

    def test_experiment_create_no_description(self):
        self.client_mock.create_experiment.return_value = mock.MagicMock(
            experiment_id='exp-123')
        result = self.invoke(['experiment', 'create', 'my-exp'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.create_experiment.assert_called_once_with(
            'my-exp', description=None)

    def test_experiment_list(self):
        self.client_mock.list_experiments.return_value = mock.MagicMock(
            experiments=[])
        result = self.invoke(['experiment', 'list'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.list_experiments.assert_called_once_with(
            page_token='',
            page_size=100,
            sort_by='created_at desc',
            filter=None)

    def test_experiment_list_with_options(self):
        self.client_mock.list_experiments.return_value = mock.MagicMock(
            experiments=[])
        result = self.invoke([
            'experiment', 'list', '--page-token', 'token123', '--max-size',
            '50', '--sort-by', 'name', '--filter', 'name=test'
        ])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.list_experiments.assert_called_once_with(
            page_token='token123',
            page_size=50,
            sort_by='name',
            filter='name=test')

    def test_experiment_get(self):
        self.client_mock.get_experiment.return_value = mock.MagicMock(
            experiment_id='exp-123')
        result = self.invoke(['experiment', 'get', 'exp-123'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.get_experiment.assert_called_once_with('exp-123')

    def test_experiment_delete(self):
        self.client_mock.delete_experiment.return_value = None
        result = self.invoke(['experiment', 'delete', 'exp-123'],
                             input_data='y\n')
        self.assertEqual(result.exit_code, 0)
        self.client_mock.delete_experiment.assert_called_once_with('exp-123')

    def test_experiment_archive_with_id(self):
        self.client_mock.archive_experiment.return_value = None
        exp_mock = mock.MagicMock(experiment_id='exp-123')
        self.client_mock.get_experiment.return_value = exp_mock
        result = self.invoke(
            ['experiment', 'archive', '--experiment-id', 'exp-123'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.archive_experiment.assert_called_once_with(
            experiment_id='exp-123')
        self.assertEqual(self.client_mock.get_experiment.call_count, 1)
        self.client_mock.get_experiment.assert_called_with(
            experiment_id='exp-123')

    def test_experiment_archive_with_name(self):
        self.client_mock.archive_experiment.return_value = None
        exp_mock = mock.MagicMock(experiment_id='exp-123')
        self.client_mock.get_experiment.return_value = exp_mock
        result = self.invoke(
            ['experiment', 'archive', '--experiment-name', 'my-exp'])
        self.assertEqual(result.exit_code, 0)
        self.assertEqual(self.client_mock.get_experiment.call_count, 2)
        self.client_mock.get_experiment.assert_any_call(
            experiment_name='my-exp')
        self.client_mock.get_experiment.assert_any_call(experiment_id='exp-123')
        self.client_mock.archive_experiment.assert_called_once_with(
            experiment_id='exp-123')

    def test_experiment_archive_requires_either(self):
        result = self.invoke(['experiment', 'archive'], catch_exceptions=True)
        self.assertEqual(result.exit_code, 1)
        self.assertIn('Either --experiment-id or --experiment-name is required',
                      str(result.exception))

    def test_experiment_archive_not_both(self):
        result = self.invoke([
            'experiment', 'archive', '--experiment-id', 'exp-1',
            '--experiment-name', 'exp-2'
        ],
                             catch_exceptions=True)
        self.assertEqual(result.exit_code, 1)
        self.assertIn('Either --experiment-id or --experiment-name is required',
                      str(result.exception))

    def test_experiment_unarchive_with_id(self):
        self.client_mock.unarchive_experiment.return_value = None
        exp_mock = mock.MagicMock(experiment_id='exp-123')
        self.client_mock.get_experiment.return_value = exp_mock
        result = self.invoke(
            ['experiment', 'unarchive', '--experiment-id', 'exp-123'])
        self.assertEqual(result.exit_code, 0)
        self.client_mock.unarchive_experiment.assert_called_once_with(
            experiment_id='exp-123')
        self.assertEqual(self.client_mock.get_experiment.call_count, 1)
        self.client_mock.get_experiment.assert_called_with(
            experiment_id='exp-123')

    def test_experiment_unarchive_with_name(self):
        self.client_mock.unarchive_experiment.return_value = None
        exp_mock = mock.MagicMock(experiment_id='exp-123')
        self.client_mock.get_experiment.return_value = exp_mock
        result = self.invoke(
            ['experiment', 'unarchive', '--experiment-name', 'my-exp'])
        self.assertEqual(result.exit_code, 0)
        self.assertEqual(self.client_mock.get_experiment.call_count, 2)
        self.client_mock.get_experiment.assert_any_call(
            experiment_name='my-exp')
        self.client_mock.get_experiment.assert_any_call(experiment_id='exp-123')
        self.client_mock.unarchive_experiment.assert_called_once_with(
            experiment_id='exp-123')

    def test_experiment_unarchive_requires_either(self):
        result = self.invoke(['experiment', 'unarchive'], catch_exceptions=True)
        self.assertEqual(result.exit_code, 1)
        self.assertIn('Either --experiment-id or --experiment-name is required',
                      str(result.exception))


if __name__ == '__main__':
    unittest.main()
