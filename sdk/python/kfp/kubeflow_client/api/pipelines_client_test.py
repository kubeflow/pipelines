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

from dataclasses import dataclass
from dataclasses import field
import json
import logging
import os
import tempfile
from typing import Any
from unittest.mock import Mock
from unittest.mock import patch
import warnings

from kfp.kubeflow_client import constants
from kfp.kubeflow_client.api.pipelines_client import PipelinesClient
from kfp.kubeflow_client.backends.kubernetes.backend import KubernetesBackend
from kfp.kubeflow_client.backends.kubernetes.types import \
    KubernetesBackendConfig
import kfp_server_api
import pytest

_AUTH_MODULE = 'kfp.kubeflow_client.backends.kubernetes.auth'
_BACKEND_MODULE = 'kfp.kubeflow_client.backends.kubernetes.backend'

SUCCESS = 'success'
FAILED = 'failed'


@dataclass
class TestCase:
    """A single test scenario for parametrized tests."""

    name: str
    expected_status: str = SUCCESS
    config: dict[str, Any] = field(default_factory=dict)
    expected_output: Any | None = None
    expected_error: type[Exception] | None = None
    expected_error_match: str | None = None
    __test__ = False


# ------------------------------------------------------------------
# Fixtures
# ------------------------------------------------------------------


@pytest.fixture
def client():
    with patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials'), \
         patch(f'{_BACKEND_MODULE}.KubernetesBackend.verify_backend'):
        return PipelinesClient(
            backend_config=KubernetesBackendConfig(
                base_url='http://localhost:8888',
                namespace='test-ns',
            ))


