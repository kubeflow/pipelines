# Copyright 2022 The Kubeflow Authors
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

import json
import os
import tempfile
import textwrap
import unittest
from unittest.mock import MagicMock
from unittest.mock import Mock
from unittest.mock import patch

from absl.testing import parameterized
from google.protobuf import json_format
from kfp.client import client
from kfp.compiler import Compiler
from kfp.dsl import component
from kfp.dsl import pipeline
from kfp.pipeline_spec import pipeline_spec_pb2
import kfp_server_api
import yaml


class TestValidatePipelineName(parameterized.TestCase):

    @parameterized.parameters([
        'pipeline',
        'my-pipeline',
        'my-pipeline-1',
        '1pipeline',
        'pipeline1',
        'my_pipeline',
        "person's-pipeline",
        'my pipeline',
        'pipeline.yaml',
    ])
    def test_valid(self, name: str):
        client.validate_pipeline_display_name(name)

    @parameterized.parameters(['', '   ', '\t'])
    def test_invalid(self, name: str):
        with self.assertRaisesRegex(
                ValueError,
                'Invalid pipeline name. Pipeline name cannot be empty or contain only whitespace.'
        ):
            client.validate_pipeline_display_name(name)


class TestOverrideCachingOptions(parameterized.TestCase):

    def test_override_caching_of_multiple_components(self):

        @component
        def hello_word(text: str) -> str:
            return text

        @component
        def to_lower(text: str) -> str:
            return text.lower()

        @pipeline(
            name='sample two-step pipeline',
            description='a minimal two-step pipeline')
        def pipeline_with_two_component(text: str = 'hi there'):

            component_1 = hello_word(text=text).set_caching_options(True)
            component_2 = to_lower(
                text=component_1.output).set_caching_options(False)

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'hello_world_pipeline.yaml')
            Compiler().compile(
                pipeline_func=pipeline_with_two_component,
                package_path=temp_filepath)

            with open(temp_filepath, 'r') as f:
                pipeline_obj = yaml.safe_load(f)
                pipeline_spec = json_format.ParseDict(
                    pipeline_obj, pipeline_spec_pb2.PipelineSpec())
                client._override_caching_options(pipeline_spec, True)
                pipeline_obj = json_format.MessageToDict(pipeline_spec)
                self.assertTrue(pipeline_obj['root']['dag']['tasks']
                                ['hello-word']['cachingOptions']['enableCache'])
                self.assertTrue(pipeline_obj['root']['dag']['tasks']['to-lower']
                                ['cachingOptions']['enableCache'])


class TestExtractPipelineYAML(parameterized.TestCase):

    def test_extract_pipeline_yaml_single_doc(self):

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'single_doc_pipeline.yaml')
            with open(temp_filepath, 'w') as f:
                f.write(
                    textwrap.dedent('''
                        components:
                          comp-foo:
                            executorLabel: exec-foo
                        deploymentSpec:
                          executors:
                            exec-foo:
                              container:
                                command:
                                - sh
                                - -c
                                - cat /data/file.txt
                                image: alpine
                        pipelineInfo:
                          name: my-pipeline
                        root:
                          dag:
                            tasks:
                              foo:
                                componentRef:
                                  name: comp-foo
                                taskInfo:
                                  name: foo
                        schemaVersion: 2.1.0
                        sdkVersion: kfp-2.0.0-beta.13
                        '''))

            pipeline_dict = client._extract_pipeline_yaml(
                temp_filepath).to_dict()
            self.assertEqual('my-pipeline',
                             pipeline_dict['pipelineInfo']['name'])

    def test_extract_pipeline_yaml_multiple_docs(self):

        with tempfile.TemporaryDirectory() as tempdir:
            temp_filepath = os.path.join(tempdir, 'multi_docs_pipeline.yaml')
            with open(temp_filepath, 'w') as f:
                f.write(
                    textwrap.dedent('''
                        components:
                          comp-foo:
                            executorLabel: exec-foo
                        deploymentSpec:
                          executors:
                            exec-foo:
                              container:
                                command:
                                - sh
                                - -c
                                - cat /data/file.txt
                                image: alpine
                        pipelineInfo:
                          name: my-pipeline
                        root:
                          dag:
                            tasks:
                              foo:
                                componentRef:
                                  name: comp-foo
                                taskInfo:
                                  name: foo
                        schemaVersion: 2.1.0
                        sdkVersion: kfp-2.0.0-beta.13
                        ---
                        platforms:
                          kubernetes:
                            deploymentSpec:
                              executors:
                                exec-foo:
                                  pvcMount:
                                  - mountPath: /data
                                    constant: my-pvc
                        '''))

            pipeline_dict = client._extract_pipeline_yaml(
                temp_filepath).to_dict()
            self.assertEqual(
                'my-pipeline',
                pipeline_dict['pipeline_spec']['pipelineInfo']['name'])
            self.assertEqual(
                'my-pvc', pipeline_dict['platform_spec']['platforms']
                ['kubernetes']['deploymentSpec']['executors']['exec-foo']
                ['pvcMount'][0]['constant'])


