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
"""Unit tests for the Kubernetes backend (config, auth, utils)."""

from dataclasses import dataclass
from dataclasses import field
import os
from typing import Any
from unittest.mock import MagicMock
from unittest.mock import mock_open
from unittest.mock import patch

from kfp.kubeflow_client.backends.kubernetes import auth
from kfp.kubeflow_client.backends.kubernetes import utils
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
    __test__ = False


# ------------------------------------------------------------------
# Fixtures
# ------------------------------------------------------------------


@pytest.fixture
def make_backend():
    """Factory fixture to create a KubernetesBackend with mocked auth."""

    def _make(base_url='http://localhost:8888', namespace='test-ns', **kwargs):
        with patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials'), \
             patch(f'{_BACKEND_MODULE}.KubernetesBackend.verify_backend'):
            return KubernetesBackend(
                KubernetesBackendConfig(
                    base_url=base_url, namespace=namespace, **kwargs))

    return _make


# ------------------------------------------------------------------
# test_backend_config_repr
# ------------------------------------------------------------------


@pytest.mark.parametrize('test_case', [
    TestCase(
        name='token is masked in repr',
        config={
            'base_url': 'http://example.com',
            'user_token': 'super-secret-token',
            'namespace': 'ns',
        },
        expected_output={
            'must_not_contain': 'super-secret-token',
            'must_contain': "'***'",
        },
    ),
    TestCase(
        name='no token shows None',
        config={
            'base_url': 'http://example.com',
        },
        expected_output={
            'must_contain': 'user_token=None',
        },
    ),
])
def test_backend_config_repr(test_case):
    """Test KubernetesBackendConfig.__repr__ across scenarios."""
    cfg = KubernetesBackendConfig(**test_case.config)
    rep = repr(cfg)

    if 'must_not_contain' in test_case.expected_output:
        assert test_case.expected_output['must_not_contain'] not in rep
    assert test_case.expected_output['must_contain'] in rep


# ------------------------------------------------------------------
# test_build_api_configuration
# ------------------------------------------------------------------


@pytest.mark.parametrize('test_case', [
    TestCase(
        name='explicit http base_url sets host directly',
        config={'base_url': 'http://my-host:9999'},
        expected_output={'host': 'http://my-host:9999'},
    ),
    TestCase(
        name='base_url without scheme defaults to https',
        config={'base_url': 'my-host:9999'},
        expected_output={'host': 'https://my-host:9999'},
    ),
    TestCase(
        name='https base_url sets host correctly',
        config={'base_url': 'https://secure.example.com'},
        expected_output={'host': 'https://secure.example.com'},
    ),
    TestCase(
        name='https url sets verify_ssl true',
        config={'base_url': 'https://secure.example.com'},
        expected_output={'verify_ssl': True},
    ),
    TestCase(
        name='http url sets verify_ssl false',
        config={'base_url': 'http://insecure.example.com'},
        expected_output={'verify_ssl': False},
    ),
    TestCase(
        name='is_secure overrides scheme-based verify_ssl',
        config={
            'base_url': 'http://insecure.example.com',
            'is_secure': True
        },
        expected_output={'verify_ssl': True},
    ),
    TestCase(
        name='custom_ca sets ssl_ca_cert',
        config={
            'base_url': 'https://host',
            'custom_ca': '/path/to/ca.crt'
        },
        expected_output={'ssl_ca_cert': '/path/to/ca.crt'},
    ),
    TestCase(
        name='user_token sets api_key and prefix',
        config={
            'base_url': 'http://host',
            'user_token': 'my-token'
        },
        expected_output={
            'api_key': 'my-token',
            'api_key_prefix': 'Bearer',
        },
    ),
])
def test_build_api_configuration(make_backend, test_case):
    """Test KubernetesBackend API configuration from config parameters."""
    backend = make_backend(**test_case.config)

    for key, expected in test_case.expected_output.items():
        if key == 'host':
            assert backend.api_config.host == expected
        elif key == 'verify_ssl':
            assert backend.api_config.verify_ssl == expected
        elif key == 'ssl_ca_cert':
            assert backend.api_config.ssl_ca_cert == expected
        elif key == 'api_key':
            assert backend.api_config.api_key['authorization'] == expected
        elif key == 'api_key_prefix':
            assert backend.api_config.api_key_prefix[
                'authorization'] == expected


# ------------------------------------------------------------------
# test_resolve_namespace
# ------------------------------------------------------------------