# ------------------------------------------------------------------
# test_get_pipeline
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='success',
            config={
                'pipeline_id': 'pid-1',
                'display_name': 'my-pipe'
            },
            expected_output={'pipeline_id': 'pid-1'},
        ),
        TestCase(
            name='not found raises ValueError',
            config={'pipelines': []},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Pipeline not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_pipeline(client, test_case):
    if test_case.expected_status == SUCCESS:
        mock_pipeline = Mock(
            pipeline_id=test_case.config['pipeline_id'],
            display_name=test_case.config['display_name'],
        )
        with patch.object(
                client._backend.pipelines_api,
                'pipeline_service_list_pipelines',
                return_value=Mock(pipelines=[mock_pipeline])):
            with patch.object(
                    client._backend.pipelines_api,
                    'pipeline_service_get_pipeline',
                    return_value=mock_pipeline) as mock_get:
                result = client.get_pipeline('my-pipe')
                mock_get.assert_called_once_with(pipeline_id='pid-1')
                assert result.pipeline_id == test_case.expected_output[
                    'pipeline_id']
    else:
        with patch.object(
                client._backend.pipelines_api,
                'pipeline_service_list_pipelines',
                return_value=Mock(pipelines=[])):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.get_pipeline('nonexistent')


# ------------------------------------------------------------------
# test_get_pipeline_version
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='success',
            config={
                'pipeline_name': 'my-pipe',
                'version_name': 'v1'
            },
            expected_output={'display_name': 'v1'},
        ),
        TestCase(
            name='not found raises ValueError',
            config={
                'pipeline_name': 'my-pipe',
                'version_name': 'bad-ver'
            },
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='version not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_pipeline_version(client, test_case):
    if test_case.expected_status == SUCCESS:
        mock_pv = Mock(
            pipeline_version_id='vid-1',
            display_name=test_case.config['version_name'],
        )
        with patch.object(
                client._backend, 'get_pipeline_version',
                return_value=mock_pv) as mock_get:
            result = client.get_pipeline_version(
                test_case.config['pipeline_name'],
                test_case.config['version_name'],
            )
            mock_get.assert_called_once_with(
                test_case.config['pipeline_name'],
                test_case.config['version_name'],
            )
            assert result.display_name == test_case.expected_output[
                'display_name']
    else:
        with patch.object(
                client._backend,
                'get_pipeline_version',
                side_effect=ValueError('version not found')):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.get_pipeline_version(
                    test_case.config['pipeline_name'],
                    test_case.config['version_name'],
                )


# ------------------------------------------------------------------
# test_list_pipelines
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='returns paginated response',
            expected_output={'next_page_token': 'tok'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_pipelines(client, test_case):
    mock_response = Mock(pipelines=[Mock()], next_page_token='tok')
    with patch.object(
            client._backend.pipelines_api,
            'pipeline_service_list_pipelines',
            return_value=mock_response) as mock_list:
        result = client.list_pipelines(page_size=5)
        mock_list.assert_called_once_with(
            namespace='test-ns', page_token='', page_size=5)
        assert result.next_page_token == test_case.expected_output[
            'next_page_token']


# ------------------------------------------------------------------
# test_list_pipeline_versions
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='success returns versions',
            config={'pipeline_id': 'pid-1'},
            expected_output={'count': 2},
        ),
        TestCase(
            name='pipeline not found raises ValueError',
            config={'pipeline_id': None},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Pipeline not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_pipeline_versions(client, test_case):
    if test_case.expected_status == SUCCESS:
        mock_resp = Mock(
            pipeline_versions=[Mock(), Mock()], next_page_token='t2')
        with patch.object(
                client._backend, '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=mock_resp) as mock_list:
                result = client.list_pipeline_versions('my-pipe')
                mock_list.assert_called_once_with(
                    pipeline_id='pid-1', page_token='', page_size=10)
                assert len(result.pipeline_versions
                          ) == test_case.expected_output['count']
    else:
        with patch.object(
                client._backend, '_get_pipeline_id_by_name', return_value=None):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.list_pipeline_versions('ghost-pipe')


# ------------------------------------------------------------------
# test_delete_pipeline
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='single version no force deletes pipeline',
            config={'scenario': 'single_version_no_force'},
        ),
        TestCase(
            name='multiple versions no force raises',
            config={'scenario': 'multi_version_no_force'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='multiple versions',
        ),
        TestCase(
            name='multiple versions with force deletes pipeline',
            config={'scenario': 'multi_version_force'},
        ),
        TestCase(
            name='specific version deletes version only',
            config={'scenario': 'specific_version'},
        ),
        TestCase(
            name='nonexistent version raises',
            config={'scenario': 'nonexistent_version'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Pipeline version not found',
        ),
        TestCase(
            name='pipeline not found raises',
            config={'scenario': 'not_found'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Pipeline not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_delete_pipeline(client, test_case):
    scenario = test_case.config['scenario']

    if scenario == 'single_version_no_force':
        with patch.object(
                client._backend, '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[Mock()])):
                with patch.object(
                        client._backend.pipelines_api,
                        'pipeline_service_delete_pipeline') as mock_del:
                    client.delete_pipeline('my-pipe', force=False)
                    mock_del.assert_called_once_with(
                        pipeline_id='pid-1', cascade=True)

    elif scenario == 'multi_version_no_force':
        with patch.object(
                client._backend, '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[Mock(), Mock()])):
                with pytest.raises(
                        test_case.expected_error,
                        match=test_case.expected_error_match):
                    client.delete_pipeline('my-pipe', force=False)

    elif scenario == 'multi_version_force':
        with patch.object(
                client._backend, '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(client._backend.pipelines_api,
                              'pipeline_service_delete_pipeline') as mock_del:
                client.delete_pipeline('my-pipe', force=True)
                mock_del.assert_called_once_with(
                    pipeline_id='pid-1', cascade=True)

    elif scenario == 'specific_version':
        with patch.object(
                client._backend, '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    client._backend,
                    '_get_version_id_by_name',
                    return_value='vid-1'):
                with patch.object(
                        client._backend.pipelines_api,
                        'pipeline_service_delete_pipeline_version',
                ) as mock_del:
                    client.delete_pipeline('my-pipe', version='v1')
                    mock_del.assert_called_once_with(
                        pipeline_id='pid-1', pipeline_version_id='vid-1')

    elif scenario == 'nonexistent_version':
        with patch.object(
                client._backend, '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    client._backend, '_get_version_id_by_name',
                    return_value=None):
                with pytest.raises(
                        test_case.expected_error,
                        match=test_case.expected_error_match):
                    client.delete_pipeline('my-pipe', version='bad-ver')

    elif scenario == 'not_found':
        with patch.object(
                client._backend, '_get_pipeline_id_by_name', return_value=None):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.delete_pipeline('ghost-pipe')


# ------------------------------------------------------------------
# test_upload_pipeline
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='from callable compiles and cleans up',
            config={'scenario': 'from_callable'},
            expected_output={'pipeline_version_id': 'vid-1'},
        ),
        TestCase(
            name='to existing pipeline creates version',
            config={'scenario': 'to_existing'},
            expected_output={'pipeline_version_id': 'vid-2'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_upload_pipeline(client, test_case):
    scenario = test_case.config['scenario']

    if scenario == 'from_callable':
        mock_version = Mock(pipeline_version_id='vid-1')
        with patch.object(
                client,
                '_resolve_pipeline_to_file',
                return_value=('/tmp/pipe.yaml', '/tmp/tmpdir')):
            with patch.object(
                    client, '_infer_pipeline_name', return_value='my-pipe'):
                with patch.object(
                        client._backend,
                        '_get_pipeline_id_by_name',
                        return_value=None):
                    with patch.object(
                            client._backend,
                            '_upload_new_pipeline',
                            return_value=mock_version):
                        with patch('shutil.rmtree') as mock_rm:
                            result = client.upload_pipeline(
                                lambda: None, name='my-pipe')
                            mock_rm.assert_called_once_with(
                                '/tmp/tmpdir', ignore_errors=True)
        assert result.pipeline_version_id == test_case.expected_output[
            'pipeline_version_id']

    elif scenario == 'to_existing':
        mock_version = Mock(pipeline_version_id='vid-2')
        with patch.object(
                client,
                '_resolve_pipeline_to_file',
                return_value=('/tmp/pipe.yaml', None)):
            with patch.object(
                    client, '_infer_pipeline_name',
                    return_value='existing-pipe'):
                with patch.object(
                        client._backend,
                        '_get_pipeline_id_by_name',
                        return_value='pid-1'):
                    with patch.object(
                            client._backend,
                            '_upload_version',
                            return_value=mock_version) as mock_uv:
                        result = client.upload_pipeline(
                            '/tmp/pipe.yaml',
                            name='existing-pipe',
                            version='v2',
                        )
                        mock_uv.assert_called_once_with(
                            '/tmp/pipe.yaml',
                            pipeline_id='pid-1',
                            version_name='v2',
                            description=None,
                        )
        assert result.pipeline_version_id == test_case.expected_output[
            'pipeline_version_id']


# ------------------------------------------------------------------
# test_run
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='callable dispatches to _run_inline',
            config={
                'input_type': 'callable',
                'kwargs': {
                    'params': {
                        'x': '1'
                    },
                    'name': 'my-run'
                },
            },
            expected_output={'run_id': 'r-1'},
        ),
        TestCase(
            name='string name delegates to backend.run_by_name',
            config={
                'input_type': 'string',
                'input_value': 'my-pipe',
                'kwargs': {
                    'name': 'run-1'
                },
            },
            expected_output={'run_id': 'r-2'},
        ),
        TestCase(
            name='string name with version',
            config={
                'input_type': 'string',
                'input_value': 'my-pipe',
                'kwargs': {
                    'version': 'v2',
                    'name': 'run-2'
                },
            },
            expected_output={'run_id': 'r-3'},
        ),
        TestCase(
            name='name nonexistent version raises',
            config={
                'input_type': 'string_version_not_found',
                'input_value': 'my-pipe',
                'kwargs': {
                    'version': 'bad'
                },
            },
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='version not found',
        ),
        TestCase(
            name='archive file raises',
            config={
                'input_type': 'archive_path',
                'input_value': '/path/to/file.tar.gz',
            },
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Archive files',
        ),
        TestCase(
            name='pipeline version object',
            config={
                'input_type': 'pipeline_version_obj',
                'kwargs': {
                    'name': 'run-pv'
                },
            },
            expected_output={'run_id': 'r-4'},
        ),
        TestCase(
            name='pipeline object',
            config={
                'input_type': 'pipeline_obj',
                'kwargs': {
                    'name': 'run-pipe'
                },
            },
            expected_output={'run_id': 'r-5'},
        ),
        TestCase(
            name='yaml path dispatches to run_from_file',
            config={
                'input_type': 'yaml_path',
                'input_value': '/tmp/my-pipeline.yaml',
                'kwargs': {
                    'name': 'yaml-run'
                },
            },
            expected_output={'run_id': 'r-6'},
        ),
        TestCase(
            name='callable with version logs warning',
            config={
                'input_type': 'callable_with_version',
                'kwargs': {
                    'version': 'v2',
                    'name': 'run-w1'
                },
                'expected_warning': 'version parameter is ignored',
            },
            expected_output={'run_id': 'r-warn-1'},
        ),
        TestCase(
            name='yaml path with version logs warning',
            config={
                'input_type': 'yaml_with_version',
                'input_value': '/tmp/pipe.yaml',
                'kwargs': {
                    'version': 'v2',
                    'name': 'run-w2'
                },
                'expected_warning': 'version parameter is ignored',
            },
            expected_output={'run_id': 'r-warn-2'},
        ),
        TestCase(
            name='pipeline version obj with version logs warning',
            config={
                'input_type': 'pv_obj_with_version',
                'kwargs': {
                    'version': 'v2',
                    'name': 'run-w3'
                },
                'expected_warning': 'version parameter is ignored',
            },
            expected_output={'run_id': 'r-warn-3'},
        ),
        TestCase(
            name='pipeline obj with version delegates to backend',
            config={
                'input_type': 'pipeline_obj_with_version',
                'kwargs': {
                    'version': 'v2',
                    'name': 'run-ver'
                },
            },
            expected_output={'run_id': 'r-ver'},
        ),
        TestCase(
            name='unsupported type raises',
            config={'input_type': 'unsupported'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Unsupported pipeline',
        ),
    ],
    ids=lambda tc: tc.name)
def test_run(client, test_case, caplog):
    input_type = test_case.config['input_type']
    kwargs = test_case.config.get('kwargs', {})

    if input_type == 'callable':
        mock_run = Mock(run_id='r-1', state='PENDING')

        def fake_pipeline():
            pass

        fake_pipeline.name = 'test-pipe'
        with patch.object(
                client, '_run_inline', return_value=mock_run) as mock_inline:
            result = client.run(fake_pipeline, **kwargs)
            mock_inline.assert_called_once()
            assert result.run_id == test_case.expected_output['run_id']

    elif input_type == 'string':
        input_value = test_case.config['input_value']
        version = kwargs.get('version')
        run_name = kwargs['name']
        run_id = test_case.expected_output['run_id']
        mock_run = Mock(run_id=run_id)
        with patch.object(
                client._backend, 'run_by_name',
                return_value=mock_run) as mock_run_by_name:
            result = client.run(input_value, **kwargs)
            mock_run_by_name.assert_called_once_with(
                pipeline_name=input_value,
                version_name=version,
                params=None,
                run_name=run_name,
                experiment=None,
            )
            assert result.run_id == run_id

    elif input_type == 'string_version_not_found':
        input_value = test_case.config['input_value']
        with patch.object(
                client._backend,
                'run_by_name',
                side_effect=ValueError('version not found')):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.run(input_value, **kwargs)

    elif input_type == 'archive_path':
        input_value = test_case.config['input_value']
        with pytest.raises(
                test_case.expected_error, match=test_case.expected_error_match):
            client.run(input_value)

    elif input_type == 'pipeline_version_obj':
        pv = Mock(
            spec=kfp_server_api.V2beta1PipelineVersion,
            pipeline_id='pid-1',
            pipeline_version_id='vid-1',
        )
        mock_run = Mock(run_id='r-4')
        with patch.object(
                client._backend, 'run_from_version',
                return_value=mock_run) as mock_ref:
            result = client.run(pv, **kwargs)
            mock_ref.assert_called_once_with(
                pipeline_id='pid-1',
                version_id='vid-1',
                params=None,
                run_name='run-pv',
                experiment=None,
            )
            assert result.run_id == test_case.expected_output['run_id']

    elif input_type == 'pipeline_obj':
        pipeline_obj = Mock(
            spec=kfp_server_api.V2beta1Pipeline, pipeline_id='pid-1')
        mock_run = Mock(run_id='r-5')
        with patch.object(
                client._backend, 'run_pipeline',
                return_value=mock_run) as mock_run_pipeline:
            result = client.run(pipeline_obj, **kwargs)
            mock_run_pipeline.assert_called_once_with(
                pipeline=pipeline_obj,
                version=None,
                params=None,
                run_name='run-pipe',
                experiment=None,
            )
            assert result.run_id == test_case.expected_output['run_id']

    elif input_type == 'yaml_path':
        input_value = test_case.config['input_value']
        mock_run = Mock(run_id='r-6')
        with patch.object(
                client._backend, 'run_from_file',
                return_value=mock_run) as mock_file:
            result = client.run(input_value, **kwargs)
            mock_file.assert_called_once_with(
                file_path=input_value,
                params=None,
                run_name='yaml-run',
                experiment=None,
            )
            assert result.run_id == test_case.expected_output['run_id']

    elif input_type == 'callable_with_version':
        mock_run = Mock(run_id='r-warn-1')
        pipeline_fn = Mock(__name__='my_pipe')
        with patch.object(client, '_run_inline', return_value=mock_run):
            with caplog.at_level(
                    logging.WARNING,
                    logger='kfp.kubeflow_client.api.pipelines_client'):
                client.run(pipeline_fn, **kwargs)
        assert test_case.config['expected_warning'] in caplog.text

    elif input_type == 'yaml_with_version':
        input_value = test_case.config['input_value']
        mock_run = Mock(run_id='r-warn-2')
        with patch.object(
                client._backend, 'run_from_file', return_value=mock_run):
            with caplog.at_level(
                    logging.WARNING,
                    logger='kfp.kubeflow_client.api.pipelines_client'):
                client.run(input_value, **kwargs)
        assert test_case.config['expected_warning'] in caplog.text

    elif input_type == 'pv_obj_with_version':
        mock_run = Mock(run_id='r-warn-3')
        pv_obj = Mock(
            spec=kfp_server_api.V2beta1PipelineVersion,
            pipeline_id='pid-1',
            pipeline_version_id='vid-1',
        )
        with patch.object(
                client._backend, 'run_from_version', return_value=mock_run):
            with caplog.at_level(
                    logging.WARNING,
                    logger='kfp.kubeflow_client.api.pipelines_client'):
                client.run(pv_obj, **kwargs)
        assert test_case.config['expected_warning'] in caplog.text

    elif input_type == 'pipeline_obj_with_version':
        pipeline_obj = Mock(
            spec=kfp_server_api.V2beta1Pipeline,
            pipeline_id='pid-1',
            display_name='my-pipe',
        )
        mock_run = Mock(run_id='r-ver')
        with patch.object(
                client._backend, 'run_pipeline',
                return_value=mock_run) as mock_run_pipeline:
            result = client.run(pipeline_obj, **kwargs)
            mock_run_pipeline.assert_called_once_with(
                pipeline=pipeline_obj,
                version='v2',
                params=None,
                run_name='run-ver',
                experiment=None,
            )
            assert result.run_id == test_case.expected_output['run_id']

    elif input_type == 'unsupported':
        with pytest.raises(
                test_case.expected_error, match=test_case.expected_error_match):
            client.run(12345)


# ------------------------------------------------------------------
# test_get_run
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='returns run by id',
            expected_output={'run_id': 'r-1'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_run(client, test_case):
    mock_run = Mock(run_id='r-1', state='SUCCEEDED')
    with patch.object(
            client._backend.run_api, 'run_service_get_run',
            return_value=mock_run) as mock_get:
        result = client.get_run('r-1')
        mock_get.assert_called_once_with(run_id='r-1')
        assert result.run_id == test_case.expected_output['run_id']


# ------------------------------------------------------------------
# test_list_runs
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='no filters',
            config={'scenario': 'no_filters'},
            expected_output={
                'count': 1,
                'next_page_token': 't2'
            },
        ),
        TestCase(
            name='pipeline filter removes non-matching runs',
            config={'scenario': 'pipeline_filter'},
            expected_output={
                'count': 1,
                'run_id': 'r-1'
            },
        ),
        TestCase(
            name='pipeline not found raises',
            config={'scenario': 'pipeline_not_found'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Pipeline not found',
        ),
        TestCase(
            name='experiment not found raises',
            config={'scenario': 'experiment_not_found'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Experiment not found',
        ),
        TestCase(
            name='status filter passed to server',
            config={'scenario': 'status_filter'},
            expected_output={'status_value': 'SUCCEEDED'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_runs(client, test_case):
    scenario = test_case.config['scenario']

    if scenario == 'no_filters':
        mock_response = Mock(runs=[Mock(run_id='r-1')], next_page_token='t2')
        with patch.object(
                client._backend.run_api,
                'run_service_list_runs',
                return_value=mock_response) as mock_list:
            result = client.list_runs(page_size=20)
            mock_list.assert_called_once_with(
                namespace='test-ns',
                experiment_id='',
                page_token='',
                page_size=20,
                filter=None,
            )
            assert len(result.runs) == test_case.expected_output['count']
            assert result.next_page_token == test_case.expected_output[
                'next_page_token']

    elif scenario == 'pipeline_filter':
        run_match = Mock(
            run_id='r-1',
            pipeline_version_reference=Mock(pipeline_id='pid-1'),
        )
        run_other = Mock(
            run_id='r-2',
            pipeline_version_reference=Mock(pipeline_id='pid-other'),
        )
        response = Mock(runs=[run_match, run_other], next_page_token='')
        with patch.object(
                client._backend, '_get_pipeline_id_by_name',
                return_value='pid-1'):
            with patch.object(
                    client._backend.run_api,
                    'run_service_list_runs',
                    return_value=response):
                result = client.list_runs(pipeline='my-pipe')
                assert len(result.runs) == test_case.expected_output['count']
                assert result.runs[0].run_id == test_case.expected_output[
                    'run_id']

    elif scenario == 'pipeline_not_found':
        with patch.object(
                client._backend, '_get_pipeline_id_by_name', return_value=None):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.list_runs(pipeline='bad-pipe')

    elif scenario == 'experiment_not_found':
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value=None):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.list_runs(experiment='bad-exp')

    elif scenario == 'status_filter':
        response = Mock(runs=[], next_page_token='')
        with patch.object(
                client._backend.run_api,
                'run_service_list_runs',
                return_value=response) as mock_list:
            client.list_runs(status='succeeded')
            call_kwargs = mock_list.call_args[1]
            filter_dict = json.loads(call_kwargs['filter'])
            assert (filter_dict['predicates'][0]['stringValue'] ==
                    test_case.expected_output['status_value'])


# ------------------------------------------------------------------
# test_wait_for_run_status
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='returns on terminal state',
            config={
                'side_effects': [Mock(run_id='r-1', state='SUCCEEDED')],
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0
                },
            },
            expected_output={'state': 'SUCCEEDED'},
        ),
        TestCase(
            name='raises timeout error',
            config={
                'side_effects': [Mock(run_id='r-1', state='RUNNING')],
                'kwargs': {
                    'timeout': 0,
                    'polling_interval': 0
                },
            },
            expected_status=FAILED,
            expected_error=TimeoutError,
        ),
        TestCase(
            name='callbacks are invoked',
            config={
                'side_effects': [Mock(run_id='r-1', state='SUCCEEDED')],
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0,
                },
                'use_callback': True,
            },
            expected_output={'callback_called': True},
        ),
        TestCase(
            name='accepts run object',
            config={
                'use_run_object': True,
                'side_effects': [Mock(run_id='r-1', state='SUCCEEDED')],
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0
                },
            },
            expected_output={'run_id': 'r-1'},
        ),
        TestCase(
            name='terminal state not in target still returns',
            config={
                'side_effects': [Mock(run_id='r-1', state='FAILED')],
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0
                },
            },
            expected_output={'state': 'FAILED'},
        ),
        TestCase(
            name='token refresh on 401',
            config={
                'side_effects': 'auth_refresh',
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0,
                },
            },
            expected_output={
                'state': 'SUCCEEDED',
                'refresh_called': True
            },
        ),
        TestCase(
            name='auth retry exhaustion raises',
            config={
                'side_effects': 'auth_exhaustion',
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0,
                },
            },
            expected_status=FAILED,
            expected_error=kfp_server_api.ApiException,
            expected_output={'status_code': 401},
        ),
        TestCase(
            name='first poll 401 raises without refresh',
            config={
                'side_effects': 'first_poll_401',
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0,
                },
            },
            expected_status=FAILED,
            expected_error=kfp_server_api.ApiException,
            expected_output={
                'status_code': 401,
                'refresh_not_called': True
            },
        ),
        TestCase(
            name='non-401 api exception propagates immediately',
            config={
                'side_effects': 'non_401',
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0,
                },
            },
            expected_status=FAILED,
            expected_error=kfp_server_api.ApiException,
            expected_output={'status_code': 500},
        ),
        TestCase(
            name='callbacks fire on timeout',
            config={
                'side_effects': [Mock(run_id='r-1', state='RUNNING')],
                'kwargs': {
                    'timeout': 0,
                    'polling_interval': 0
                },
                'use_callback': True,
            },
            expected_status=FAILED,
            expected_error=TimeoutError,
            expected_output={'callback_called': True},
        ),
        TestCase(
            name='callbacks fire on terminal non-target state',
            config={
                'side_effects': [Mock(run_id='r-1', state='FAILED')],
                'kwargs': {
                    'status': {constants.RUN_COMPLETE},
                    'polling_interval': 0,
                },
                'use_callback': True,
            },
            expected_output={
                'state': 'FAILED',
                'callback_called': True
            },
        ),
    ],
    ids=lambda tc: tc.name)
