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

import sys
from typing import Any, Dict, List, Mapping, Optional, Tuple, Type, Union

from google.protobuf import struct_pb2
from kfp.dsl import _container_op, _pipeline_param, dsl_utils
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.v2.components.types import artifact_types, type_utils
from kfp.components import _naming

URI_KEY = 'uri'
OUTPUT_KEY = 'artifact'
METADATA_KEY = 'metadata'


def make_input_parameter_placeholder(key: str) -> str:
    return f"{{{{$.inputs.parameters['{key}']}}}}"


def transform_metadata_and_get_inputs(
    metadata: Dict[Union[str, _pipeline_param.PipelineParam],
                   Union[_pipeline_param.PipelineParam, Any]]
) -> Tuple[Dict[str, Any], Dict[str, _pipeline_param.PipelineParam]]:
    metadata_inputs: Dict[str, _pipeline_param.PipelineParam] = {}

    def traverse_dict_and_create_metadata_inputs(d: Any) -> Any:
        if isinstance(d, _pipeline_param.PipelineParam):
            reversed_metadata_inputs = {
                pipeline_param: name
                for name, pipeline_param in metadata_inputs.items()
            }
            unique_name = reversed_metadata_inputs.get(
                d,
                _naming._make_name_unique_by_adding_index(
                    METADATA_KEY, metadata_inputs, '-'),
            )
            metadata_inputs[unique_name] = d
            return make_input_parameter_placeholder(unique_name)
        elif isinstance(d, dict):
            res = {}
            for k, v in d.items():
                new_k = traverse_dict_and_create_metadata_inputs(k)
                new_v = traverse_dict_and_create_metadata_inputs(v)
                res[new_k] = new_v
            return res

        elif isinstance(d, list):
            return [traverse_dict_and_create_metadata_inputs(el) for el in d]
        else:
            return d

    new_metadata = traverse_dict_and_create_metadata_inputs(metadata)
    return new_metadata, metadata_inputs


def _build_importer_spec(
    artifact_uri: Union[_pipeline_param.PipelineParam, str],
    artifact_type_schema: pipeline_spec_pb2.ArtifactTypeSchema,
    metadata_with_placeholders: Optional[Mapping[str, Any]] = None,
) -> pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec:
    """Builds an importer executor spec.

    Args:
      artifact_uri: The artifact uri to import from.
      artifact_type_schema: The user specified artifact type schema of the
        artifact to be imported.
      metadata_with_placeholders: Metadata to assign to the artifact, with pipeline parameters 
        replaced with string placeholders.

    Returns:
      An importer spec.
    """
    importer_spec = pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec()
    importer_spec.type_schema.CopyFrom(artifact_type_schema)

    if isinstance(artifact_uri, _pipeline_param.PipelineParam):
        importer_spec.artifact_uri.runtime_parameter = URI_KEY
    elif isinstance(artifact_uri, str):
        importer_spec.artifact_uri.constant_value.string_value = artifact_uri

    if metadata_with_placeholders:
        metadata_protobuf_struct = struct_pb2.Struct()
        metadata_protobuf_struct.update(metadata_with_placeholders)
        importer_spec.metadata.CopyFrom(metadata_protobuf_struct)

    return importer_spec


def _build_importer_task_spec(
    importer_base_name: str,
    artifact_uri: Union[_pipeline_param.PipelineParam, str],
    metadata_inputs: Dict[str, _pipeline_param.PipelineParam],
) -> pipeline_spec_pb2.PipelineTaskSpec:
    """Builds an importer task spec.

    Args:
      importer_base_name: The base name of the importer node.
      artifact_uri: The artifact uri to import from.
      metadata_inputs: Pipeline parameters for metadata.

    Returns:
      An importer node task spec.
    """
    result = pipeline_spec_pb2.PipelineTaskSpec()
    result.component_ref.name = dsl_utils.sanitize_component_name(
        importer_base_name)

    if isinstance(artifact_uri, _pipeline_param.PipelineParam):
        param = artifact_uri
        if param.op_name:
            result.inputs.parameters[
                URI_KEY].task_output_parameter.producer_task = (
                    dsl_utils.sanitize_task_name(param.op_name))
            result.inputs.parameters[
                URI_KEY].task_output_parameter.output_parameter_key = param.name
        else:
            result.inputs.parameters[
                URI_KEY].component_input_parameter = param.full_name
    elif isinstance(artifact_uri, str):
        result.inputs.parameters[
            URI_KEY].runtime_value.constant_value.string_value = artifact_uri

    for unique_name, pipeline_param in metadata_inputs.items():
        if pipeline_param.op_name:
            result.inputs.parameters[
                unique_name].task_output_parameter.producer_task = (
                    dsl_utils.sanitize_task_name(pipeline_param.op_name))
            result.inputs.parameters[
                unique_name].task_output_parameter.output_parameter_key = pipeline_param.name
        else:
            result.inputs.parameters[
                unique_name].component_input_parameter = pipeline_param.full_name

    return result


