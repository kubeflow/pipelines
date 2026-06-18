# Copyright The Kubeflow Authors
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
"""Unit tests for PipelinesClient."""

import os
import unittest
from unittest.mock import Mock
from unittest.mock import patch

from absl.testing import parameterized
from kfp.kubeflow_client import constants
from kfp.kubeflow_client.api.pipelines_client import PipelinesClient
from kfp.kubeflow_client.backends.kubernetes.types import \
    KubernetesBackendConfig
import kfp_server_api

_AUTH_MODULE = 'kfp.kubeflow_client.backends.kubernetes.auth'

_BACKEND_MODULE = 'kfp.kubeflow_client.backends.kubernetes.backend'


def _make_client(base_url='http://localhost:8888', namespace='test-ns'):
    """Create a PipelinesClient with mocked credentials for testing."""
    with patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials'), \
         patch(f'{_BACKEND_MODULE}.KubernetesBackend.verify_backend'):
        return PipelinesClient(
            backend_config=KubernetesBackendConfig(
                base_url=base_url,
                namespace=namespace,
            ))


class TestPipelineOperations(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_get_pipeline_not_found_raises_value_error(self):
        with patch.object(
                self.client._backend.pipelines_api,
                'pipeline_service_list_pipelines',
                return_value=Mock(pipelines=[])):
            with self.assertRaisesRegex(ValueError, 'Pipeline not found'):
                self.client.get_pipeline('nonexistent')

    def test_get_pipeline_success(self):
        mock_pipeline = Mock(pipeline_id='pid-1', display_name='my-pipe')
        with patch.object(
                self.client._backend.pipelines_api,
                'pipeline_service_list_pipelines',
                return_value=Mock(pipelines=[mock_pipeline])):
            with patch.object(
                    self.client._backend.pipelines_api,
                    'pipeline_service_get_pipeline',
                    return_value=mock_pipeline) as mock_get:
                result = self.client.get_pipeline('my-pipe')
                mock_get.assert_called_once_with(pipeline_id='pid-1')
                self.assertEqual(result.pipeline_id, 'pid-1')

    def test_list_pipelines(self):
        mock_response = Mock(pipelines=[Mock()], next_page_token='tok')
        with patch.object(
                self.client._backend.pipelines_api,
                'pipeline_service_list_pipelines',
                return_value=mock_response) as mock_list:
            result = self.client.list_pipelines(page_size=5)
            mock_list.assert_called_once_with(
                namespace='test-ns', page_token='', page_size=5)
            self.assertEqual(result.next_page_token, 'tok')

    def test_delete_pipeline_single_version_no_force(self):
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[Mock()])):
                with patch.object(
                        self.client._backend.pipelines_api,
                        'pipeline_service_delete_pipeline') as mock_del:
                    self.client.delete_pipeline('my-pipe', force=False)
                    mock_del.assert_called_once_with(
                        pipeline_id='pid-1', cascade=True)

    def test_delete_pipeline_multiple_versions_no_force_raises(self):
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[Mock(), Mock()])):
                with self.assertRaisesRegex(ValueError, 'multiple versions'):
                    self.client.delete_pipeline('my-pipe', force=False)

    def test_delete_pipeline_multiple_versions_with_force(self):
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(self.client._backend.pipelines_api,
                              'pipeline_service_delete_pipeline') as mock_del:
                self.client.delete_pipeline('my-pipe', force=True)
                mock_del.assert_called_once_with(
                    pipeline_id='pid-1', cascade=True)

    def test_delete_pipeline_specific_version(self):
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend,
                    '_get_version_id_by_name',
                    return_value='vid-1'):
                with patch.object(
                        self.client._backend.pipelines_api,
                        'pipeline_service_delete_pipeline_version',
                ) as mock_del:
                    self.client.delete_pipeline('my-pipe', version='v1')
                    mock_del.assert_called_once_with(
                        pipeline_id='pid-1', pipeline_version_id='vid-1')

    def test_delete_pipeline_nonexistent_version_raises(self):
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend,
                    '_get_version_id_by_name',
                    return_value=None):
                with self.assertRaisesRegex(ValueError,
                                            'Pipeline version not found'):
                    self.client.delete_pipeline('my-pipe', version='bad-ver')

    def test_delete_pipeline_not_found_raises(self):
        with patch.object(
                self.client._backend, '_get_pipeline_id_by_name',
                return_value=None):
            with self.assertRaisesRegex(ValueError, 'Pipeline not found'):
                self.client.delete_pipeline('ghost-pipe')


