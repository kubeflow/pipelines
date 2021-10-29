# Copyright 2020 The Kubeflow Authors
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
"""Utility function for building Importer Node spec."""

from typing import Union, Type

from kfp.v2.components import utils as component_utils
from kfp.v2.components.experimental import pipeline_task, structures
from kfp.v2.components.experimental import pipeline_channel

from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.v2.components.types import artifact_types, type_utils

INPUT_KEY = 'uri'
OUTPUT_KEY = 'artifact'


def _build_importer_spec(
    artifact_uri: Union[pipeline_channel.PipelineParameterChannel, str],
    artifact_type_schema: pipeline_spec_pb2.ArtifactTypeSchema,
) -> pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec:
    """Builds an importer executor spec.

    Args:
      artifact_uri: The artifact uri to import from.
      artifact_type_schema: The user specified artifact type schema of the
        artifact to be imported.

    Returns:
      An importer spec.
    """
    importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec()
    importer_spec.type_schema.CopyFrom(artifact_type_schema)

    if isinstance(artifact_uri, pipeline_channel.PipelineParameterChannel):
        importer_spec.artifact_uri.runtime_parameter = INPUT_KEY
    elif isinstance(artifact_uri, str):
        importer_spec.artifact_uri.constant_value.string_value = artifact_uri

    return importer_spec


def _build_importer_task_spec(
    importer_base_name: str,
    artifact_uri: Union[pipeline_channel.PipelineParameterChannel, str],
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Builds an importer task spec.

    Args:
      importer_base_name: The base name of the importer node.
      artifact_uri: The artifact uri to import from.

    Returns:
      An importer node task spec.
    """
    result = pipeline_spec_pb2.PipelineTaskSpec()
    result.component_ref.name = component_utils.sanitize_component_name(
        importer_base_name)

    if isinstance(artifact_uri, pipeline_channel.PipelineParameterChannel):
        param = artifact_uri
        if param.task_name:
            result.inputs.parameters[
                INPUT_KEY].task_output_parameter.producer_task = (
                    component_utils.sanitize_task_name(param.task_name))
            result.inputs.parameters[
                INPUT_KEY].task_output_parameter.output_parameter_key = param.name
        else:
            result.inputs.parameters[
                INPUT_KEY].component_input_parameter = param.full_name
    elif isinstance(artifact_uri, str):
        result.inputs.parameters[
            INPUT_KEY].runtime_value.constant_value.string_value = artifact_uri

    return result


def _build_importer_component_spec(
    importer_base_name: str,
    artifact_type_schema: pipeline_spec_pb2.ArtifactTypeSchema,
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds an importer component spec.

    Args:
      importer_base_name: The base name of the importer node.
      artifact_type_schema: The user specified artifact type schema of the
        artifact to be imported.

    Returns:
      An importer node component spec.
    """
    result = pipeline_spec_pb2.ComponentSpec()
    result.executor_label = component_utils.sanitize_executor_label(
        importer_base_name)
    result.input_definitions.parameters[
        INPUT_KEY].type = pipeline_spec_pb2.PrimitiveType.STRING
    result.output_definitions.artifacts[OUTPUT_KEY].artifact_type.CopyFrom(
        artifact_type_schema)

    return result


def importer(artifact_uri: Union[pipeline_channel.PipelineParameterChannel, str],
             artifact_class: Type[artifact_types.Artifact],
             reimport: bool = False) -> pipeline_task.PipelineTask:
    """dsl.importer for importing an existing artifact.

    Args:
      artifact_uri: The artifact uri to import from.
      artifact_type_schema: The user specified artifact type schema of the
        artifact to be imported.
      reimport: Whether to reimport the artifact. Defaults to False.

    Returns:
      A PipelineTask instance.

    Raises:
      ValueError if the passed in artifact_uri is neither a PipelineParam nor a
        constant string value.
    """

    if isinstance(artifact_uri, pipeline_channel.PipelineParameterChannel):
        input_param = artifact_uri
    elif isinstance(artifact_uri, str):
        input_param = pipeline_channel.PipelineParameterChannel(
            name='uri', value=artifact_uri, channel_type='String')
    else:
        raise ValueError(
            'Importer got unexpected artifact_uri: {} of type: {}.'.format(
                artifact_uri, type(artifact_uri)))

    task = pipeline_task.create_pipeline_task(
        component_spec=structures.ComponentSpec(
            name='importer',
            implementation=structures.Implementation(
                importer=structures.ImporterSpec(
                    artifact_uri=input_param,
                    type_schema=artifact_class,
                    reimport=reimport
                )
            ),
        ),
        arguments={}
    )

    artifact_type_schema = type_utils.get_artifact_type_schema(artifact_class)
    task.importer_spec = _build_importer_spec(
        artifact_uri=artifact_uri, artifact_type_schema=artifact_type_schema)
    task.component_spec = _build_importer_component_spec(
        importer_base_name=task.name, artifact_type_schema=artifact_type_schema)
    task.task_spec = _build_importer_task_spec(
        importer_base_name=task.name, artifact_uri=artifact_uri)

    return task
