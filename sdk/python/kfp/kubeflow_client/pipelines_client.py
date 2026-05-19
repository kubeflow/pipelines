# Copyright 2026 The Kubeflow Authors
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
"""PipelinesClient — simplified, name-first client for Kubeflow Pipelines.

This module provides a streamlined interface over the KFP backend API,
designed for re-export by the Kubeflow SDK at ``kubeflow.pipelines``.
"""

from __future__ import annotations

import dataclasses
import datetime
import json
import logging
import os
import shutil
import tempfile
import time
from typing import Any, Callable, TYPE_CHECKING

if TYPE_CHECKING:
    from kfp.client import Client

from google.protobuf import json_format
from kfp import compiler
from kfp.kubeflow_client import constants
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

__all__ = ['PipelinesClient', 'PipelinesBackendConfig']

_IN_CLUSTER_DNS_NAME = 'http://ml-pipeline.{}.svc.cluster.local:8888'
_KUBE_PROXY_PATH = 'api/v1/namespaces/{}/services/ml-pipeline:http/proxy/'
_NAMESPACE_PATH = '/var/run/secrets/kubernetes.io/serviceaccount/namespace'
_DEFAULT_NAMESPACE = 'kubeflow'
_VALID_UPLOAD_EXTENSIONS = ('.yaml', '.yml', '.tar.gz', '.tgz', '.zip')


@dataclasses.dataclass
class PipelinesBackendConfig:
    """Connection configuration for the KFP API server.

    Args:
        base_url: KFP API server URL including scheme and port
            (e.g. ``https://ml-pipeline.example.com:8080``). If omitted,
            auto-discovered following kfp.Client conventions (in-cluster DNS
            or kubeconfig proxy).
        user_token: Bearer token for authentication.
        is_secure: Whether to verify TLS certificates (controls
            ``verify_ssl`` on the underlying HTTP client). Does not control
            whether the connection uses TLS — that is determined by the URL
            scheme. Inferred from scheme if omitted (``True`` for https,
            ``False`` for http).
        custom_ca: Path to PEM-encoded root certificates.
        namespace: Kubernetes namespace. If omitted, auto-detected.
    """

    base_url: str | None = None
    user_token: str | None = None
    is_secure: bool | None = None
    custom_ca: str | None = None
    namespace: str | None = None

    def __repr__(self) -> str:
        token_display = '***' if self.user_token else None
        return (f'PipelinesBackendConfig('
                f'base_url={self.base_url!r}, '
                f'user_token={token_display!r}, '
                f'is_secure={self.is_secure!r}, '
                f'custom_ca={self.custom_ca!r}, '
                f'namespace={self.namespace!r})')