class TestRunOperations(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_run_callable_dispatches_to_inline(self):
        mock_run = Mock(run_id='r-1', state='PENDING')
        with patch.object(
                self.client, '_run_inline',
                return_value=mock_run) as mock_inline:
            with patch.object(
                    self.client._backend,
                    '_get_experiment_id_by_name',
                    return_value='exp-1'):

                def fake_pipeline():
                    pass

                fake_pipeline.name = 'test-pipe'
                result = self.client.run(
                    fake_pipeline, params={'x': '1'}, name='my-run')
                mock_inline.assert_called_once()
                self.assertEqual(result.run_id, 'r-1')

    def test_run_by_name_resolves_latest_version(self):
        mock_run = Mock(run_id='r-2')
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend,
                    '_get_latest_version_id',
                    return_value='vid-latest'):
                with patch.object(
                        self.client._backend,
                        '_run_from_version_reference',
                        return_value=mock_run) as mock_ref:
                    result = self.client.run('my-pipe', name='run-1')
                    mock_ref.assert_called_once_with(
                        pipeline_id='pid-1',
                        version_id='vid-latest',
                        params=None,
                        run_name='run-1',
                        experiment_id=None,
                    )
                    self.assertEqual(result.run_id, 'r-2')

    def test_run_by_name_with_version(self):
        mock_run = Mock(run_id='r-3')
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend,
                    '_get_version_id_by_name',
                    return_value='vid-specific'):
                with patch.object(
                        self.client._backend,
                        '_run_from_version_reference',
                        return_value=mock_run) as mock_ref:
                    result = self.client.run(
                        'my-pipe', version='v2', name='run-2')
                    mock_ref.assert_called_once_with(
                        pipeline_id='pid-1',
                        version_id='vid-specific',
                        params=None,
                        run_name='run-2',
                        experiment_id=None,
                    )

    def test_run_by_name_nonexistent_version_raises(self):
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend,
                    '_get_version_id_by_name',
                    return_value=None):
                with patch.object(
                        self.client._backend,
                        '_get_experiment_id_by_name',
                        return_value='exp-1'):
                    with self.assertRaisesRegex(ValueError,
                                                'version not found'):
                        self.client.run('my-pipe', version='bad')

    def test_run_archive_file_raises(self):
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value='exp-1'):
            with self.assertRaisesRegex(ValueError, 'Archive files'):
                self.client.run('/path/to/file.tar.gz')

    def test_run_pipeline_version_object(self):
        pv = Mock(
            spec=kfp_server_api.V2beta1PipelineVersion,
            pipeline_id='pid-1',
            pipeline_version_id='vid-1')
        mock_run = Mock(run_id='r-4')
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value='exp-1'):
            with patch.object(
                    self.client._backend,
                    '_run_from_version_reference',
                    return_value=mock_run) as mock_ref:
                result = self.client.run(pv, name='run-pv')
                mock_ref.assert_called_once()
                self.assertEqual(result.run_id, 'r-4')

    def test_run_pipeline_object_resolves_latest_version(self):
        pipeline_obj = Mock(
            spec=kfp_server_api.V2beta1Pipeline, pipeline_id='pid-1')
        mock_run = Mock(run_id='r-5')
        with patch.object(
                self.client._backend,
                '_get_latest_version_id',
                return_value='vid-latest') as mock_latest:
            with patch.object(
                    self.client._backend,
                    '_run_from_version_reference',
                    return_value=mock_run) as mock_ref:
                result = self.client.run(pipeline_obj, name='run-pipe')
                mock_latest.assert_called_once_with('pid-1')
                mock_ref.assert_called_once_with(
                    pipeline_id='pid-1',
                    version_id='vid-latest',
                    params=None,
                    run_name='run-pipe',
                    experiment_id=None,
                )
                self.assertEqual(result.run_id, 'r-5')

    def test_run_yaml_path_dispatches_to_run_from_file(self):
        mock_run = Mock(run_id='r-6')
        with patch.object(
                self.client._backend, '_run_from_file',
                return_value=mock_run) as mock_file:
            result = self.client.run('/tmp/my-pipeline.yaml', name='yaml-run')
            mock_file.assert_called_once_with(
                file_path='/tmp/my-pipeline.yaml',
                params=None,
                run_name='yaml-run',
                experiment_id=None,
            )
            self.assertEqual(result.run_id, 'r-6')

    def test_run_callable_with_version_logs_warning(self):
        mock_run = Mock(run_id='r-warn-1')
        pipeline_fn = Mock(__name__='my_pipe')
        with patch.object(self.client, '_run_inline', return_value=mock_run):
            with self.assertLogs(
                    'kfp.kubeflow_client.api.pipelines_client',
                    level='WARNING') as log:
                self.client.run(pipeline_fn, version='v2', name='run-w1')
            self.assertIn('version parameter is ignored', log.output[0])

    def test_run_yaml_path_with_version_logs_warning(self):
        mock_run = Mock(run_id='r-warn-2')
        with patch.object(
                self.client._backend, '_run_from_file', return_value=mock_run):
            with self.assertLogs(
                    'kfp.kubeflow_client.api.pipelines_client',
                    level='WARNING') as log:
                self.client.run('/tmp/pipe.yaml', version='v2', name='run-w2')
            self.assertIn('version parameter is ignored', log.output[0])

    def test_run_pipeline_version_obj_with_version_logs_warning(self):
        mock_run = Mock(run_id='r-warn-3')
        pv_obj = Mock(
            spec=kfp_server_api.V2beta1PipelineVersion,
            pipeline_id='pid-1',
            pipeline_version_id='vid-1')
        with patch.object(
                self.client._backend,
                '_run_from_version_reference',
                return_value=mock_run):
            with self.assertLogs(
                    'kfp.kubeflow_client.api.pipelines_client',
                    level='WARNING') as log:
                self.client.run(pv_obj, version='v2', name='run-w3')
            self.assertIn('version parameter is ignored', log.output[0])

    def test_run_pipeline_obj_with_version_resolves_named_version(self):
        pipeline_obj = Mock(
            spec=kfp_server_api.V2beta1Pipeline,
            pipeline_id='pid-1',
            display_name='my-pipe')
        mock_run = Mock(run_id='r-ver')
        with patch.object(
                self.client._backend,
                '_get_version_id_by_name',
                return_value='vid-named') as mock_ver:
            with patch.object(
                    self.client._backend,
                    '_run_from_version_reference',
                    return_value=mock_run) as mock_ref:
                result = self.client.run(
                    pipeline_obj, version='v2', name='run-ver')
                mock_ver.assert_called_once_with('pid-1', 'v2')
                mock_ref.assert_called_once_with(
                    pipeline_id='pid-1',
                    version_id='vid-named',
                    params=None,
                    run_name='run-ver',
                    experiment_id=None,
                )
                self.assertEqual(result.run_id, 'r-ver')

    def test_run_unsupported_type_raises(self):
        with self.assertRaisesRegex(ValueError, 'Unsupported pipeline'):
            self.client.run(12345)


