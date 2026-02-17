import copy
import logging
from typing import Any, Dict, List, Tuple

from kfp.local import config
from kfp.local import importer_handler
from kfp.local import io
from kfp.local import status
from kfp.local import task_dispatcher
from kfp.pipeline_spec import pipeline_spec_pb2

Outputs = Dict[str, Any]


class OrchestratorUtils:

    @classmethod
    def _apply_expression_selector(cls, value: Any, expression: str) -> Any:
        """Apply a parameter expression selector to extract a field from a
        value.

        Args:
            value: The value to extract from (typically a dict)
            expression: The expression selector (e.g., 'parseJson(string_value)["A_a"]')

        Returns:
            The extracted value
        """
        import json as json_module

        try:
            # Handle parseJson expressions
            if expression.startswith('parseJson('):
                # Extract the field path from the expression
                # Expression format: parseJson(string_value)["field_name"]
                if '["' in expression and '"]' in expression:
                    start_idx = expression.find('["') + 2
                    end_idx = expression.find('"]')
                    field_name = expression[start_idx:end_idx]

                    # If value is already a dict, extract the field directly
                    if isinstance(value, dict):
                        if field_name in value:
                            return value[field_name]
                        else:
                            logging.warning(
                                f'Field "{field_name}" not found in value: {value}'
                            )
                            return value
                    # If value is a JSON string, parse it first
                    elif isinstance(value, str):
                        parsed_value = json_module.loads(value)
                        if field_name in parsed_value:
                            return parsed_value[field_name]
                        else:
                            logging.warning(
                                f'Field "{field_name}" not found in parsed value: {parsed_value}'
                            )
                            return value
                    else:
                        logging.warning(
                            f'Cannot parse JSON from value type: {type(value)}')
                        return value
                else:
                    # No field extraction, just parse JSON if string
                    if isinstance(value, str):
                        return json_module.loads(value)
                    else:
                        return value

            # Handle direct field access like item["field"]
            elif '[' in expression and ']' in expression:
                start_idx = expression.find('["') + 2
                end_idx = expression.find('"]')
                if start_idx > 1 and end_idx > start_idx:
                    field_name = expression[start_idx:end_idx]
                    if isinstance(value, dict):
                        if field_name in value:
                            return value[field_name]
                        else:
                            logging.warning(
                                f'Field "{field_name}" not found in value: {value}'
                            )

            # Default: return the value as-is
            return value

        except Exception as e:
            logging.warning(
                f'Failed to apply expression selector "{expression}": {e}')
            return value

    @classmethod
    def execute_single_task(
        cls,
        task_name: str,
        task_spec: pipeline_spec_pb2.PipelineTaskSpec,
        pipeline_resource_name: str,
        components: Dict[str, pipeline_spec_pb2.ComponentSpec],
        executors: Dict[
            str, pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec],
        io_store: io.IOStore,
        pipeline_root: str,
        runner: config.LocalRunnerType,
        unique_pipeline_id: str,
        fail_stack: List[str],
    ) -> Tuple[Outputs, status.Status]:
        """Execute a single task (used by parallel executor)."""
        component_name = task_spec.component_ref.name
        component_spec = components[component_name]
        implementation = component_spec.WhichOneof('implementation')

        if implementation == 'executor_label':
            executor_spec = executors[component_spec.executor_label]
            task_arguments = cls.make_task_arguments(
                task_inputs_spec=task_spec.inputs,
                io_store=io_store,
            )

            if executor_spec.WhichOneof('spec') == 'importer':
                return importer_handler.run_importer(
                    pipeline_resource_name=pipeline_resource_name,
                    component_name=component_name,
                    component_spec=component_spec,
                    executor_spec=executor_spec,
                    arguments=task_arguments,
                    pipeline_root=pipeline_root,
                    unique_pipeline_id=unique_pipeline_id,
                )
            elif executor_spec.WhichOneof('spec') == 'container':
                return task_dispatcher.run_single_task_implementation(
                    pipeline_resource_name=pipeline_resource_name,
                    component_name=component_name,
                    component_spec=component_spec,
                    executor_spec=executor_spec,
                    arguments=task_arguments,
                    pipeline_root=pipeline_root,
                    runner=runner,
                    raise_on_error=False,
                    block_input_artifact=False,
                    unique_pipeline_id=unique_pipeline_id,
                )
            else:
                raise ValueError(
                    "Got unknown spec in ExecutorSpec. Only 'dsl.component', 'dsl.container_component', and 'dsl.importer' are supported."
                )
        else:
            raise ValueError(
                f'Got unknown component implementation: {implementation}')

    @classmethod
    def join_user_inputs_and_defaults(
        cls,
        dag_arguments: Dict[str, Any],
        dag_inputs_spec: pipeline_spec_pb2.ComponentInputsSpec,
    ) -> Dict[str, Any]:
        """Collects user-provided arguments and default arguments (when no
        user- provided argument) into a dictionary. Returns the dictionary.

        Args:
            dag_arguments: The user-provided arguments to the DAG.
            dag_inputs_spec: The ComponentInputSpec for the DAG.

        Returns:
            The complete DAG inputs, with defaults included where the user-provided argument is missing.
        """
        from kfp.local import executor_output_utils

        copied_dag_arguments = copy.deepcopy(dag_arguments)

        for input_name, input_spec in dag_inputs_spec.parameters.items():
            if input_name not in copied_dag_arguments:
                copied_dag_arguments[
                    input_name] = executor_output_utils.pb2_value_to_python(
                        input_spec.default_value)
        return copied_dag_arguments

    @classmethod
    def make_task_arguments(
        cls,
        task_inputs_spec: pipeline_spec_pb2.TaskInputsSpec,
        io_store: io.IOStore,
    ) -> Dict[str, Any]:
        """Obtains a dictionary of arguments required to execute the task
        corresponding to TaskInputsSpec.

        Args:
            task_inputs_spec: The TaskInputsSpec for the task.
            io_store: The IOStore of the current DAG. Used to obtain task arguments which come from upstream task outputs and parent component inputs.

        Returns:
            The arguments for the task.
        """
        from kfp.local import executor_output_utils

        task_arguments = {}
        # handle parameters
        for input_name, input_spec in task_inputs_spec.parameters.items():

            # handle constants
            if input_spec.HasField('runtime_value'):
                # runtime_value's value should always be constant for the v2 compiler
                if input_spec.runtime_value.WhichOneof('value') != 'constant':
                    raise ValueError('Expected constant.')
                task_arguments[
                    input_name] = executor_output_utils.pb2_value_to_python(
                        input_spec.runtime_value.constant)

            # handle upstream outputs
            elif input_spec.HasField('task_output_parameter'):
                task_arguments[input_name] = io_store.get_task_output(
                    input_spec.task_output_parameter.producer_task,
                    input_spec.task_output_parameter.output_parameter_key,
                )

            # handle parent pipeline input parameters
            elif input_spec.HasField('component_input_parameter'):
                parent_value = io_store.get_parent_input(
                    input_spec.component_input_parameter)

                # Check if there's a parameter expression selector to apply
                # (used in ParallelFor to extract specific fields from structured loop items)
                expression_selector = input_spec.parameter_expression_selector
                if expression_selector:
                    logging.info(
                        f"Applying expression selector '{expression_selector}' to parent value: {parent_value}"
                    )
                    parent_value = cls._apply_expression_selector(
                        parent_value, expression_selector)
                    logging.info(f"After expression selector: {parent_value}")

                task_arguments[input_name] = parent_value

            # TODO: support dsl.ExitHandler
            elif input_spec.HasField('task_final_status'):
                raise NotImplementedError(
                    "'dsl.ExitHandler' is not yet support for local execution.")

            else:
                raise ValueError(f'Missing input for parameter {input_name}.')

        # handle artifacts
        for input_name, input_spec in task_inputs_spec.artifacts.items():
            if input_spec.HasField('task_output_artifact'):
                task_arguments[input_name] = io_store.get_task_output(
                    input_spec.task_output_artifact.producer_task,
                    input_spec.task_output_artifact.output_artifact_key,
                )
            elif input_spec.HasField('component_input_artifact'):
                task_arguments[input_name] = io_store.get_parent_input(
                    input_spec.component_input_artifact)
            else:
                raise ValueError(f'Missing input for artifact {input_name}.')

        return task_arguments

    @classmethod
    def get_dag_output_parameters(
        cls,
        dag_outputs_spec: pipeline_spec_pb2.DagOutputsSpec,
        io_store: io.IOStore,
    ) -> Dict[str, Any]:
        """Gets the DAG output parameter values from a DagOutputsSpec and the
        DAG's IOStore.

        Args:
            dag_outputs_spec: DagOutputsSpec corresponding to the DAG.
            io_store: IOStore corresponding to the DAG.

        Returns:
            The DAG output parameters.
        """
        outputs = {}
        for root_output_key, parameter_selector_spec in dag_outputs_spec.parameters.items(
        ):
            kind = parameter_selector_spec.WhichOneof('kind')
            if kind == 'value_from_parameter':
                value_from_parameter = parameter_selector_spec.value_from_parameter
                outputs[root_output_key] = io_store.get_task_output(
                    value_from_parameter.producer_subtask,
                    value_from_parameter.output_parameter_key,
                )
            elif kind == 'value_from_oneof':
                raise NotImplementedError(
                    "'dsl.OneOf' is not yet supported in local execution.")
            else:
                raise ValueError(
                    f"Got unknown 'parameter_selector_spec' kind: {kind}")
        return outputs

    @classmethod
    def get_dag_output_artifacts(
        cls,
        dag_outputs_spec: pipeline_spec_pb2.DagOutputsSpec,
        io_store: io.IOStore,
    ) -> Dict[str, Any]:
        """Gets the DAG output artifact values from a DagOutputsSpec and the
        DAG's IOStore.

        Args:
            dag_outputs_spec: DagOutputsSpec corresponding to the DAG.
            io_store: IOStore corresponding to the DAG.

        Returns:
            The DAG output artifacts.
        """
        outputs = {}
        for root_output_key, artifact_selector_spec in dag_outputs_spec.artifacts.items(
        ):
            len_artifact_selectors = len(
                artifact_selector_spec.artifact_selectors)
            if len_artifact_selectors != 1:
                raise ValueError(
                    f'Expected 1 artifact in ArtifactSelectorSpec. Got: {len_artifact_selectors}'
                )
            artifact_selector = artifact_selector_spec.artifact_selectors[0]
            outputs[root_output_key] = io_store.get_task_output(
                artifact_selector.producer_subtask,
                artifact_selector.output_artifact_key,
            )
        return outputs

    @classmethod
    def get_dag_outputs(
        cls,
        dag_outputs_spec: pipeline_spec_pb2.DagOutputsSpec,
        io_store: io.IOStore,
    ) -> Dict[str, Any]:
        """Gets the DAG output values from a DagOutputsSpec and the DAG's
        IOStore.

        Args:
            dag_outputs_spec: DagOutputsSpec corresponding to the DAG.
            io_store: IOStore corresponding to the DAG.

        Returns:
            The DAG outputs.
        """
        output_params = cls.get_dag_output_parameters(
            dag_outputs_spec=dag_outputs_spec,
            io_store=io_store,
        )
        output_artifacts = cls.get_dag_output_artifacts(
            dag_outputs_spec=dag_outputs_spec,
            io_store=io_store,
        )
        return {**output_params, **output_artifacts}
