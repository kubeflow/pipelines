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
"""Tests for kfp.cli.diagnose_me_cli."""

import json
import unittest
from unittest import mock

from click import testing
from kfp.cli import diagnose_me_cli
from kfp.cli.diagnose_me import gcp


def make_executor_response(has_error=False,
                           json_output=None,
                           parsed_output='',
                           stderr=''):
    """Helper to create mock ExecutorResponse."""
    m = mock.MagicMock()
    m.has_error = has_error
    m.json_output = json_output or {}
    m.parsed_output = parsed_output
    m.stderr = stderr
    return m


class TestDiagnoseMeCLI(unittest.TestCase):

    def setUp(self):
        self.runner = testing.CliRunner()

    def invoke(self, args, catch_exceptions=False):
        return self.runner.invoke(
            diagnose_me_cli.diagnose_me,
            args,
            catch_exceptions=catch_exceptions,
        )

    @mock.patch('kfp.cli.diagnose_me_cli.gcp.get_gcp_configuration')
    def test_diagnose_me_missing_gcloud(self, mock_get_gcp):
        mock_get_gcp.return_value = make_executor_response(json_output={})
        result = self.invoke([], catch_exceptions=True)
        self.assertEqual(result.exit_code, 1)
        self.assertIn('gcloud, gsutil and kubectl are required',
                      str(result.exception))

    def _make_side_effect(self, validation_response, gcp_responses,
                          k8s_responses, dev_responses):
        """Create side_effect that returns responses in order and then
        cycles."""
        all_responses = [validation_response
                        ] + gcp_responses + k8s_responses + dev_responses

        def side_effect(*args, **kwargs):
            return all_responses.pop(0)

        return side_effect

    @mock.patch('kfp.cli.diagnose_me_cli.gcp.get_gcp_configuration')
    @mock.patch(
        'kfp.cli.diagnose_me_cli.kubernetes_cluster.get_kubectl_configuration')
    @mock.patch('kfp.cli.diagnose_me_cli.dev_env.get_dev_env_configuration')
    def test_diagnose_me_human_readable(self, mock_dev_env, mock_k8s, mock_gcp):
        validation_response = make_executor_response(json_output={
            'Google Cloud SDK': '1.0',
            'gsutil': '1.0',
            'kubectl': '1.0'
        })

        # gcp.Commands has 12 values, k8.Commands has 8, dev_env.Commands has 6
        gcp_responses = [
            make_executor_response(
                json_output={'test': f'gcp{i}'},
                parsed_output=f'gcp output {i}') for i in range(12)
        ]
        k8s_responses = [
            make_executor_response(
                json_output={'test': f'k8s{i}'},
                parsed_output=f'k8s output {i}') for i in range(8)
        ]
        dev_responses = [
            make_executor_response(
                json_output={'test': f'dev{i}'},
                parsed_output=f'dev output {i}') for i in range(6)
        ]

        mock_gcp.side_effect = self._make_side_effect(validation_response,
                                                      gcp_responses,
                                                      k8s_responses,
                                                      dev_responses)
        mock_k8s.side_effect = k8s_responses[:]  # copy
        mock_dev_env.side_effect = dev_responses[:]

        result = self.invoke([])
        self.assertEqual(result.exit_code, 0)
        self.assertIn('================', result.output)

    @mock.patch('kfp.cli.diagnose_me_cli.gcp.get_gcp_configuration')
    @mock.patch(
        'kfp.cli.diagnose_me_cli.kubernetes_cluster.get_kubectl_configuration')
    @mock.patch('kfp.cli.diagnose_me_cli.dev_env.get_dev_env_configuration')
    def test_diagnose_me_json(self, mock_dev_env, mock_k8s, mock_gcp):
        validation_response = make_executor_response(json_output={
            'Google Cloud SDK': '1.0',
            'gsutil': '1.0',
            'kubectl': '1.0'
        })

        gcp_responses = [
            make_executor_response(
                json_output={'test': f'gcp{i}'},
                parsed_output=f'gcp output {i}') for i in range(12)
        ]
        k8s_responses = [
            make_executor_response(
                json_output={'test': f'k8s{i}'},
                parsed_output=f'k8s output {i}') for i in range(8)
        ]
        dev_responses = [
            make_executor_response(
                json_output={'test': f'dev{i}'},
                parsed_output=f'dev output {i}') for i in range(6)
        ]

        mock_gcp.side_effect = self._make_side_effect(validation_response,
                                                      gcp_responses,
                                                      k8s_responses,
                                                      dev_responses)
        mock_k8s.side_effect = k8s_responses[:]
        mock_dev_env.side_effect = dev_responses[:]

        result = self.invoke(['--json'])
        self.assertEqual(result.exit_code, 0)
        # Output has "Collecting diagnostic information ...\n" prefix
        json_start = result.output.find('{')
        output_data = json.loads(result.output[json_start:])
        self.assertIn('GET_GCLOUD_VERSION', output_data)

    @mock.patch('kfp.cli.diagnose_me_cli.gcp.get_gcp_configuration')
    @mock.patch(
        'kfp.cli.diagnose_me_cli.kubernetes_cluster.get_kubectl_configuration')
    @mock.patch('kfp.cli.diagnose_me_cli.dev_env.get_dev_env_configuration')
    def test_diagnose_me_with_error(self, mock_dev_env, mock_k8s, mock_gcp):
        validation_response = make_executor_response(json_output={
            'Google Cloud SDK': '1.0',
            'gsutil': '1.0',
            'kubectl': '1.0'
        })

        gcp_responses = [
            make_executor_response(has_error=True, stderr='command failed'),
            make_executor_response(
                json_output={'test': 'gcp1'}, parsed_output='gcp output 1'), *[
                    make_executor_response(
                        json_output={'test': f'gcp{i}'},
                        parsed_output=f'gcp output {i}') for i in range(2, 12)
                ]
        ]
        k8s_responses = [
            make_executor_response(
                json_output={'test': f'k8s{i}'},
                parsed_output=f'k8s output {i}') for i in range(8)
        ]
        dev_responses = [
            make_executor_response(
                json_output={'test': f'dev{i}'},
                parsed_output=f'dev output {i}') for i in range(6)
        ]

        mock_gcp.side_effect = self._make_side_effect(validation_response,
                                                      gcp_responses,
                                                      k8s_responses,
                                                      dev_responses)
        mock_k8s.side_effect = k8s_responses[:]
        mock_dev_env.side_effect = dev_responses[:]

        result = self.invoke(['--json'])
        self.assertEqual(result.exit_code, 0)
        # Check JSON output for error
        json_start = result.output.find('{')
        output_data = json.loads(result.output[json_start:])
        # Find the key with error
        error_found = any(
            'Following error occurred during the diagnoses' in str(v)
            for v in output_data.values())
        self.assertTrue(error_found)

    @mock.patch('kfp.cli.diagnose_me_cli.gcp.get_gcp_configuration')
    @mock.patch(
        'kfp.cli.diagnose_me_cli.kubernetes_cluster.get_kubectl_configuration')
    @mock.patch('kfp.cli.diagnose_me_cli.dev_env.get_dev_env_configuration')
    def test_diagnose_me_with_project_and_namespace(self, mock_dev_env,
                                                    mock_k8s, mock_gcp):
        validation_response = make_executor_response(json_output={
            'Google Cloud SDK': '1.0',
            'gsutil': '1.0',
            'kubectl': '1.0'
        })

        gcp_responses = [
            make_executor_response(
                json_output={'test': f'gcp{i}'},
                parsed_output=f'gcp output {i}') for i in range(12)
        ]
        k8s_responses = [
            make_executor_response(
                json_output={'test': f'k8s{i}'},
                parsed_output=f'k8s output {i}') for i in range(8)
        ]
        dev_responses = [
            make_executor_response(
                json_output={'test': f'dev{i}'},
                parsed_output=f'dev output {i}') for i in range(6)
        ]

        mock_gcp.side_effect = self._make_side_effect(validation_response,
                                                      gcp_responses,
                                                      k8s_responses,
                                                      dev_responses)
        mock_k8s.side_effect = k8s_responses[:]
        mock_dev_env.side_effect = dev_responses[:]

        result = self.invoke(
            ['--project-id', 'my-project', '--namespace', 'my-namespace'])
        self.assertEqual(result.exit_code, 0)
        # Verify the project_id was passed to gcp calls
        for call in mock_gcp.call_args_list:
            self.assertEqual(call.kwargs.get('project_id'), 'my-project')


if __name__ == '__main__':
    unittest.main()