def test_wait_for_run_status(client, test_case):
    config = test_case.config
    kwargs = config['kwargs']
    side_effects = config['side_effects']
    use_callback = config.get('use_callback', False)
    use_run_object = config.get('use_run_object', False)

    callback = Mock() if use_callback else None
    if use_callback:
        kwargs = {**kwargs, 'callbacks': [callback]}

    run_input = 'r-1'
    if use_run_object:
        run_input = Mock(run_id='r-1', state='SUCCEEDED')

    if side_effects == 'auth_refresh':
        mock_run_ok = Mock(run_id='r-1', state='SUCCEEDED')
        mock_side_effects = [
            Mock(run_id='r-1', state='RUNNING'),
            kfp_server_api.ApiException(status=401),
            mock_run_ok,
        ]
        with patch.object(
                client._backend.run_api,
                'run_service_get_run',
                side_effect=mock_side_effects):
            with patch.object(client._backend,
                              'refresh_credentials') as mock_ref:
                result = client.wait_for_run_status(run_input, **kwargs)
                mock_ref.assert_called_once()
                assert result.state == test_case.expected_output['state']

    elif side_effects == 'auth_exhaustion':
        mock_side_effects = [
            Mock(run_id='r-1', state='RUNNING'),
            kfp_server_api.ApiException(status=401),
            kfp_server_api.ApiException(status=401),
            kfp_server_api.ApiException(status=401),
        ]
        with patch.object(
                client._backend.run_api,
                'run_service_get_run',
                side_effect=mock_side_effects):
            with patch.object(client._backend, 'refresh_credentials'):
                with pytest.raises(test_case.expected_error) as exc_info:
                    client.wait_for_run_status(run_input, **kwargs)
                assert exc_info.value.status == test_case.expected_output[
                    'status_code']

    elif side_effects == 'first_poll_401':
        with patch.object(
                client._backend.run_api,
                'run_service_get_run',
                side_effect=kfp_server_api.ApiException(status=401)):
            with patch.object(client._backend,
                              'refresh_credentials') as mock_ref:
                with pytest.raises(test_case.expected_error) as exc_info:
                    client.wait_for_run_status(run_input, **kwargs)
                assert exc_info.value.status == test_case.expected_output[
                    'status_code']
                mock_ref.assert_not_called()

    elif side_effects == 'non_401':
        mock_side_effects = [
            Mock(run_id='r-1', state='RUNNING'),
            kfp_server_api.ApiException(status=500),
        ]
        with patch.object(
                client._backend.run_api,
                'run_service_get_run',
                side_effect=mock_side_effects):
            with pytest.raises(test_case.expected_error) as exc_info:
                client.wait_for_run_status(run_input, **kwargs)
            assert exc_info.value.status == test_case.expected_output[
                'status_code']

    elif isinstance(side_effects, list):
        with patch.object(
                client._backend.run_api,
                'run_service_get_run',
                side_effect=side_effects if len(side_effects) > 1 else None,
                return_value=side_effects[0]
                if len(side_effects) == 1 else None):
            if test_case.expected_status == FAILED:
                with pytest.raises(test_case.expected_error):
                    client.wait_for_run_status(run_input, **kwargs)
                if use_callback:
                    callback.assert_called_once_with(side_effects[0])
            else:
                result = client.wait_for_run_status(run_input, **kwargs)
                if 'state' in (test_case.expected_output or {}):
                    assert result.state == test_case.expected_output['state']
                if 'run_id' in (test_case.expected_output or {}):
                    assert result.run_id == test_case.expected_output['run_id']
                if use_callback:
                    callback.assert_called_once_with(side_effects[0])


