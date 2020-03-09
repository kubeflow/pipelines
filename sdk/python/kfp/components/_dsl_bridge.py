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
from .structures import ComponentSpec, ComponentReference
from ._components import _default_component_name, _resolve_command_line_and_paths
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

    resolved_cmd = _resolve_command_line_and_paths(
        component_spec=component_spec,
        arguments=arguments,
    )

    #Renaming outputs to conform with ContainerOp/Argo
    from ._naming import _sanitize_python_function_name, generate_unique_name_conversion_table
    output_names = (resolved_cmd.output_paths or {}).keys()
    output_name_to_python = generate_unique_name_conversion_table(output_names, _sanitize_python_function_name)
    output_paths_for_container_op = {output_name_to_python[name]: path for name, path in resolved_cmd.output_paths.items()}

    container_spec = component_spec.implementation.container

    task = dsl.ContainerOp(
        name=component_spec.name or _default_component_name,
        image=container_spec.image,
        command=resolved_cmd.command,
        arguments=resolved_cmd.args,
        file_outputs=output_paths_for_container_op,
        artifact_argument_paths=[
            dsl.InputArgumentPath(
                argument=arguments[input_name],
                input=input_name,
                path=path,
            )
            for input_name, path in resolved_cmd.input_paths.items()
        ],
    )
    # Fixing ContainerOp output types
    if component_spec.outputs:
        for output in component_spec.outputs:
            pythonic_name = output_name_to_python[output.name]
            if pythonic_name in task.outputs:
                task.outputs[pythonic_name].param_type = output.type

    component_meta = copy.copy(component_spec)
    component_meta.implementation = None
    task._set_metadata(component_meta)

    if container_spec.env:
        from kubernetes import client as k8s_client
        for name, value in container_spec.env.items():
            task.container.add_env_variable(k8s_client.V1EnvVar(name=name, value=value))

    if component_spec.metadata:
        for key, value in (component_spec.metadata.annotations or {}).items():
            task.add_pod_annotation(key, value)
        for key, value in (component_spec.metadata.labels or {}).items():
            task.add_pod_label(key, value)

    return task
