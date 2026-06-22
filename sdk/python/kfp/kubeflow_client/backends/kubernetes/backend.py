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

from collections.abc import Callable
import datetime
import json
import logging
import os
import shutil
import tempfile
import time
from typing import Any
import warnings

from google.protobuf import json_format
from kfp import compiler
from kfp.kubeflow_client import constants
from kfp.kubeflow_client.backends.kubernetes import auth
from kfp.kubeflow_client.backends.kubernetes import utils
from kfp.kubeflow_client.backends.kubernetes.types import \
    KubernetesBackendConfig
from kfp.kubeflow_client.types import Experiment
from kfp.kubeflow_client.types import ListExperimentsResponse
from kfp.kubeflow_client.types import ListPipelinesResponse
from kfp.kubeflow_client.types import ListPipelineVersionsResponse
from kfp.kubeflow_client.types import ListRunsResponse
from kfp.kubeflow_client.types import Pipeline
from kfp.kubeflow_client.types import PipelineVersion
from kfp.kubeflow_client.types import Run
from kfp.pipeline_spec import pipeline_spec_pb2
import kfp_server_api
import yaml

logger = logging.getLogger(__name__)

_VALID_UPLOAD_EXTENSIONS = ('.yaml', '.yml', '.tar.gz', '.tgz', '.zip')


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
    # Pipeline operations
    # ------------------------------------------------------------------

    def get_pipeline(self, name: str) -> Pipeline:
        """Get a pipeline by name."""
        pipeline_id = self._get_pipeline_id_by_name(name)
        if pipeline_id is None:
            raise ValueError(f'Pipeline not found: {name!r}. '
                             'Use list_pipelines() to see available pipelines.')
        return self._pipelines_api.pipeline_service_get_pipeline(
            pipeline_id=pipeline_id)

    def get_pipeline_version(
        self,
        name: str,
        version: str,
    ) -> PipelineVersion:
        """Get a specific pipeline version by pipeline name and version
        name."""
        pipeline = self.get_pipeline(name)
        version_id = self._get_version_id_by_name(pipeline.pipeline_id, version)
        if version_id is None:
            raise ValueError(f'Pipeline version not found: {version!r} '
                             f'for pipeline {name!r}.')
        return self._pipelines_api.pipeline_service_get_pipeline_version(
            pipeline_id=pipeline.pipeline_id,
            pipeline_version_id=version_id,
        )

    def list_pipelines(
        self,
        *,
        page_token: str = '',
        page_size: int = 10,
    ) -> ListPipelinesResponse:
        """List pipelines available on the server."""
        return self._pipelines_api.pipeline_service_list_pipelines(
            namespace=self.namespace,
            page_token=page_token,
            page_size=page_size,
        )

    def list_pipeline_versions(
        self,
        name: str,
        *,
        page_token: str = '',
        page_size: int = 10,
    ) -> ListPipelineVersionsResponse:
        """List versions of a pipeline by name."""
        pipeline_id = self._get_pipeline_id_by_name(name)
        if pipeline_id is None:
            raise ValueError(f'Pipeline not found: {name!r}. '
                             'Use list_pipelines() to see available pipelines.')
        return self._pipelines_api.pipeline_service_list_pipeline_versions(
            pipeline_id=pipeline_id,
            page_token=page_token,
            page_size=page_size,
        )

    def delete_pipeline(
        self,
        name: str,
        *,
        version: str | None = None,
        force: bool = False,
    ) -> None:
        """Delete a pipeline or a specific pipeline version."""
        pipeline_id = self._get_pipeline_id_by_name(name)
        if pipeline_id is None:
            raise ValueError(f'Pipeline not found: {name!r}. '
                             'Use list_pipelines() to see available pipelines.')

        if version is not None:
            version_id = self._get_version_id_by_name(pipeline_id, version)
            if version_id is None:
                raise ValueError(f'Pipeline version not found: {version!r} '
                                 f'for pipeline {name!r}.')
            self._pipelines_api.pipeline_service_delete_pipeline_version(
                pipeline_id=pipeline_id,
                pipeline_version_id=version_id,
            )
            return

        if not force:
            versions_response = (
                self._pipelines_api.pipeline_service_list_pipeline_versions(
                    pipeline_id=pipeline_id,
                    page_size=2,
                ))
            versions = versions_response.pipeline_versions or []
            if len(versions) > 1:
                raise ValueError(
                    f'Pipeline {name!r} has multiple versions. '
                    'Use force=True to delete the pipeline and all versions, '
                    'or specify version= to delete a single version.')

        self._pipelines_api.pipeline_service_delete_pipeline(
            pipeline_id=pipeline_id, cascade=True)

    def upload_pipeline(
        self,
        pipeline: Callable | str,
        *,
        name: str | None = None,
        version_name: str | None = None,
        description: str | None = None,
    ) -> PipelineVersion:
        """Upload a pipeline (or new version) to the server."""
        package_path, temp_dir = self._resolve_pipeline_to_file(pipeline)

        try:
            if name is None:
                name = self._infer_pipeline_name(pipeline, package_path)
            self._validate_pipeline_name(name)

            existing_pipeline_id = self._get_pipeline_id_by_name(name)

            if existing_pipeline_id is not None:
                return self._upload_version(
                    package_path,
                    pipeline_id=existing_pipeline_id,
                    version_name=version_name,
                    description=description,
                )
            else:
                return self._upload_new_pipeline(
                    package_path,
                    name=name,
                    version_name=version_name,
                    description=description,
                )
        finally:
            if temp_dir is not None:
                shutil.rmtree(temp_dir, ignore_errors=True)

    # ------------------------------------------------------------------
    # Run operations
    # ------------------------------------------------------------------

    def run(
        self,
        pipeline: str | Callable | Pipeline | PipelineVersion,
        *,
        params: dict[str, Any] | None = None,
        name: str | None = None,
        experiment: str | None = None,
        version: str | None = None,
    ) -> Run:
        """Run a pipeline with full dispatch logic."""
        run_name = name or self._generate_run_name(pipeline)

        version_is_usable = (
            isinstance(pipeline, Pipeline) or
            (isinstance(pipeline, str) and not self._is_yaml_path(pipeline) and
             not self._is_archive_path(pipeline)))
        if version is not None and not version_is_usable:
            logger.warning(
                'The version parameter is ignored when pipeline is not a '
                'name string (got %s).',
                type(pipeline).__name__
                if not isinstance(pipeline, str) else repr(pipeline))

        if isinstance(pipeline, PipelineVersion):
            return self.run_from_version(
                pipeline_id=pipeline.pipeline_id,
                version_id=pipeline.pipeline_version_id,
                params=params,
                run_name=run_name,
                experiment=experiment,
            )

        if isinstance(pipeline, Pipeline):
            return self.run_pipeline(
                pipeline=pipeline,
                version=version,
                params=params,
                run_name=run_name,
                experiment=experiment,
            )

        if callable(pipeline):
            return self._run_inline(
                pipeline_callable=pipeline,
                params=params,
                run_name=run_name,
                experiment=experiment,
            )

        if isinstance(pipeline, str) and self._is_yaml_path(pipeline):
            return self.run_from_file(
                file_path=pipeline,
                params=params,
                run_name=run_name,
                experiment=experiment,
            )

        if isinstance(pipeline, str) and self._is_archive_path(pipeline):
            raise ValueError(
                f'Archive files ({pipeline!r}) cannot be used for inline '
                'runs. Use upload_pipeline() first, then run by name.')

        if isinstance(pipeline, str):
            return self.run_by_name(
                pipeline_name=pipeline,
                version_name=version,
                params=params,
                run_name=run_name,
                experiment=experiment,
            )

        raise ValueError(f'Unsupported pipeline type: {type(pipeline)!r}. '
                         'Expected a pipeline name, file path, callable, '
                         'Pipeline, or PipelineVersion.')

    def get_run(self, run_id: str) -> Run:
        """Get a run by ID."""
        return self._run_api.run_service_get_run(run_id=run_id)

    def list_runs(
        self,
        *,
        pipeline: str | None = None,
        experiment: str | None = None,
        status: str | None = None,
        page_token: str = '',
        page_size: int = 10,
    ) -> ListRunsResponse:
        """List runs, optionally filtered by pipeline, experiment, or
        status."""
        pipeline_id = None
        if pipeline is not None:
            pipeline_id = self._get_pipeline_id_by_name(pipeline)
            if pipeline_id is None:
                raise ValueError(f'Pipeline not found: {pipeline!r}.')

        experiment_id = None
        if experiment is not None:
            experiment_id = self._get_experiment_id_by_name(experiment)
            if experiment_id is None:
                raise ValueError(f'Experiment not found: {experiment!r}. '
                                 'Use create_experiment() first.')

        filter_predicates = []
        if status is not None:
            filter_predicates.append({
                'operation': 'EQUALS',
                'key': 'state',
                'stringValue': status.upper(),
            })

        filter_str = None
        if filter_predicates:
            filter_str = json.dumps({'predicates': filter_predicates})

        response = self._run_api.run_service_list_runs(
            namespace=self.namespace,
            experiment_id=experiment_id or '',
            page_token=page_token,
            page_size=page_size,
            filter=filter_str,
        )

        if pipeline_id is not None and response.runs:
            original_count = len(response.runs)
            response.runs = [
                run for run in response.runs
                if (run.pipeline_version_reference and
                    run.pipeline_version_reference.pipeline_id == pipeline_id)
            ]
            filtered_count = original_count - len(response.runs)
            if filtered_count > 0:
                logger.info(
                    'Client-side pipeline filter removed %d of %d runs.',
                    filtered_count, original_count)

        return response

    def wait_for_run_status(
        self,
        run: str | Run,
        *,
        status: set[str] | None = None,
        timeout: int = 600,
        polling_interval: int = 5,
        callbacks: list[Callable[[Run], None]] | None = None,
    ) -> Run:
        """Wait for a run to reach a target state."""
        if status is None:
            status = {constants.RUN_COMPLETE}
        target_states = {state.lower() for state in status}

        run_id = run.run_id if isinstance(run, Run) else run
        start_time = time.monotonic()
        first_poll_succeeded = False
        max_auth_retries = 2
        auth_retries = 0

        while True:
            try:
                run_response = self._run_api.run_service_get_run(run_id=run_id)
                first_poll_succeeded = True
                auth_retries = 0
            except kfp_server_api.ApiException as api_error:
                if (first_poll_succeeded and api_error.status == 401 and
                        auth_retries < max_auth_retries):
                    auth_retries += 1
                    logger.info(
                        'Access token expired, refreshing '
                        '(attempt %d/%d)...', auth_retries, max_auth_retries)
                    self.refresh_credentials()
                    continue
                raise

            current_state = (run_response.state or '').lower()

            if current_state in target_states:
                self._invoke_callbacks(callbacks, run_response)
                return run_response

            if current_state in constants.TERMINAL_STATES:
                logger.info(
                    'Run %s reached terminal state %r before target %s.',
                    run_id, current_state, target_states)
                self._invoke_callbacks(callbacks, run_response)
                return run_response

            if timeout is not None:
                elapsed = time.monotonic() - start_time
                if elapsed >= timeout:
                    self._invoke_callbacks(callbacks, run_response)
                    raise TimeoutError(f'Run {run_id} did not reach state '
                                       f'{target_states} within {timeout}s. '
                                       f'Current state: {current_state!r}.')

            time.sleep(polling_interval)

    def run_by_name(
        self,
        pipeline_name: str,
        version_name: str | None,
        params: dict[str, Any] | None,
        run_name: str,
        experiment: str | None,
    ) -> Run:
        """Run an uploaded pipeline by name."""
        experiment_id = self._resolve_experiment_id(experiment)
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

    def run_pipeline(
        self,
        pipeline: Pipeline,
        version: str | None,
        params: dict[str, Any] | None,
        run_name: str,
        experiment: str | None,
    ) -> Run:
        """Run a pipeline from a Pipeline object."""
        experiment_id = self._resolve_experiment_id(experiment)
        if version:
            version_id = self._get_version_id_by_name(pipeline.pipeline_id,
                                                      version)
            if version_id is None:
                raise ValueError(f'Pipeline version not found: {version!r} '
                                 f'for pipeline {pipeline.display_name!r}.')
        else:
            version_id = self._get_latest_version_id(pipeline.pipeline_id)
        return self._run_from_version_reference(
            pipeline_id=pipeline.pipeline_id,
            version_id=version_id,
            params=params,
            run_name=run_name,
            experiment_id=experiment_id,
        )

    def run_from_version(
        self,
        pipeline_id: str,
        version_id: str,
        params: dict[str, Any] | None,
        run_name: str,
        experiment: str | None,
    ) -> Run:
        """Run a pipeline from a direct version reference."""
        experiment_id = self._resolve_experiment_id(experiment)
        return self._run_from_version_reference(
            pipeline_id=pipeline_id,
            version_id=version_id,
            params=params,
            run_name=run_name,
            experiment_id=experiment_id,
        )

    def run_from_file(
        self,
        file_path: str,
        params: dict[str, Any] | None,
        run_name: str,
        experiment: str | None,
    ) -> Run:
        """Submit a run from a compiled pipeline YAML file."""
        experiment_id = self._resolve_experiment_id(experiment)
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

    # ------------------------------------------------------------------
    # Experiment operations
    # ------------------------------------------------------------------

    def create_experiment(
        self,
        name: str,
        *,
        description: str | None = None,
    ) -> Experiment:
        """Create a new experiment.

        Returns existing if name matches.
        """
        existing_id = self._get_experiment_id_by_name(name)
        if existing_id is not None:
            if description:
                logger.warning(
                    'Experiment %r already exists; provided description will '
                    'not be applied.', name)
            return self._experiment_api.experiment_service_get_experiment(
                experiment_id=existing_id)

        experiment_body = kfp_server_api.V2beta1Experiment(
            display_name=name,
            description=description,
            namespace=self.namespace,
        )
        return self._experiment_api.experiment_service_create_experiment(
            experiment=experiment_body)

    def get_experiment(self, name: str) -> Experiment:
        """Get an experiment by name."""
        experiment_id = self._get_experiment_id_by_name(name)
        if experiment_id is None:
            raise ValueError(f'Experiment not found: {name!r}. '
                             'Use create_experiment() to create one.')
        return self._experiment_api.experiment_service_get_experiment(
            experiment_id=experiment_id)

    def list_experiments(
        self,
        *,
        page_token: str = '',
        page_size: int = 10,
    ) -> ListExperimentsResponse:
        """List experiments."""
        return self._experiment_api.experiment_service_list_experiments(
            namespace=self.namespace,
            page_token=page_token,
            page_size=page_size,
        )

    def delete_experiment(self, name: str) -> None:
        """Delete an experiment by name."""
        experiment_id = self._get_experiment_id_by_name(name)
        if experiment_id is None:
            raise ValueError(f'Experiment not found: {name!r}.')
        self._experiment_api.experiment_service_delete_experiment(
            experiment_id=experiment_id)

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

    @staticmethod
    def _invoke_callbacks(
        callbacks: list[Callable[[Run], None]] | None,
        run: Run,
    ) -> None:
        """Invoke user-provided callbacks with the final run state."""
        if not callbacks:
            return
        for callback in callbacks:
            try:
                callback(run)
            except Exception as error:
                raise RuntimeError(
                    f'Callback {callback!r} raised an exception: '
                    f'{error}') from error

    def _run_inline(
        self,
        pipeline_callable: Callable,
        params: dict[str, Any] | None,
        run_name: str,
        experiment: str | None,
    ) -> Run:
        """Compile a callable and submit inline (no upload)."""
        package_path, temp_dir = self._resolve_pipeline_to_file(
            pipeline_callable)
        try:
            return self.run_from_file(
                file_path=package_path,
                params=params,
                run_name=run_name,
                experiment=experiment,
            )
        finally:
            if temp_dir is not None:
                shutil.rmtree(temp_dir, ignore_errors=True)

    # ------------------------------------------------------------------
    # Private helpers — SDK-level preprocessing
    # ------------------------------------------------------------------

    def _resolve_pipeline_to_file(
        self,
        pipeline: Callable | str,
    ) -> tuple[str, str | None]:
        """Resolve a pipeline source to a compiled YAML file path.

        Returns:
            A tuple of (package_path, temp_dir). ``temp_dir`` is ``None``
            when the input is a user-provided file path (no cleanup needed).
        """
        if callable(pipeline):
            temp_dir = tempfile.mkdtemp()
            package_path = os.path.join(temp_dir, 'pipeline.yaml')
            try:
                compiler.Compiler().compile(
                    pipeline_func=pipeline,
                    package_path=package_path,
                )
            except Exception as error:
                shutil.rmtree(temp_dir, ignore_errors=True)
                raise ValueError(
                    f'Failed to compile pipeline: {error}') from error
            return package_path, temp_dir
        if isinstance(pipeline, str):
            if not os.path.isfile(pipeline):
                raise ValueError(f'Pipeline file not found: {pipeline!r}.')
            if not any(
                    pipeline.endswith(ext) for ext in _VALID_UPLOAD_EXTENSIONS):
                raise ValueError(f'Unsupported file type: {pipeline!r}. '
                                 f'Expected one of: '
                                 f'{", ".join(_VALID_UPLOAD_EXTENSIONS)}')
            return pipeline, None
        raise ValueError(
            f'Expected a callable or file path, got {type(pipeline)!r}.')

    def _infer_pipeline_name(
        self,
        pipeline: Callable | str,
        package_path: str,
    ) -> str:
        """Infer a pipeline name from the source."""
        if callable(pipeline) and hasattr(pipeline, 'name') and pipeline.name:
            return pipeline.name
        if callable(pipeline) and hasattr(pipeline,
                                          '__name__') and pipeline.__name__:
            return pipeline.__name__.replace('_', '-')

        name_from_spec = self._read_pipeline_name_from_yaml(package_path)
        if name_from_spec:
            return name_from_spec

        if isinstance(pipeline, str):
            basename = os.path.basename(pipeline)
            return self._strip_pipeline_extension(basename)
        return 'pipeline'

    @staticmethod
    def _validate_pipeline_name(name: str) -> None:
        """Validate that a pipeline name is non-empty."""
        if not name or name.isspace():
            raise ValueError(
                'Invalid pipeline name. Pipeline name cannot be empty '
                'or contain only whitespace.')

    @staticmethod
    def _read_pipeline_name_from_yaml(package_path: str,) -> str | None:
        """Try to extract the pipeline name from a compiled YAML file."""
        try:
            if not package_path.endswith(('.yaml', '.yml')):
                return None
            with open(package_path, 'r') as f:
                doc = yaml.safe_load(f)
            if isinstance(doc, dict):
                pipeline_info = doc.get('pipelineInfo', {})
                name = pipeline_info.get('name')
                if name and isinstance(name, str) and not name.isspace():
                    return name
        except (OSError, yaml.YAMLError):
            logger.debug(
                'Could not read pipeline name from %s.',
                package_path,
                exc_info=True)
        return None

    # ------------------------------------------------------------------
    # Private helpers — utilities
    # ------------------------------------------------------------------

    @staticmethod
    def _is_yaml_path(value: str) -> bool:
        """Check if a string looks like a YAML pipeline file path."""
        return value.endswith('.yaml') or value.endswith('.yml')

    @staticmethod
    def _is_archive_path(value: str) -> bool:
        """Check if a string looks like an archive pipeline file path."""
        return (value.endswith('.tar.gz') or value.endswith('.tgz') or
                value.endswith('.zip'))

    @staticmethod
    def _strip_pipeline_extension(filename: str) -> str:
        """Strip known pipeline file extensions including compound ones."""
        for suffix in ('.tar.gz', '.tgz', '.zip', '.yaml', '.yml'):
            if filename.endswith(suffix):
                return filename[:-len(suffix)]
        return filename

    @staticmethod
    def _generate_run_name(
        pipeline: str | Callable | Pipeline | PipelineVersion,) -> str:
        """Generate a default run display name."""
        timestamp = (
            datetime.datetime.now().astimezone().strftime('%Y-%m-%d %H-%M-%S'))
        if callable(pipeline):
            base_name = getattr(pipeline, 'name', None)
            if base_name is None:
                base_name = getattr(pipeline, '__name__', 'pipeline')
            return f'{base_name} {timestamp}'
        if isinstance(pipeline, str):
            if (KubernetesBackend._is_yaml_path(pipeline) or
                    KubernetesBackend._is_archive_path(pipeline)):
                basename = os.path.basename(pipeline)
                name = KubernetesBackend._strip_pipeline_extension(basename)
                return f'{name} {timestamp}'
            return f'{pipeline} {timestamp}'
        if isinstance(pipeline, (Pipeline, PipelineVersion)):
            display_name = getattr(pipeline, 'display_name', 'pipeline')
            return f'{display_name} {timestamp}'
        return f'pipeline {timestamp}'

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
