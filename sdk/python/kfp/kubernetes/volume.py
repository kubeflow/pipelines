# Copyright 2023 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
"""Declarative volume bindings. PVC creation/deletion belongs to Kubernetes/Argo."""

from typing import Dict, List, Optional, Union

from google.protobuf import json_format, message

from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


def mount_pvc(
    task: PipelineTask,
    pvc_name: Union[str, 'PipelineChannel'],
    mount_path: str,
    sub_path: str = '',
) -> PipelineTask:
    """Mount an existing PVC. Native compilation currently requires a literal name."""
    msg = common.get_existing_kubernetes_config_as_message(task)
    mount = pb.PvcMount(mount_path=mount_path, sub_path=sub_path)
    mount.pvc_name_parameter.CopyFrom(
        common.parse_k8s_parameter_input(pvc_name, task)
    )
    if _assign_pvc_name_to_msg(mount, pvc_name):
        task.after(pvc_name.task)
    msg.pvc_mount.append(mount)
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)
    return task


def _assign_pvc_name_to_msg(
    msg: message.Message, pvc_name: Union[str, 'PipelineChannel']
) -> bool:
    if isinstance(pvc_name, str):
        msg.constant = pvc_name
        return False
    if hasattr(pvc_name, 'task_name'):
        if pvc_name.task_name is None:
            msg.component_input_parameter = pvc_name.name
            return False
        msg.task_output_parameter.producer_task = pvc_name.task_name
        msg.task_output_parameter.output_parameter_key = pvc_name.name
        return True
    raise ValueError(
        f'PVC name must be a string or pipeline channel, got {type(pvc_name)!r}.'
    )


def add_ephemeral_volume(
    task: PipelineTask,
    volume_name: str,
    mount_path: str,
    access_modes: List[str],
    size: str,
    storage_class_name: Optional[str] = None,
    labels: Dict[str, str] = None,
    annotations: Dict[str, str] = None,
):
    """Capture an ephemeral-volume declaration; native lowering is not yet supported."""
    if isinstance(access_modes, str):
        raise TypeError(
            f"'access_modes' must be a list of strings, got str: {access_modes!r}."
        )
    msg = common.get_existing_kubernetes_config_as_message(task)
    msg.generic_ephemeral_volume.append(
        pb.GenericEphemeralVolume(
            volume_name=volume_name,
            mount_path=mount_path,
            access_modes=access_modes,
            size=size,
            default_storage_class=storage_class_name is None,
            storage_class_name=storage_class_name,
            metadata=pb.PodMetadata(
                annotations=annotations or {}, labels=labels or {}
            )
            if annotations or labels
            else None,
        )
    )
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)
    return task
