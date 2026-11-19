# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Portable container component definitions, not pipeline execution IR.

The shared schema remains useful for component YAML and Kubernetes argument
bindings. This serializer never builds a DAG, registers tasks or selects a backend.
"""

from google.protobuf import json_format, struct_pb2

import kfp
from kfp.dsl import placeholders, utils
from kfp.dsl.types import type_utils
from kfp.pipeline_spec import pipeline_spec_pb2 as pb


def to_protobuf_value(value):
    return json_format.ParseDict(value, struct_pb2.Value())


def serialize(component):
    container = component.implementation.container
    if container is None:
        raise ValueError(
            'Only container component definitions can be exported; use Compiler.compile for native pipeline resources.'
        )
    spec = pb.PipelineSpec()
    spec.pipeline_info.name = component.name
    spec.pipeline_info.description = component.description or ''
    spec.schema_version = '2.1.0'
    spec.sdk_version = f'kfp-{kfp.__version__}'
    for direction, ports in [
        ('input', component.inputs),
        ('output', component.outputs),
    ]:
        definitions = getattr(spec.root, direction + '_definitions')
        for name, port in (ports or {}).items():
            if type_utils.is_parameter_type(port.type):
                declaration = definitions.parameters[name]
                special = (
                    pb.ParameterType.TASK_FINAL_STATUS
                    if type_utils.is_task_final_status_type(port.type)
                    else pb.ParameterType.TASK_CONFIG
                    if type_utils.is_task_config_type(port.type)
                    else None
                )
                declaration.parameter_type = (
                    special or type_utils.get_parameter_type(port.type)
                )
                if direction == 'input':
                    declaration.is_optional = (
                        port.optional or special is not None
                    )
                    if port.optional and port.default is not None:
                        declaration.default_value.CopyFrom(
                            to_protobuf_value(port.default)
                        )
            else:
                declaration = definitions.artifacts[name]
                declaration.artifact_type.CopyFrom(
                    type_utils.bundled_artifact_to_artifact_proto(port.type)
                )
                declaration.is_artifact_list = port.is_artifact_list
                if direction == 'input':
                    declaration.is_optional = port.optional
            declaration.description = port.description or ''
    for passthrough in component.task_config_passthroughs or []:
        spec.root.task_config_passthroughs.append(passthrough.to_proto())
    label = utils.sanitize_executor_label(component.name)
    spec.root.executor_label = label
    convert = placeholders.convert_command_line_element_to_string
    implementation = {'image': container.image}
    for field in ('command', 'args'):
        if getattr(container, field):
            implementation[field] = [
                convert(value) for value in getattr(container, field)
            ]
    if container.env:
        implementation['env'] = [
            {'name': name, 'value': convert(value)}
            for name, value in container.env.items()
        ]
    spec.deployment_spec.update(
        {'executors': {label: {'container': implementation}}}
    )
    return spec