@pytest.mark.parametrize('test_case', [
    TestCase(
        name='explicit namespace returned as-is',
        config={'namespace': 'explicit-ns'},
        expected_output='explicit-ns',
    ),
    TestCase(
        name='in-cluster namespace file read',
        config={
            'namespace': None,
            'mock_file': 'file-namespace\n'
        },
        expected_output='file-namespace',
    ),
    TestCase(
        name='default fallback to kubeflow',
        config={
            'namespace': None,
            'mock_file': None
        },
        expected_output='kubeflow',
    ),
])
def test_resolve_namespace(test_case):
    """Test utils.resolve_namespace across discovery scenarios."""
    ns = test_case.config['namespace']
    mock_file_content = test_case.config.get('mock_file')

    if ns is not None:
        assert utils.resolve_namespace(ns) == test_case.expected_output
    elif mock_file_content is not None:
        with patch('builtins.open', mock_open(read_data=mock_file_content)):
            assert utils.resolve_namespace(None) == test_case.expected_output
    else:
        original_open = open

        def mock_open_fn(path, *args, **kwargs):
            if 'serviceaccount' in str(path):
                raise FileNotFoundError
            return original_open(path, *args, **kwargs)

        import kubernetes
        mock_k8s = MagicMock()
        mock_k8s.config.ConfigException = kubernetes.config.ConfigException
        mock_k8s.config.list_kube_config_contexts.side_effect = (
            FileNotFoundError)
        with patch('builtins.open', side_effect=mock_open_fn):
            with patch.dict('sys.modules', {'kubernetes': mock_k8s}):
                assert utils.resolve_namespace(
                    None) == test_case.expected_output


# ------------------------------------------------------------------
# test_apply_in_cluster_credentials
# ------------------------------------------------------------------


@pytest.mark.parametrize('test_case', [
    TestCase(
        name='no token files skips silently',
        expected_output={'authorization_absent': True},
    ),
])
def test_apply_in_cluster_credentials(test_case):
    """Test auth.apply_in_cluster_credentials behavior."""
    api_config = kfp_server_api.Configuration()
    with patch('os.path.exists', return_value=False):
        with patch('os.environ.get', return_value=None):
            auth.apply_in_cluster_credentials(api_config)
    assert 'authorization' not in api_config.api_key


# ------------------------------------------------------------------
# test_discover_host
# ------------------------------------------------------------------


@pytest.mark.parametrize('test_case', [
    TestCase(
        name='env var takes priority',
        config={'env': {
            'KF_PIPELINES_ENDPOINT': 'http://my-endpoint'
        }},
        expected_output='http://my-endpoint',
    ),
    TestCase(
        name='env var without scheme adds https',
        config={'env': {
            'KF_PIPELINES_ENDPOINT': 'my-endpoint:8080'
        }},
        expected_output='https://my-endpoint:8080',
    ),
    TestCase(
        name='in-cluster returns DNS name',
        config={'mode': 'in_cluster'},
        expected_output=('http://ml-pipeline.test-ns.svc.cluster.local:8888'),
    ),
    TestCase(
        name='kube proxy fallback',
        config={'mode': 'kube_proxy'},
        expected_output=(
            'http://localhost:8001/'
            'api/v1/namespaces/test-ns/services/ml-pipeline:http/proxy/'),
    ),
])
def test_discover_host(test_case):
    """Test utils.discover_host across discovery scenarios."""
    env_vars = test_case.config.get('env')
    mode = test_case.config.get('mode')

    if env_vars:
        with patch.dict(os.environ, env_vars):
            result = utils.discover_host('test-ns')
    elif mode == 'in_cluster':
        env_clean = {
            k: v for k, v in os.environ.items() if k != 'KF_PIPELINES_ENDPOINT'
        }
        import kubernetes
        mock_k8s = MagicMock()
        mock_k8s.config.ConfigException = kubernetes.config.ConfigException
        mock_k8s.config.load_incluster_config.return_value = None
        with patch.dict(os.environ, env_clean, clear=True):
            with patch.dict('sys.modules', {'kubernetes': mock_k8s}):
                result = utils.discover_host('test-ns')
    elif mode == 'kube_proxy':
        env_clean = {
            k: v for k, v in os.environ.items() if k != 'KF_PIPELINES_ENDPOINT'
        }
        import kubernetes
        mock_k8s = MagicMock()
        mock_k8s.config.ConfigException = kubernetes.config.ConfigException
        mock_k8s.config.load_incluster_config.side_effect = (
            kubernetes.config.ConfigException('no incluster'))
        mock_k8s_client_config = MagicMock()
        mock_k8s_client_config.host = 'http://localhost:8001'
        mock_k8s.client.Configuration.return_value = mock_k8s_client_config
        mock_k8s.config.load_kube_config.return_value = None
        with patch.dict(os.environ, env_clean, clear=True):
            with patch.dict('sys.modules', {'kubernetes': mock_k8s}):
                result = utils.discover_host('test-ns')

    assert result == test_case.expected_output
