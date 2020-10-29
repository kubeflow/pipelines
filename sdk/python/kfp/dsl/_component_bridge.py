# Copyright 2018 Google LLC
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

import copy
from typing import Any, Mapping
from ..components.structures import ComponentSpec, ComponentReference
from ..components._components import _default_component_name, _resolve_command_line_and_paths
from ..components._naming import _sanitize_python_function_name, generate_unique_name_conversion_table
from .. import dsl


def _create_container_op_from_component_and_arguments(
    component_spec: ComponentSpec,
    arguments: Mapping[str, Any],
    component_ref: ComponentReference = None,
) -> 'dsl.ContainerOp':
    # Check types of the reference arguments and serialize PipelineParams
    arguments = arguments.copy()
    for input_name, argument_value in arguments.items():
        if isinstance(argument_value, dsl.PipelineParam):
            input_type = component_spec._inputs_dict[input_name].type
            reference_type = argument_value.param_type
            dsl.types.verify_type_compatibility(reference_type, input_type, 'Incompatible argument passed to the input "{}" of component "{}": '.format(input_name, component_spec.name))

            arguments[input_name] = str(argument_value)
        if isinstance(argument_value, dsl.ContainerOp):
            raise TypeError('ContainerOp object was passed to component as an input argument. Pass a single output instead.')

    resolved_cmd = _resolve_command_line_and_paths(
        component_spec=component_spec,
        arguments=arguments,
    )

    container_spec = component_spec.implementation.container

    old_warn_value = dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING
    dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = True
    task = dsl.ContainerOp(
        name=component_spec.name or _default_component_name,
        image=container_spec.image,
        command=resolved_cmd.command,
        arguments=resolved_cmd.args,
        file_outputs=resolved_cmd.output_paths,
        artifact_argument_paths=[
            dsl.InputArgumentPath(
                argument=arguments[input_name],
                input=input_name,
                path=path,
            )
            for input_name, path in resolved_cmd.input_paths.items()
        ],
    )
    dsl.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = old_warn_value

    component_meta = copy.copy(component_spec)
    task._set_metadata(component_meta)
    component_ref_without_spec = copy.copy(component_ref)
    component_ref_without_spec.spec = None
    task._component_ref = component_ref_without_spec

    task._parameter_arguments = resolved_cmd.inputs_consumed_by_value

    # Previously, ContainerOp had strict requirements for the output names, so we had to
    # convert all the names before passing them to the ContainerOp constructor.
    # Outputs with non-pythonic names could not be accessed using their original names.
    # Now ContainerOp supports any output names, so we're now using the original output names.
    # However to support legacy pipelines, we're also adding output references with pythonic names.
    # TODO: Add warning when people use the legacy output names.
    output_names = [output_spec.name for output_spec in component_spec.outputs or []] # Stabilizing the ordering
    output_name_to_python = generate_unique_name_conversion_table(output_names, _sanitize_python_function_name)
    for output_name in output_names:
        pythonic_output_name = output_name_to_python[output_name]
        # Note: Some component outputs are currently missing from task.outputs (e.g. MLPipeline UI Metadata)
        if pythonic_output_name not in task.outputs and output_name in task.outputs:
            task.outputs[pythonic_output_name] = task.outputs[output_name]

    if container_spec.env:
        from kubernetes import client as k8s_client
        for name, value in container_spec.env.items():
            task.container.add_env_variable(k8s_client.V1EnvVar(name=name, value=value))

    if component_spec.metadata:
        annotations = component_spec.metadata.annotations or {}
        for key, value in annotations.items():
            task.add_pod_annotation(key, value)
        for key, value in (component_spec.metadata.labels or {}).items():
            task.add_pod_label(key, value)
        # Disabling the caching for the volatile components by default
        if annotations.get('volatile_component', 'false') == 'true':
            task.execution_options.caching_strategy.max_cache_staleness = 'P0D'

    return task