class PipelinesClient:
    """Simplified, name-first client for Kubeflow Pipelines.

    Provides the core author → compile → upload → run → monitor workflow
    from a single import. Designed to be re-exported by the Kubeflow SDK
    at ``kubeflow.pipelines.PipelinesClient``.

    Args:
        backend_config: Connection parameters for the KFP API server.
            When ``None``, uses ``PipelinesBackendConfig()`` (zero-arg
            construction with auto-discovery).
    """

    def __init__(
        self,
        backend_config: PipelinesBackendConfig | None = None,
    ) -> None:
        if backend_config is None:
            backend_config = PipelinesBackendConfig()
        self._config = backend_config
        self._namespace = backend_config.namespace

        self._api_config = self._build_api_configuration(backend_config)
        api_client = kfp_server_api.ApiClient(self._api_config)

        self._pipelines_api = kfp_server_api.PipelineServiceApi(api_client)
        self._run_api = kfp_server_api.RunServiceApi(api_client)
        self._experiment_api = kfp_server_api.ExperimentServiceApi(api_client)
        self._upload_api = kfp_server_api.PipelineUploadServiceApi(api_client)
        self._kfp_client_instance = None

    # ------------------------------------------------------------------
    # Pipeline operations
    # ------------------------------------------------------------------

    def upload_pipeline(
        self,
        pipeline: Callable | str,
        *,
        name: str | None = None,
        version: str | None = None,
        description: str | None = None,
    ) -> PipelineVersion:
        """Upload a pipeline (or new version) to the server.

        Handles callable functions, file paths, new pipelines, and new
        versions through a single unified method.

        Note:
            When creating a **new** pipeline with an explicit ``version``
            name, the server auto-generates the first version name during
            upload. A best-effort rename is attempted afterward; if the rename
            fails (e.g. due to permissions), a warning is logged and the
            returned version retains the server-generated name.

        Args:
            pipeline: A ``@dsl.pipeline``-decorated function or a path to a
                compiled pipeline YAML file.
            name: Display name for the pipeline. If omitted, auto-generated
                from the function's ``@dsl.pipeline(name=...)`` value or
                the filename without extension.
            version: Version label. If omitted, auto-generated. Calling
                ``upload_pipeline`` again with the same ``name`` and no
                explicit ``version`` creates a new version each time.
            description: Pipeline description.

        Returns:
            A ``PipelineVersion`` object representing the uploaded version.
        """
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
                    version_name=version,
                    description=description,
                )
            else:
                return self._upload_new_pipeline(
                    package_path,
                    name=name,
                    version_name=version,
                    description=description,
                )
        finally:
            if temp_dir is not None:
                shutil.rmtree(temp_dir, ignore_errors=True)

    def get_pipeline(self, name: str) -> Pipeline:
        """Get a pipeline by name.

        Args:
            name: Pipeline display name.

        Returns:
            A ``Pipeline`` object.

        Raises:
            ValueError: If no pipeline matches or multiple pipelines match.
        """
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
        """Get a specific pipeline version by pipeline name and version name.

        Args:
            name: Pipeline display name.
            version: Version display name.

        Returns:
            A ``PipelineVersion`` object.

        Raises:
            ValueError: If the pipeline or version is not found.
        """
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
        """List pipelines available on the server.

        Args:
            page_token: Token for obtaining the next page.
            page_size: Number of results per page.

        Returns:
            A ``ListPipelinesResponse`` with ``.pipelines`` and
            ``.next_page_token``.
        """
        return self._pipelines_api.pipeline_service_list_pipelines(
            namespace=self._get_namespace(),
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
        """List versions of a pipeline by name.

        Args:
            name: Pipeline display name.
            page_token: Token for obtaining the next page.
            page_size: Number of results per page.

        Returns:
            A ``ListPipelineVersionsResponse`` with ``.pipeline_versions``
            and ``.next_page_token``.
        """
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
        """Delete a pipeline or a specific pipeline version.

        Args:
            name: Pipeline display name.
            version: If provided, delete only this version. If ``None``,
                delete the entire pipeline and all versions.
            force: When deleting an entire pipeline, required if the pipeline
                has more than one version. Ignored when ``version`` is set.

        Raises:
            ValueError: If the pipeline has multiple versions and
                ``force=False``.
        """
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
        """Run a pipeline.

        Supports multiple input types:
        - A pipeline name (``str`` without file extension): resolves the
          uploaded pipeline on the server.
        - A path to a compiled YAML file (``str`` ending in ``.yaml``/
          ``.yml``): compile-and-submit inline, no upload.
        - A ``@dsl.pipeline``-decorated callable: compile-and-submit inline.
        - A ``Pipeline`` or ``PipelineVersion`` object (from
          ``get_pipeline``/``upload_pipeline``).

        Note:
            String inputs are classified as file paths when they end in
            ``.yaml`` or ``.yml``. If you have an uploaded pipeline whose
            display name ends with such an extension, pass the ``Pipeline``
            object from ``get_pipeline()`` instead.

        Args:
            pipeline: Pipeline to run (see above).
            params: Pipeline parameters as a dict.
            name: Run display name. Auto-generated if omitted.
            experiment: Experiment name. If ``None``, a ``Default`` experiment
                is used (created automatically if it does not already exist).
                If provided and the experiment does not exist, raises
                ``ValueError``.
            version: Pipeline version name (used only when ``pipeline`` is a
                name string). Uses latest version if omitted.

        Returns:
            A ``Run`` object.
        """
        experiment_id = self._resolve_experiment_id(experiment)
        run_name = name or self._generate_run_name(pipeline)

        version_is_usable = (
            isinstance(pipeline, str) and not self._is_yaml_path(pipeline) and
            not self._is_archive_path(pipeline))
        if version is not None and not version_is_usable:
            logger.warning(
                'The version parameter is ignored when pipeline is not a '
                'name string (got %s).',
                type(pipeline).__name__
                if not isinstance(pipeline, str) else repr(pipeline))

        if isinstance(pipeline, PipelineVersion):
            return self._run_from_version_reference(
                pipeline_id=pipeline.pipeline_id,
                version_id=pipeline.pipeline_version_id,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )

        if isinstance(pipeline, Pipeline):
            latest_version_id = self._get_latest_version_id(
                pipeline.pipeline_id)
            return self._run_from_version_reference(
                pipeline_id=pipeline.pipeline_id,
                version_id=latest_version_id,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )

        if callable(pipeline):
            return self._run_inline(
                pipeline_callable=pipeline,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )

        # Dispatch YAML and archive separately because YAML triggers inline
        # run while archives must be rejected with guidance.
        if isinstance(pipeline, str) and self._is_yaml_path(pipeline):
            return self._run_from_file(
                file_path=pipeline,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )

        if isinstance(pipeline, str) and self._is_archive_path(pipeline):
            raise ValueError(
                f'Archive files ({pipeline!r}) cannot be used for inline '
                'runs. Use upload_pipeline() first, then run by name.')

        if isinstance(pipeline, str):
            return self._run_by_name(
                pipeline_name=pipeline,
                version_name=version,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )

        raise ValueError(f'Unsupported pipeline type: {type(pipeline)!r}. '
                         'Expected a pipeline name, file path, callable, '
                         'Pipeline, or PipelineVersion.')

    def get_run(self, run_id: str) -> Run:
        """Get a run by ID.

        Args:
            run_id: The run identifier.

        Returns:
            A ``Run`` object.
        """
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
        """List runs, optionally filtered by pipeline, experiment, or status.

        Note:
            The ``pipeline`` filter is applied client-side because the KFP
            v2beta1 API does not support server-side filtering by pipeline ID.
            When used, the returned page may contain fewer items than
            ``page_size`` — including zero items with a non-empty
            ``next_page_token`` if all runs on that server page belong to
            other pipelines. Callers should continue paginating until
            ``next_page_token`` is empty.

        Args:
            pipeline: Filter by pipeline display name (client-side).
            experiment: Filter by experiment display name.
            status: Filter by run state (e.g. ``"succeeded"``).
            page_token: Token for obtaining the next page.
            page_size: Number of results per page.

        Returns:
            A ``ListRunsResponse`` with ``.runs`` and ``.next_page_token``.
        """
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
            # Server-side filter expects uppercase enum values (e.g. SUCCEEDED)
            filter_predicates.append({
                'operation': 'EQUALS',
                'key': 'state',
                'stringValue': status.upper(),
            })

        filter_str = None
        if filter_predicates:
            filter_str = json.dumps({'predicates': filter_predicates})

        response = self._run_api.run_service_list_runs(
            namespace=self._get_namespace(),
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
                logger.debug(
                    'Client-side pipeline filter removed %d of %d runs.',
                    filtered_count, original_count)

        return response

    def wait_for_run_status(
        self,
        run: str | Run,
        *,
        status: set[str] | None = None,
        timeout: int | None = None,
        polling_interval: int = 5,
        callbacks: list[Callable[[Run], None]] | None = None,
    ) -> Run:
        """Wait for a run to reach a target state.

        Args:
            run: A ``Run`` object or a run ID string.
            status: Set of states to wait for. Defaults to
                ``{constants.RUN_COMPLETE}`` (``"succeeded"``). The wait
                always exits immediately on any terminal state regardless
                of this parameter.
            timeout: Maximum seconds to wait. If ``None``, wait indefinitely.
            polling_interval: Seconds between status checks.
            callbacks: Called with the final ``Run`` object when the wait
                ends (on any stop condition).

        Returns:
            The ``Run`` object at the time the wait concluded.

        Raises:
            TimeoutError: If ``timeout`` expires before reaching a stop
                condition.
        """
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
                    self._refresh_credentials()
                    continue
                raise

            # Normalize to lowercase for comparison against our constants
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

        If an experiment with the given name already exists, returns it.

        Args:
            name: Experiment display name.
            description: Experiment description.

        Returns:
            An ``Experiment`` object.
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
            namespace=self._get_namespace(),
        )
        return self._experiment_api.experiment_service_create_experiment(
            experiment=experiment_body)

    def get_experiment(self, name: str) -> Experiment:
        """Get an experiment by name.

        Args:
            name: Experiment display name.

        Returns:
            An ``Experiment`` object.

        Raises:
            ValueError: If no experiment with that name is found.
        """
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
        """List experiments.

        Args:
            page_token: Token for obtaining the next page.
            page_size: Number of results per page.

        Returns:
            A ``ListExperimentsResponse`` with ``.experiments`` and
            ``.next_page_token``.
        """
        return self._experiment_api.experiment_service_list_experiments(
            namespace=self._get_namespace(),
            page_token=page_token,
            page_size=page_size,
        )

    def delete_experiment(self, name: str) -> None:
        """Delete an experiment by name.

        Args:
            name: Experiment display name.

        Raises:
            ValueError: If no experiment with that name is found.
        """
        experiment_id = self._get_experiment_id_by_name(name)
        if experiment_id is None:
            raise ValueError(f'Experiment not found: {name!r}.')
        self._experiment_api.experiment_service_delete_experiment(
            experiment_id=experiment_id)

    # ------------------------------------------------------------------
    # Escape hatch
    # ------------------------------------------------------------------

    @property
    def kfp_client(self) -> Client:
        """Access the underlying ``kfp.Client`` for advanced operations.

        Lazily constructed on first access, sharing connection
        parameters from ``PipelinesBackendConfig``.
        """
        if self._kfp_client_instance is None:
            from kfp.client import Client
            kwargs: dict[str, Any] = {}
            if self._config.base_url:
                kwargs['host'] = self._config.base_url
            if self._config.user_token:
                kwargs['existing_token'] = self._config.user_token
            if self._config.namespace:
                kwargs['namespace'] = self._config.namespace
            if self._config.custom_ca:
                kwargs['ssl_ca_cert'] = self._config.custom_ca
            if self._config.is_secure is not None:
                kwargs['verify_ssl'] = self._config.is_secure
            self._kfp_client_instance = Client(**kwargs)
        return self._kfp_client_instance

    # ------------------------------------------------------------------
    # Private helpers — connection
    # ------------------------------------------------------------------

    def _build_api_configuration(
        self,
        config: PipelinesBackendConfig,
    ) -> kfp_server_api.Configuration:
        """Build a kfp_server_api.Configuration from PipelinesBackendConfig."""
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
            api_config.host = self._discover_host()

        if config.is_secure is not None:
            api_config.verify_ssl = config.is_secure
        elif api_config.host:
            api_config.verify_ssl = api_config.host.startswith('https')

        if config.user_token:
            api_config.api_key['authorization'] = config.user_token
            api_config.api_key_prefix['authorization'] = 'Bearer'
        else:
            self._apply_in_cluster_credentials(api_config)

        return api_config

    def _discover_host(self) -> str:
        """Auto-discover the KFP API server endpoint."""
        endpoint_from_env = os.environ.get('KF_PIPELINES_ENDPOINT')
        if endpoint_from_env:
            host = endpoint_from_env.rstrip('/')
            if not (host.startswith('http://') or host.startswith('https://')):
                logger.warning(
                    'No scheme in KF_PIPELINES_ENDPOINT %r, defaulting '
                    'to https.', endpoint_from_env)
                host = 'https://' + host
            return host

        namespace = self._get_namespace()

        try:
            import kubernetes as k8s
        except ImportError:
            logger.debug('kubernetes package not installed.')
            return _IN_CLUSTER_DNS_NAME.format(namespace)

        try:
            k8s.config.load_incluster_config()
            return _IN_CLUSTER_DNS_NAME.format(namespace)
        except (k8s.config.ConfigException, FileNotFoundError):
            logger.debug('In-cluster config not available.', exc_info=True)

        # Only the host URL is extracted from kubeconfig; kubeconfig auth
        # credentials are not applied to the API configuration. This matches
        # kfp.Client behavior and assumes kubectl proxy handles auth.
        try:
            k8s_config = k8s.client.Configuration()
            k8s.config.load_kube_config(client_configuration=k8s_config)
            if k8s_config.host:
                return (k8s_config.host.rstrip('/') + '/' +
                        _KUBE_PROXY_PATH.format(namespace))
        except (k8s.config.ConfigException, FileNotFoundError):
            logger.debug('Kubeconfig not available.', exc_info=True)

        fallback = _IN_CLUSTER_DNS_NAME.format(namespace)
        logger.warning(
            'Could not detect KFP endpoint via in-cluster config or '
            'kubeconfig. Falling back to %s. Set base_url in '
            'PipelinesBackendConfig or KF_PIPELINES_ENDPOINT to override.',
            fallback)
        return fallback

    def _apply_in_cluster_credentials(
        self,
        api_config: kfp_server_api.Configuration,
    ) -> None:
        """Apply default in-cluster service account credentials."""
        _kfp_sa_token_path = '/var/run/secrets/kubeflow/pipelines/token'
        _k8s_sa_token_path = '/var/run/secrets/kubernetes.io/serviceaccount/token'
        _token_path_env = 'KF_PIPELINES_SA_TOKEN_PATH'

        token_path = os.environ.get(_token_path_env)
        if not token_path:
            if os.path.exists(_kfp_sa_token_path):
                token_path = _kfp_sa_token_path
            elif os.path.exists(_k8s_sa_token_path):
                token_path = _k8s_sa_token_path
                logger.debug(
                    'Kubeflow pipelines token path missing; using Kubernetes '
                    'service account token at %s for API auth.', token_path)

        if not token_path:
            logger.debug(
                'No in-cluster token file found; skipping credential setup.')
            return

        try:
            from kfp.client.set_volume_credentials import \
                ServiceAccountTokenVolumeCredentials
        except ImportError:
            logger.debug(
                'In-cluster credential module not available.', exc_info=True)
            return
        try:
            credentials = ServiceAccountTokenVolumeCredentials(path=token_path)
            credentials.refresh_api_key_hook(api_config)
            api_config.api_key_prefix['authorization'] = 'Bearer'
            api_config.refresh_api_key_hook = credentials.refresh_api_key_hook
        except Exception:
            logger.warning(
                'Could not set up in-cluster credentials. '
                'Proceeding without authentication.',
                exc_info=True)

    def _refresh_credentials(self) -> None:
        """Refresh the API token using the configured refresh hook."""
        if hasattr(self._api_config, 'refresh_api_key_hook') and callable(
                self._api_config.refresh_api_key_hook):
            self._api_config.refresh_api_key_hook(self._api_config)
        else:
            logger.warning('Token expired but no refresh hook is configured. '
                           'Re-create the client with a fresh token.')

    def _get_namespace(self) -> str:
        """Return the configured namespace, auto-detecting if needed.

        Resolution order:
            1. Explicitly configured via ``PipelinesBackendConfig.namespace``.
            2. In-cluster: ``/var/run/secrets/kubernetes.io/serviceaccount/namespace``.
            3. Out-of-cluster: namespace from the current kubeconfig context.
            4. Fallback: ``"kubeflow"``.

        Note: Does not read ~/.config/kfp/context.json (used by kfp.Client's
        set_user_namespace). This will be implemented in further phases.
        """
        if self._namespace:
            return self._namespace
        try:
            with open(_NAMESPACE_PATH, 'r') as f:
                self._namespace = f.read().strip()
                return self._namespace
        except FileNotFoundError:
            pass
        try:
            import kubernetes as k8s
            _, active_context = k8s.config.list_kube_config_contexts()
            namespace = active_context.get('context', {}).get('namespace')
            if namespace:
                self._namespace = namespace
                logger.debug('Namespace resolved from kubeconfig context: %r.',
                             namespace)
                return self._namespace
        except (k8s.config.ConfigException, FileNotFoundError):
            logger.debug(
                'Could not read namespace from kubeconfig.', exc_info=True)
        self._namespace = _DEFAULT_NAMESPACE
        logger.debug(
            'Namespace not resolved from cluster or kubeconfig; '
            'using default %r.', _DEFAULT_NAMESPACE)
        return self._namespace

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
            namespace=self._get_namespace(),
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
            namespace=self._get_namespace(),
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
    # Private helpers — pipeline compilation and upload
    # ------------------------------------------------------------------

    def _resolve_pipeline_to_file(
        self,
        pipeline: Callable | str,
    ) -> tuple[str, str | None]:
        """Resolve a pipeline source to a compiled YAML file path.

        If ``pipeline`` is callable, compile it to a temporary directory.
        If it's a string, treat it as a file path and validate existence.

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
        """Infer a pipeline name from the source.

        Resolution order: callable's .name attribute, callable's
        __name__, pipeline name from compiled YAML, filename stem, or
        'pipeline'.
        """
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
        except Exception:
            logger.debug(
                'Could not read pipeline name from %s.',
                package_path,
                exc_info=True)
        return None

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
            'namespace': self._get_namespace(),
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
            except (kfp_server_api.ApiException, AttributeError):
                logger.warning(
                    'Could not rename the first pipeline version to %r. '
                    'The upload API does not accept a version name for the '
                    'initial version; subsequent versions will use the '
                    'provided name.', version_name)

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
    ) -> str:
        """Resolve an experiment name to its ID, using default if None."""
        if experiment is None:
            default_id = self._get_experiment_id_by_name('Default')
            if default_id is None:
                default_experiment = self.create_experiment('Default')
                return default_experiment.experiment_id
            return default_id
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
        experiment_id: str,
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
        experiment_id: str,
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

    def _run_inline(
        self,
        pipeline_callable: Callable,
        params: dict[str, Any] | None,
        run_name: str,
        experiment_id: str,
    ) -> Run:
        """Compile a callable and submit inline (no upload)."""
        package_path, temp_dir = self._resolve_pipeline_to_file(
            pipeline_callable)
        try:
            return self._run_from_file(
                file_path=package_path,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )
        finally:
            if temp_dir is not None:
                shutil.rmtree(temp_dir, ignore_errors=True)

    def _run_from_file(
        self,
        file_path: str,
        params: dict[str, Any] | None,
        run_name: str,
        experiment_id: str,
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
            docs = list(yaml.safe_load_all(f))

        if not docs or not docs[0]:
            raise ValueError(
                f'Pipeline file is empty or contains no valid YAML '
                f'documents: {file_path}')
        pipeline_spec_dict = docs[0]
        platform_spec_dict = {}
        for doc in docs[1:]:
            if doc:
                platform_spec_dict.update(doc)

        pipeline_spec = json_format.ParseDict(
            pipeline_spec_dict,
            pipeline_spec_pb2.PipelineSpec(),
            ignore_unknown_fields=True)
        platform_spec = json_format.ParseDict(
            platform_spec_dict,
            pipeline_spec_pb2.PlatformSpec(),
            ignore_unknown_fields=True)

        # The API expects different shapes depending on whether a platform spec
        # is present: bare pipeline spec dict vs. wrapped {pipeline_spec, platform_spec}.
        if platform_spec == pipeline_spec_pb2.PlatformSpec():
            return json_format.MessageToDict(pipeline_spec)
        return {
            'pipeline_spec': json_format.MessageToDict(pipeline_spec),
            'platform_spec': json_format.MessageToDict(platform_spec),
        }

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
            if (PipelinesClient._is_yaml_path(pipeline) or
                    PipelinesClient._is_archive_path(pipeline)):
                basename = os.path.basename(pipeline)
                name = PipelinesClient._strip_pipeline_extension(basename)
                return f'{name} {timestamp}'
            return f'{pipeline} {timestamp}'
        if isinstance(pipeline, (Pipeline, PipelineVersion)):
            display_name = getattr(pipeline, 'display_name', 'pipeline')
            return f'{display_name} {timestamp}'
        return f'pipeline {timestamp}'

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