class TestExperimentOperations(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_create_experiment(self):
        mock_exp = Mock(experiment_id='exp-1', display_name='my-exp')
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value=None):
            with patch.object(
                    self.client._backend.experiment_api,
                    'experiment_service_create_experiment',
                    return_value=mock_exp) as mock_create:
                result = self.client.create_experiment('my-exp')
                mock_create.assert_called_once()
                self.assertEqual(result.experiment_id, 'exp-1')

    def test_create_experiment_idempotent(self):
        existing = Mock(experiment_id='exp-1', display_name='my-exp')
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value='exp-1'):
            with patch.object(
                    self.client._backend.experiment_api,
                    'experiment_service_get_experiment',
                    return_value=existing):
                result = self.client.create_experiment('my-exp')
                self.assertEqual(result.experiment_id, 'exp-1')

    def test_get_experiment_not_found_raises(self):
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value=None):
            with self.assertRaisesRegex(ValueError, 'Experiment not found'):
                self.client.get_experiment('nonexistent')

    def test_get_experiment_success(self):
        mock_exp = Mock(experiment_id='exp-1', display_name='my-exp')
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value='exp-1'):
            with patch.object(
                    self.client._backend.experiment_api,
                    'experiment_service_get_experiment',
                    return_value=mock_exp):
                result = self.client.get_experiment('my-exp')
                self.assertEqual(result.experiment_id, 'exp-1')

    def test_delete_experiment(self):
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value='exp-1'):
            with patch.object(
                    self.client._backend.experiment_api,
                    'experiment_service_delete_experiment') as mock_del:
                self.client.delete_experiment('my-exp')
                mock_del.assert_called_once_with(experiment_id='exp-1')

    def test_delete_experiment_not_found_raises(self):
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value=None):
            with self.assertRaisesRegex(ValueError, 'Experiment not found'):
                self.client.delete_experiment('bad-exp')

    def test_list_experiments(self):
        mock_response = Mock(
            experiments=[Mock(experiment_id='e-1')], next_page_token='tok2')
        with patch.object(
                self.client._backend.experiment_api,
                'experiment_service_list_experiments',
                return_value=mock_response) as mock_list:
            result = self.client.list_experiments(page_size=5)
            mock_list.assert_called_once_with(
                namespace='test-ns', page_token='', page_size=5)
            self.assertEqual(result.next_page_token, 'tok2')