# ------------------------------------------------------------------
# test_create_experiment
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='new experiment',
            config={'existing_id': None},
            expected_output={'experiment_id': 'exp-1'},
        ),
        TestCase(
            name='idempotent returns existing',
            config={'existing_id': 'exp-1'},
            expected_output={'experiment_id': 'exp-1'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_create_experiment(client, test_case):
    existing_id = test_case.config['existing_id']

    if existing_id is None:
        mock_exp = Mock(experiment_id='exp-1', display_name='my-exp')
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value=None):
            with patch.object(
                    client._backend.experiment_api,
                    'experiment_service_create_experiment',
                    return_value=mock_exp) as mock_create:
                result = client.create_experiment('my-exp')
                mock_create.assert_called_once()
                assert result.experiment_id == test_case.expected_output[
                    'experiment_id']
    else:
        existing = Mock(experiment_id='exp-1', display_name='my-exp')
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value='exp-1'):
            with patch.object(
                    client._backend.experiment_api,
                    'experiment_service_get_experiment',
                    return_value=existing):
                result = client.create_experiment('my-exp')
                assert result.experiment_id == test_case.expected_output[
                    'experiment_id']


# ------------------------------------------------------------------
# test_get_experiment
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='success',
            config={'existing_id': 'exp-1'},
            expected_output={'experiment_id': 'exp-1'},
        ),
        TestCase(
            name='not found raises',
            config={'existing_id': None},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Experiment not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_experiment(client, test_case):
    if test_case.expected_status == SUCCESS:
        mock_exp = Mock(experiment_id='exp-1', display_name='my-exp')
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value='exp-1'):
            with patch.object(
                    client._backend.experiment_api,
                    'experiment_service_get_experiment',
                    return_value=mock_exp):
                result = client.get_experiment('my-exp')
                assert result.experiment_id == test_case.expected_output[
                    'experiment_id']
    else:
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value=None):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.get_experiment('nonexistent')


