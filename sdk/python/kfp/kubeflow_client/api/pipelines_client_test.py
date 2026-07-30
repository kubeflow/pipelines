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
"""Unit tests for PipelinesClient (delegation to backend)."""

from __future__ import annotations

from dataclasses import dataclass
from dataclasses import field
from typing import Any
from unittest.mock import Mock
from unittest.mock import patch

from kfp.kubeflow_client import constants
from kfp.kubeflow_client.api.pipelines_client import PipelinesClient
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
# test_upload_pipeline
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend upload_pipeline',
            expected_output={'pipeline_version_id': 'vid-1'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_upload_pipeline(client, test_case):
    mock_version = Mock(pipeline_version_id='vid-1')
    with patch.object(
            client._backend, 'upload_pipeline',
            return_value=mock_version) as mock_upload:
        result = client.upload_pipeline(
            lambda: None, name='my-pipe', version='v1', description='desc')
        mock_upload.assert_called_once()
        call_kwargs = mock_upload.call_args[1]
        assert call_kwargs['name'] == 'my-pipe'
        assert call_kwargs['version_name'] == 'v1'
        assert call_kwargs['description'] == 'desc'
    assert result.pipeline_version_id == test_case.expected_output[
        'pipeline_version_id']


# ------------------------------------------------------------------
# test_get_pipeline
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'pipeline_id': 'pid-1'},
        ),
        TestCase(
            name='not found raises ValueError',
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='Pipeline not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_pipeline(client, test_case):
    if test_case.expected_status == SUCCESS:
        mock_pipeline = Mock(pipeline_id='pid-1')
        with patch.object(
                client._backend, 'get_pipeline',
                return_value=mock_pipeline) as mock_get:
            result = client.get_pipeline('my-pipe')
            mock_get.assert_called_once_with('my-pipe')
            assert result.pipeline_id == test_case.expected_output[
                'pipeline_id']
    else:
        with patch.object(
                client._backend,
                'get_pipeline',
                side_effect=ValueError('Pipeline not found')):
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
            name='delegates to backend',
            expected_output={'display_name': 'v1'},
        ),
        TestCase(
            name='not found raises ValueError',
            expected_status=FAILED,
            expected_error=ValueError,
            expected_error_match='version not found',
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_pipeline_version(client, test_case):
    if test_case.expected_status == SUCCESS:
        mock_pv = Mock(display_name='v1')
        with patch.object(
                client._backend, 'get_pipeline_version',
                return_value=mock_pv) as mock_get:
            result = client.get_pipeline_version('my-pipe', 'v1')
            mock_get.assert_called_once_with('my-pipe', 'v1')
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
                client.get_pipeline_version('my-pipe', 'bad-ver')


# ------------------------------------------------------------------
# test_list_pipelines
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'next_page_token': 'tok'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_pipelines(client, test_case):
    mock_response = Mock(pipelines=[Mock()], next_page_token='tok')
    with patch.object(
            client._backend, 'list_pipelines',
            return_value=mock_response) as mock_list:
        result = client.list_pipelines(page_size=5)
        mock_list.assert_called_once_with(page_token='', page_size=5)
        assert result.next_page_token == test_case.expected_output[
            'next_page_token']


# ------------------------------------------------------------------
# test_list_pipeline_versions
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'count': 2},
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_pipeline_versions(client, test_case):
    mock_resp = Mock(pipeline_versions=[Mock(), Mock()], next_page_token='t2')
    with patch.object(
            client._backend, 'list_pipeline_versions',
            return_value=mock_resp) as mock_list:
        result = client.list_pipeline_versions('my-pipe')
        mock_list.assert_called_once_with(
            'my-pipe', page_token='', page_size=10)
        assert len(
            result.pipeline_versions) == test_case.expected_output['count']


# ------------------------------------------------------------------
# test_delete_pipeline
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            config={
                'version': 'v1',
                'force': True
            },
        ),
    ],
    ids=lambda tc: tc.name)
def test_delete_pipeline(client, test_case):
    with patch.object(client._backend, 'delete_pipeline') as mock_del:
        client.delete_pipeline('my-pipe', version='v1', force=True)
        mock_del.assert_called_once_with('my-pipe', version='v1', force=True)


