# Copyright 2019 Google LLC
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

import re
import warnings
import yaml
from collections import OrderedDict
from typing import Union, List, Any, Callable, TypeVar, Dict

from ._k8s_helper import K8sHelper
from .. import dsl
from ..dsl._container_op import BaseOp
from ..dsl._artifact_location import ArtifactLocation

# generics
T = TypeVar('T')


def _process_obj(obj: Any, map_to_tmpl_var: dict):
    """Recursively sanitize and replace any PipelineParam (instances and serialized strings)
    in the object with the corresponding template variables
    (i.e. '{{inputs.parameters.<PipelineParam.full_name>}}').
    
    Args:
      obj: any obj that may have PipelineParam
      map_to_tmpl_var: a dict that maps an unsanitized pipeline
                       params signature into a template var
    """
    # serialized str might be unsanitized
    if isinstance(obj, str):
        # get signature
        param_tuples = dsl.match_serialized_pipelineparam(obj)
        if not param_tuples:
            return obj
        # replace all unsanitized signature with template var
        for param_tuple in param_tuples:
            obj = re.sub(param_tuple.pattern, map_to_tmpl_var[param_tuple.pattern], obj)

    # list
    if isinstance(obj, list):
        return [_process_obj(item, map_to_tmpl_var) for item in obj]

    # tuple
    if isinstance(obj, tuple):
        return tuple((_process_obj(item, map_to_tmpl_var) for item in obj))

    # dict
    if isinstance(obj, dict):
        return {
            key: _process_obj(value, map_to_tmpl_var)
            for key, value in obj.items()
        }

    # pipelineparam
    if isinstance(obj, dsl.PipelineParam):
        # if not found in unsanitized map, then likely to be sanitized
        return map_to_tmpl_var.get(str(obj), '{{inputs.parameters.%s}}' % obj.full_name)

    # k8s objects (generated from swaggercodegen)
    if hasattr(obj, 'swagger_types') and isinstance(obj.swagger_types, dict):
        # process everything inside recursively
        for key in obj.swagger_types.keys():
            setattr(obj, key, _process_obj(getattr(obj, key), map_to_tmpl_var))
        # return json representation of the k8s obj
        return K8sHelper.convert_k8s_obj_to_json(obj)

    # k8s objects (generated from openapi)
    if hasattr(obj, 'openapi_types') and isinstance(obj.openapi_types, dict):
        # process everything inside recursively
        for key in obj.openapi_types.keys():
            setattr(obj, key, _process_obj(getattr(obj, key), map_to_tmpl_var))
        # return json representation of the k8s obj
        return K8sHelper.convert_k8s_obj_to_json(obj)

    # do nothing
    return obj


def _process_base_ops(op: BaseOp):
    """Recursively go through the attrs listed in `attrs_with_pipelineparams`
    and sanitize and replace pipeline params with template var string.

    Returns a processed `BaseOp`.

    NOTE this is an in-place update to `BaseOp`'s attributes (i.e. the ones
    specified in `attrs_with_pipelineparams`, all `PipelineParam` are replaced
    with the corresponding template variable strings).

    Args:
        op {BaseOp}: class that inherits from BaseOp

    Returns:
        BaseOp
    """

    # map param's (unsanitized pattern or serialized str pattern) -> input param var str
    map_to_tmpl_var = {
        (param.pattern or str(param)): '{{inputs.parameters.%s}}' % param.full_name
        for param in op.inputs
    }

    # process all attr with pipelineParams except inputs and outputs parameters
    for key in op.attrs_with_pipelineparams:
        setattr(op, key, _process_obj(getattr(op, key), map_to_tmpl_var))

    return op


def _parameters_to_json(params: List[dsl.PipelineParam]):
    """Converts a list of PipelineParam into an argo `parameter` JSON obj."""
    _to_json = (lambda param: dict(name=param.full_name, value=param.value)
                if param.value else dict(name=param.full_name))
    params = [_to_json(param) for param in params]
    # Sort to make the results deterministic.
    params.sort(key=lambda x: x['name'])
    return params


# TODO: artifacts?
def _inputs_to_json(inputs_params: List[dsl.PipelineParam], _artifacts=None):
    """Converts a list of PipelineParam into an argo `inputs` JSON obj."""
    parameters = _parameters_to_json(inputs_params)
    return {'parameters': parameters} if parameters else None