class TestWaitForRunStatus(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_returns_on_terminal_state(self):
        mock_run = Mock(run_id='r-1', state='SUCCEEDED')
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=mock_run):
            result = self.client.wait_for_run_status(
                'r-1', status={constants.RUN_COMPLETE}, polling_interval=0)
            self.assertEqual(result.state, 'SUCCEEDED')

    def test_raises_timeout_error(self):
        mock_run = Mock(run_id='r-1', state='RUNNING')
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=mock_run):
            with self.assertRaises(TimeoutError):
                self.client.wait_for_run_status(
                    'r-1', timeout=0, polling_interval=0)

    def test_callbacks_are_invoked(self):
        mock_run = Mock(run_id='r-1', state='SUCCEEDED')
        callback = Mock()
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=mock_run):
            self.client.wait_for_run_status(
                'r-1',
                status={constants.RUN_COMPLETE},
                polling_interval=0,
                callbacks=[callback],
            )
        callback.assert_called_once_with(mock_run)

    def test_accepts_run_object(self):
        run_obj = Mock(run_id='r-1', state='SUCCEEDED')
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=run_obj):
            result = self.client.wait_for_run_status(
                run_obj, status={constants.RUN_COMPLETE}, polling_interval=0)
            self.assertEqual(result.run_id, 'r-1')

    def test_terminal_state_not_in_target_still_returns(self):
        mock_run = Mock(run_id='r-1', state='FAILED')
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=mock_run):
            result = self.client.wait_for_run_status(
                'r-1', status={constants.RUN_COMPLETE}, polling_interval=0)
            self.assertEqual(result.state, 'FAILED')

    def test_token_refresh_on_401(self):
        mock_run_ok = Mock(run_id='r-1', state='SUCCEEDED')
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                side_effect=[
                    Mock(run_id='r-1', state='RUNNING'),
                    kfp_server_api.ApiException(status=401),
                    mock_run_ok,
                ]):
            with patch.object(self.client._backend,
                              'refresh_credentials') as mock_ref:
                result = self.client.wait_for_run_status(
                    'r-1',
                    status={constants.RUN_COMPLETE},
                    polling_interval=0,
                )
                mock_ref.assert_called_once()
                self.assertEqual(result.state, 'SUCCEEDED')

    def test_auth_retry_exhaustion_raises(self):
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                side_effect=[
                    Mock(run_id='r-1', state='RUNNING'),
                    kfp_server_api.ApiException(status=401),
                    kfp_server_api.ApiException(status=401),
                    kfp_server_api.ApiException(status=401),
                ]):
            with patch.object(self.client._backend, 'refresh_credentials'):
                with self.assertRaises(kfp_server_api.ApiException) as ctx:
                    self.client.wait_for_run_status(
                        'r-1',
                        status={constants.RUN_COMPLETE},
                        polling_interval=0,
                    )
                self.assertEqual(ctx.exception.status, 401)

    def test_first_poll_401_raises_without_refresh(self):
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                side_effect=kfp_server_api.ApiException(status=401)):
            with patch.object(self.client._backend,
                              'refresh_credentials') as mock_ref:
                with self.assertRaises(kfp_server_api.ApiException) as ctx:
                    self.client.wait_for_run_status(
                        'r-1',
                        status={constants.RUN_COMPLETE},
                        polling_interval=0,
                    )
                self.assertEqual(ctx.exception.status, 401)
                mock_ref.assert_not_called()

    def test_non_401_api_exception_propagates_immediately(self):
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                side_effect=[
                    Mock(run_id='r-1', state='RUNNING'),
                    kfp_server_api.ApiException(status=500),
                ]):
            with self.assertRaises(kfp_server_api.ApiException) as ctx:
                self.client.wait_for_run_status(
                    'r-1',
                    status={constants.RUN_COMPLETE},
                    polling_interval=0,
                )
            self.assertEqual(ctx.exception.status, 500)

    def test_callbacks_fire_on_timeout(self):
        mock_run = Mock(run_id='r-1', state='RUNNING')
        callback = Mock()
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=mock_run):
            with self.assertRaises(TimeoutError):
                self.client.wait_for_run_status(
                    'r-1',
                    timeout=0,
                    polling_interval=0,
                    callbacks=[callback],
                )
        callback.assert_called_once_with(mock_run)

    def test_callbacks_fire_on_terminal_non_target_state(self):
        mock_run = Mock(run_id='r-1', state='FAILED')
        callback = Mock()
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=mock_run):
            result = self.client.wait_for_run_status(
                'r-1',
                status={constants.RUN_COMPLETE},
                polling_interval=0,
                callbacks=[callback],
            )
        callback.assert_called_once_with(mock_run)
        self.assertEqual(result.state, 'FAILED')


