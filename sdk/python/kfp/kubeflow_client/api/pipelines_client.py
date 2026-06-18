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
"""PipelinesClient — simplified, name-first client for Kubeflow Pipelines.

This module provides a streamlined interface over the KFP backend API,
designed for re-export by the Kubeflow SDK at ``kubeflow.pipelines``.
"""

from __future__ import annotations

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

from kfp import compiler
from kfp.kubeflow_client import constants
from kfp.kubeflow_client.backends.kubernetes import KubernetesBackend
from kfp.kubeflow_client.backends.kubernetes import KubernetesBackendConfig
from kfp.kubeflow_client.types import Experiment
from kfp.kubeflow_client.types import ListExperimentsResponse
from kfp.kubeflow_client.types import ListPipelinesResponse
from kfp.kubeflow_client.types import ListPipelineVersionsResponse
from kfp.kubeflow_client.types import ListRunsResponse
from kfp.kubeflow_client.types import Pipeline
from kfp.kubeflow_client.types import PipelineVersion
from kfp.kubeflow_client.types import Run
import kfp_server_api
import yaml

logger = logging.getLogger(__name__)

__all__ = ['PipelinesClient']

_VALID_UPLOAD_EXTENSIONS = ('.yaml', '.yml', '.tar.gz', '.tgz', '.zip')


