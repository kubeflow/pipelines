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

import datetime
import json
import logging
import os
from typing import Any
import warnings

from google.protobuf import json_format
from kfp.kubeflow_client.backends.kubernetes import auth
from kfp.kubeflow_client.backends.kubernetes import utils
from kfp.kubeflow_client.backends.kubernetes.types import \
    KubernetesBackendConfig
from kfp.kubeflow_client.types import PipelineVersion
from kfp.kubeflow_client.types import Run
from kfp.pipeline_spec import pipeline_spec_pb2
import kfp_server_api
import yaml

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
        self._namespace: str | None = None

        self._api_config = self._build_api_configuration(config)
        api_client = kfp_server_api.ApiClient(self._api_config)

        self._pipelines_api = kfp_server_api.PipelineServiceApi(api_client)
        self._run_api = kfp_server_api.RunServiceApi(api_client)
        self._experiment_api = kfp_server_api.ExperimentServiceApi(api_client)
        self._upload_api = kfp_server_api.PipelineUploadServiceApi(api_client)
        self._healthz_api = kfp_server_api.HealthzServiceApi(api_client)

        self.verify_backend()

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

    def verify_backend(self) -> None:
        """Verify that the KFP API server is reachable.

        Logs a warning on failure but never raises, matching the
        TrainerClient.verify_backend() pattern.
        """
        try:
            self._healthz_api.healthz_service_get_healthz()
        except Exception as e:
            logger.warning(
                'KFP API server is not reachable at %s: %s. '
                'Requests will fail until the server becomes available.',
                self._api_config.host, e)

    # ------------------------------------------------------------------
    # Private helpers — name resolution
    # ------------------------------------------------------------------

    @staticmethod
    def _equals_filter(key: str, value: str) -> str:
        """Build a JSON filter string for an equality predicate."""
        return json.dumps({
            'predicates': [{
                'operation': 'EQUALS',
                'key': key,
                'stringValue': value,
            }]
        })

    def _get_pipeline_id_by_name(self, name: str) -> str | None:
        """Resolve a pipeline display name to its ID."""
        result = self._pipelines_api.pipeline_service_list_pipelines(
            namespace=self.namespace,
            filter=self._equals_filter('display_name', name))
        pipelines = result.pipelines or []
        if len(pipelines) == 0:
            return None
        if len(pipelines) == 1:
            return pipelines[0].pipeline_id
        pipeline_ids = [p.pipeline_id for p in pipelines]
        raise ValueError(
            f'Multiple pipelines found with name {name!r}: {pipeline_ids}. '
            'Use kfp.Client directly to operate by pipeline ID.')

    def _get_version_id_by_name(
        self,
        pipeline_id: str,
        version_name: str,
    ) -> str | None:
        """Resolve a version display name to its ID within a pipeline."""
        result = self._pipelines_api.pipeline_service_list_pipeline_versions(
            pipeline_id=pipeline_id,
            filter=self._equals_filter('display_name', version_name),
        )
        versions = result.pipeline_versions or []
        if len(versions) == 0:
            return None
        if len(versions) == 1:
            return versions[0].pipeline_version_id
        version_ids = [v.pipeline_version_id for v in versions]
        raise ValueError(f'Multiple versions found with name {version_name!r}: '
                         f'{version_ids}.')

    def _get_latest_version_id(self, pipeline_id: str) -> str:
        """Get the latest (most recently created) version ID for a pipeline."""
        result = self._pipelines_api.pipeline_service_list_pipeline_versions(
            pipeline_id=pipeline_id,
            page_size=1,
            sort_by='created_at desc',
        )
        versions = result.pipeline_versions or []
        if not versions:
            raise ValueError(f'Pipeline {pipeline_id!r} has no versions.')
        return versions[0].pipeline_version_id

    def _get_experiment_id_by_name(self, name: str) -> str | None:
        """Resolve an experiment display name to its ID."""
        result = self._experiment_api.experiment_service_list_experiments(
            namespace=self.namespace,
            filter=self._equals_filter('display_name', name),
        )
        experiments = result.experiments or []
        if len(experiments) == 0:
            return None
        if len(experiments) == 1:
            return experiments[0].experiment_id
        experiment_ids = [e.experiment_id for e in experiments]
        raise ValueError(f'Multiple experiments found with name {name!r}: '
                         f'{experiment_ids}.')

    # ------------------------------------------------------------------
    # Private helpers — pipeline upload
    # ------------------------------------------------------------------

    def _upload_new_pipeline(
        self,
        package_path: str,
        *,
        name: str,
        version_name: str | None,
        description: str | None,
    ) -> PipelineVersion:
        """Upload a brand-new pipeline and return its first version."""
        upload_kwargs: dict[str, Any] = {
            'name': name,
            'namespace': self.namespace,
        }
        if description:
            upload_kwargs['description'] = description

        pipeline_response = self._upload_api.upload_pipeline(
            package_path, **upload_kwargs)

        versions_response = (
            self._pipelines_api.pipeline_service_list_pipeline_versions(
                pipeline_id=pipeline_response.pipeline_id,
                page_size=1,
                sort_by='created_at desc',
            ))
        versions = versions_response.pipeline_versions or []
        if not versions:
            raise RuntimeError(
                f'Pipeline {name!r} was uploaded but no version was '
                'created by the server. This is unexpected.')

        first_version = versions[0]

        if version_name and first_version.display_name != version_name:
            try:
                update_body = kfp_server_api.V2beta1PipelineVersion(
                    pipeline_id=first_version.pipeline_id,
                    pipeline_version_id=first_version.pipeline_version_id,
                    display_name=version_name,
                )
                self._pipelines_api.pipeline_service_update_pipeline_version(
                    pipeline_version_pipeline_id=first_version.pipeline_id,
                    pipeline_version_pipeline_version_id=(
                        first_version.pipeline_version_id),
                    pipeline_version=update_body,
                )
                first_version.display_name = version_name
            except kfp_server_api.ApiException:
                warnings.warn(
                    f'Could not rename the first pipeline version to '
                    f'{version_name!r}. The upload API does not accept a '
                    f'version name for the initial version; subsequent '
                    f'versions will use the provided name.',
                    stacklevel=2,
                )

        return first_version

    def _upload_version(
        self,
        package_path: str,
        *,
        pipeline_id: str,
        version_name: str | None,
        description: str | None,
    ) -> PipelineVersion:
        """Upload a new version to an existing pipeline."""
        if not version_name:
            version_name = (
                datetime.datetime.now().astimezone().strftime(
                    '%Y-%m-%d %H-%M-%S'))
        upload_kwargs: dict[str, Any] = {
            'pipelineid': pipeline_id,
            'name': version_name,
        }
        if description:
            upload_kwargs['description'] = description

        return self._upload_api.upload_pipeline_version(package_path,
                                                        **upload_kwargs)

    # ------------------------------------------------------------------
    # Private helpers — run creation
    # ------------------------------------------------------------------

    def _resolve_experiment_id(
        self,
        experiment: str | None,
    ) -> str | None:
        """Resolve an experiment name to its ID.

        Returns None when experiment is None, letting the server use its
        default experiment.
        """
        if experiment is None:
            return None
        experiment_id = self._get_experiment_id_by_name(experiment)
        if experiment_id is None:
            raise ValueError(f'Experiment not found: {experiment!r}. '
                             'Use create_experiment() first.')
        return experiment_id

    def _run_by_name(
        self,
        pipeline_name: str,
        version_name: str | None,
        params: dict[str, Any] | None,
        run_name: str,
        experiment_id: str | None,
    ) -> Run:
        """Run an uploaded pipeline by name."""
        pipeline_id = self._get_pipeline_id_by_name(pipeline_name)
        if pipeline_id is None:
            raise ValueError(f'Pipeline not found: {pipeline_name!r}. '
                             'Upload it first with upload_pipeline().')

        if version_name:
            version_id = self._get_version_id_by_name(pipeline_id, version_name)
            if version_id is None:
                raise ValueError(
                    f'Pipeline version not found: {version_name!r} '
                    f'for pipeline {pipeline_name!r}.')
        else:
            version_id = self._get_latest_version_id(pipeline_id)

        return self._run_from_version_reference(
            pipeline_id=pipeline_id,
            version_id=version_id,
            params=params,
            run_name=run_name,
            experiment_id=experiment_id,
        )

    def _run_from_version_reference(
        self,
        pipeline_id: str,
        version_id: str,
        params: dict[str, Any] | None,
        run_name: str,
        experiment_id: str | None,
    ) -> Run:
        """Create a run from a pipeline version reference (ID-based)."""
        runtime_config = kfp_server_api.V2beta1RuntimeConfig(
            parameters=params or {},)
        pipeline_version_reference = (
            kfp_server_api.V2beta1PipelineVersionReference(
                pipeline_id=pipeline_id,
                pipeline_version_id=version_id,
            ))
        run_body = kfp_server_api.V2beta1Run(
            display_name=run_name,
            experiment_id=experiment_id,
            pipeline_version_reference=pipeline_version_reference,
            runtime_config=runtime_config,
        )
        return self._run_api.run_service_create_run(run=run_body)

    def _run_from_file(
        self,
        file_path: str,
        params: dict[str, Any] | None,
        run_name: str,
        experiment_id: str | None,
    ) -> Run:
        """Submit a run from a compiled pipeline YAML file."""
        if not os.path.isfile(file_path):
            raise ValueError(f'Pipeline file not found: {file_path}')
        pipeline_spec_dict = self._load_pipeline_spec(file_path)
        runtime_config = kfp_server_api.V2beta1RuntimeConfig(
            parameters=params or {},)
        run_body = kfp_server_api.V2beta1Run(
            display_name=run_name,
            experiment_id=experiment_id,
            pipeline_spec=pipeline_spec_dict,
            runtime_config=runtime_config,
        )
        return self._run_api.run_service_create_run(run=run_body)

    def _load_pipeline_spec(self, file_path: str) -> dict:
        """Load a pipeline spec from a YAML file and return as dict."""
        with open(file_path, 'r') as f:
            try:
                docs = list(yaml.safe_load_all(f))
            except yaml.YAMLError as e:
                raise ValueError(
                    f'Failed to parse pipeline YAML at {file_path!r}: {e}'
                ) from e

        if not docs or not docs[0]:
            raise ValueError(
                f'Pipeline file is empty or contains no valid YAML '
                f'documents: {file_path}')
        pipeline_spec_dict = docs[0]
        if len(docs) > 2:
            raise ValueError(
                f'Expected at most 2 YAML documents (pipeline spec + '
                f'platform spec), got {len(docs)} in {file_path!r}.')
        platform_spec_dict = docs[1] if len(docs) > 1 and docs[1] else {}

        pipeline_spec = json_format.ParseDict(
            pipeline_spec_dict,
            pipeline_spec_pb2.PipelineSpec(),
            ignore_unknown_fields=True)
        platform_spec = json_format.ParseDict(
            platform_spec_dict,
            pipeline_spec_pb2.PlatformSpec(),
            ignore_unknown_fields=True)

        if platform_spec == pipeline_spec_pb2.PlatformSpec():
            return json_format.MessageToDict(pipeline_spec)
        return {
            'pipeline_spec': json_format.MessageToDict(pipeline_spec),
            'platform_spec': json_format.MessageToDict(platform_spec),
        }

    # ------------------------------------------------------------------
    # Private helpers — configuration
    # ------------------------------------------------------------------

    def _build_api_configuration(
        self,
        config: KubernetesBackendConfig,
    ) -> kfp_server_api.Configuration:
        """Build a kfp_server_api.Configuration from
        KubernetesBackendConfig."""
        api_config = kfp_server_api.Configuration()

        if config.custom_ca:
            api_config.ssl_ca_cert = config.custom_ca

        if config.base_url:
            host = config.base_url
            if not (host.startswith('http://') or host.startswith('https://')):
                logger.warning('No scheme in base_url %r, defaulting to https.',
                               config.base_url)
                host = 'https://' + host
            api_config.host = host.rstrip('/')
        else:
            api_config.host = utils.discover_host(self.namespace)

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
