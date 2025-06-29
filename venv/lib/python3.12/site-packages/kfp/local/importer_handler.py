# Copyright 2024 The Kubeflow Authors
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
"""Code for running a dsl.importer locally."""
import logging
from typing import Any, Dict, Tuple
import warnings

from google.protobuf import json_format
from kfp import dsl
from kfp.dsl.types import artifact_types
from kfp.dsl.types import type_utils
from kfp.local import logging_utils
from kfp.local import placeholder_utils
from kfp.local import status
from kfp.pipeline_spec import pipeline_spec_pb2

Outputs = Dict[str, Any]


def run_importer(
    pipeline_resource_name: str,
    component_name: str,
    component_spec: pipeline_spec_pb2.ComponentSpec,
    executor_spec: pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec,
    arguments: Dict[str, Any],
    pipeline_root: str,
    unique_pipeline_id: str,
) -> Tuple[Outputs, status.Status]:
    """Runs an importer component and returns a two-tuple of (outputs, status).

    Args:
        pipeline_resource_name: The root pipeline resource name.
        component_name: The name of the component.
        component_spec: The ComponentSpec of the importer.
        executor_spec: The ExecutorSpec of the importer.
        arguments: The arguments to the importer, as determined by the TaskInputsSpec for the importer.
        pipeline_root: The local pipeline root directory of the current pipeline.
        unique_pipeline_id: A unique identifier for the pipeline for placeholder resolution.

    Returns:
        A two-tuple of the output dictionary ({"artifact": <the-artifact>}) and the status. The outputs dictionary will be empty when status is failure.
    """
    from kfp.local import executor_input_utils

    task_resource_name = executor_input_utils.get_local_task_resource_name(
        component_name)
    task_name_for_logs = logging_utils.format_task_name(task_resource_name)
    with logging_utils.local_logger_context():
        logging.info(f'Executing task {task_name_for_logs}')

    task_root = executor_input_utils.construct_local_task_root(
        pipeline_root=pipeline_root,
        pipeline_resource_name=pipeline_resource_name,
        task_resource_name=task_resource_name,
    )
    executor_input = executor_input_utils.construct_executor_input(
        component_spec=component_spec,
        arguments=arguments,
        task_root=task_root,
        block_input_artifact=True,
    )
    uri = get_importer_uri(
        importer_spec=executor_spec.importer,
        executor_input=executor_input,
    )
    metadata = json_format.MessageToDict(executor_spec.importer.metadata)
    executor_input_dict = executor_input_utils.executor_input_to_dict(
        executor_input=executor_input,
        component_spec=component_spec,
    )
    metadata = placeholder_utils.recursively_resolve_json_dict_placeholders(
        metadata,
        executor_input_dict=executor_input_dict,
        pipeline_resource_name=pipeline_resource_name,
        task_resource_name=task_resource_name,
        pipeline_root=pipeline_root,
        pipeline_job_id=unique_pipeline_id,
        pipeline_task_id=placeholder_utils.make_random_id(),
    )
    ArtifactCls = get_artifact_class_from_schema_title(
        executor_spec.importer.type_schema.schema_title)
    outputs = {
        'artifact': ArtifactCls(
            name='artifact',
            uri=uri,
            metadata=metadata,
        )
    }
    with logging_utils.local_logger_context():
        logging.info(
            f'Task {task_name_for_logs} finished with status {logging_utils.format_status(status.Status.SUCCESS)}'
        )
        output_string = [
            f'Task {task_name_for_logs} outputs:',
            *logging_utils.make_log_lines_for_outputs(outputs),
        ]
        logging.info('\n'.join(output_string))
        logging_utils.print_horizontal_line()

    return outputs, status.Status.SUCCESS


def get_importer_uri(
    importer_spec: pipeline_spec_pb2.PipelineDeploymentConfig.ImporterSpec,
    executor_input: pipeline_spec_pb2.ExecutorInput,
) -> str:
    value_or_runtime_param = importer_spec.artifact_uri.WhichOneof('value')
    if value_or_runtime_param == 'constant':
        uri = importer_spec.artifact_uri.constant.string_value
    elif value_or_runtime_param == 'runtime_parameter':
        uri = executor_input.inputs.parameter_values['uri'].string_value
    else:
        raise ValueError(
            f'Got unknown value of artifact_uri: {value_or_runtime_param}')

    if any(
            uri.startswith(prefix)
            for prefix in [p.value for p in artifact_types.RemotePrefix]):
        warnings.warn(
            f"It looks like you're using the remote file '{uri}' in a 'dsl.importer'. Note that you will only be able to read and write to/from local files using 'artifact.path' in local executed pipelines."
        )

    return uri


def get_artifact_class_from_schema_title(schema_title: str) -> dsl.Artifact:
    system_prefix = 'system.'
    if schema_title.startswith(system_prefix):
        return type_utils.ARTIFACT_CLASSES_MAPPING[schema_title.lstrip(
            system_prefix).lower()]
    return dsl.Artifact
