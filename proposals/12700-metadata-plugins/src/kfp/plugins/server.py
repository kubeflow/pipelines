from __future__ import annotations

import concurrent.futures
from typing import Any, Dict, List, Literal, Union

from fastapi import Depends, FastAPI, HTTPException, Request
from pydantic import BaseModel

# These imports are not part of the design. During implementation, the kfp sdk
# will need to ensure these models are available and properly exported.
from backend.api.v2beta1.python_http_client.kfp_server_api.models.v2beta1_run import Metadata
from kubernetes.client import V1PodSpec


def _verify_authorization(request: Request) -> None:
    """
    Verify the caller has RBAC permission: update on runs.pipelines.kubeflow.org.

    Extracts the Bearer token from the Authorization header and performs a
    SubjectAccessReview to validate the caller's permissions.
    """
    # TODO: Implement during implementation
    # 1. Extract token from Authorization header
    # 2. Create SubjectAccessReview with:
    #    - resource: "runs"
    #    - group: "pipelines.kubeflow.org"
    #    - verb: "update"
    # 3. If denied, raise HTTPException(status_code=403)
    raise NotImplementedError("Authorization to be implemented")

from .api import (
    ExecutorHookRequest,
    InputFieldsResponse,
    RunHookRequest,
    TaskHookRequest,
    ValidationResult,
)
from .loader import load_plugins


class PluginHookResult(BaseModel):
    """Result from a single plugin's hook execution."""
    plugin: str
    status: Literal["ok", "error", "skipped", "async"] = "ok"
    error: str | None = None
    metadata: Metadata | None = None
    pod_spec_patch: V1PodSpec | None = None
    pre_execution_code: str | None = None
    post_execution_code: str | None = None

    model_config = {"arbitrary_types_allowed": True}


class HookResponse(BaseModel):
    """Aggregated response from all plugins for a hook."""
    hook: str
    results: List[PluginHookResult]
    aggregated_metadata: Metadata | None = None
    merged_pod_spec_patch: V1PodSpec | None = None

    model_config = {"arbitrary_types_allowed": True}


class InputFieldsHookResponse(BaseModel):
    """Aggregated response from all plugins for input fields."""
    plugins: Dict[str, InputFieldsResponse]


class ValidateInputsRequest(BaseModel):
    """Request to validate inputs across all plugins."""
    inputs: Dict[str, Dict[str, Any]]  # plugin_name -> {field_id -> value}


class PluginValidationResult(BaseModel):
    """Validation result from a single plugin."""
    valid: bool = True
    errors: List[Dict[str, str]] = []  # List of {field_id, message}


class ValidationHookResponse(BaseModel):
    """Aggregated response from all plugins for validation."""
    valid: bool = True
    results: Dict[str, PluginValidationResult]


def _prefix_metadata_keys(metadata: Metadata | None, plugin_name: str) -> Metadata | None:
    """Prefix all metadata keys with the plugin name to prevent conflicts."""
    if metadata is None:
        return None
    prefixed = Metadata()
    for key, value in metadata.entries.items():
        prefixed.entries[f"{plugin_name}.{key}"] = value
    return prefixed


def _merge_metadata(existing: Metadata | None, new: Metadata | None) -> Metadata | None:
    """Merge metadata from multiple plugins. Keys should already be prefixed."""
    if existing is None:
        return new
    if new is None:
        return existing
    merged = Metadata()
    merged.entries.update(existing.entries)
    merged.entries.update(new.entries)
    return merged


def _merge_pod_specs(existing: V1PodSpec | None, new: V1PodSpec | None) -> V1PodSpec | None:
    """
    Merge pod specs from multiple plugins using strategic merge patch.

    Pod patches are applied sequentially in plugin order. Plugin authors must
    ensure their patches are additive (e.g., append to lists rather than replace).

    TODO: Implement strategic merge patch for V1PodSpec during implementation.
    Consider using kubernetes strategic merge patch library or manual field merging.
    """
    if existing is None:
        return new
    if new is None:
        return existing
    # Placeholder: actual implementation should use strategic merge patch
    raise NotImplementedError("Pod spec merging to be implemented")


# Type alias for any hook request
AnyHookRequest = Union[RunHookRequest, TaskHookRequest, ExecutorHookRequest]


