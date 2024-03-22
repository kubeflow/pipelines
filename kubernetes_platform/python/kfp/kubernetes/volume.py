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

from typing import Dict, List, Optional, Union

from google.protobuf import json_format
from google.protobuf import message
from kfp import dsl
from kfp.dsl import PipelineTask
from kfp.kubernetes import common
from kfp.kubernetes import kubernetes_executor_config_pb2 as pb


@dsl.container_component
def CreatePVC(
    name: dsl.OutputPath(str),
    access_modes: List[str],
    size: str,
    pvc_name: Optional[str] = None,
    pvc_name_suffix: Optional[str] = None,
    storage_class_name: Optional[str] = '',
    volume_name: Optional[str] = None,
    annotations: Optional[Dict[str, str]] = None,
):
    """Create a PersistentVolumeClaim, which can be used by downstream tasks.
    See `PersistentVolume <https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistent-volumes>`_ and `PersistentVolumeClaim <https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims>`_ documentation for more information about
    the component input parameters.

    Args:
        access_modes: AccessModes to request for the provisioned PVC. May
            be one or more of ``'ReadWriteOnce'``, ``'ReadOnlyMany'``, ``'ReadWriteMany'``, or
            ``'ReadWriteOncePod'``. Corresponds to `PersistentVolumeClaim.spec.accessModes <https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes>`_.
        size: The size of storage requested by the PVC that will be provisioned. For example, ``'5Gi'``. Corresponds to `PersistentVolumeClaim.spec.resources.requests.storage <https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#PersistentVolumeClaimSpec>`_.
        pvc_name: Name of the PVC. Corresponds to `PersistentVolumeClaim.metadata.name <https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#PersistentVolumeClaim>`_. Only one of ``pvc_name`` and ``pvc_name_suffix`` can
            be provided.
        pvc_name_suffix: Prefix to use for a dynamically generated name, which
            will take the form ``<argo-workflow-name>-<pvc_name_suffix>``. Only one
            of ``pvc_name`` and ``pvc_name_suffix`` can be provided.
        storage_class_name: Name of StorageClass from which to provision the PV
            to back the PVC. ``None`` indicates to use the cluster's default
            storage_class_name. Set to ``''`` for a statically specified PVC.
        volume_name: Pre-existing PersistentVolume that should back the
            provisioned PersistentVolumeClaim. Used for statically
            specified PV only. Corresponds to `PersistentVolumeClaim.spec.volumeName <https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#PersistentVolumeClaimSpec>`_.
        annotations: Annotations for the PVC's metadata. Corresponds to `PersistentVolumeClaim.metadata.annotations <https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#PersistentVolumeClaim>`_.

    Returns:
        ``name: str`` \n\t\t\tName of the generated PVC.
    """

    return dsl.ContainerSpec(image='argostub/createpvc')


def mount_pvc(
    task: PipelineTask,
    pvc_name: Union[str, 'PipelineChannel'],
    mount_path: str,
) -> PipelineTask:
    """Mount a PersistentVolumeClaim to the task's container.

    Args:
        task: Pipeline task.
        pvc_name: Name of the PVC to mount. Supports passing a runtime-generated name, such as a name provided by ``kubernetes.CreatePvcOp().outputs['name']``.
        mount_path: Path to which the PVC should be mounted as a volume.

    Returns:
        Task object with updated PVC mount configuration.
    """

    msg = common.get_existing_kubernetes_config_as_message(task)

    pvc_mount = pb.PvcMount(mount_path=mount_path)
    pvc_name_from_task = _assign_pvc_name_to_msg(pvc_mount, pvc_name)
    if pvc_name_from_task:
        task.after(pvc_name.task)

    msg.pvc_mount.append(pvc_mount)
    task.platform_config['kubernetes'] = json_format.MessageToDict(msg)

    return task


@dsl.container_component
def DeletePVC(pvc_name: str):
    """Delete a PersistentVolumeClaim.

    Args:
        pvc_name: Name of the PVC to delete. Supports passing a runtime-generated name, such as a name provided by ``kubernetes.CreatePvcOp().outputs['name']``.
    """
    return dsl.ContainerSpec(image='argostub/deletepvc')


def _assign_pvc_name_to_msg(
    msg: message.Message,
    pvc_name: Union[str, 'PipelineChannel'],
) -> bool:
    """Assigns pvc_name to the msg's pvc_reference oneof. Returns True if pvc_name is an upstream task output. Else, returns False."""
    if isinstance(pvc_name, str):
        msg.constant = pvc_name
        return False
    elif hasattr(pvc_name, 'task_name'):
        if pvc_name.task_name is None:
            msg.component_input_parameter = pvc_name.name
            return False
        else:
            msg.task_output_parameter.producer_task = pvc_name.task_name
            msg.task_output_parameter.output_parameter_key = pvc_name.name
            return True
    else:
        raise ValueError(
            f'Argument for {"pvc_name"!r} must be an instance of str or PipelineChannel. Got unknown input type: {type(pvc_name)!r}. '
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
    """Add a `generic ephemeral volume
    <https://kubernetes.io/docs/concepts/storage/ephemeral-volumes/#generic-ephemeral-volumes>`_. to a task.

    Args:
        task:
            Pipeline task.
        volume_name:
            name to be given to the created ephemeral volume. Corresponds to Pod.spec.volumes[*].name
        mount_path:
            local path in the main container where the PVC should be mounted as a volume
        access_modes:
            AccessModes to request for the provisioned PVC. May be one or more of ``'ReadWriteOnce'``,
            ``'ReadOnlyMany'``, ``'ReadWriteMany'``, or``'ReadWriteOncePod'``. Corresponds to
            `Pod.spec.volumes[*].ephemeral.volumeClaimTemplate.spec.accessModes
            <https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes>`_.
        size:
            The size of storage requested by the PVC that will be provisioned. For example, ``'5Gi'``. Corresponds to
            `Pod.spec.volumes[*].ephemeral.volumeClaimTemplate.spec.resources.requests.storage
            <https://kubernetes.io/docs/reference/kubernetes-api/config-and-storage-resources/persistent-volume-claim-v1/#PersistentVolumeClaimSpec>`_.
        storage_class_name:
            Name of StorageClass from which to provision the PV to back the PVC. ``None`` indicates to use the
            cluster's default storage_class_name.
        labels:
            The labels to attach to the created PVC. Corresponds to
            `Pod.spec.volumes[*].ephemeral.volumeClaimTemplate.metadata.labels
        annotations:
            The annotation to attach to the created PVC. Corresponds to
            `Pod.spec.volumes[*].ephemeral.volumeClaimTemplate.metadata.annotations
    Returns:
        Task object with added toleration.
    """

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
                annotations=annotations or {},
                labels=labels or {},
            ) if annotations or labels else None,
        )
    )
    task.platform_config["kubernetes"] = json_format.MessageToDict(msg)

    return task