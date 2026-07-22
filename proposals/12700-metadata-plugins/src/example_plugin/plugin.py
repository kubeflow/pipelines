from __future__ import annotations

from kubernetes.client import V1Container, V1EnvVar, V1PodSpec

from typing import Any, Dict

from ..kfp.plugins.api import (
    ExecutorHookRequest,
    ExecutorStartResponse,
    InputFieldDefinition,
    InputFieldsResponse,
    KfpLifecyclePlugin,
    RunHookRequest,
    RunHookResponse,
    TaskEndResponse,
    TaskHookRequest,
    TaskStartResponse,
    ValidationError,
    ValidationResult,
)


class ExamplePlugin(KfpLifecyclePlugin):
    """
    Example plugin demonstrating all available hooks.

    This plugin logs lifecycle events and returns example metadata.
    """

    @property
    def name(self) -> str:
        return "example-plugin"

    @property
    def enable_asynchronous(self) -> bool:
        # This plugin runs synchronously so we can return metadata
        return False

    def on_run_start(self, req: RunHookRequest) -> RunHookResponse:
        run = req.ctx.run
        print(f"[example-plugin] Run started: {run.run_id}")
        print(f"  Display name: {run.display_name}")
        print(f"  Experiment ID: {run.experiment_id}")

        # Access validated inputs from run creation form
        exp_name = req.plugin_inputs.get("experiment_name", "default")
        tracking_enabled = req.plugin_inputs.get("tracking_enabled", True)
        log_level = req.plugin_inputs.get("log_level", "INFO")

        print(f"  Plugin inputs: experiment={exp_name}, tracking={tracking_enabled}, log_level={log_level}")

        metadata = {
            "example.run_start_hook": {"value": "processed", "type": "text"},
            "example.experiment_name": {"value": exp_name, "type": "text"},
            "example.log_level": {"value": log_level, "type": "text"},
        }

        # Only add tracking URL if tracking is enabled
        if tracking_enabled:
            metadata["example.tracking_url"] = {
                "value": f"https://example.com/experiments/{exp_name}/runs/{run.run_id}",
                "type": "url",
            }

        return RunHookResponse(metadata=metadata)

    def on_run_end(self, req: RunHookRequest) -> RunHookResponse:
        run = req.ctx.run
        print(f"[example-plugin] Run ended: {run.run_id}")
        # Access run state from the V2beta1Run object
        state = run.state if run.state else "unknown"
        print(f"  State: {state}")

        return RunHookResponse(
            metadata={
                "example.run_end_hook": {"value": "processed", "type": "text"},
                "example.final_state": {"value": str(state), "type": "text"},
            }
        )

    def on_task_start(self, req: TaskHookRequest) -> TaskStartResponse:
        run = req.ctx.run
        task_id = req.task_ctx.task_id

        print(f"[example-plugin] Task started: {task_id}")
        print(f"  Run: {run.run_id}")

        return TaskStartResponse(
            metadata={
                "example.task_start_hook": {"value": "processed", "type": "text"},
            },
            # Example: inject an environment variable into the task pod
            pod_spec_patch=V1PodSpec(
                containers=[
                    V1Container(
                        name="main",
                        env=[V1EnvVar(name="EXAMPLE_PLUGIN_ENABLED", value="true")],
                    )
                ]
            ),
        )

    def on_task_end(self, req: TaskHookRequest) -> TaskEndResponse:
        task_id = req.task_ctx.task_id

        print(f"[example-plugin] Task ended: {task_id}")

        return TaskEndResponse(
            metadata={
                "example.task_end_hook": {"value": "processed", "type": "text"},
            }
        )

    def on_executor_start(self, req: ExecutorHookRequest) -> ExecutorStartResponse:
        task_id = req.task_ctx.task_id

        print(f"[example-plugin] Executor starting for task: {task_id}")

        return ExecutorStartResponse(
            # Could add additional env vars, volume mounts, etc.
            pod_spec_patch=V1PodSpec(
                containers=[
                    V1Container(
                        name="main",
                        env=[V1EnvVar(name="EXAMPLE_EXECUTOR_HOOK", value="active")],
                    )
                ]
            ),
            # Optional: wrap user code with setup/teardown
            pre_execution_code="print('[example-plugin] Pre-execution hook')",
            post_execution_code="print('[example-plugin] Post-execution hook')",
        )

    def get_input_fields(self) -> InputFieldsResponse:
        """
        Return custom input fields for the run creation UI.

        This example defines fields for tracking configuration.
        """
        return InputFieldsResponse(
            group_label="Example Plugin Settings",
            order=10,
            fields=[
                InputFieldDefinition(
                    field_id="tracking_enabled",
                    label="Enable Tracking",
                    field_type="checkbox",
                    default_value="true",
                    description="Enable experiment tracking for this run",
                    order=0,
                ),
                InputFieldDefinition(
                    field_id="experiment_name",
                    label="Experiment Name",
                    field_type="text",
                    required=True,
                    description="Name of the experiment to track this run under",
                    order=1,
                ),
                InputFieldDefinition(
                    field_id="log_level",
                    label="Log Level",
                    field_type="select",
                    options=["DEBUG", "INFO", "WARNING", "ERROR"],
                    default_value="INFO",
                    description="Logging verbosity level",
                    order=2,
                ),
            ]
        )

    def validate_inputs(self, inputs: Dict[str, Any]) -> ValidationResult:
        """
        Validate the plugin inputs before run creation.

        Returns errors if experiment_name is missing or too long.
        """
        errors = []

        # Validate experiment_name
        exp_name = inputs.get("experiment_name", "")
        if not exp_name:
            errors.append(ValidationError(
                field_id="experiment_name",
                message="Experiment name is required"
            ))
        elif len(exp_name) > 100:
            errors.append(ValidationError(
                field_id="experiment_name",
                message="Experiment name must be 100 characters or less"
            ))

        # Validate log_level if provided
        log_level = inputs.get("log_level", "INFO")
        valid_levels = ["DEBUG", "INFO", "WARNING", "ERROR"]
        if log_level not in valid_levels:
            errors.append(ValidationError(
                field_id="log_level",
                message=f"Log level must be one of: {', '.join(valid_levels)}"
            ))

        return ValidationResult(valid=len(errors) == 0, errors=errors)


def create_plugin() -> KfpLifecyclePlugin:
    """Entry point factory function for plugin discovery."""
    return ExamplePlugin()
