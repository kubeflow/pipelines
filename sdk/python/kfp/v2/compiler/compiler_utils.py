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
"""KFP v2 DSL compiler utility functions."""

from multiprocessing.sharedctypes import Value
import re
from typing import Mapping, Optional

from google.protobuf import json_format
from kfp.containers import _component_builder
from kfp.dsl import _container_op
from kfp.dsl import _pipeline_param
from kfp.pipeline_spec import pipeline_spec_pb2
from kfp.v2.components.types import type_utils

# Alias for PipelineContainerSpec
PipelineContainerSpec = pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec


def build_runtime_config_spec(
    output_directory: str,
    pipeline_parameters: Optional[Mapping[
        str, _pipeline_param.PipelineParam]] = None,
) -> pipeline_spec_pb2.PipelineJob.RuntimeConfig:
    """Converts pipeine parameters to runtime parameters mapping.

    Args:
      output_directory: The root of pipeline outputs.
      pipeline_parameters: The mapping from parameter names to PipelineParam
        objects. Optional.

    Returns:
      A pipeline job RuntimeConfig object.
    """

    def _get_value(
            param: _pipeline_param.PipelineParam) -> pipeline_spec_pb2.Value:
        assert param.value is not None, 'None values should be filterd out.'

        result = pipeline_spec_pb2.Value()
        # TODO(chensun): remove defaulting to 'String' for None param_type once we
        # fix importer behavior.
        param_type = type_utils.get_parameter_type(param.param_type or 'String')
        if param_type == pipeline_spec_pb2.PrimitiveType.INT:
            result.int_value = int(param.value)
        elif param_type == pipeline_spec_pb2.PrimitiveType.DOUBLE:
            result.double_value = float(param.value)
        elif param_type == pipeline_spec_pb2.PrimitiveType.STRING:
            result.string_value = str(param.value)
        else:
            # For every other type, defaults to 'String'.
            # TODO(chensun): remove this default behavior once we migrate from
            # `pipeline_spec_pb2.Value` to `protobuf.Value`.
            result.string_value = str(param.value)

        return result

    parameters = pipeline_parameters or {}
    return pipeline_spec_pb2.PipelineJob.RuntimeConfig(
        gcs_output_directory=output_directory,
        parameters={
            k: _get_value(v)
            for k, v in parameters.items()
            if v.value is not None
        })


def validate_pipeline_name(name: str) -> None:
    """Validate pipeline name.

    A valid pipeline name should match ^[a-z0-9][a-z0-9-]{0,127}$.

    Args:
      name: The pipeline name.

    Raises:
      ValueError if the pipeline name doesn't conform to the regular expression.
    """
    pattern = re.compile(r'^[a-z0-9][a-z0-9-]{0,127}$')
    if not pattern.match(name):
        raise ValueError(
            'Invalid pipeline name: %s.\n'
            'Please specify a pipeline name that matches the regular '
            'expression "^[a-z0-9][a-z0-9-]{0,127}$" using '
            '`dsl.pipeline(name=...)` decorator.' % name)


def is_v2_component(op: _container_op.ContainerOp) -> bool:
    """Determines whether a component is a KFP v2 component."""

    # TODO: migrate v2 component to PipelineTask
    if not isinstance(op, _container_op.ContainerOp):
        return False

    component_spec = op._metadata
    return (component_spec and component_spec.metadata and
            component_spec.metadata.annotations and
            component_spec.metadata.annotations.get(
                _component_builder.V2_COMPONENT_ANNOTATION) == 'true')


def refactor_v2_container_spec(container_spec: PipelineContainerSpec) -> None:
    """Refactor the container spec for a v2 component."""
    if not '--function_name' in container_spec.args:
        raise RuntimeError(
            'V2 component is expected to have function_name as a '
            'command line arg.')
    fn_name_idx = list(container_spec.args).index('--function_name') + 1
    fn_name = container_spec.args[fn_name_idx]
    container_spec.ClearField('command')
    container_spec.ClearField('args')
    container_spec.command.extend(['python', '-m', 'kfp.container.entrypoint'])
    container_spec.args.extend(
        ['--executor_input_str', '{{$}}', '--function_name', fn_name])


