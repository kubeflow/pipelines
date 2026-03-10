# Copyright 2018 The Kubeflow Authors
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

import collections
import copy
import inspect
import json
import pathlib
from typing import Any, Mapping, Optional

import kfp
from kfp.components import _structures
from kfp.components import _components
from kfp.components import _naming
from kfp import dsl
from kfp.dsl import _container_op
from kfp.dsl import _for_loop
from kfp.dsl import _pipeline_param
from kfp.dsl import dsl_utils
from kfp.dsl import types

# Placeholder to represent the output directory hosting all the generated URIs.
# Its actual value will be specified during pipeline compilation.
# The format of OUTPUT_DIR_PLACEHOLDER is serialized dsl.PipelineParam, to
# ensure being extracted as a pipeline parameter during compilation.
# Note that we cannot direclty import dsl module here due to circular
# dependencies.
OUTPUT_DIR_PLACEHOLDER = '{{pipelineparam:op=;name=pipeline-root}}'
# Placeholder to represent to UID of the current pipeline at runtime.
# Will be replaced by engine-specific placeholder during compilation.
RUN_ID_PLACEHOLDER = '{{kfp.run_uid}}'
# Format of the Argo parameter used to pass the producer's Pod ID to
# the consumer.
PRODUCER_POD_NAME_PARAMETER = '{}-producer-pod-id-'
# Format of the input output port name placeholder.
INPUT_OUTPUT_NAME_PATTERN = '{{{{kfp.input-output-name.{}}}}}'
# Fixed name for per-task output metadata json file.
OUTPUT_METADATA_JSON = '/tmp/outputs/executor_output.json'
# Executor input placeholder.
_EXECUTOR_INPUT_PLACEHOLDER = '{{$}}'


# TODO(chensun): block URI placeholder usage in v1.
def _generate_output_uri_placeholder(port_name: str) -> str:
    """Generates the URI placeholder for an output."""
    return "{{{{$.outputs.artifacts['{}'].uri}}}}".format(port_name)


def _generate_input_uri_placeholder(port_name: str) -> str:
    """Generates the URI placeholder for an input."""
    return "{{{{$.inputs.artifacts['{}'].uri}}}}".format(port_name)


def _generate_output_metadata_path() -> str:
    """Generates the URI to write the output metadata JSON file."""

    return OUTPUT_METADATA_JSON


def _generate_input_metadata_path(port_name: str) -> str:
    """Generates the placeholder for input artifact metadata file."""

    # Return a placeholder for path to input artifact metadata, which will be
    # rewritten during pipeline compilation.
    return str(
        pathlib.PurePosixPath(
            OUTPUT_DIR_PLACEHOLDER, RUN_ID_PLACEHOLDER,
            '{{{{inputs.parameters.{input}}}}}'.format(
                input=PRODUCER_POD_NAME_PARAMETER.format(port_name)),
            OUTPUT_METADATA_JSON))


def _generate_input_output_name(port_name: str) -> str:
    """Generates the placeholder for input artifact's output name."""

    # Return a placeholder for the output port name of the input artifact, which
    # will be rewritten during pipeline compilation.
    return INPUT_OUTPUT_NAME_PATTERN.format(port_name)


def _generate_executor_input() -> str:
    """Generates the placeholder for serialized executor input."""
    return _EXECUTOR_INPUT_PLACEHOLDER


class ExtraPlaceholderResolver:

    def __init__(self):
        self.input_paths = {}
        self.input_metadata_paths = {}
        self.output_paths = {}

    def resolve_placeholder(
        self,
        arg,
        component_spec: _structures.ComponentSpec,
        arguments: dict,
    ) -> str:
        inputs_dict = {
            input_spec.name: input_spec
            for input_spec in component_spec.inputs or []
        }

        if isinstance(arg, _structures.InputUriPlaceholder):
            input_name = arg.input_name
            if input_name in arguments:
                input_uri = _generate_input_uri_placeholder(input_name)
                self.input_paths[
                    input_name] = _components._generate_input_file_name(
                        input_name)
                return input_uri
            else:
                input_spec = inputs_dict[input_name]
                if input_spec.optional:
                    return None
                else:
                    raise ValueError(
                        'No value provided for input {}'.format(input_name))

        elif isinstance(arg, _structures.OutputUriPlaceholder):
            output_name = arg.output_name
            output_uri = _generate_output_uri_placeholder(output_name)
            self.output_paths[
                output_name] = _components._generate_output_file_name(
                    output_name)
            return output_uri

        elif isinstance(arg, _structures.InputMetadataPlaceholder):
            input_name = arg.input_name
            if input_name in arguments:
                input_metadata_path = _generate_input_metadata_path(input_name)
                self.input_metadata_paths[input_name] = input_metadata_path
                return input_metadata_path
            else:
                input_spec = inputs_dict[input_name]
                if input_spec.optional:
                    return None
                else:
                    raise ValueError(
                        'No value provided for input {}'.format(input_name))

        elif isinstance(arg, _structures.InputOutputPortNamePlaceholder):
            input_name = arg.input_name
            if input_name in arguments:
                return _generate_input_output_name(input_name)
            else:
                input_spec = inputs_dict[input_name]
                if input_spec.optional:
                    return None
                else:
                    raise ValueError(
                        'No value provided for input {}'.format(input_name))

        elif isinstance(arg, _structures.OutputMetadataPlaceholder):
            # TODO: Consider making the output metadata per-artifact.
            return _generate_output_metadata_path()
        elif isinstance(arg, _structures.ExecutorInputPlaceholder):
            return _generate_executor_input()

        return None


