"""Definition for TaskConfig."""

import dataclasses
from enum import IntEnum

from kfp.pipeline_spec import pipeline_spec_pb2


class TaskConfigField(IntEnum):
    # Indicates that the resource limits and requests should be passed through to the external workload.
    # Be cautious about also setting apply_to_task=True since that will double the resources required
    # for the task.
    RESOURCES = (
        pipeline_spec_pb2.TaskConfigPassthroughType
        .TaskConfigPassthroughTypeEnum.RESOURCES)
    # Indicates that the environment variables should be passed through to the external workload.
    # It is generally safe to always set apply_to_task=True on this field.
    ENV = (
        pipeline_spec_pb2.TaskConfigPassthroughType
        .TaskConfigPassthroughTypeEnum.ENV)
    # Indicates that the Kubernetes node affinity should be passed through to the external workload.
    KUBERNETES_AFFINITY = (
        pipeline_spec_pb2.TaskConfigPassthroughType
        .TaskConfigPassthroughTypeEnum.KUBERNETES_AFFINITY)
    # Indicates that the Kubernetes node tolerations should be passed through to the external workload.
    KUBERNETES_TOLERATIONS = (
        pipeline_spec_pb2.TaskConfigPassthroughType
        .TaskConfigPassthroughTypeEnum.KUBERNETES_TOLERATIONS)
    # Indicates that the Kubernetes node selector should be passed through to the external workload.
    KUBERNETES_NODE_SELECTOR = (
        pipeline_spec_pb2.TaskConfigPassthroughType
        .TaskConfigPassthroughTypeEnum.KUBERNETES_NODE_SELECTOR)
    # Indicates that the Kubernetes persistent volumes and ConfigMaps/Secrets mounted as volumes should be passed
    # through to the external workload. Be sure that when setting apply_to_task=True, the volumes are ReadWriteMany or ReadOnlyMany or else
    # the task's pod may not start.
    # This is useful when the task prepares a shared volume for the external workload or defines output artifact
    # (e.g. dsl.Model) that is created by the external workload.
    KUBERNETES_VOLUMES = (
        pipeline_spec_pb2.TaskConfigPassthroughType
        .TaskConfigPassthroughTypeEnum.KUBERNETES_VOLUMES)


@dataclasses.dataclass
class TaskConfigPassthrough:
    field: TaskConfigField
    apply_to_task: bool = False

    def to_proto(self) -> pipeline_spec_pb2.TaskConfigPassthrough:
        """Converts this object to its proto representation."""
        proto = pipeline_spec_pb2.TaskConfigPassthrough()
        proto.field = int(self.field)
        proto.apply_to_task = self.apply_to_task
        return proto