def create_app(*, continue_on_error: bool = True) -> FastAPI:
    app = FastAPI(
        title="KFP Plugin Hook Server",
        dependencies=[Depends(_verify_authorization)],
    )

    plugins = load_plugins()
    executor = concurrent.futures.ThreadPoolExecutor(max_workers=10)

    @app.get("/v1/plugins")
    def list_plugins() -> Dict[str, Any]:
        return {
            "plugins": [
                {
                    "name": p.name,
                    "async": p.enable_asynchronous,
                }
                for p in plugins
            ]
        }

    def _dispatch(hook_name: str, req: AnyHookRequest) -> HookResponse:
        """
        Dispatch a hook to all registered plugins.

        For async plugins, the hook is fired in a background thread (fire-and-forget).
        For sync plugins, all are executed in parallel and we wait for all to complete.
        """
        results: list[PluginHookResult] = []
        aggregated_metadata: Metadata | None = None
        merged_pod_spec_patch: V1PodSpec | None = None

        # Collect sync plugin futures for parallel execution
        sync_futures: list[tuple[str, concurrent.futures.Future]] = []

        for plugin in plugins:
            hook = getattr(plugin, hook_name, None)
            if hook is None:
                results.append(PluginHookResult(plugin=plugin.name, status="skipped"))
                continue

            if plugin.enable_asynchronous:
                # Fire and forget
                executor.submit(hook, req)
                results.append(PluginHookResult(plugin=plugin.name, status="async"))
            else:
                # Submit for parallel execution
                future = executor.submit(hook, req)
                sync_futures.append((plugin.name, future))

        # Wait for all sync plugins to complete
        for plugin_name, future in sync_futures:
            try:
                response = future.result()
                result = PluginHookResult(plugin=plugin_name, status="ok")

                if response is not None:
                    if hasattr(response, "metadata") and response.metadata:
                        # Prefix metadata keys with plugin name to prevent conflicts
                        prefixed_metadata = _prefix_metadata_keys(response.metadata, plugin_name)
                        result.metadata = prefixed_metadata
                        aggregated_metadata = _merge_metadata(
                            aggregated_metadata, prefixed_metadata
                        )

                    if hasattr(response, "pod_spec_patch") and response.pod_spec_patch:
                        result.pod_spec_patch = response.pod_spec_patch
                        merged_pod_spec_patch = _merge_pod_specs(
                            merged_pod_spec_patch, response.pod_spec_patch
                        )

                    if hasattr(response, "pre_execution_code"):
                        result.pre_execution_code = response.pre_execution_code

                    if hasattr(response, "post_execution_code"):
                        result.post_execution_code = response.post_execution_code

                results.append(result)

            except Exception as e:
                results.append(
                    PluginHookResult(plugin=plugin_name, status="error", error=str(e))
                )
                if not continue_on_error:
                    raise

        return HookResponse(
            hook=hook_name,
            results=results,
            aggregated_metadata=aggregated_metadata,
            merged_pod_spec_patch=merged_pod_spec_patch,
        )

    @app.post("/v1/hooks/on_run_start")
    def on_run_start(req: RunHookRequest) -> HookResponse:
        try:
            return _dispatch("on_run_start", req)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Hook dispatch failed: {e}")

    @app.post("/v1/hooks/on_run_end")
    def on_run_end(req: RunHookRequest) -> HookResponse:
        try:
            return _dispatch("on_run_end", req)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Hook dispatch failed: {e}")

    @app.post("/v1/hooks/on_task_start")
    def on_task_start(req: TaskHookRequest) -> HookResponse:
        try:
            return _dispatch("on_task_start", req)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Hook dispatch failed: {e}")

    @app.post("/v1/hooks/on_task_end")
    def on_task_end(req: TaskHookRequest) -> HookResponse:
        try:
            return _dispatch("on_task_end", req)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Hook dispatch failed: {e}")

    @app.post("/v1/hooks/on_executor_start")
    def on_executor_start(req: ExecutorHookRequest) -> HookResponse:
        try:
            return _dispatch("on_executor_start", req)
        except Exception as e:
            raise HTTPException(status_code=500, detail=f"Hook dispatch failed: {e}")

    @app.get("/v1/hooks/input_fields")
    def get_input_fields() -> InputFieldsHookResponse:
        """Aggregate input field definitions from all plugins."""
        plugin_fields: Dict[str, InputFieldsResponse] = {}

        for plugin in plugins:
            try:
                response = plugin.get_input_fields()
                if response is not None:
                    plugin_fields[plugin.name] = response
            except Exception as e:
                # Log error but continue - a plugin failure shouldn't block UI
                print(f"Error getting input fields from {plugin.name}: {e}")

        return InputFieldsHookResponse(plugins=plugin_fields)

    @app.post("/v1/hooks/validate_inputs")
    def validate_inputs(req: ValidateInputsRequest) -> ValidationHookResponse:
        """
        Validate inputs across all plugins.

        Each plugin only receives its own inputs (scoped by plugin name).
        If ANY plugin fails validation, run creation should be blocked.
        """
        results: Dict[str, PluginValidationResult] = {}
        all_valid = True

        for plugin in plugins:
            plugin_inputs = req.inputs.get(plugin.name, {})

            try:
                validation = plugin.validate_inputs(plugin_inputs)

                plugin_result = PluginValidationResult(
                    valid=validation.valid,
                    errors=[
                        {"field_id": e.field_id, "message": e.message}
                        for e in validation.errors
                    ]
                )
                results[plugin.name] = plugin_result

                if not validation.valid:
                    all_valid = False

            except Exception as e:
                # Validation error counts as failure
                results[plugin.name] = PluginValidationResult(
                    valid=False,
                    errors=[{"field_id": "_plugin", "message": str(e)}]
                )
                all_valid = False
                if not continue_on_error:
                    raise HTTPException(
                        status_code=500,
                        detail=f"Validation failed for {plugin.name}: {e}"
                    )

        return ValidationHookResponse(valid=all_valid, results=results)

    return app