def _build_importer_component_spec(
    importer_base_name: str,
    artifact_type_schema: pipeline_spec_pb2.ArtifactTypeSchema,
    metadata_inputs: Dict[str, _pipeline_param.PipelineParam],
) -> pipeline_spec_pb2.ComponentSpec:
    """Builds an importer component spec.

    Args:
      importer_base_name: The base name of the importer node.
      artifact_type_schema: The user specified artifact type schema of the
        artifact to be imported.
      metadata_inputs: Pipeline parameters for metadata.

    Returns:
      An importer node component spec.
    """
    result = pipeline_spec_pb2.ComponentSpec()
    result.executor_label = dsl_utils.sanitize_executor_label(
        importer_base_name)
    result.input_definitions.parameters[
        URI_KEY].type = pipeline_spec_pb2.PrimitiveType.STRING
    result.output_definitions.artifacts[OUTPUT_KEY].artifact_type.CopyFrom(
        artifact_type_schema)

    for unique_name, pipeline_param in metadata_inputs.items():
        result.input_definitions.parameters[
            unique_name].type = type_utils._PARAMETER_TYPES_MAPPING.get(
                pipeline_param.param_type.lower())

    return result


def importer(
        artifact_uri: Union[_pipeline_param.PipelineParam, str],
        artifact_class: Type[artifact_types.Artifact],
        reimport: bool = False,
        metadata: Optional[Mapping[str,
                                   Any]] = None) -> _container_op.ContainerOp:
    """dsl.importer for importing an existing artifact. Only for v2 pipeline.

    Args:
      artifact_uri: The artifact uri to import from.
      artifact_type_schema: The user specified artifact type schema of the
        artifact to be imported.
      reimport: Whether to reimport the artifact. Defaults to False.
      metadata: Metadata to assign to the artifact.

    Returns:
      A ContainerOp instance.

    Raises:
      ValueError if the passed in artifact_uri is neither a PipelineParam nor a
        constant string value.
    """

    if isinstance(artifact_uri, _pipeline_param.PipelineParam):
        input_param = artifact_uri

    elif isinstance(artifact_uri, str):
        input_param = _pipeline_param.PipelineParam(
            name=URI_KEY, value=artifact_uri, param_type='String')
    else:
        raise ValueError(
            'Importer got unexpected artifact_uri: {} of type: {}.'.format(
                artifact_uri, type(artifact_uri)))

    old_warn_value = _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING
    _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = True

    task = _container_op.ContainerOp(
        name='importer',
        image='importer_image',  # TODO: need a v1 implementation of importer.
        file_outputs={
            OUTPUT_KEY:
                "{{{{$.outputs.artifacts['{}'].uri}}}}".format(OUTPUT_KEY)
        },
    )
    _container_op.ContainerOp._DISABLE_REUSABLE_COMPONENT_WARNING = old_warn_value

    metadata_with_placeholders, metadata_inputs = transform_metadata_and_get_inputs(
        metadata)

    artifact_type_schema = type_utils.get_artifact_type_schema(artifact_class)
    task.importer_spec = _build_importer_spec(
        artifact_uri=artifact_uri,
        artifact_type_schema=artifact_type_schema,
        metadata_with_placeholders=metadata_with_placeholders,
    )
    task.task_spec = _build_importer_task_spec(
        importer_base_name=task.name,
        artifact_uri=artifact_uri,
        metadata_inputs=metadata_inputs,
    )
    task.component_spec = _build_importer_component_spec(
        importer_base_name=task.name,
        artifact_type_schema=artifact_type_schema,
        metadata_inputs=metadata_inputs,
    )

    task.inputs = [input_param]
    task.inputs += list(metadata_inputs.values())

    return task
