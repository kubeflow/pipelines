# Copyright 2023 The Kubeflow Authors
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
"""Utilities for reading and processing the ExecutorOutput message."""
import json
import os
from typing import Any, Dict, List, Union

from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp import dsl
from kfp.compiler import pipeline_spec_builder
from kfp.dsl import executor
from kfp.pipeline_spec import pipeline_spec_pb2


def load_executor_output(
        executor_output_path: str) -> pipeline_spec_pb2.ExecutorOutput:
    """Loads the ExecutorOutput message from a path."""
    executor_output = pipeline_spec_pb2.ExecutorOutput()

    if not os.path.isfile(executor_output_path):
        return executor_output

    with open(executor_output_path) as f:
        json_format.Parse(f.read(), executor_output)
    return executor_output


def cast_protobuf_numbers(
    output_parameters: Dict[str, Any],
    output_parameter_types: Dict[
        str, pipeline_spec_pb2.ComponentOutputsSpec.ParameterSpec],
) -> Dict[str, Any]:
    """Casts output fields that are typed as NUMBER_INTEGER to a Python int.

    This is required, since google.protobuf.struct_pb2.Value uses
    number_value to represent both floats and ints. When converting
    struct_pb2.Value to a dict/json, int will be upcast to float, even
    if the component output specifies int.
    """
    int_output_keys = [
        output_param_name
        for output_param_name, parameter_spec in output_parameter_types.items()
        if parameter_spec.parameter_type ==
        pipeline_spec_pb2.ParameterType.ParameterTypeEnum.NUMBER_INTEGER
    ]
    for int_output_key in int_output_keys:
        # avoid KeyError when the user never writes to the dsl.OutputPath
        if int_output_key in output_parameters:
            output_parameters[int_output_key] = int(
                output_parameters[int_output_key])
    return output_parameters


def get_outputs_from_executor_output(
    executor_output: pipeline_spec_pb2.ExecutorOutput,
    executor_input: pipeline_spec_pb2.ExecutorInput,
    component_spec: pipeline_spec_pb2.ComponentSpec,
) -> Dict[str, Any]:
    """Obtains a dictionary of output key to output value from several messages
    corresponding to the executed a component/task."""
    executor_output = add_type_to_executor_output(
        executor_input=executor_input,
        executor_output=executor_output,
    )

    # merge any parameter outputs written using dsl.OutputPath with the rest of ExecutorOutput
    executor_output = merge_dsl_output_file_parameters_to_executor_output(
        executor_input=executor_input,
        executor_output=executor_output,
        component_spec=component_spec,
    )

    # collect parameter outputs from executor output
    output_parameters = {
        param_name: pb2_value_to_python(value)
        for param_name, value in executor_output.parameter_values.items()
    }
    # process the special case of protobuf ints
    output_parameters = cast_protobuf_numbers(
        output_parameters,
        component_spec.output_definitions.parameters,
    )

    # collect artifact outputs from executor output
    output_artifact_definitions = component_spec.output_definitions.artifacts
    output_artifacts = {
        artifact_name: artifact_list_to_dsl_artifact(
            artifact_list,
            is_artifact_list=output_artifact_definitions[artifact_name]
            .is_artifact_list,
        ) for artifact_name, artifact_list in executor_output.artifacts.items()
    }
    return {**output_parameters, **output_artifacts}


def special_dsl_outputpath_read(
    parameter_name: str,
    output_file: str,
    dtype: pipeline_spec_pb2.ParameterType.ParameterTypeEnum,
) -> Any:
    """Reads the text in dsl.OutputPath files in the same way as the remote
    backend.

    In brief: read strings as strings and JSON load everything else.
    """
    try:
        with open(output_file) as f:
            value = f.read()

        if dtype == pipeline_spec_pb2.ParameterType.ParameterTypeEnum.STRING:
            value = value
        elif dtype == pipeline_spec_pb2.ParameterType.ParameterTypeEnum.BOOLEAN:
            # permit true/True and false/False, consistent with remote BE
            value = json.loads(value.lower())
        else:
            value = json.loads(value)
        return value
    except Exception as e:
        raise ValueError(
            f'Could not deserialize output {parameter_name!r} from path {output_file}'
        ) from e


