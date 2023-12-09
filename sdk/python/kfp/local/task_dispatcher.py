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
"""Code for dispatching a local task execution."""
from typing import Any, Dict

from google.protobuf import json_format
from kfp import local
from kfp.dsl import structures
from kfp.dsl.types import type_utils
from kfp.local import config
from kfp.pipeline_spec import pipeline_spec_pb2


def run_single_component(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    arguments: Dict[str, Any],
) -> Dict[str, Any]:
    """Runs a single component from its compiled PipelineSpec.

    Args:
        pipeline_spec: The PipelineSpec of the component to run.
        arguments: The runtime arguments.

    Returns:
        A LocalTask instance.
    """
    if config.LocalExecutionConfig.instance is None:
        raise RuntimeError(
            f'You must initiatize the local execution session using {local.__name__}.{local.init.__name__}().'
        )

    return _run_single_component_implementation(
        pipeline_spec=pipeline_spec,
        arguments=arguments,
        pipeline_root=config.LocalExecutionConfig.instance.pipeline_root,
        runner=config.LocalExecutionConfig.instance.runner,
    )


def _run_single_component_implementation(
    pipeline_spec: pipeline_spec_pb2.PipelineSpec,
    arguments: Dict[str, Any],
    pipeline_root: str,
    runner: config.LocalRunnerType,
) -> Dict[str, Any]:
    """The implementation of a single component runner."""

    if len(pipeline_spec.components) != 1:
        raise NotImplementedError(
            'Local pipeline execution is not currently supported.')

    component_name, component_spec = list(pipeline_spec.components.items())[0]

    if component_spec.input_definitions != pipeline_spec.root.input_definitions or component_spec.output_definitions != pipeline_spec.root.output_definitions:
        raise NotImplementedError(
            'Local pipeline execution is not currently supported.')
    # include the input artifact constant check above the validate_arguments check
    # for a better error message
    # we also perform this check further downstream in the executor input
    # construction utilities, since that's where the logic needs to be
    # implemented as well
    validate_arguments(
        arguments=arguments,
        component_spec=component_spec,
        component_name=component_name,
    )
    return {}


def validate_arguments(
    arguments: Dict[str, Any],
    component_spec: pipeline_spec_pb2.ComponentSpec,
    component_name: str,
) -> None:
    """Validates arguments provided for the execution of component_spec."""

    input_specs = {}
    for artifact_name, artifact_spec in component_spec.input_definitions.artifacts.items(
    ):
        input_specs[
            artifact_name] = structures.InputSpec.from_ir_component_inputs_dict(
                json_format.MessageToDict(artifact_spec))

    for parameter_name, parameter_spec in component_spec.input_definitions.parameters.items(
    ):
        input_specs[
            parameter_name] = structures.InputSpec.from_ir_component_inputs_dict(
                json_format.MessageToDict(parameter_spec))

    for input_name, argument_value in arguments.items():
        if input_name in input_specs:
            type_utils.verify_type_compatibility(
                given_value=argument_value,
                expected_spec=input_specs[input_name],
                error_message_prefix=(
                    f'Incompatible argument passed to the input '
                    f'{input_name!r} of component {component_name!r}: '),
            )
        else:
            raise ValueError(
                f'Component {component_name!r} got an unexpected input:'
                f' {input_name!r}.')
