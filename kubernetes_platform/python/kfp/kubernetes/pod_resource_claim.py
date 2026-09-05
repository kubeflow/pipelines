# Copyright 2026 The Kubeflow Authors
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

from typing import List, Union

from google.protobuf import json_format
from kfp.dsl import pipeline_channel
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


class ResourceClaimConfig(dict):
    """Typed configuration for a DRA resource claim.

    Requires Kubernetes 1.31 or later with the ``DynamicResourceAllocation``
    feature gate enabled (GA and enabled by default in Kubernetes 1.34), an
    installed DRA driver, and a compatible KFP backend from the same release as
    the SDK support.

    Args:
        resource_claim_template_name: Name of a ResourceClaimTemplate in the
            same namespace.
    """

    def __init__(self, resource_claim_template_name: str):
        if not resource_claim_template_name:
            raise ValueError(
                'resource_claim_template_name must be a non-empty string.')
        super().__init__(
            resourceClaimTemplateName=resource_claim_template_name,)

    @property
    def resource_claim_template_name(self) -> str:
        return self['resourceClaimTemplateName']

    def to_dict(self) -> dict:
        return dict(self)


def add_resource_claim(
    task: PipelineTask,
    resource_claim_template_name: str,
) -> PipelineTask:
    """Add a DRA resource claim to a task.

    Requires Kubernetes 1.31 or later with the ``DynamicResourceAllocation``
    feature gate enabled (GA and enabled by default in Kubernetes 1.34), an
    installed DRA driver, and a compatible KFP backend from the same release as
    the SDK support. The referenced ``ResourceClaimTemplate`` must be in the
    task's namespace.

    Args:
        task: Pipeline task.
        resource_claim_template_name: Name of a ResourceClaimTemplate in the
            same namespace. Must be non-empty.

    Returns:
        Task object with added resource claim.
    """
    if not resource_claim_template_name:
        raise ValueError(
            'resource_claim_template_name must be a non-empty string.')

    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.pod_resource_claims.append(
        pb.PodResourceClaim(
            resource_claim_template_name=resource_claim_template_name,
        ))
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task


def add_resource_claim_json(
    task: PipelineTask,
    resource_claim_json: Union[pipeline_channel.PipelineParameterChannel,
                               'ResourceClaimConfig',
                               List['ResourceClaimConfig']],
) -> PipelineTask:
    """Add a DRA resource claim in JSON form to a task.

    Requires Kubernetes 1.31 or later with the ``DynamicResourceAllocation``
    feature gate enabled (GA and enabled by default in Kubernetes 1.34), an
    installed DRA driver, and a compatible KFP backend from the same release as
    the SDK support. Referenced ``ResourceClaimTemplate`` objects must be in the
    task's namespace.

    Args:
        task: Pipeline task.
        resource_claim_json: A pipeline parameter, a ResourceClaimConfig, or a
            list of ResourceClaimConfig instances.

    Returns:
        Task object with added resource claim.
    """
    if isinstance(resource_claim_json,
                  pipeline_channel.PipelineParameterChannel):
        msg = common.get_existing_kubernetes_config_as_message(task)
        claim = pb.PodResourceClaim()
        claim.resource_claim_json.CopyFrom(
            common.parse_k8s_parameter_input(resource_claim_json, task))
        msg.pod_resource_claims.append(claim)
        task.platform_config['kubernetes'] = json_format.MessageToDict(msg)
    elif isinstance(resource_claim_json, list):
        for config in resource_claim_json:
            if not isinstance(config, ResourceClaimConfig):
                raise ValueError(
                    'Each element in resource_claim_json list must be a '
                    'ResourceClaimConfig instance.')
        msg = common.get_existing_kubernetes_config_as_message(task)
        for config in resource_claim_json:
            claim = pb.PodResourceClaim()
            claim.resource_claim_json.CopyFrom(
                common.parse_k8s_parameter_input(config.to_dict(), task))
            msg.pod_resource_claims.append(claim)
        task.platform_config['kubernetes'] = json_format.MessageToDict(msg)
    elif isinstance(resource_claim_json, ResourceClaimConfig):
        _add_resource_claim_config(task, resource_claim_json)
    else:
        raise ValueError(
            'resource_claim_json must be a PipelineParameterChannel, '
            'ResourceClaimConfig, or list of ResourceClaimConfig.')

    return task


def _add_resource_claim_config(task: PipelineTask,
                               config: ResourceClaimConfig):
    msg = common.get_existing_kubernetes_config_as_message(task)
    claim = pb.PodResourceClaim()
    claim.resource_claim_json.CopyFrom(
        common.parse_k8s_parameter_input(config.to_dict(), task))
    msg.pod_resource_claims.append(claim)
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)