class TestClient(parameterized.TestCase):

    def setUp(self):
        self.client = client.Client(namespace='ns1')

    def test__is_ipython_return_false(self):
        mock = MagicMock()
        with patch.dict('sys.modules', IPython=mock):
            mock.get_ipython.return_value = None
            self.assertFalse(self.client._is_ipython())

    def test__is_ipython_return_true(self):
        mock = MagicMock()
        with patch.dict('sys.modules', IPython=mock):
            mock.get_ipython.return_value = 'Something'
            self.assertTrue(self.client._is_ipython())

    def test__is_ipython_should_raise_error(self):
        mock = MagicMock()
        with patch.dict('sys.modules', mock):
            mock.side_effect = ImportError
            self.assertFalse(self.client._is_ipython())

    def test_wait_for_run_completion_invalid_token_should_raise_error(self):
        with self.assertRaises(kfp_server_api.ApiException):
            with patch.object(
                    self.client._run_api,
                    'get_run',
                    side_effect=kfp_server_api.ApiException) as mock_get_run:
                self.client.wait_for_run_completion(
                    run_id='foo', timeout=1, sleep_duration=0)
                mock_get_run.assert_called_once()

    def test_wait_for_run_completion_expired_access_token(self):
        with patch.object(self.client._run_api, 'get_run') as mock_get_run:
            # We need to iterate through multiple side effects in order to test this logic.
            mock_get_run.side_effect = [
                Mock(state='unknown state'),
                kfp_server_api.ApiException(status=401),
                Mock(state='succeeded'),
            ]

            with patch.object(self.client, '_refresh_api_client_token'
                             ) as mock_refresh_api_client_token:
                self.client.wait_for_run_completion(
                    run_id='foo', timeout=1, sleep_duration=0)
                mock_get_run.assert_called_with(run_id='foo')
                mock_refresh_api_client_token.assert_called_once()

    def test_wait_for_run_completion_valid_token(self):
        with patch.object(self.client._run_api, 'get_run') as mock_get_run:
            mock_get_run.return_value = Mock(state='succeeded')
            response = self.client.wait_for_run_completion(
                run_id='foo', timeout=1, sleep_duration=0)
            mock_get_run.assert_called_once_with(run_id='foo')
            assert response == mock_get_run.return_value

    def test_wait_for_run_completion_run_timeout_should_raise_error(self):
        with self.assertRaises(TimeoutError):
            with patch.object(self.client._run_api, 'get_run') as mock_get_run:
                mock_get_run.return_value = Mock(run=Mock(status='foo'))
                self.client.wait_for_run_completion(
                    run_id='foo', timeout=1, sleep_duration=0)
                mock_get_run.assert_called_once_with(run_id='foo')

    @patch('kfp.Client.get_experiment', side_effect=ValueError)
    def test_create_experiment_no_experiment_should_raise_error(
            self, mock_get_experiment):
        with self.assertRaises(ValueError):
            self.client.create_experiment(name='foo', namespace='ns1')
            mock_get_experiment.assert_called_once_with(
                name='foo', namespace='ns1')

    @patch('kfp.Client.get_experiment', return_value=Mock(id='foo'))
    @patch('kfp.Client._get_url_prefix', return_value='/pipeline')
    def test_create_experiment_existing_experiment(self, mock_get_url_prefix,
                                                   mock_get_experiment):
        self.client.create_experiment(name='foo')
        mock_get_experiment.assert_called_once_with(
            experiment_name='foo', namespace='ns1')
        mock_get_url_prefix.assert_called_once()

    @patch('kfp_server_api.V2beta1Experiment')
    @patch(
        'kfp.Client.get_experiment',
        side_effect=ValueError('No experiment is found with name'))
    @patch('kfp.Client._get_url_prefix', return_value='/pipeline')
    def test__create_experiment_name_not_found(self, mock_get_url_prefix,
                                               mock_get_experiment,
                                               mock_api_experiment):
        # experiment with the specified name is not found, so a new experiment
        # is created.
        with patch.object(
                self.client._experiment_api,
                'create_experiment',
                return_value=Mock(
                    experiment_id='foo')) as mock_create_experiment:
            self.client.create_experiment(name='foo')
            mock_get_experiment.assert_called_once_with(
                experiment_name='foo', namespace='ns1')
            mock_api_experiment.assert_called_once()
            mock_create_experiment.assert_called_once()
            mock_get_url_prefix.assert_called_once()

    def test_get_experiment_no_experiment_id_or_name_should_raise_error(self):
        with self.assertRaises(ValueError):
            self.client.get_experiment()

    @patch('kfp.Client.get_user_namespace', return_value=None)
    def test_get_experiment_does_not_exist_should_raise_error(
            self, mock_get_user_namespace):
        with self.assertRaises(ValueError):
            with patch.object(
                    self.client._experiment_api,
                    'list_experiments',
                    return_value=Mock(
                        experiments=None)) as mock_list_experiments:
                self.client.get_experiment(experiment_name='foo')
                mock_list_experiments.assert_called_once()
                mock_get_user_namespace.assert_called_once()

    @patch('kfp.Client.get_user_namespace', return_value=None)
    def test_get_experiment_multiple_experiments_with_name_should_raise_error(
            self, mock_get_user_namespace):
        with self.assertRaises(ValueError):
            with patch.object(
                    self.client._experiment_api,
                    'list_experiments',
                    return_value=Mock(
                        experiments=['foo', 'foo'])) as mock_list_experiments:
                self.client.get_experiment(experiment_name='foo')
                mock_list_experiments.assert_called_once()
                mock_get_user_namespace.assert_called_once()

    def test_get_experiment_with_experiment_id(self):
        with patch.object(self.client._experiment_api,
                          'get_experiment') as mock_get_experiment:
            self.client.get_experiment(experiment_id='foo')
            mock_get_experiment.assert_called_once_with(experiment_id='foo')

    def test_get_experiment_with_experiment_name_and_namespace(self):
        with patch.object(self.client._experiment_api,
                          'list_experiments') as mock_list_experiments:
            self.client.get_experiment(experiment_name='foo', namespace='ns1')
            mock_list_experiments.assert_called_once()

    @patch('kfp.Client.get_user_namespace', return_value=None)
    def test_get_experiment_with_experiment_name_and_no_namespace(
            self, mock_get_user_namespace):
        with patch.object(self.client._experiment_api,
                          'list_experiments') as mock_list_experiments:
            self.client.get_experiment(experiment_name='foo')
            mock_list_experiments.assert_called_once()
            mock_get_user_namespace.assert_called_once()

    @patch('kfp_server_api.HealthzServiceApi.get_healthz')
    def test_get_kfp_healthz(self, mock_get_kfp_healthz):
        mock_get_kfp_healthz.return_value = json.dumps([{'foo': 'bar'}])
        response = self.client.get_kfp_healthz()
        mock_get_kfp_healthz.assert_called_once()
        assert (response == mock_get_kfp_healthz.return_value)

    @patch(
        'kfp_server_api.HealthzServiceApi.get_healthz',
        side_effect=kfp_server_api.ApiException)
    def test_get_kfp_healthz_should_raise_error(self, mock_get_kfp_healthz):
        with self.assertRaises(TimeoutError):
            self.client.get_kfp_healthz(sleep_duration=0)
            mock_get_kfp_healthz.assert_called()

    def test_upload_pipeline_without_name(self):

        @component
        def return_bool(boolean: bool) -> bool:
            return boolean

        @pipeline(name='test-upload-without-name', description='description')
        def pipeline_test_upload_without_name(boolean: bool = True):
            return_bool(boolean=boolean)

        with patch.object(self.client._upload_api,
                          'upload_pipeline') as mock_upload_pipeline:
            with patch.object(self.client, '_is_ipython', return_value=False):
                with tempfile.TemporaryDirectory() as tmp_path:
                    pipeline_test_path = os.path.join(tmp_path, 'test.yaml')
                    Compiler().compile(
                        pipeline_func=pipeline_test_upload_without_name,
                        package_path=pipeline_test_path)
                    self.client.upload_pipeline(
                        pipeline_package_path=pipeline_test_path,
                        description='description',
                        namespace='ns1')
                    mock_upload_pipeline.assert_called_once_with(
                        pipeline_test_path,
                        name='test-upload-without-name',
                        description='description',
                        namespace='ns1')

    @parameterized.parameters([
        'pipeline',
        'my-pipeline',
        'my-pipeline-1',
        '1pipeline',
        'pipeline1',
        'my_pipeline',
        "person's-pipeline",
        'my pipeline',
        'pipeline.yaml',
    ])
    def test_upload_pipeline_with_name(self, pipeline_name):
        with patch.object(self.client._upload_api,
                          'upload_pipeline') as mock_upload_pipeline:
            with patch.object(self.client, '_is_ipython', return_value=False):
                self.client.upload_pipeline(
                    pipeline_package_path='fake.yaml',
                    pipeline_name=pipeline_name,
                    description='description',
                    namespace='ns1')
                mock_upload_pipeline.assert_called_once_with(
                    'fake.yaml',
                    name=pipeline_name,
                    description='description',
                    namespace='ns1')

    @parameterized.parameters([
        '',
        '   ',
        '\t',
    ])
    def test_upload_pipeline_with_name_invalid(self, pipeline_name):
        with patch.object(self.client._upload_api,
                          'upload_pipeline') as mock_upload_pipeline:
            with patch.object(self.client, '_is_ipython', return_value=False):
                with self.assertRaisesRegex(
                        ValueError,
                        'Invalid pipeline name. Pipeline name cannot be empty or contain only whitespace.'
                ):
                    self.client.upload_pipeline(
                        pipeline_package_path='fake.yaml',
                        pipeline_name=pipeline_name,
                        description='description',
                        namespace='ns1')


if __name__ == '__main__':
    unittest.main()
