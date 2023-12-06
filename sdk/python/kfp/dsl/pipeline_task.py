# Copyright 2021-2022 The Kubeflow Authors
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
"""Pipeline task class and operations."""

import copy
import itertools
from typing import Any, Dict, List, Mapping, Optional, Union
import warnings

from kfp.dsl import final_task_state
from kfp.dsl import future_task_state
from kfp.dsl import pipeline_channel
from kfp.dsl import pipeline_task_state_abc
from kfp.dsl import placeholders
from kfp.dsl import structures
from kfp.dsl import utils
from kfp.dsl.types import type_utils
from kfp.local import task_dispatcher
from kfp.pipeline_spec import pipeline_spec_pb2

TEMPORARILY_BLOCK_LOCAL_EXECUTION = True

_register_task_handler = lambda task: utils.maybe_rename_for_k8s(
    task.component_spec.name)


class PipelineTask:
    """Represents a pipeline task (instantiated component).

    **Note:** ``PipelineTask`` should not be constructed by pipeline authors directly, but instead obtained via an instantiated component (see example).

    Replaces ``ContainerOp`` from ``kfp`` v1. Holds operations available on a task object, such as
    ``.after()``, ``.set_memory_limit()``, ``.enable_caching()``, etc.

    Args:
        component_spec: The component definition.
        args: The dictionary of arguments on which the component was called to instantiate this task.

    Example:
      ::

        @dsl.component
        def identity(message: str) -> str:
            return message

        @dsl.pipeline(name='my_pipeline')
        def my_pipeline():
            # task is an instance of PipelineTask
            task = identity(message='my string')
    """
    _register_task_handler = _register_task_handler

    # Fallback behavior for compiling a component. This should be overriden by
    # pipeline `register_task_and_generate_id` if compiling a pipeline (more
    # than one component).

    def __init__(
        self,
        component_spec: structures.ComponentSpec,
        args: Dict[str, Any],
    ):
        """Initilizes a PipelineTask instance."""

        from kfp.dsl import pipeline_context
        from kfp.dsl import pipeline_task_state_abc
        from kfp.dsl.tasks_group import TasksGroup

        self.state: pipeline_task_state_abc.PipelineTaskState = future_task_state.FutureTaskState(
            self)

        self.parent_task_group: Union[None, TasksGroup] = None
        args = args or {}

        for input_name, argument_value in args.items():

            if input_name not in component_spec.inputs:
                raise ValueError(
                    f'Component {component_spec.name!r} got an unexpected input:'
                    f' {input_name!r}.')

            input_spec = component_spec.inputs[input_name]

            type_utils.verify_type_compatibility(
                given_value=argument_value,
                expected_spec=input_spec,
                error_message_prefix=(
                    f'Incompatible argument passed to the input '
                    f'{input_name!r} of component {component_spec.name!r}: '),
            )

        self.component_spec = component_spec

        self._task_spec = structures.TaskSpec(
            name=self._register_task_handler(),
            inputs=dict(args.items()),
            dependent_tasks=[],
            component_ref=component_spec.name,
            enable_caching=True)
        self._run_after: List[str] = []

        self.importer_spec = None
        self.container_spec = None
        self.pipeline_spec = None
        self._ignore_upstream_failure_tag = False
        # platform_config for this primitive task; empty if task is for a graph component
        self.platform_config = {}

        def validate_placeholder_types(
                component_spec: structures.ComponentSpec) -> None:
            inputs_dict = component_spec.inputs or {}
            outputs_dict = component_spec.outputs or {}
            for arg in itertools.chain(
                (component_spec.implementation.container.command or []),
                (component_spec.implementation.container.args or [])):
                check_primitive_placeholder_is_used_for_correct_io_type(
                    inputs_dict, outputs_dict, arg)

        if component_spec.implementation.container is not None:
            validate_placeholder_types(component_spec)
            self.container_spec = self._extract_container_spec_and_convert_placeholders(
                component_spec=component_spec)
        elif component_spec.implementation.importer is not None:
            self.importer_spec = component_spec.implementation.importer
            self.importer_spec.artifact_uri = args['uri']
        else:
            self.pipeline_spec = self.component_spec.implementation.graph

        self._outputs = {
            output_name: pipeline_channel.create_pipeline_channel(
                name=output_name,
                channel_type=output_spec.type,
                task_name=self._task_spec.name,
                is_artifact_list=output_spec.is_artifact_list,
            ) for output_name, output_spec in (
                component_spec.outputs or {}).items()
        }

        self._inputs = args

        self.channel_inputs = [
            value for _, value in args.items()
            if isinstance(value, pipeline_channel.PipelineChannel)
        ] + pipeline_channel.extract_pipeline_channels_from_any([
            value for _, value in args.items()
            if not isinstance(value, pipeline_channel.PipelineChannel)
        ])

        if not TEMPORARILY_BLOCK_LOCAL_EXECUTION and pipeline_context.Pipeline.get_default_pipeline(
        ) is None:
            self._outputs = task_dispatcher.run_single_component(
                pipeline_spec=self.pipeline_spec,
                arguments=args,
            )
            self.state: pipeline_task_state_abc.PipelineTaskState = final_task_state.FinalTaskState(
                self)

    def _extract_container_spec_and_convert_placeholders(
        self, component_spec: structures.ComponentSpec
    ) -> structures.ContainerSpecImplementation:
        """Extracts a ContainerSpec from a ComponentSpec and converts
        placeholder objects to strings.

        Args:
            component_spec: The component definition.
        """
        container_spec = copy.deepcopy(component_spec.implementation.container)
        if container_spec is None:
            raise ValueError(
                '_extract_container_spec_and_convert_placeholders used incorrectly. ComponentSpec.implementation.container is None.'
            )
        container_spec.command = [
            placeholders.convert_command_line_element_to_string(e)
            for e in container_spec.command or []
        ]
        container_spec.args = [
            placeholders.convert_command_line_element_to_string(e)
            for e in container_spec.args or []
        ]
        return container_spec

    @property
    def platform_spec(self) -> pipeline_spec_pb2.PlatformSpec:
        """PlatformSpec for all tasks in the pipeline as task.

        Only for use on tasks created from GraphComponents.
        """
        return self.state.platform_spec

    @property
    def name(self) -> str:
        """The name of the task.

        Unique within its parent group.
        """
        return self.state.name

    @property
    def inputs(
        self
    ) -> Dict[str, Union[type_utils.PARAMETER_TYPES,
                         pipeline_channel.PipelineChannel]]:
        """The inputs passed to the task."""
        return self.state.inputs

    @property
    def output(self) -> pipeline_channel.PipelineChannel:
        """The single output of the task.

        Used when a task has exactly one output parameter.
        """
        return self.state.output

    @property
    def outputs(self) -> Mapping[str, pipeline_channel.PipelineChannel]:
        """The dictionary of outputs of the task.

        Used when a task has more the one output or uses an
        ``OutputPath`` or ``Output[Artifact]`` type annotation.
        """
        return self.state.outputs

    @property
    def dependent_tasks(self) -> List[str]:
        """A list of the dependent task names."""
        return self.state.dependent_tasks

    def set_caching_options(self, enable_caching: bool) -> 'PipelineTask':
        """Sets caching options for the task.

        Args:
            enable_caching: Whether to enable caching.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_caching_options(enable_caching=enable_caching)
        return self

    def set_cpu_request(self, cpu: str) -> 'PipelineTask':
        """Sets CPU request (minimum) for the task.

        Args:
            cpu: Minimum CPU requests required. This string should be a number
                or a number followed by an "m" to indicate millicores (1/1000).
                For more information, see `Specify a CPU Request and a CPU Limit
                <https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#specify-a-cpu-request-and-a-cpu-limit>`_.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_cpu_request(cpu=cpu)
        return self

    def set_cpu_limit(self, cpu: str) -> 'PipelineTask':
        """Sets CPU limit (maximum) for the task.

        Args:
            cpu: Maximum CPU requests allowed. This string should be a number
                or a number followed by an "m" to indicate millicores (1/1000).
                For more information, see `Specify a CPU Request and a CPU Limit
                <https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#specify-a-cpu-request-and-a-cpu-limit>`_.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_cpu_limit(cpu=cpu)
        return self

    def set_accelerator_limit(self, limit: int) -> 'PipelineTask':
        """Sets accelerator limit (maximum) for the task. Only applies if
        accelerator type is also set via .set_accelerator_type().

        Args:
            limit: Maximum number of accelerators allowed.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_accelerator_limit(limit=limit)
        return self

    def set_gpu_limit(self, gpu: str) -> 'PipelineTask':
        """Sets GPU limit (maximum) for the task. Only applies if accelerator
        type is also set via .add_accelerator_type().

        Args:
            gpu: The maximum GPU reuqests allowed. This string should be a positive integer number of GPUs.

        Returns:
            Self return to allow chained setting calls.

        :meta private:
        """
        warnings.warn(
            f'{self.set_gpu_limit.__name__!r} is deprecated. Please use {self.set_accelerator_limit.__name__!r} instead.',
            category=DeprecationWarning)
        self.state.set_accelerator_limit(limit=gpu)
        return self

    def set_memory_request(self, memory: str) -> 'PipelineTask':
        """Sets memory request (minimum) for the task.

        Args:
            memory: The minimum memory requests required. This string should be
                a number or a number followed by one of "E", "Ei", "P", "Pi",
                "T", "Ti", "G", "Gi", "M", "Mi", "K", or "Ki".

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_memory_request(memory=memory)
        return self

    def set_memory_limit(self, memory: str) -> 'PipelineTask':
        """Sets memory limit (maximum) for the task.

        Args:
            memory: The maximum memory requests allowed. This string should be
                a number or a number followed by one of "E", "Ei", "P", "Pi",
                "T", "Ti", "G", "Gi", "M", "Mi", "K", or "Ki".

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_memory_limit(memory=memory)
        return self

    def set_retry(self,
                  num_retries: int,
                  backoff_duration: Optional[str] = None,
                  backoff_factor: Optional[float] = None,
                  backoff_max_duration: Optional[str] = None) -> 'PipelineTask':
        """Sets task retry parameters.

        Args:
            num_retries : Number of times to retry on failure.
            backoff_duration: Number of seconds to wait before triggering a retry. Defaults to ``'0s'`` (immediate retry).
            backoff_factor: Exponential backoff factor applied to ``backoff_duration``. For example, if ``backoff_duration="60"`` (60 seconds) and ``backoff_factor=2``, the first retry will happen after 60 seconds, then again after 120, 240, and so on. Defaults to ``2.0``.
            backoff_max_duration: Maximum duration during which the task will be retried. Maximum duration is 1 hour (3600s). Defaults to ``'3600s'``.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_retry(
            backoff_duration=backoff_duration,
            backoff_factor=backoff_factor,
            backoff_max_duration=backoff_max_duration,
        )

    def add_node_selector_constraint(self, accelerator: str) -> 'PipelineTask':
        """Sets accelerator type to use when executing this task.

        Args:
            accelerator: The name of the accelerator, such as ``'NVIDIA_TESLA_K80'``, ``'TPU_V3'``, ``'nvidia.com/gpu'`` or ``'cloud-tpus.google.com/v3'``.

        Returns:
            Self return to allow chained setting calls.
        """
        warnings.warn(
            f'{self.add_node_selector_constraint.__name__!r} is deprecated. Please use {self.set_accelerator_type.__name__!r} instead.',
            category=DeprecationWarning)
        self.state.set_accelerator_type(accelerator=accelerator)
        return self

    def set_accelerator_type(self, accelerator: str) -> 'PipelineTask':
        """Sets accelerator type to use when executing this task.

        Args:
            accelerator: The name of the accelerator, such as ``'NVIDIA_TESLA_K80'``, ``'TPU_V3'``, ``'nvidia.com/gpu'`` or ``'cloud-tpus.google.com/v3'``.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_accelerator_type(accelerator=accelerator)
        return self

    def set_display_name(self, name: str) -> 'PipelineTask':
        """Sets display name for the task.

        Args:
            name: Display name.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_display_name(name=name)
        return self

    def set_env_variable(self, name: str, value: str) -> 'PipelineTask':
        """Sets environment variable for the task.

        Args:
            name: Environment variable name.
            value: Environment variable value.

        Returns:
            Self return to allow chained setting calls.
        """
        self.state.set_env_variable(name=name, value=value)

    def after(self, *tasks) -> 'PipelineTask':
        """Specifies an explicit dependency on other tasks by requiring this
        task be executed after other tasks finish completion.

        Args:
            *tasks: Tasks after which this task should be executed.

        Returns:
            Self return to allow chained setting calls.

        Example:
          ::

            @dsl.pipeline(name='my-pipeline')
            def my_pipeline():
                task1 = my_component(text='1st task')
                task2 = my_component(text='2nd task').after(task1)
        """
        self.state.after(*tasks)
        return self

    def ignore_upstream_failure(self) -> 'PipelineTask':
        """If called, the pipeline task will run when any specified upstream
        tasks complete, even if unsuccessful.

        This method effectively turns the caller task into an exit task
        if the caller task has upstream dependencies.

        If the task has no upstream tasks, either via data exchange or an explicit dependency via .after(), this method has no effect.

        Returns:
            Self return to allow chained setting calls.

        Example:
          ::

            @dsl.pipeline()
            def my_pipeline(text: str = 'message'):
                task = fail_op(message=text)
                clean_up_task = print_op(
                    message=task.output).ignore_upstream_failure()
        """
        self.state.ignore_upstream_failure()


# TODO: this function should ideally be in the function kfp.dsl.structures.check_placeholder_references_valid_io_name, which does something similar, but this causes the exception to be raised at component definition time, rather than compile time. This would break tests that load v1 component YAML, even though that YAML is invalid.
def check_primitive_placeholder_is_used_for_correct_io_type(
    inputs_dict: Dict[str, structures.InputSpec],
    outputs_dict: Dict[str, structures.OutputSpec],
    arg: Union[placeholders.CommandLineElement, Any],
):
    """Validates input/output placeholders refer to an input/output with an
    appropriate type for the placeholder. This should only apply to components
    loaded from v1 component YAML, where the YAML is authored directly. For v2
    YAML, this is encapsulated in the DSL logic which does not permit writing
    incorrect placeholders.

    Args:
        inputs_dict: The existing input names.
        outputs_dict: The existing output names.
        arg: The command line element, which may be a placeholder.
    """

    if isinstance(arg, placeholders.InputValuePlaceholder):
        input_name = arg.input_name
        if not type_utils.is_parameter_type(inputs_dict[input_name].type):
            raise TypeError(
                f'Input "{input_name}" with type '
                f'"{inputs_dict[input_name].type}" cannot be paired with '
                'InputValuePlaceholder.')

    elif isinstance(
            arg,
        (placeholders.InputUriPlaceholder, placeholders.InputPathPlaceholder)):
        input_name = arg.input_name
        if type_utils.is_parameter_type(inputs_dict[input_name].type):
            raise TypeError(
                f'Input "{input_name}" with type '
                f'"{inputs_dict[input_name].type}" cannot be paired with '
                f'{arg.__class__.__name__}.')

    elif isinstance(arg, placeholders.OutputUriPlaceholder):
        output_name = arg.output_name
        if type_utils.is_parameter_type(outputs_dict[output_name].type):
            raise TypeError(
                f'Output "{output_name}" with type '
                f'"{outputs_dict[output_name].type}" cannot be paired with '
                f'{arg.__class__.__name__}.')
    elif isinstance(arg, placeholders.IfPresentPlaceholder):
        all_normalized_args: List[placeholders.CommandLineElement] = []
        if arg.then is None:
            pass
        elif isinstance(arg.then, list):
            all_normalized_args.extend(arg.then)
        else:
            all_normalized_args.append(arg.then)

        if arg.else_ is None:
            pass
        elif isinstance(arg.else_, list):
            all_normalized_args.extend(arg.else_)
        else:
            all_normalized_args.append(arg.else_)

        for arg in all_normalized_args:
            check_primitive_placeholder_is_used_for_correct_io_type(
                inputs_dict, outputs_dict, arg)
    elif isinstance(arg, placeholders.ConcatPlaceholder):
        for arg in arg.items:
            check_primitive_placeholder_is_used_for_correct_io_type(
                inputs_dict, outputs_dict, arg)