class TestHelperMethods(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    @parameterized.parameters([
        ('my_pipeline', 'my-pipeline'),
        ('hello_world_pipe', 'hello-world-pipe'),
        ('already-dashed', 'already-dashed'),
    ])
    def test_infer_pipeline_name_from_function_name(self, func_name, expected):
        func = Mock()
        func.name = None
        func.__name__ = func_name
        func.__class__ = type('function', (), {})
        with patch('os.path.isfile', return_value=False):
            result = self.client._infer_pipeline_name(func, '/tmp/f.yaml')
        self.assertEqual(result, expected)

    def test_infer_pipeline_name_from_dsl_name(self):
        func = Mock()
        func.name = 'dsl-assigned-name'
        result = self.client._infer_pipeline_name(func, '/tmp/f.yaml')
        self.assertEqual(result, 'dsl-assigned-name')

    def test_infer_pipeline_name_from_file_path(self):
        result = self.client._infer_pipeline_name('/tmp/my-pipeline.yaml',
                                                  '/tmp/my-pipeline.yaml')
        self.assertEqual(result, 'my-pipeline')

    @parameterized.parameters(
        ['pipeline.yaml', 'pipeline.yml', 'path/to/spec.yaml'])
    def test_is_yaml_path(self, path):
        self.assertTrue(self.client._is_yaml_path(path))

    @parameterized.parameters(
        ['pipeline.tar.gz', 'pipeline.zip', 'pipeline-name', 'pipeline.tgz'])
    def test_is_not_yaml_path(self, path):
        self.assertFalse(self.client._is_yaml_path(path))

    def test_validate_pipeline_name_valid(self):
        PipelinesClient._validate_pipeline_name('my-pipeline')

    def test_validate_pipeline_name_empty_raises(self):
        with self.assertRaisesRegex(ValueError, 'cannot be empty'):
            PipelinesClient._validate_pipeline_name('')

    def test_validate_pipeline_name_whitespace_raises(self):
        with self.assertRaisesRegex(ValueError, 'cannot be empty'):
            PipelinesClient._validate_pipeline_name('   ')

    def test_get_latest_version_id(self):
        mock_version = Mock(
            pipeline_version_id='vid-latest', created_at='2026-01-01T00:00:00Z')
        with patch.object(
                self.client._backend.pipelines_api,
                'pipeline_service_list_pipeline_versions',
                return_value=Mock(pipeline_versions=[mock_version])):
            result = self.client._backend._get_latest_version_id('pid-1')
            self.assertEqual(result, 'vid-latest')

    def test_get_latest_version_id_no_versions_raises(self):
        with patch.object(
                self.client._backend.pipelines_api,
                'pipeline_service_list_pipeline_versions',
                return_value=Mock(pipeline_versions=[])):
            with self.assertRaisesRegex(ValueError, 'has no versions'):
                self.client._backend._get_latest_version_id('pid-1')

    def test_equals_filter(self):
        result = self.client._backend._equals_filter('display_name', 'test')
        expected = '{"predicates": [{"operation": "EQUALS", "key": "display_name", "stringValue": "test"}]}'
        import json
        self.assertEqual(json.loads(result), json.loads(expected))

    def test_get_pipeline_id_by_name_multiple_matches_raises(self):
        pipelines = [
            Mock(pipeline_id='pid-1'),
            Mock(pipeline_id='pid-2'),
        ]
        with patch.object(
                self.client._backend.pipelines_api,
                'pipeline_service_list_pipelines',
                return_value=Mock(pipelines=pipelines)):
            with self.assertRaisesRegex(ValueError, 'Multiple pipelines'):
                self.client._backend._get_pipeline_id_by_name('dup-name')

    def test_invoke_callbacks_wraps_exception_in_runtime_error(self):

        def bad_callback(run):
            raise ValueError('callback boom')

        mock_run = Mock(run_id='r-1', state='SUCCEEDED')
        with self.assertRaisesRegex(RuntimeError, 'callback boom'):
            PipelinesClient._invoke_callbacks([bad_callback], mock_run)

    def test_resolve_experiment_id_returns_none_when_not_specified(self):
        result = self.client._backend._resolve_experiment_id(None)
        self.assertIsNone(result)

    def test_resolve_experiment_id_named_not_found_raises(self):
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value=None):
            with self.assertRaisesRegex(ValueError, 'Experiment not found'):
                self.client._backend._resolve_experiment_id('nonexistent')

    def test_generate_run_name_from_callable(self):
        func = Mock()
        func.name = 'my-pipe'
        result = PipelinesClient._generate_run_name(func)
        self.assertTrue(result.startswith('my-pipe '))

    def test_generate_run_name_from_string(self):
        result = PipelinesClient._generate_run_name('some-pipeline')
        self.assertTrue(result.startswith('some-pipeline '))

    def test_generate_run_name_from_yaml_path(self):
        result = PipelinesClient._generate_run_name('/tmp/train.yaml')
        self.assertTrue(result.startswith('train '))

    def test_generate_run_name_from_pipeline_object(self):
        pipe = kfp_server_api.V2beta1Pipeline(display_name='uploaded-pipe')
        result = PipelinesClient._generate_run_name(pipe)
        self.assertTrue(result.startswith('uploaded-pipe '))