# ------------------------------------------------------------------
# test_run
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend run',
            expected_output={'run_id': 'r-1'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_run(client, test_case):
    mock_run = Mock(run_id='r-1')
    with patch.object(
            client._backend, 'run', return_value=mock_run) as mock_backend_run:
        result = client.run(
            'my-pipe', params={'x': '1'}, name='run-1', version='v2')
        mock_backend_run.assert_called_once_with(
            'my-pipe',
            params={'x': '1'},
            name='run-1',
            experiment=None,
            version='v2',
        )
        assert result.run_id == test_case.expected_output['run_id']


# ------------------------------------------------------------------
# test_get_run
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'run_id': 'r-1'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_run(client, test_case):
    mock_run = Mock(run_id='r-1')
    with patch.object(
            client._backend, 'get_run', return_value=mock_run) as mock_get:
        result = client.get_run('r-1')
        mock_get.assert_called_once_with('r-1')
        assert result.run_id == test_case.expected_output['run_id']


# ------------------------------------------------------------------
# test_list_runs
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'count': 1},
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_runs(client, test_case):
    mock_response = Mock(runs=[Mock(run_id='r-1')], next_page_token='t2')
    with patch.object(
            client._backend, 'list_runs',
            return_value=mock_response) as mock_list:
        result = client.list_runs(
            pipeline='my-pipe', status='succeeded', page_size=20)
        mock_list.assert_called_once_with(
            pipeline='my-pipe',
            experiment=None,
            status='succeeded',
            page_token='',
            page_size=20,
        )
        assert len(result.runs) == test_case.expected_output['count']


# ------------------------------------------------------------------
# test_wait_for_run_status
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'state': 'SUCCEEDED'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_wait_for_run_status(client, test_case):
    mock_run = Mock(run_id='r-1', state='SUCCEEDED')
    with patch.object(
            client._backend, 'wait_for_run_status',
            return_value=mock_run) as mock_wait:
        result = client.wait_for_run_status(
            'r-1', status={constants.RUN_COMPLETE}, timeout=30)
        mock_wait.assert_called_once_with(
            'r-1',
            status={constants.RUN_COMPLETE},
            timeout=30,
            polling_interval=5,
            callbacks=None,
        )
        assert result.state == test_case.expected_output['state']


# ------------------------------------------------------------------
# test_create_experiment
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'experiment_id': 'exp-1'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_create_experiment(client, test_case):
    mock_exp = Mock(experiment_id='exp-1')
    with patch.object(
            client._backend, 'create_experiment',
            return_value=mock_exp) as mock_create:
        result = client.create_experiment('my-exp', description='desc')
        mock_create.assert_called_once_with('my-exp', description='desc')
        assert result.experiment_id == test_case.expected_output[
            'experiment_id']


# ------------------------------------------------------------------
# test_get_experiment
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'experiment_id': 'exp-1'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_get_experiment(client, test_case):
    mock_exp = Mock(experiment_id='exp-1')
    with patch.object(
            client._backend, 'get_experiment',
            return_value=mock_exp) as mock_get:
        result = client.get_experiment('my-exp')
        mock_get.assert_called_once_with('my-exp')
        assert result.experiment_id == test_case.expected_output[
            'experiment_id']


# ------------------------------------------------------------------
# test_list_experiments
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(
            name='delegates to backend',
            expected_output={'next_page_token': 'tok2'},
        ),
    ],
    ids=lambda tc: tc.name)
def test_list_experiments(client, test_case):
    mock_response = Mock(
        experiments=[Mock(experiment_id='e-1')], next_page_token='tok2')
    with patch.object(
            client._backend, 'list_experiments',
            return_value=mock_response) as mock_list:
        result = client.list_experiments(page_size=5)
        mock_list.assert_called_once_with(page_token='', page_size=5)
        assert result.next_page_token == test_case.expected_output[
            'next_page_token']


# ------------------------------------------------------------------
# test_delete_experiment
# ------------------------------------------------------------------


@pytest.mark.parametrize(
    'test_case', [
        TestCase(name='delegates to backend',),
    ],
    ids=lambda tc: tc.name)
def test_delete_experiment(client, test_case):
    with patch.object(client._backend, 'delete_experiment') as mock_del:
        client.delete_experiment('my-exp')
        mock_del.assert_called_once_with('my-exp')


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