# ------------------------------------------------------------------
# test_list_experiments
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='returns paginated response',
            expected_output={'next_page_token': 'tok2'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_experiments(client, test_case):
    mock_response = Mock(
        experiments=[Mock(experiment_id='e-1')], next_page_token='tok2')
    with patch.object(
            client._backend.experiment_api,
            'experiment_service_list_experiments',
            return_value=mock_response) as mock_list:
        result = client.list_experiments(page_size=5)
        mock_list.assert_called_once_with(
            namespace='test-ns', page_token='', page_size=5)
        assert result.next_page_token == test_case.expected_output[
            'next_page_token']


# ------------------------------------------------------------------
# test_delete_experiment
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='success',
            config={'existing_id': 'exp-1'},
        ),
        TestCase(
            name='not found raises',
            config={'existing_id': None},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Experiment not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_delete_experiment(client, test_case):
    if test_case.expected_status == SUCCESS:
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value='exp-1'):
            with patch.object(
                    client._backend.experiment_api,
                    'experiment_service_delete_experiment') as mock_del:
                client.delete_experiment('my-exp')
                mock_del.assert_called_once_with(experiment_id='exp-1')
    else:
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value=None):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client.delete_experiment('bad-exp')


# ------------------------------------------------------------------
# test_kfp_client
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(name='creates instance once (cached property)',),
    ],
    ids=lambda tc: tc.name)