def _create_container_op_from_component_and_arguments(
    component_spec: _structures.ComponentSpec,
    arguments: Mapping[str, Any],
    component_ref: Optional[_structures.ComponentReference] = None,
) -> _container_op.ContainerOp:
    """Instantiates ContainerOp object.

    Args:
      component_spec: The component spec object.
      arguments: The dictionary of component arguments.
      component_ref: (only for v1) The component references.

    Returns:
      A ContainerOp instance.
    """
    # Add component inputs with default value to the arguments dict if they are not
    # in the arguments dict already.
    arguments = arguments.copy()
    for input_spec in component_spec.inputs or []:
        if input_spec.name not in arguments and input_spec.default is not None:
            default_value = input_spec.default
            if input_spec.type == 'Integer':
                default_value = int(default_value)
            elif input_spec.type == 'Float':
                default_value = float(default_value)
            arguments[input_spec.name] = default_value

    # Check types of the reference arguments and serialize PipelineParams
    original_arguments = arguments
    arguments = arguments.copy()
    for input_name, argument_value in arguments.items():
        if isinstance(argument_value, _pipeline_param.PipelineParam):
            input_type = component_spec._inputs_dict[input_name].type
            argument_type = argument_value.param_type
            types.verify_type_compatibility(
                argument_type, input_type,
                'Incompatible argument passed to the input "{}" of component "{}": '
                .format(input_name, component_spec.name))

            arguments[input_name] = str(argument_value)
        if isinstance(argument_value, _container_op.ContainerOp):
            raise TypeError(
                'ContainerOp object was passed to component as an input argument. '
                'Pass a single output instead.')
    placeholder_resolver = ExtraPlaceholderResolver()
    resolved_cmd = _components._resolve_command_line_and_paths(
        component_spec=component_spec,
        arguments=arguments,
        placeholder_resolver=placeholder_resolver.resolve_placeholder,
    )

    container_spec = component_spec.implementation.container

    old_warn_value = _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING
    _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = True

    output_paths = collections.OrderedDict(resolved_cmd.output_paths or {})
    output_paths.update(placeholder_resolver.output_paths)
    input_paths = collections.OrderedDict(resolved_cmd.input_paths or {})
    input_paths.update(placeholder_resolver.input_paths)

    artifact_argument_paths = [
        dsl.InputArgumentPath(
            argument=arguments[input_name],
            input=input_name,
            path=path,
        ) for input_name, path in input_paths.items()
    ]

    task = _container_op.ContainerOp(
        name=component_spec.name or _components._default_component_name,
        image=container_spec.image,
        command=resolved_cmd.command,
        arguments=resolved_cmd.args,
        file_outputs=output_paths,
        artifact_argument_paths=artifact_argument_paths,
    )
    _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = old_warn_value

    component_meta = copy.copy(component_spec)
    task._set_metadata(component_meta, original_arguments)
    if component_ref:
        component_ref_without_spec = copy.copy(component_ref)
        component_ref_without_spec.spec = None
        task._component_ref = component_ref_without_spec

    task._parameter_arguments = resolved_cmd.inputs_consumed_by_value
    name_to_spec_type = {}
    if component_meta.inputs:
        name_to_spec_type = {
            input.name: input.type for input in component_meta.inputs
        }

    for name in list(task.artifact_arguments.keys()):
        if name in task._parameter_arguments:
            del task.artifact_arguments[name]

    for name in list(task.input_artifact_paths.keys()):
        if name in task._parameter_arguments:
            del task.input_artifact_paths[name]

    # Previously, ContainerOp had strict requirements for the output names, so we
    # had to convert all the names before passing them to the ContainerOp
    # constructor.
    # Outputs with non-pythonic names could not be accessed using their original
    # names. Now ContainerOp supports any output names, so we're now using the
    # original output names. However to support legacy pipelines, we're also
    # adding output references with pythonic names.
    # TODO: Add warning when people use the legacy output names.
    output_names = [
        output_spec.name for output_spec in component_spec.outputs or []
    ]  # Stabilizing the ordering
    output_name_to_python = _naming.generate_unique_name_conversion_table(
        output_names, _naming._sanitize_python_function_name)
    for output_name in output_names:
        pythonic_output_name = output_name_to_python[output_name]
        # Note: Some component outputs are currently missing from task.outputs
        # (e.g. MLPipeline UI Metadata)
        if pythonic_output_name not in task.outputs and output_name in task.outputs:
            task.outputs[pythonic_output_name] = task.outputs[output_name]

    if container_spec.env:
        from kubernetes import client as k8s_client
        for name, value in container_spec.env.items():
            task.container.add_env_variable(
                k8s_client.V1EnvVar(name=name, value=value))

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