def merge_dsl_output_file_parameters_to_executor_output(
    executor_input: pipeline_spec_pb2.ExecutorInput,
    executor_output: pipeline_spec_pb2.ExecutorOutput,
    component_spec: pipeline_spec_pb2.ComponentSpec,
) -> pipeline_spec_pb2.ExecutorOutput:
    """Merges and output parameters specified via dsl.OutputPath with the rest
    of the ExecutorOutput message."""
    for parameter_key, output_parameter in executor_input.outputs.parameters.items(
    ):
        if os.path.exists(output_parameter.output_file):
            parameter_value = special_dsl_outputpath_read(
                parameter_name=parameter_key,
                output_file=output_parameter.output_file,
                dtype=component_spec.output_definitions
                .parameters[parameter_key].parameter_type,
            )
            executor_output.parameter_values[parameter_key].CopyFrom(
                pipeline_spec_builder.to_protobuf_value(parameter_value))

    return executor_output


def pb2_value_to_python(value: struct_pb2.Value) -> Any:
    """Converts protobuf Value to the corresponding Python type."""
    if value.HasField('null_value'):
        return None
    elif value.HasField('number_value'):
        return value.number_value
    elif value.HasField('string_value'):
        return value.string_value
    elif value.HasField('bool_value'):
        return value.bool_value
    elif value.HasField('struct_value'):
        return pb2_struct_to_python(value.struct_value)
    elif value.HasField('list_value'):
        return [pb2_value_to_python(v) for v in value.list_value.values]
    else:
        raise ValueError(f'Unknown value type: {value}')


def pb2_struct_to_python(struct: struct_pb2.Struct) -> Dict[str, Any]:
    """Converts protobuf Struct to a dict."""
    return {k: pb2_value_to_python(v) for k, v in struct.fields.items()}


def runtime_artifact_to_dsl_artifact(
        runtime_artifact: pipeline_spec_pb2.RuntimeArtifact) -> dsl.Artifact:
    """Converts a single RuntimeArtifact instance to the corresponding
    dsl.Artifact instance."""
    return executor.create_artifact_instance(
        json_format.MessageToDict(runtime_artifact))


def artifact_list_to_dsl_artifact(
    artifact_list: pipeline_spec_pb2.ArtifactList,
    is_artifact_list: bool,
) -> Union[dsl.Artifact, List[dsl.Artifact]]:
    """Converts an ArtifactList instance to a single dsl.Artifact or a list of
    dsl.Artifacts, depending on whether the ArtifactList is a true list or
    simply a container for single artifact element."""
    dsl_artifacts = [
        runtime_artifact_to_dsl_artifact(artifact_spec)
        for artifact_spec in artifact_list.artifacts
    ]
    return dsl_artifacts if is_artifact_list else dsl_artifacts[0]


def add_type_to_executor_output(
    executor_input: pipeline_spec_pb2.ExecutorInput,
    executor_output: pipeline_spec_pb2.ExecutorOutput,
) -> pipeline_spec_pb2.ExecutorOutput:
    """Adds artifact type information (ArtifactTypeSchema) from the
    ExecutorInput message to the ExecutorOutput message.

    This information is not present in the ExecutorOutput message
    written by a task, though it is useful to have it for constructing
    local outputs. We don't want to change the executor logic and the
    serialized outputs of all tasks for this case, so we add this extra
    info to ExecutorOutput in memory.
    """
    for key, artifact_list in executor_output.artifacts.items():
        for artifact in artifact_list.artifacts:
            artifact.type.CopyFrom(
                executor_input.outputs.artifacts[key].artifacts[0].type)
    return executor_output


def get_outputs_for_task(
    executor_input: pipeline_spec_pb2.ExecutorInput,
    component_spec: pipeline_spec_pb2.ComponentSpec,
) -> Dict[str, Any]:
    """Gets outputs from a recently executed task, if available, using the
    ExecutorInput and ComponentSpec of the task."""
    executor_output = load_executor_output(
        executor_output_path=executor_input.outputs.output_file)
    return get_outputs_from_executor_output(
        executor_output=executor_output,
        executor_input=executor_input,
        component_spec=component_spec,
    )