def test_kfp_client(client, test_case):
    with patch('kfp.client.Client') as MockClient:
        MockClient.return_value = Mock()
        first = client.kfp_client
        second = client.kfp_client
        assert first is second
        MockClient.assert_called_once()


# ------------------------------------------------------------------
# test__infer_pipeline_name
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='dsl name attribute',
            config={
                'source': 'dsl_name',
                'dsl_name': 'dsl-assigned-name'
            },
            expected_output='dsl-assigned-name',
        ),
        TestCase(
            name='function name my_pipeline',
            config={
                'source': 'func_name',
                'func_name': 'my_pipeline'
            },
            expected_output='my-pipeline',
        ),
        TestCase(
            name='function name hello_world_pipe',
            config={
                'source': 'func_name',
                'func_name': 'hello_world_pipe'
            },
            expected_output='hello-world-pipe',
        ),
        TestCase(
            name='function name already-dashed',
            config={
                'source': 'func_name',
                'func_name': 'already-dashed'
            },
            expected_output='already-dashed',
        ),
        TestCase(
            name='file path fallback',
            config={'source': 'file_path'},
            expected_output='my-pipeline',
        ),
    ],
    ids=lambda tc: tc.name)
def test__infer_pipeline_name(client, test_case):
    source = test_case.config['source']

    if source == 'dsl_name':
        func = Mock()
        func.name = test_case.config['dsl_name']
        result = client._infer_pipeline_name(func, '/tmp/f.yaml')
        assert result == test_case.expected_output

    elif source == 'func_name':
        func = Mock()
        func.name = None
        func.__name__ = test_case.config['func_name']
        func.__class__ = type('function', (), {})
        with patch('os.path.isfile', return_value=False):
            result = client._infer_pipeline_name(func, '/tmp/f.yaml')
        assert result == test_case.expected_output

    elif source == 'file_path':
        result = client._infer_pipeline_name('/tmp/my-pipeline.yaml',
                                             '/tmp/my-pipeline.yaml')
        assert result == test_case.expected_output


# ------------------------------------------------------------------
# test__is_yaml_path
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='pipeline.yaml',
            config={'path': 'pipeline.yaml'},
            expected_output=True),
        TestCase(
            name='pipeline.yml',
            config={'path': 'pipeline.yml'},
            expected_output=True),
        TestCase(
            name='path/to/spec.yaml',
            config={'path': 'path/to/spec.yaml'},
            expected_output=True),
        TestCase(
            name='pipeline.tar.gz',
            config={'path': 'pipeline.tar.gz'},
            expected_output=False),
        TestCase(
            name='pipeline.zip',
            config={'path': 'pipeline.zip'},
            expected_output=False),
        TestCase(
            name='pipeline-name',
            config={'path': 'pipeline-name'},
            expected_output=False),
        TestCase(
            name='pipeline.tgz',
            config={'path': 'pipeline.tgz'},
            expected_output=False),
    ],
    ids=lambda tc: tc.name)
def test__is_yaml_path(test_case):
    assert PipelinesClient._is_yaml_path(
        test_case.config['path']) == test_case.expected_output


# ------------------------------------------------------------------
# test__validate_pipeline_name
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='valid name passes',
            config={'name': 'my-pipeline'},
        ),
        TestCase(
            name='empty raises',
            config={'name': ''},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='cannot be empty',
        ),
        TestCase(
            name='whitespace raises',
            config={'name': '   '},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='cannot be empty',
        ),
    ],
    ids=lambda tc: tc.name)
def test__validate_pipeline_name(test_case):
    if test_case.expected_status == SUCCESS:
        PipelinesClient._validate_pipeline_name(test_case.config['name'])
    else:
        with pytest.raises(
                test_case.expected_error, match=test_case.expected_error_match):
            PipelinesClient._validate_pipeline_name(test_case.config['name'])


# ------------------------------------------------------------------
# test__generate_run_name
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='from callable',
            config={'source': 'callable'},
            expected_output='my-pipe ',
        ),
        TestCase(
            name='from string',
            config={'source': 'string'},
            expected_output='some-pipeline ',
        ),
        TestCase(
            name='from yaml path',
            config={'source': 'yaml_path'},
            expected_output='train ',
        ),
        TestCase(
            name='from pipeline object',
            config={'source': 'pipeline_object'},
            expected_output='uploaded-pipe ',
        ),
    ],
    ids=lambda tc: tc.name)
