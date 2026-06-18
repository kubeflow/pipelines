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

import os
import unittest
from unittest.mock import MagicMock
from unittest.mock import patch

from absl.testing import parameterized
from kfp.kubeflow_client.backends.kubernetes import auth
from kfp.kubeflow_client.backends.kubernetes import utils
from kfp.kubeflow_client.backends.kubernetes.backend import KubernetesBackend
from kfp.kubeflow_client.backends.kubernetes.types import KubernetesBackendConfig
import kfp_server_api

_AUTH_MODULE = 'kfp.kubeflow_client.backends.kubernetes.auth'
_BACKEND_MODULE = 'kfp.kubeflow_client.backends.kubernetes.backend'


class TestKubernetesBackendConfig(parameterized.TestCase):

    def test_repr_masks_token(self):
        cfg = KubernetesBackendConfig(
            base_url='http://example.com',
            user_token='super-secret-token',
            namespace='ns',
        )
        rep = repr(cfg)
        self.assertNotIn('super-secret-token', rep)
        self.assertIn("'***'", rep)

    def test_repr_no_token(self):
        cfg = KubernetesBackendConfig(base_url='http://example.com')
        rep = repr(cfg)
        self.assertIn('user_token=None', rep)


@patch(f'{_BACKEND_MODULE}.KubernetesBackend.verify_backend')
class TestBuildApiConfiguration(parameterized.TestCase):

    @patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials')
    def test_explicit_base_url_sets_host(self, _mock_creds, _mock_verify):
        backend = KubernetesBackend(
            KubernetesBackendConfig(
                base_url='http://my-host:9999',
                namespace='ns',
            ))
        self.assertEqual(backend.api_config.host, 'http://my-host:9999')

    @patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials')
    def test_base_url_without_scheme_defaults_to_https(self, _mock_creds,
                                                       _mock_verify):
        backend = KubernetesBackend(
            KubernetesBackendConfig(
                base_url='my-host:9999',
                namespace='ns',
            ))
        self.assertEqual(backend.api_config.host, 'https://my-host:9999')

    @patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials')
    def test_https_url_sets_verify_ssl_true(self, _mock_creds, _mock_verify):
        backend = KubernetesBackend(
            KubernetesBackendConfig(
                base_url='https://secure.example.com',
                namespace='ns',
            ))
        self.assertTrue(backend.api_config.verify_ssl)

    @patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials')
    def test_http_url_sets_verify_ssl_false(self, _mock_creds, _mock_verify):
        backend = KubernetesBackend(
            KubernetesBackendConfig(
                base_url='http://insecure.example.com',
                namespace='ns',
            ))
        self.assertFalse(backend.api_config.verify_ssl)

    @patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials')
    def test_custom_ca_sets_ssl_ca_cert(self, _mock_creds, _mock_verify):
        backend = KubernetesBackend(
            KubernetesBackendConfig(
                base_url='https://host',
                custom_ca='/path/to/ca.crt',
                namespace='ns',
            ))
        self.assertEqual(backend.api_config.ssl_ca_cert, '/path/to/ca.crt')

    @patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials')
    def test_is_secure_overrides_scheme(self, _mock_creds, _mock_verify):
        backend = KubernetesBackend(
            KubernetesBackendConfig(
                base_url='http://insecure.example.com',
                is_secure=True,
                namespace='ns',
            ))
        self.assertTrue(backend.api_config.verify_ssl)

    @patch(f'{_AUTH_MODULE}.apply_in_cluster_credentials')
    def test_user_token_sets_api_key(self, _mock_creds, _mock_verify):
        backend = KubernetesBackend(
            KubernetesBackendConfig(
                base_url='http://host',
                user_token='my-token',
                namespace='ns',
            ))
        self.assertEqual(backend.api_config.api_key['authorization'],
                         'my-token')
        self.assertEqual(backend.api_config.api_key_prefix['authorization'],
                         'Bearer')
        _mock_creds.assert_not_called()


class TestNamespaceResolution(parameterized.TestCase):

    def test_explicit_namespace(self):
        result = utils.resolve_namespace('explicit-ns')
        self.assertEqual(result, 'explicit-ns')

    @patch('builtins.open',
           unittest.mock.mock_open(read_data='file-namespace\n'))
    def test_in_cluster_namespace_file(self):
        result = utils.resolve_namespace(None)
        self.assertEqual(result, 'file-namespace')

    def test_default_namespace_fallback(self):
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
                result = utils.resolve_namespace(None)
        self.assertEqual(result, 'kubeflow')


class TestApplyInClusterCredentials(parameterized.TestCase):

    def test_no_token_files_skips_silently(self):
        api_config = kfp_server_api.Configuration()
        with patch('os.path.exists', return_value=False):
            with patch('os.environ.get', return_value=None):
                auth.apply_in_cluster_credentials(api_config)
        self.assertNotIn('authorization', api_config.api_key)


class TestHostDiscovery(parameterized.TestCase):

    @patch.dict(os.environ, {'KF_PIPELINES_ENDPOINT': 'http://my-endpoint'})
    def test_env_var_takes_priority(self):
        result = utils.discover_host('test-ns')
        self.assertEqual(result, 'http://my-endpoint')

    @patch.dict(os.environ, {'KF_PIPELINES_ENDPOINT': 'my-endpoint:8080'})
    def test_env_var_without_scheme_adds_https(self):
        result = utils.discover_host('test-ns')
        self.assertEqual(result, 'https://my-endpoint:8080')

    @patch.dict(os.environ, {}, clear=False)
    def test_in_cluster_returns_dns_name(self):
        if 'KF_PIPELINES_ENDPOINT' in os.environ:
            del os.environ['KF_PIPELINES_ENDPOINT']
        import kubernetes
        mock_k8s = MagicMock()
        mock_k8s.config.ConfigException = kubernetes.config.ConfigException
        mock_k8s.config.load_incluster_config.return_value = None
        with patch.dict('sys.modules', {'kubernetes': mock_k8s}):
            result = utils.discover_host('test-ns')
        self.assertEqual(result,
                         'http://ml-pipeline.test-ns.svc.cluster.local:8888')

    @patch.dict(os.environ, {}, clear=False)
    def test_kube_proxy_fallback(self):
        if 'KF_PIPELINES_ENDPOINT' in os.environ:
            del os.environ['KF_PIPELINES_ENDPOINT']
        import kubernetes
        mock_k8s = MagicMock()
        mock_k8s.config.ConfigException = kubernetes.config.ConfigException
        mock_k8s.config.load_incluster_config.side_effect = (
            kubernetes.config.ConfigException('no incluster'))

        mock_k8s_client_config = MagicMock()
        mock_k8s_client_config.host = 'http://localhost:8001'
        mock_k8s.client.Configuration.return_value = mock_k8s_client_config
        mock_k8s.config.load_kube_config.return_value = None

        with patch.dict('sys.modules', {'kubernetes': mock_k8s}):
            result = utils.discover_host('test-ns')
        expected = (
            'http://localhost:8001/'
            'api/v1/namespaces/test-ns/services/ml-pipeline:http/proxy/')
        self.assertEqual(result, expected)


if __name__ == '__main__':
    unittest.main()
