from __future__ import annotations

from abc import ABC
from typing import Any, Dict, List, Literal

from pydantic import BaseModel

# These imports are not part of the design. During implementation, the kfp sdk
# will need to ensure these models are available and properly exported.
from backend.api.v2beta1.python_http_client.kfp_server_api.models.v2beta1_run import V2beta1Run, Metadata
from kubernetes.client import V1PodSpec


class RunContext(BaseModel):
    """Context about the pipeline run."""
    run: V2beta1Run

    model_config = {"frozen": True, "arbitrary_types_allowed": True}


class TaskContext(BaseModel):
    """Context about a specific task within a run."""
    task_id: str

    model_config = {"frozen": True}


class RunHookRequest(BaseModel):
    """
    Request for run-level hooks (on_run_start, on_run_end).
    Keep this stable and versioned if you expect external plugins.
    """
    ctx: RunContext
    plugin_inputs: Dict[str, Any] = {}  # This plugin's validated inputs

    model_config = {"frozen": True, "arbitrary_types_allowed": True}


class TaskHookRequest(BaseModel):
    """
    Request for task-level hooks (on_task_start, on_task_end).
    Keep this stable and versioned if you expect external plugins.
    """
    ctx: RunContext
    task_ctx: TaskContext

    model_config = {"frozen": True, "arbitrary_types_allowed": True}


class ExecutorHookRequest(BaseModel):
    """
    Request for executor-level hooks (on_executor_start).
    Keep this stable and versioned if you expect external plugins.
    """
    ctx: RunContext
    task_ctx: TaskContext

    model_config = {"frozen": True, "arbitrary_types_allowed": True}


class RunHookResponse(BaseModel):
    """Response from run-level hooks (on_run_start, on_run_end)."""
    metadata: Metadata | None = None

    model_config = {"arbitrary_types_allowed": True}


class TaskStartResponse(BaseModel):
    """Response from on_task_start hook."""
    metadata: Metadata | None = None
    pod_spec_patch: V1PodSpec | None = None

    model_config = {"arbitrary_types_allowed": True}


class TaskEndResponse(BaseModel):
    """Response from on_task_end hook."""
    metadata: Metadata | None = None

    model_config = {"arbitrary_types_allowed": True}


class ExecutorStartResponse(BaseModel):
    """
    Response from on_executor_start hook.

    This hook is responsible for setting up the runtime environment.
    The pod_spec_patch will be strategically merged into the executor pod spec.
    """
    pod_spec_patch: V1PodSpec | None = None
    pre_execution_code: str | None = None
    post_execution_code: str | None = None

    model_config = {"arbitrary_types_allowed": True}


# Input field types for run creation UI
class InputFieldDefinition(BaseModel):
    """Definition of a single input field for the run creation UI."""
    field_id: str                    # Unique within plugin, e.g., "experiment_name"
    label: str                       # UI display label
    field_type: Literal["text", "number", ""] # List all KFP supported UI Types
    required: bool = False
    default_value: str | None = None
    options: List[str] | None = None  # For select type
    description: str | None = None
    order: int = 0                   # Lower = appears first

    model_config = {"frozen": True}


class InputFieldsResponse(BaseModel):
    """Response from get_input_fields hook."""
    fields: List[InputFieldDefinition] = []
    group_label: str | None = None   # Display name for grouping in UI
    order: int = 0                   # Plugin ordering (lower = appears first)


class ValidationError(BaseModel):
    """A single validation error for an input field."""
    field_id: str
    message: str


class ValidationResult(BaseModel):
    """Result of input validation."""
    valid: bool = True
    errors: List[ValidationError] = []


class KfpLifecyclePlugin(ABC):
    """
    Base class plugins must inherit from.
    Hooks are optional by default (no-op).
    """

    @property
    def name(self) -> str:
        """Unique-ish plugin name. Override to customize."""
        return self.__class__.__name__

    @property
    def enable_asynchronous(self) -> bool:
        """
        Whether this plugin supports asynchronous execution.

        When True, the plugin server will not block requests waiting for
        this plugin's hooks to complete - it fires and forgets.
        When False (default), the server waits for hook completion.
        """
        return False

    def on_run_start(self, req: RunHookRequest) -> RunHookResponse | None:
        """
        Called when a pipeline run starts.

        Args:
            req: Request containing run context and payload.

        Returns:
            Optional response with metadata to attach to the run.
        """
        return None

    def on_run_end(self, req: RunHookRequest) -> RunHookResponse | None:
        """
        Called when a pipeline run completes (success or failure).

        Args:
            req: Request containing run context and payload.

        Returns:
            Optional response with metadata to attach to the run.
        """
        return None

    def on_task_start(self, req: TaskHookRequest) -> TaskStartResponse | None:
        """
        Called when a task within a run starts.

        Args:
            req: Request containing run context, task context, and payload.

        Returns:
            Optional response with metadata and/or pod spec patches.
            The pod_spec_patch will be strategically merged into the task pod.
        """
        return None

    def on_task_end(self, req: TaskHookRequest) -> TaskEndResponse | None:
        """
        Called when a task within a run completes.

        Args:
            req: Request containing run context, task context, and payload.

        Returns:
            Optional response with metadata to attach to the task.
        """
        return None

    def on_executor_start(self, req: ExecutorHookRequest) -> ExecutorStartResponse | None:
        """
        Called when an executor (the actual user code container) starts.

        This hook is responsible for setting up and tearing down the
        runtime environment. Use this to inject environment variables,
        mount secrets, or wrap user code with pre/post execution logic.

        Args:
            req: Request containing run context, task context, and payload.

        Returns:
            Optional response with pod spec patches and/or code injection.
        """
        return None

    def get_input_fields(self) -> InputFieldsResponse | None:
        """
        Return custom input field definitions for run creation UI.

        Called when frontend loads the run creation page. Override this
        method to define custom input fields that users can fill out
        before creating a run.

        Returns:
            Optional InputFieldsResponse with field definitions and grouping.
        """
        return None

    def validate_inputs(self, inputs: Dict[str, Any]) -> ValidationResult:
        """
        Validate plugin inputs before run creation.

        Called when user submits run creation. Each plugin receives only
        the inputs for fields it defined. If any plugin returns
        valid=False, run creation is blocked.

        Args:
            inputs: Values for fields defined by this plugin (keyed by field_id).

        Returns:
            ValidationResult - if valid=False, run creation is blocked
            and errors are displayed to the user.
        """
        return ValidationResult(valid=True)