def test__generate_run_name(test_case):
    source = test_case.config['source']

    if source == 'callable':
        func = Mock()
        func.name = 'my-pipe'
        result = PipelinesClient._generate_run_name(func)
    elif source == 'string':
        result = PipelinesClient._generate_run_name('some-pipeline')
    elif source == 'yaml_path':
        result = PipelinesClient._generate_run_name('/tmp/train.yaml')
    elif source == 'pipeline_object':
        pipe = kfp_server_api.V2beta1Pipeline(display_name='uploaded-pipe')
        result = PipelinesClient._generate_run_name(pipe)

    assert result.startswith(test_case.expected_output)


# ------------------------------------------------------------------
# test__resolve_pipeline_to_file
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='file not found raises',
            config={'scenario': 'file_not_found'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Pipeline file not found',
        ),
        TestCase(
            name='unsupported extension raises',
            config={'scenario': 'unsupported_extension'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Unsupported file type',
        ),
        TestCase(
            name='valid yaml returns path no cleanup',
            config={'scenario': 'valid_yaml'},
            expected_output={
                'path': '/tmp/my-pipeline.yaml',
                'temp_dir': None
            },
        ),
        TestCase(
            name='valid tar.gz returns path no cleanup',
            config={'scenario': 'valid_tar_gz'},
            expected_output={
                'path': '/tmp/my-pipeline.tar.gz',
                'temp_dir': None
            },
        ),
        TestCase(
            name='callable compilation failure cleans temp dir',
            config={'scenario': 'callable_failure'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Failed to compile pipeline',
        ),
        TestCase(
            name='callable compilation success returns temp path',
            config={'scenario': 'callable_success'},
            expected_output={
                'path': '/tmp/fake-tmpdir/pipeline.yaml',
                'temp_dir': '/tmp/fake-tmpdir',
            },
        ),
        TestCase(
            name='non callable non string raises',
            config={'scenario': 'non_callable_non_string'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Expected a callable',
        ),
    ],
    ids=lambda tc: tc.name)
def test__resolve_pipeline_to_file(client, test_case):
    scenario = test_case.config['scenario']

    if scenario == 'file_not_found':
        with patch('os.path.isfile', return_value=False):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client._resolve_pipeline_to_file('/no/such/file.yaml')

    elif scenario == 'unsupported_extension':
        with patch('os.path.isfile', return_value=True):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client._resolve_pipeline_to_file('/tmp/pipeline.json')

    elif scenario == 'valid_yaml':
        with patch('os.path.isfile', return_value=True):
            path, temp_dir = client._resolve_pipeline_to_file(
                '/tmp/my-pipeline.yaml')
        assert path == test_case.expected_output['path']
        assert temp_dir == test_case.expected_output['temp_dir']

    elif scenario == 'valid_tar_gz':
        with patch('os.path.isfile', return_value=True):
            path, temp_dir = client._resolve_pipeline_to_file(
                '/tmp/my-pipeline.tar.gz')
        assert path == test_case.expected_output['path']
        assert temp_dir == test_case.expected_output['temp_dir']

    elif scenario == 'callable_failure':

        def bad_pipeline():
            pass

        with patch('tempfile.mkdtemp', return_value='/tmp/fake-tmpdir'):
            with patch(
                    'kfp.compiler.Compiler.compile',
                    side_effect=RuntimeError('compile failed')):
                with patch('shutil.rmtree') as mock_rm:
                    with pytest.raises(
                            test_case.expected_error,
                            match=test_case.expected_error_match):
                        client._resolve_pipeline_to_file(bad_pipeline)
                    mock_rm.assert_called_once_with(
                        '/tmp/fake-tmpdir', ignore_errors=True)

    elif scenario == 'callable_success':

        def good_pipeline():
            pass

        with patch('tempfile.mkdtemp', return_value='/tmp/fake-tmpdir'):
            with patch('kfp.compiler.Compiler.compile'):
                path, temp_dir = client._resolve_pipeline_to_file(good_pipeline)
        assert path == test_case.expected_output['path']
        assert temp_dir == test_case.expected_output['temp_dir']

    elif scenario == 'non_callable_non_string':
        with pytest.raises(
                test_case.expected_error, match=test_case.expected_error_match):
            client._resolve_pipeline_to_file(12345)


# ------------------------------------------------------------------
# test__upload_new_pipeline
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='rename version success',
            config={'scenario': 'rename_success'},
            expected_output={'display_name': 'v1'},
        ),
        TestCase(
            name='rename version failure warns',
            config={'scenario': 'rename_failure'},
            expected_output={'display_name': 'auto-generated-name'},
        ),
        TestCase(
            name='no version created raises RuntimeError',
            config={'scenario': 'no_version'},
            expected_status=FAILED,
            expected_error=RuntimeError,
            expected_error_match='no version was created',
        ),
    ],
    ids=lambda tc: tc.name)
