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
"""Kubernetes backend for PipelinesClient."""

from __future__ import annotations

import logging

from kfp.kubeflow_client.backends.kubernetes import auth
from kfp.kubeflow_client.backends.kubernetes import utils
from kfp.kubeflow_client.backends.kubernetes.types import KubernetesBackendConfig
import kfp_server_api

logger = logging.getLogger(__name__)


class KubernetesBackend:
    """Kubernetes backend providing API connectivity for PipelinesClient.

    Manages the ``kfp_server_api`` configuration, service API instances,
    namespace resolution, and credential lifecycle.

    Args:
        config: Connection parameters for the KFP API server.
    """

    def __init__(self, config: KubernetesBackendConfig) -> None:
        self._config = config
        self._namespace: str | None = config.namespace

        self._api_config = self._build_api_configuration(config)
        api_client = kfp_server_api.ApiClient(self._api_config)

        self._pipelines_api = kfp_server_api.PipelineServiceApi(api_client)
        self._run_api = kfp_server_api.RunServiceApi(api_client)
        self._experiment_api = kfp_server_api.ExperimentServiceApi(api_client)
        self._upload_api = kfp_server_api.PipelineUploadServiceApi(api_client)

    @property
    def config(self) -> KubernetesBackendConfig:
        """The backend configuration."""
        return self._config

    @property
    def api_config(self) -> kfp_server_api.Configuration:
        """The underlying kfp_server_api configuration."""
        return self._api_config

    @property
    def pipelines_api(self) -> kfp_server_api.PipelineServiceApi:
        """Pipeline service API instance."""
        return self._pipelines_api

    @property
    def run_api(self) -> kfp_server_api.RunServiceApi:
        """Run service API instance."""
        return self._run_api

    @property
    def experiment_api(self) -> kfp_server_api.ExperimentServiceApi:
        """Experiment service API instance."""
        return self._experiment_api

    @property
    def upload_api(self) -> kfp_server_api.PipelineUploadServiceApi:
        """Pipeline upload service API instance."""
        return self._upload_api

    @property
    def namespace(self) -> str:
        """Resolved target namespace (cached after first resolution)."""
        if self._namespace is None:
            self._namespace = utils.resolve_namespace(self._config.namespace)
        return self._namespace

    def refresh_credentials(self) -> None:
        """Refresh the API token using the configured refresh hook."""
        auth.refresh_credentials(self._api_config)

    def _build_api_configuration(
        self,
        config: KubernetesBackendConfig,
    ) -> kfp_server_api.Configuration:
        """Build a kfp_server_api.Configuration from
        KubernetesBackendConfig."""
        api_config = kfp_server_api.Configuration()

        if config.custom_ca:
            api_config.ssl_ca_cert = config.custom_ca

        namespace = utils.resolve_namespace(config.namespace)
        self._namespace = namespace

        if config.base_url:
            host = config.base_url
            if not (host.startswith('http://') or host.startswith('https://')):
                logger.warning('No scheme in base_url %r, defaulting to https.',
                               config.base_url)
                host = 'https://' + host
            api_config.host = host.rstrip('/')
        else:
            api_config.host = utils.discover_host(namespace)

        if config.is_secure is not None:
            api_config.verify_ssl = config.is_secure
        elif api_config.host:
            api_config.verify_ssl = api_config.host.startswith('https')

        if config.user_token:
            api_config.api_key['authorization'] = config.user_token
            api_config.api_key_prefix['authorization'] = 'Bearer'
        else:
            auth.apply_in_cluster_credentials(api_config)

        return api_config