class PipelinesClient:
    """Simplified, name-first client for Kubeflow Pipelines.

    Provides the core author → compile → upload → run → monitor workflow
    from a single import. Designed to be re-exported by the Kubeflow SDK
    at ``kubeflow.pipelines.PipelinesClient``.

    Args:
        backend_config: Connection parameters for the KFP API server.
            When ``None``, uses ``KubernetesBackendConfig()`` (zero-arg
            construction with auto-discovery).
    """

    def __init__(
        self,
        backend_config: KubernetesBackendConfig | None = None,
    ) -> None:
        if backend_config is None:
            backend_config = KubernetesBackendConfig()
        self._backend = KubernetesBackend(backend_config)
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

            existing_pipeline_id = self._backend._get_pipeline_id_by_name(name)

            if existing_pipeline_id is not None:
                return self._backend._upload_version(
                    package_path,
                    pipeline_id=existing_pipeline_id,
                    version_name=version,
                    description=description,
                )
            else:
                return self._backend._upload_new_pipeline(
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
        pipeline_id = self._backend._get_pipeline_id_by_name(name)
        if pipeline_id is None:
            raise ValueError(f'Pipeline not found: {name!r}. '
                             'Use list_pipelines() to see available pipelines.')
        return self._backend.pipelines_api.pipeline_service_get_pipeline(
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
        version_id = self._backend._get_version_id_by_name(
            pipeline.pipeline_id, version)
        if version_id is None:
            raise ValueError(f'Pipeline version not found: {version!r} '
                             f'for pipeline {name!r}.')
        return self._backend.pipelines_api.pipeline_service_get_pipeline_version(
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
        return self._backend.pipelines_api.pipeline_service_list_pipelines(
            namespace=self._backend.namespace,
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
        pipeline_id = self._backend._get_pipeline_id_by_name(name)
        if pipeline_id is None:
            raise ValueError(f'Pipeline not found: {name!r}. '
                             'Use list_pipelines() to see available pipelines.')
        return self._backend.pipelines_api.pipeline_service_list_pipeline_versions(
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
        pipeline_id = self._backend._get_pipeline_id_by_name(name)
        if pipeline_id is None:
            raise ValueError(f'Pipeline not found: {name!r}. '
                             'Use list_pipelines() to see available pipelines.')

        if version is not None:
            version_id = self._backend._get_version_id_by_name(
                pipeline_id, version)
            if version_id is None:
                raise ValueError(f'Pipeline version not found: {version!r} '
                                 f'for pipeline {name!r}.')
            self._backend.pipelines_api.pipeline_service_delete_pipeline_version(
                pipeline_id=pipeline_id,
                pipeline_version_id=version_id,
            )
            return

        if not force:
            versions_response = (
                self._backend.pipelines_api
                .pipeline_service_list_pipeline_versions(
                    pipeline_id=pipeline_id,
                    page_size=2,
                ))
            versions = versions_response.pipeline_versions or []
            if len(versions) > 1:
                raise ValueError(
                    f'Pipeline {name!r} has multiple versions. '
                    'Use force=True to delete the pipeline and all versions, '
                    'or specify version= to delete a single version.')

        self._backend.pipelines_api.pipeline_service_delete_pipeline(
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
            experiment: Experiment name. If ``None``, the server's default
                experiment is used. If provided and the experiment does not
                exist, raises ``ValueError``.
            version: Pipeline version name (used when ``pipeline`` is a
                name string or a ``Pipeline`` object). Uses latest version
                if omitted.

        Returns:
            A ``Run`` object.
        """
        experiment_id = self._backend._resolve_experiment_id(experiment)
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
            return self._backend._run_from_version_reference(
                pipeline_id=pipeline.pipeline_id,
                version_id=pipeline.pipeline_version_id,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )

        if isinstance(pipeline, Pipeline):
            if version:
                version_id = self._backend._get_version_id_by_name(
                    pipeline.pipeline_id, version)
                if version_id is None:
                    raise ValueError(f'Pipeline version not found: {version!r} '
                                     f'for pipeline {pipeline.display_name!r}.')
            else:
                version_id = self._backend._get_latest_version_id(
                    pipeline.pipeline_id)
            return self._backend._run_from_version_reference(
                pipeline_id=pipeline.pipeline_id,
                version_id=version_id,
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
            return self._backend._run_from_file(
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
            return self._backend._run_by_name(
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
        return self._backend.run_api.run_service_get_run(run_id=run_id)

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
            pipeline_id = self._backend._get_pipeline_id_by_name(pipeline)
            if pipeline_id is None:
                raise ValueError(f'Pipeline not found: {pipeline!r}.')

        experiment_id = None
        if experiment is not None:
            experiment_id = self._backend._get_experiment_id_by_name(experiment)
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

        response = self._backend.run_api.run_service_list_runs(
            namespace=self._backend.namespace,
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
        """Wait for a run to reach a target state.

        Args:
            run: A ``Run`` object or a run ID string.
            status: Set of states to wait for. Defaults to
                ``{constants.RUN_COMPLETE}`` (``"succeeded"``). The wait
                always exits immediately on any terminal state regardless
                of this parameter.
            timeout: Maximum seconds to wait. Defaults to 600 (10 minutes).
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
                run_response = self._backend.run_api.run_service_get_run(
                    run_id=run_id)
                first_poll_succeeded = True
                auth_retries = 0
            except kfp_server_api.ApiException as api_error:
                if (first_poll_succeeded and api_error.status == 401 and
                        auth_retries < max_auth_retries):
                    auth_retries += 1
                    logger.info(
                        'Access token expired, refreshing '
                        '(attempt %d/%d)...', auth_retries, max_auth_retries)
                    self._backend.refresh_credentials()
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
        existing_id = self._backend._get_experiment_id_by_name(name)
        if existing_id is not None:
            if description:
                logger.warning(
                    'Experiment %r already exists; provided description will '
                    'not be applied.', name)
            return self._backend.experiment_api.experiment_service_get_experiment(
                experiment_id=existing_id)

        experiment_body = kfp_server_api.V2beta1Experiment(
            display_name=name,
            description=description,
            namespace=self._backend.namespace,
        )
        return self._backend.experiment_api.experiment_service_create_experiment(
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
        experiment_id = self._backend._get_experiment_id_by_name(name)
        if experiment_id is None:
            raise ValueError(f'Experiment not found: {name!r}. '
                             'Use create_experiment() to create one.')
        return self._backend.experiment_api.experiment_service_get_experiment(
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
        return self._backend.experiment_api.experiment_service_list_experiments(
            namespace=self._backend.namespace,
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
        experiment_id = self._backend._get_experiment_id_by_name(name)
        if experiment_id is None:
            raise ValueError(f'Experiment not found: {name!r}.')
        self._backend.experiment_api.experiment_service_delete_experiment(
            experiment_id=experiment_id)

    # ------------------------------------------------------------------
    # Escape hatch
    # ------------------------------------------------------------------

    @property
    def kfp_client(self) -> Client:
        """Access the underlying ``kfp.Client`` for advanced operations.

        Lazily constructed on first access, sharing connection
        parameters from ``KubernetesBackendConfig``.
        """
        if self._kfp_client_instance is None:
            from kfp.client import Client
            config = self._backend.config
            kwargs: dict[str, Any] = {}
            if config.base_url:
                kwargs['host'] = config.base_url
            if config.user_token:
                kwargs['existing_token'] = config.user_token
            if config.namespace:
                kwargs['namespace'] = config.namespace
            if config.custom_ca:
                kwargs['ssl_ca_cert'] = config.custom_ca
            if config.is_secure is not None:
                kwargs['verify_ssl'] = config.is_secure
            self._kfp_client_instance = Client(**kwargs)
        return self._kfp_client_instance

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
        except (OSError, yaml.YAMLError):
            logger.debug(
                'Could not read pipeline name from %s.',
                package_path,
                exc_info=True)
        return None

    # ------------------------------------------------------------------
    # Private helpers — run creation
    # ------------------------------------------------------------------

    def _run_inline(
        self,
        pipeline_callable: Callable,
        params: dict[str, Any] | None,
        run_name: str,
        experiment_id: str | None,
    ) -> Run:
        """Compile a callable and submit inline (no upload)."""
        package_path, temp_dir = self._resolve_pipeline_to_file(
            pipeline_callable)
        try:
            return self._backend._run_from_file(
                file_path=package_path,
                params=params,
                run_name=run_name,
                experiment_id=experiment_id,
            )
        finally:
            if temp_dir is not None:
                shutil.rmtree(temp_dir, ignore_errors=True)

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