def test__upload_new_pipeline(client, test_case):
    scenario = test_case.config['scenario']

    if scenario == 'rename_success':
        mock_pipeline = Mock(pipeline_id='pid-1')
        mock_version = Mock(
            pipeline_id='pid-1',
            pipeline_version_id='vid-1',
            display_name='auto-generated-name',
        )
        with patch.object(
                client._backend.upload_api,
                'upload_pipeline',
                return_value=mock_pipeline):
            with patch.object(
                    client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[mock_version])):
                with patch.object(
                        client._backend.pipelines_api,
                        'pipeline_service_update_pipeline_version',
                        create=True,
                ) as mock_update:
                    result = client._backend._upload_new_pipeline(
                        '/tmp/pipe.yaml',
                        name='my-pipe',
                        version_name='v1',
                        description=None,
                    )
                    mock_update.assert_called_once()
                    assert result.display_name == test_case.expected_output[
                        'display_name']

    elif scenario == 'rename_failure':
        mock_pipeline = Mock(pipeline_id='pid-1')
        mock_version = Mock(
            pipeline_id='pid-1',
            pipeline_version_id='vid-1',
            display_name='auto-generated-name',
        )
        with patch.object(
                client._backend.upload_api,
                'upload_pipeline',
                return_value=mock_pipeline):
            with patch.object(
                    client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[mock_version])):
                with patch.object(
                        client._backend.pipelines_api,
                        'pipeline_service_update_pipeline_version',
                        create=True,
                        side_effect=kfp_server_api.ApiException(
                            status=404, reason='Not Found')):
                    with warnings.catch_warnings(record=True) as caught:
                        warnings.simplefilter('always')
                        result = client._backend._upload_new_pipeline(
                            '/tmp/pipe.yaml',
                            name='my-pipe',
                            version_name='v1',
                            description=None,
                        )
                    assert len(caught) == 1
                    assert 'Could not rename' in str(caught[0].message)
                    assert result.display_name == test_case.expected_output[
                        'display_name']

    elif scenario == 'no_version':
        mock_pipeline = Mock(pipeline_id='pid-1')
        with patch.object(
                client._backend.upload_api,
                'upload_pipeline',
                return_value=mock_pipeline):
            with patch.object(
                    client._backend.pipelines_api,
                    'pipeline_service_list_pipeline_versions',
                    return_value=Mock(pipeline_versions=[])):
                with pytest.raises(
                        test_case.expected_error,
                        match=test_case.expected_error_match):
                    client._backend._upload_new_pipeline(
                        '/tmp/pipe.yaml',
                        name='my-pipe',
                        version_name=None,
                        description=None,
                    )


# ------------------------------------------------------------------
# test__load_pipeline_spec
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='valid pipeline spec',
            config={'scenario': 'valid'},
            expected_output={'contains': 'pipelineInfo'},
        ),
        TestCase(
            name='empty file raises',
            config={'scenario': 'empty'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='empty',
        ),
        TestCase(
            name='multi doc yaml with platform spec',
            config={'scenario': 'multi_doc'},
            expected_output={
                'contains_keys': ['pipeline_spec', 'platform_spec']
            },
        ),
    ],
    ids=lambda tc: tc.name)
def test__load_pipeline_spec(client, test_case):
    scenario = test_case.config['scenario']

    if scenario == 'valid':
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
        try:
            result = client._backend._load_pipeline_spec(f.name)
            assert test_case.expected_output['contains'] in result
        finally:
            os.unlink(f.name)

    elif scenario == 'empty':
        with tempfile.NamedTemporaryFile(
                mode='w', suffix='.yaml', delete=False) as f:
            f.write('')
            f.flush()
        try:
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client._backend._load_pipeline_spec(f.name)
        finally:
            os.unlink(f.name)

    elif scenario == 'multi_doc':
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
        try:
            result = client._backend._load_pipeline_spec(f.name)
            for key in test_case.expected_output['contains_keys']:
                assert key in result
        finally:
            os.unlink(f.name)


# ------------------------------------------------------------------
# test__invoke_callbacks
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='wraps exception in RuntimeError',
            expected_status=FAILED,
            expected_error=RuntimeError,
            expected_error_match='callback boom',
        ),
    ],
    ids=lambda tc: tc.name)
def test__invoke_callbacks(test_case):

    def bad_callback(run):
        raise ValueError('callback boom')

    mock_run = Mock(run_id='r-1', state='SUCCEEDED')
    with pytest.raises(
            test_case.expected_error, match=test_case.expected_error_match):
        KubernetesBackend._invoke_callbacks([bad_callback], mock_run)


# ------------------------------------------------------------------
# test__get_latest_version_id
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='returns latest version id',
            config={'has_versions': True},
            expected_output='vid-latest',
        ),
        TestCase(
            name='no versions raises',
            config={'has_versions': False},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='has no versions',
        ),
    ],
    ids=lambda tc: tc.name)
def test__get_latest_version_id(client, test_case):
    if test_case.config['has_versions']:
        mock_version = Mock(
            pipeline_version_id='vid-latest',
            created_at='2026-01-01T00:00:00Z',
        )
        with patch.object(
                client._backend.pipelines_api,
                'pipeline_service_list_pipeline_versions',
                return_value=Mock(pipeline_versions=[mock_version])):
            result = client._backend._get_latest_version_id('pid-1')
            assert result == test_case.expected_output
    else:
        with patch.object(
                client._backend.pipelines_api,
                'pipeline_service_list_pipeline_versions',
                return_value=Mock(pipeline_versions=[])):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client._backend._get_latest_version_id('pid-1')


# ------------------------------------------------------------------
# test__equals_filter
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='produces correct filter JSON',
            config={
                'key': 'display_name',
                'value': 'test'
            },
            expected_output=(
                '{"predicates": [{"operation": "EQUALS",'
                ' "key": "display_name", "stringValue": "test"}]}'),
        ),
    ],
    ids=lambda tc: tc.name)
def test__equals_filter(client, test_case):
    result = client._backend._equals_filter(test_case.config['key'],
                                            test_case.config['value'])
    assert json.loads(result) == json.loads(test_case.expected_output)


# ------------------------------------------------------------------
# test__get_pipeline_id_by_name
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='multiple matches raises',
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Multiple pipelines',
        ),
    ],
    ids=lambda tc: tc.name)
def test__get_pipeline_id_by_name(client, test_case):
    pipelines = [
        Mock(pipeline_id='pid-1'),
        Mock(pipeline_id='pid-2'),
    ]
    with patch.object(
            client._backend.pipelines_api,
            'pipeline_service_list_pipelines',
            return_value=Mock(pipelines=pipelines)):
        with pytest.raises(
                test_case.expected_error, match=test_case.expected_error_match):
            client._backend._get_pipeline_id_by_name('dup-name')


# ------------------------------------------------------------------
# test__resolve_experiment_id
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='returns None when not specified',
            config={'experiment': None},
            expected_output=None,
        ),
        TestCase(
            name='named not found raises',
            config={'experiment': 'nonexistent'},
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Experiment not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test__resolve_experiment_id(client, test_case):
    experiment = test_case.config['experiment']

    if test_case.expected_status == SUCCESS:
        result = client._backend._resolve_experiment_id(experiment)
        assert result == test_case.expected_output
    else:
        with patch.object(
                client._backend, '_get_experiment_id_by_name',
                return_value=None):
            with pytest.raises(
                    test_case.expected_error,
                    match=test_case.expected_error_match):
                client._backend._resolve_experiment_id(experiment)