def make_retry_policy_proto(
    max_retry_count: Optional[int] = None,
    backoff_duration: Optional[str] = None,
    backoff_factor: Optional[float] = None,
    backoff_max_duration: Optional[str] = None
) -> pipeline_spec_pb2.PipelineTaskSpec.RetryPolicy:
    """Makes a retry policy proto from constituent parts. Applies defaults and
    value caps so that IR reflects true behavior.

    Args:
        max_retry_count (Optional[int], optional): Number of retries before
            considering a task as failed. Defaults to 0, or no retries.
        backoff_duration (Optional[str], optional): The time interval between
            retries. Defaults to '0s', or immediate retry.
        backoff_factor (Optional[float], optional): The exponential backoff factor
            applied to backoff_duration. Defaults to 2.0.
        backoff_max_duration (Optional[str], optional): The maximum duration
            during which the task will be retried according to the backoff strategy. Max allowed is 1 hour - higher value will be capped to this limit. If unspecified, will set to 1 hour. Defaults to '3600s'.

    Returns:
        pipeline_spec_pb2.PipelineTaskSpec.RetryPolicy: RetryPolicy proto message.
    """
    max_retry_count = max_retry_count or 0
    backoff_duration = backoff_duration or '0s'
    backoff_factor = backoff_factor or 2.0
    backoff_max_duration = backoff_max_duration or '3600s'

    backoff_duration_seconds = f'{convert_duration_to_seconds(backoff_duration)}s'
    backoff_max_duration_seconds = f'{min(convert_duration_to_seconds(backoff_max_duration), 3600)}s'

    return json_format.ParseDict(
        {
            'max_retry_count': max_retry_count,
            'backoff_duration': backoff_duration_seconds,
            'backoff_factor': backoff_factor,
            'backoff_max_duration': backoff_max_duration_seconds,
        }, pipeline_spec_pb2.PipelineTaskSpec.RetryPolicy())


def normalize_time_string(duration: str) -> str:
    """Normalizes a time string.
        Examples:
            - '1 hour' -> '1h'
            - '2 hours' -> '2h'
            - '2hours' -> '2h'
            - '2 w' -> '2w'
            - '2w' -> '2w'

    Args:
        duration (str): The unnormalized duration string.

    Returns:
        str: The normalized duration string.
    """
    no_ws_duration = duration.replace(' ', '')
    duration_split = [el for el in re.split(r'(\D+)', no_ws_duration) if el]

    if len(duration_split) != 2:
        raise ValueError(
            f"Invalid duration string: '{duration}'. Expected one value (as integer in string) and one unit, such as '1 hour'."
        )

    value = duration_split[0]
    unit = duration_split[1]

    first_letter_of_unit = unit[0]
    return value + first_letter_of_unit


def convert_duration_to_seconds(duration: str) -> int:
    """Converts a duration string to seconds.

    Args:
        duration (str): The unnormalized duration string. (e.g. '1h', '1 hour', '2
            hours', '2w', '2 weeks', '2d', etc.)

    Raises:
        ValueError: If the time unit is not one of seconds, minutes, hours, days,
            or weeks.

    Returns:
        int: The number of seconds in the duration.
    """
    duration = normalize_time_string(duration)
    seconds_per_unit = {'s': 1, 'm': 60, 'h': 3_600, 'd': 86_400, 'w': 604_800}
    if duration[-1] not in seconds_per_unit.keys():
        raise ValueError(
            f"Unsupported duration unit: '{duration[-1]}' for '{duration}'.")
    return int(duration[:-1]) * seconds_per_unit[duration[-1]]


class NoOpWarning(UserWarning):
    pass
