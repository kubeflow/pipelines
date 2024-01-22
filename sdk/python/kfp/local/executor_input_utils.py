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
"""Utilities for constructing the ExecutorInput message."""
import datetime
import os
from typing import Any, Dict

from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp.compiler import pipeline_spec_builder
from kfp.dsl import utils
from kfp.pipeline_spec import pipeline_spec_pb2

_EXECUTOR_OUTPUT_FILE = 'executor_output.json'


def construct_executor_input(
    component_spec: pipeline_spec_pb2.ComponentSpec,
    arguments: Dict[str, Any],
    task_root: str,
) -> pipeline_spec_pb2.ExecutorInput:
    """Constructs the executor input message for a task execution."""
    input_parameter_keys = list(
        component_spec.input_definitions.parameters.keys())
    input_artifact_keys = list(
        component_spec.input_definitions.artifacts.keys())
    if input_artifact_keys:
        raise ValueError(
            'Input artifacts are not yet supported for local execution.')

    output_parameter_keys = list(
        component_spec.output_definitions.parameters.keys())
    output_artifact_specs_dict = component_spec.output_definitions.artifacts

    inputs = pipeline_spec_pb2.ExecutorInput.Inputs(
        parameter_values={
            param_name:
            pipeline_spec_builder.to_protobuf_value(arguments[param_name])
            if param_name in arguments else component_spec.input_definitions
            .parameters[param_name].default_value
            for param_name in input_parameter_keys
        },
        # input artifact constants are not supported yet
        artifacts={},
    )
    outputs = pipeline_spec_pb2.ExecutorInput.Outputs(
        parameters={
            param_name: pipeline_spec_pb2.ExecutorInput.OutputParameter(
                output_file=os.path.join(task_root, param_name))
            for param_name in output_parameter_keys
        },
        artifacts={
            artifact_name: artifact_type_schema_to_artifact_list(
                name=artifact_name,
                artifact_type=artifact_spec.artifact_type,
                task_root=task_root,
            ) for artifact_name, artifact_spec in
            output_artifact_specs_dict.items()
        },
        output_file=os.path.join(task_root, _EXECUTOR_OUTPUT_FILE),
    )
    return pipeline_spec_pb2.ExecutorInput(
        inputs=inputs,
        outputs=outputs,
    )


def get_local_pipeline_resource_name(pipeline_name: str) -> str:
    """Gets the local pipeline resource name from the pipeline name in
    PipelineSpec.

    Args:
        pipeline_name: The pipeline name provided by PipelineSpec.pipelineInfo.name.

    Returns:
        The local pipeline resource name. Includes timestamp.
    """
    timestamp = datetime.datetime.now().strftime('%Y-%m-%d-%H-%M-%S-%f')
    return f'{pipeline_name}-{timestamp}'


def get_local_task_resource_name(component_name: str) -> str:
    """Gets the local task resource name from the component name in
    PipelineSpec.

    Args:
        component_name: The component name provided as the key for the component's ComponentSpec
    message. Takes the form comp-*.

    Returns:
        The local task resource name.
    """
    return component_name[len(utils.COMPONENT_NAME_PREFIX):]


def construct_local_task_root(
    pipeline_root: str,
    pipeline_resource_name: str,
    task_resource_name: str,
) -> str:
    """Constructs the local task root directory for a task."""
    return os.path.join(
        pipeline_root,
        pipeline_resource_name,
        task_resource_name,
    )


def artifact_type_schema_to_artifact_list(
    name: str,
    artifact_type: pipeline_spec_pb2.ArtifactTypeSchema,
    task_root: str,
) -> pipeline_spec_pb2.ArtifactList:
    """Constructs an ArtifactList instance for an artifact in ExecutorInput."""
    return pipeline_spec_pb2.ArtifactList(artifacts=[
        pipeline_spec_pb2.RuntimeArtifact(
            name=name,
            type=artifact_type,
            uri=os.path.join(task_root, name),
            # metadata always starts empty for output artifacts
            metadata=struct_pb2.Struct(),
        )
    ])


def executor_input_to_dict(
    executor_input: pipeline_spec_pb2.ExecutorInput,
    component_spec: pipeline_spec_pb2.ComponentSpec,
) -> Dict[str, Any]:
    """Converts the executor input to a dictionary.

    Since protobuf value represents ints and floats the same way, we
    cast ints to their correct type. This should be called before
    replacing placeholders with values.

    This is consistent with the remote backend behavior.
    """
    executor_input_dict = json_format.MessageToDict(executor_input)
    inputs_typed_int = [
        in_param_name for in_param_name, parameter_spec in
        component_spec.input_definitions.parameters.items()
        if parameter_spec.parameter_type ==
        pipeline_spec_pb2.ParameterType.ParameterTypeEnum.NUMBER_INTEGER
    ]
    for param_name, param_value in executor_input_dict.get('inputs', {}).get(
            'parameterValues', {}).items():
        if param_name in inputs_typed_int:
            executor_input_dict['inputs']['parameterValues'][param_name] = int(
                param_value)
    return executor_input_dict