def _outputs_to_json(op: BaseOp,
                     outputs: Dict[str, dsl.PipelineParam],
                     param_outputs: Dict[str, str],
                     output_artifacts: List[dict]):
    """Creates an argo `outputs` JSON obj."""
    if isinstance(op, dsl.ResourceOp):
        value_from_key = "jsonPath"
    else:
        value_from_key = "path"
    output_parameters = []
    for param in outputs.values():
        output_parameters.append({
            'name': param.full_name,
            'valueFrom': {
                value_from_key: param_outputs[param.name]
            }
        })
    output_parameters.sort(key=lambda x: x['name'])
    ret = {}
    if output_parameters:
        ret['parameters'] = output_parameters
    if output_artifacts:
        ret['artifacts'] = output_artifacts

    return ret


# TODO: generate argo python classes from swagger and use convert_k8s_obj_to_json??
def _op_to_template(op: BaseOp):
    """Generate template given an operator inherited from BaseOp."""

    # NOTE in-place update to BaseOp
    # replace all PipelineParams with template var strings
    processed_op = _process_base_ops(op)

    if isinstance(op, dsl.ContainerOp):
        # default output artifacts
        output_artifact_paths = OrderedDict(op.output_artifact_paths)
        output_artifact_paths.setdefault('mlpipeline-ui-metadata', '/mlpipeline-ui-metadata.json')
        output_artifact_paths.setdefault('mlpipeline-metrics', '/mlpipeline-metrics.json')

        output_artifacts = [
             K8sHelper.convert_k8s_obj_to_json(
                 ArtifactLocation.create_artifact_for_s3(
                     op.artifact_location, 
                     name=name, 
                     path=path, 
                     key='runs/{{workflow.uid}}/{{pod.name}}/' + name + '.tgz'))
            for name, path in output_artifact_paths.items()
        ]

        for output_artifact in output_artifacts:
            if output_artifact['name'] in ['mlpipeline-ui-metadata', 'mlpipeline-metrics']:
                output_artifact['optional'] = True

        # workflow template
        template = {
            'name': processed_op.name,
            'container': K8sHelper.convert_k8s_obj_to_json(
                processed_op.container
            )
        }
    elif isinstance(op, dsl.ResourceOp):
        # no output artifacts
        output_artifacts = []

        # workflow template
        processed_op.resource["manifest"] = yaml.dump(
            K8sHelper.convert_k8s_obj_to_json(processed_op.k8s_resource),
            default_flow_style=False
        )
        template = {
            'name': processed_op.name,
            'resource': K8sHelper.convert_k8s_obj_to_json(
                processed_op.resource
            )
        }

    # inputs
    inputs = _inputs_to_json(processed_op.inputs)
    if inputs:
        template['inputs'] = inputs

    # outputs
    if isinstance(op, dsl.ContainerOp):
        param_outputs = processed_op.file_outputs
    elif isinstance(op, dsl.ResourceOp):
        param_outputs = processed_op.attribute_outputs
    template['outputs'] = _outputs_to_json(op, processed_op.outputs,
                                           param_outputs, output_artifacts)

    # node selector
    if processed_op.node_selector:
        template['nodeSelector'] = processed_op.node_selector

    # tolerations
    if processed_op.tolerations:
        template['tolerations'] = processed_op.tolerations

    # affinity
    if processed_op.affinity:
        template['affinity'] = K8sHelper.convert_k8s_obj_to_json(processed_op.affinity)

    # metadata
    if processed_op.pod_annotations or processed_op.pod_labels:
        template['metadata'] = {}
        if processed_op.pod_annotations:
            template['metadata']['annotations'] = processed_op.pod_annotations
        if processed_op.pod_labels:
            template['metadata']['labels'] = processed_op.pod_labels
    # retries
    if processed_op.num_retries:
        template['retryStrategy'] = {'limit': processed_op.num_retries}

    # timeout
    if processed_op.timeout:
        template['activeDeadlineSeconds'] = processed_op.timeout

    # initContainers
    if processed_op.init_containers:
        template['initContainers'] = processed_op.init_containers

    # sidecars
    if processed_op.sidecars:
        template['sidecars'] = processed_op.sidecars

    # Display name
    if processed_op.display_name:
        template.setdefault('metadata', {}).setdefault('annotations', {})['pipelines.kubeflow.org/task_display_name'] = processed_op.display_name

    return template
