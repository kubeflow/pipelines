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

from collections.abc import Callable
from typing import Any, TYPE_CHECKING

if TYPE_CHECKING:
    from kfp.client import Client

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

__all__ = ['PipelinesClient']


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
        return self._backend.upload_pipeline(
            pipeline, name=name, version_name=version, description=description)

    def get_pipeline(self, name: str) -> Pipeline:
        """Get a pipeline by name.

        Args:
            name: Pipeline display name.

        Returns:
            A ``Pipeline`` object.

        Raises:
            ValueError: If no pipeline matches or multiple pipelines match.
        """
        return self._backend.get_pipeline(name)

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
        return self._backend.get_pipeline_version(name, version)

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
        return self._backend.list_pipelines(
            page_token=page_token, page_size=page_size)

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
        return self._backend.list_pipeline_versions(
            name, page_token=page_token, page_size=page_size)

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
        self._backend.delete_pipeline(name, version=version, force=force)

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
        return self._backend.run(
            pipeline,
            params=params,
            name=name,
            experiment=experiment,
            version=version,
        )

    def get_run(self, run_id: str) -> Run:
        """Get a run by ID.

        Args:
            run_id: The run identifier.

        Returns:
            A ``Run`` object.
        """
        return self._backend.get_run(run_id)

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
        return self._backend.list_runs(
            pipeline=pipeline,
            experiment=experiment,
            status=status,
            page_token=page_token,
            page_size=page_size,
        )

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
        return self._backend.wait_for_run_status(
            run,
            status=status,
            timeout=timeout,
            polling_interval=polling_interval,
            callbacks=callbacks,
        )

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
        return self._backend.create_experiment(name, description=description)

    def get_experiment(self, name: str) -> Experiment:
        """Get an experiment by name.

        Args:
            name: Experiment display name.

        Returns:
            An ``Experiment`` object.

        Raises:
            ValueError: If no experiment with that name is found.
        """
        return self._backend.get_experiment(name)

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
        return self._backend.list_experiments(
            page_token=page_token, page_size=page_size)

    def delete_experiment(self, name: str) -> None:
        """Delete an experiment by name.

        Args:
            name: Experiment display name.

        Raises:
            ValueError: If no experiment with that name is found.
        """
        self._backend.delete_experiment(name)

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