class TestUploadNewPipeline(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_rename_version_success(self):
        mock_pipeline = Mock(pipeline_id='pid-1')
        mock_version = Mock(
            pipeline_id='pid-1',
            pipeline_version_id='vid-1',
            display_name='auto-generated-name',
        )
        with patch.object(
                self.client._backend.upload_api,
                'upload_pipeline',
                return_value=mock_pipeline):
            with patch.object(
                    self.client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[mock_version])):
                with patch.object(
                        self.client._backend.pipelines_api,
                        'pipeline_service_update_pipeline_version',
                        create=True,
                ) as mock_update:
                    result = self.client._backend._upload_new_pipeline(
                        '/tmp/pipe.yaml',
                        name='my-pipe',
                        version_name='v1',
                        description=None,
                    )
                    mock_update.assert_called_once()
                    self.assertEqual(result.display_name, 'v1')

    def test_rename_version_failure_warns(self):
        mock_pipeline = Mock(pipeline_id='pid-1')
        mock_version = Mock(
            pipeline_id='pid-1',
            pipeline_version_id='vid-1',
            display_name='auto-generated-name',
        )
        with patch.object(
                self.client._backend.upload_api,
                'upload_pipeline',
                return_value=mock_pipeline):
            with patch.object(
                    self.client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[mock_version])):
                with patch.object(
                        self.client._backend.pipelines_api,
                        'pipeline_service_update_pipeline_version',
                        create=True,
                        side_effect=kfp_server_api.ApiException(
                            status=404, reason='Not Found')):
                    import warnings
                    with warnings.catch_warnings(record=True) as caught:
                        warnings.simplefilter('always')
                        result = self.client._backend._upload_new_pipeline(
                            '/tmp/pipe.yaml',
                            name='my-pipe',
                            version_name='v1',
                            description=None,
                        )
                    self.assertEqual(len(caught), 1)
                    self.assertIn('Could not rename', str(caught[0].message))
                    self.assertEqual(result.display_name, 'auto-generated-name')

    def test_no_version_created_raises_runtime_error(self):
        mock_pipeline = Mock(pipeline_id='pid-1')
        with patch.object(
                self.client._backend.upload_api,
                'upload_pipeline',
                return_value=mock_pipeline):
            with patch.object(
                    self.client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[])):
                with self.assertRaisesRegex(RuntimeError,
                                            'no version was created'):
                    self.client._backend._upload_new_pipeline(
                        '/tmp/pipe.yaml',
                        name='my-pipe',
                        version_name=None,
                        description=None,
                    )


class TestLoadPipelineSpec(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_load_valid_pipeline_spec(self):
        import tempfile
        spec_content = ('pipelineInfo:\n'
                        '  name: test-pipeline\n'
                        'root:\n'
                        '  dag: {}\n'
                        'schemaVersion: 2.1.0\n'
                        'sdkVersion: kfp-2.0.0\n')
        with tempfile.NamedTemporaryFile(
                mode='w', suffix='.yaml', delete=False) as f:
            f.write(spec_content)
            f.flush()
        self.addCleanup(os.unlink, f.name)
        result = self.client._backend._load_pipeline_spec(f.name)
        self.assertIn('pipelineInfo', result)

    def test_load_empty_file_raises(self):
        import tempfile
        with tempfile.NamedTemporaryFile(
                mode='w', suffix='.yaml', delete=False) as f:
            f.write('')
            f.flush()
        self.addCleanup(os.unlink, f.name)
        with self.assertRaisesRegex(ValueError, 'empty'):
            self.client._backend._load_pipeline_spec(f.name)

    def test_load_multi_doc_yaml_with_platform_spec(self):
        import tempfile
        spec_content = ('pipelineInfo:\n'
                        '  name: test-pipeline\n'
                        'root:\n'
                        '  dag: {}\n'
                        'schemaVersion: 2.1.0\n'
                        'sdkVersion: kfp-2.0.0\n'
                        '---\n'
                        'platforms:\n'
                        '  kubernetes:\n'
                        '    deploymentSpec: {}\n')
        with tempfile.NamedTemporaryFile(
                mode='w', suffix='.yaml', delete=False) as f:
            f.write(spec_content)
            f.flush()
        self.addCleanup(os.unlink, f.name)
        result = self.client._backend._load_pipeline_spec(f.name)
        self.assertIn('pipeline_spec', result)
        self.assertIn('platform_spec', result)


class TestListRunsClientSideFilter(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_pipeline_filter_removes_non_matching_runs(self):
        run_match = Mock(
            run_id='r-1', pipeline_version_reference=Mock(pipeline_id='pid-1'))
        run_other = Mock(
            run_id='r-2',
            pipeline_version_reference=Mock(pipeline_id='pid-other'))
        response = Mock(runs=[run_match, run_other], next_page_token='')

        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend.run_api,
                    'run_service_list_runs',
                    return_value=response):
                result = self.client.list_runs(pipeline='my-pipe')
                self.assertEqual(len(result.runs), 1)
                self.assertEqual(result.runs[0].run_id, 'r-1')

    def test_pipeline_filter_nonexistent_raises(self):
        with patch.object(
                self.client._backend, '_get_pipeline_id_by_name',
                return_value=None):
            with self.assertRaisesRegex(ValueError, 'Pipeline not found'):
                self.client.list_runs(pipeline='bad-pipe')

    def test_experiment_filter_nonexistent_raises(self):
        with patch.object(
                self.client._backend,
                '_get_experiment_id_by_name',
                return_value=None):
            with self.assertRaisesRegex(ValueError, 'Experiment not found'):
                self.client.list_runs(experiment='bad-exp')

    def test_status_filter_passed_to_server(self):
        response = Mock(runs=[], next_page_token='')
        with patch.object(
                self.client._backend.run_api,
                'run_service_list_runs',
                return_value=response) as mock_list:
            self.client.list_runs(status='succeeded')
            call_kwargs = mock_list.call_args[1]
            import json
            filter_dict = json.loads(call_kwargs['filter'])
            self.assertEqual(filter_dict['predicates'][0]['stringValue'],
                             'SUCCEEDED')


class TestUploadPipeline(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_upload_from_callable_compiles_and_cleans_up(self):
        mock_version = Mock(pipeline_version_id='vid-1')
        with patch.object(
                self.client,
                '_resolve_pipeline_to_file',
                return_value=('/tmp/pipe.yaml', '/tmp/tmpdir')):
            with patch.object(
                    self.client, '_infer_pipeline_name',
                    return_value='my-pipe'):
                with patch.object(
                        self.client._backend,
                        '_get_pipeline_id_by_name',
                        return_value=None):
                    with patch.object(
                            self.client._backend,
                            '_upload_new_pipeline',
                            return_value=mock_version):
                        with patch('shutil.rmtree') as mock_rm:
                            result = self.client.upload_pipeline(
                                lambda: None, name='my-pipe')
                            mock_rm.assert_called_once_with(
                                '/tmp/tmpdir', ignore_errors=True)
        self.assertEqual(result.pipeline_version_id, 'vid-1')

    def test_upload_to_existing_pipeline_creates_version(self):
        mock_version = Mock(pipeline_version_id='vid-2')
        with patch.object(
                self.client,
                '_resolve_pipeline_to_file',
                return_value=('/tmp/pipe.yaml', None)):
            with patch.object(
                    self.client,
                    '_infer_pipeline_name',
                    return_value='existing-pipe'):
                with patch.object(
                        self.client._backend,
                        '_get_pipeline_id_by_name',
                        return_value='pid-1'):
                    with patch.object(
                            self.client._backend,
                            '_upload_version',
                            return_value=mock_version) as mock_uv:
                        result = self.client.upload_pipeline(
                            '/tmp/pipe.yaml',
                            name='existing-pipe',
                            version='v2')
                        mock_uv.assert_called_once_with(
                            '/tmp/pipe.yaml',
                            pipeline_id='pid-1',
                            version_name='v2',
                            description=None,
                        )
        self.assertEqual(result.pipeline_version_id, 'vid-2')


class TestResolvePipelineToFile(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_file_not_found_raises(self):
        with patch('os.path.isfile', return_value=False):
            with self.assertRaisesRegex(ValueError, 'Pipeline file not found'):
                self.client._resolve_pipeline_to_file('/no/such/file.yaml')

    def test_unsupported_extension_raises(self):
        with patch('os.path.isfile', return_value=True):
            with self.assertRaisesRegex(ValueError, 'Unsupported file type'):
                self.client._resolve_pipeline_to_file('/tmp/pipeline.json')

    def test_valid_yaml_returns_path_no_cleanup(self):
        with patch('os.path.isfile', return_value=True):
            path, temp_dir = self.client._resolve_pipeline_to_file(
                '/tmp/my-pipeline.yaml')
        self.assertEqual(path, '/tmp/my-pipeline.yaml')
        self.assertIsNone(temp_dir)

    def test_valid_tar_gz_returns_path_no_cleanup(self):
        with patch('os.path.isfile', return_value=True):
            path, temp_dir = self.client._resolve_pipeline_to_file(
                '/tmp/my-pipeline.tar.gz')
        self.assertEqual(path, '/tmp/my-pipeline.tar.gz')
        self.assertIsNone(temp_dir)

    def test_callable_compilation_failure_cleans_temp_dir(self):

        def bad_pipeline():
            pass

        with patch('tempfile.mkdtemp', return_value='/tmp/fake-tmpdir'):
            with patch(
                    'kfp.compiler.Compiler.compile',
                    side_effect=RuntimeError('compile failed')):
                with patch('shutil.rmtree') as mock_rm:
                    with self.assertRaisesRegex(ValueError,
                                                'Failed to compile pipeline'):
                        self.client._resolve_pipeline_to_file(bad_pipeline)
                    mock_rm.assert_called_once_with(
                        '/tmp/fake-tmpdir', ignore_errors=True)

    def test_callable_compilation_success_returns_temp_path(self):

        def good_pipeline():
            pass

        with patch('tempfile.mkdtemp', return_value='/tmp/fake-tmpdir'):
            with patch('kfp.compiler.Compiler.compile'):
                path, temp_dir = self.client._resolve_pipeline_to_file(
                    good_pipeline)
        self.assertEqual(path, '/tmp/fake-tmpdir/pipeline.yaml')
        self.assertEqual(temp_dir, '/tmp/fake-tmpdir')

    def test_non_callable_non_string_raises(self):
        with self.assertRaisesRegex(ValueError, 'Expected a callable'):
            self.client._resolve_pipeline_to_file(12345)


class TestGetPipelineVersion(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_get_pipeline_version_success(self):
        mock_pipeline = Mock(pipeline_id='pid-1', display_name='my-pipe')
        mock_pv = Mock(pipeline_version_id='vid-1', display_name='v1')
        with patch.object(
                self.client, 'get_pipeline', return_value=mock_pipeline):
            with patch.object(
                    self.client._backend,
                    '_get_version_id_by_name',
                    return_value='vid-1'):
                with patch.object(
                        self.client._backend.pipelines_api,
                        'pipeline_service_get_pipeline_version',
                        return_value=mock_pv) as mock_get:
                    result = self.client.get_pipeline_version('my-pipe', 'v1')
                    mock_get.assert_called_once_with(
                        pipeline_id='pid-1', pipeline_version_id='vid-1')
                    self.assertEqual(result.display_name, 'v1')

    def test_get_pipeline_version_not_found_raises(self):
        mock_pipeline = Mock(pipeline_id='pid-1', display_name='my-pipe')
        with patch.object(
                self.client, 'get_pipeline', return_value=mock_pipeline):
            with patch.object(
                    self.client._backend,
                    '_get_version_id_by_name',
                    return_value=None):
                with self.assertRaisesRegex(ValueError, 'version not found'):
                    self.client.get_pipeline_version('my-pipe', 'bad-ver')


class TestListPipelineVersions(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_list_versions(self):
        mock_resp = Mock(
            pipeline_versions=[Mock(), Mock()], next_page_token='t2')
        with patch.object(
                self.client._backend,
                '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    self.client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=mock_resp) as mock_list:
                result = self.client.list_pipeline_versions('my-pipe')
                mock_list.assert_called_once_with(
                    pipeline_id='pid-1', page_token='', page_size=10)
                self.assertEqual(len(result.pipeline_versions), 2)

    def test_list_versions_pipeline_not_found_raises(self):
        with patch.object(
                self.client._backend, '_get_pipeline_id_by_name',
                return_value=None):
            with self.assertRaisesRegex(ValueError, 'Pipeline not found'):
                self.client.list_pipeline_versions('ghost-pipe')


class TestGetRun(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_get_run_by_id(self):
        mock_run = Mock(run_id='r-1', state='SUCCEEDED')
        with patch.object(
                self.client._backend.run_api,
                'run_service_get_run',
                return_value=mock_run) as mock_get:
            result = self.client.get_run('r-1')
            mock_get.assert_called_once_with(run_id='r-1')
            self.assertEqual(result.run_id, 'r-1')


class TestKfpClientEscapeHatch(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_kfp_client_property_creates_instance_once(self):
        with patch('kfp.client.Client') as MockClient:
            MockClient.return_value = Mock()
            first = self.client.kfp_client
            second = self.client.kfp_client
            self.assertIs(first, second)
            MockClient.assert_called_once()


class TestListRunsBasic(parameterized.TestCase):

    def setUp(self):
        self.client = _make_client()

    def test_list_runs_no_filters(self):
        mock_response = Mock(runs=[Mock(run_id='r-1')], next_page_token='t2')
        with patch.object(
                self.client._backend.run_api,
                'run_service_list_runs',
                return_value=mock_response) as mock_list:
            result = self.client.list_runs(page_size=20)
            mock_list.assert_called_once_with(
                namespace='test-ns',
                experiment_id='',
                page_token='',
                page_size=20,
                filter=None,
            )
            self.assertEqual(len(result.runs), 1)
            self.assertEqual(result.next_page_token, 't2')


if __name__ == '__main__':
    unittest.main()
