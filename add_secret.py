from typing import Dict, List, Optional
from kfp import dsl
from kfp import kubernetes


# secret -- convenience
@dsl.pipeline
def pipeline():
    task = comp()
    kubernetes.use_secret_as_env(task,
                                 secret_name='my-secret',
                                 secret_key_to_env={'password', 'SECRET_VAR'})

    kubernetes.use_secret_as_volume(task,
                                    secret_name='my-secret',
                                    mount_path='/mtn/my_vol')


# volume -- convenience
@dsl.pipeline
def pipeline():
    task1 = comp()
    pvc_name = kubernetes.use_pvc(
        task1,
        kind='PersistentVolumeClaim',
        generate_name='my-pvc-prefix-',
        access_modes=['ReadWriteMany'],
        resources='1Gi',
        storage_class_name='standard',
        mount_path='/data',
    )

    task2 = comp().after(task1)
    kubernetes.use_pvc(task2, pvc_name=pvc_name)


# secret -- object
## as env
from kubernetes.client import V1SecretEnvSource

env_source = V1SecretEnvSource(
    name='my-secret',
    optional=False,
)


@dsl.pipeline
def pipeline():
    task = comp()

    kubernetes.use_secret_as_env(
        task,
        secret_env_source=env_source,
    )


## as vol
from kubernetes.client import V1SecretVolumeSource, V1VolumeMount, V1Volume

volume = V1Volume(name='my-volume',
                  secret=V1SecretVolumeSource(secret_name='my-secret'))
volume_mount = V1VolumeMount(name='my-volume', mount_path='/mnt/my_vol')


@dsl.pipeline
def pipeline():
    task = comp()
    kubernetes.use_secret_as_volume(
        task,
        volume=volume,
        volume_mount=volume_mount,
    )


# volume -- object
from kubernetes.client import V1PersistentVolumeClaim, V1ObjectMeta, V1PersistentVolumeClaimSpec, V1Volume, V1VolumeMount

pvc_spec = V1PersistentVolumeClaimSpec(
    access_modes=['ReadWriteMany'],
    resources='1Gi',
    storage_class_name='standard',
)
pvc = V1PersistentVolumeClaim(
    api_version='v1',
    kind='PersistentVolumeClaim',
    metadata=V1ObjectMeta(generate_name='my-pvc-prefix-'),
    spec=pvc_spec,
)
volume = V1Volume(
    name='my-volume',
    claim_name=...,
)
volume_mount = V1VolumeMount(
    name='my-volume',
    mount_path='/data',
)


@dsl.pipeline
def pipeline():
    task1 = comp()
    pvc = kubernetes.mount_pvc(
        task1,
        volume=volume,
        volume_mount=volume_mount,
    )

    task2 = comp().after(task1)
    kubernetes.mount_pvc(task2, pvc=pvc)


# kubernetes.py -- convenience
def use_secret_as_env(
    task: PipelineTask,
    secret_name: str,
    secret_key_to_env: Dict[str, str],
) -> None:
    ...


def use_secret_as_volume(
    task: PipelineTask,
    secret_name: str,
    mount_path: str,
) -> None:
    ...


def mount_pvc(
    task: PipelineTask,
    generate_name: str,
    access_modes: List[str],
    size: str,
    mount_path: str,
    storage_class: Optional[str] = None,
    volume_name: Optional[str] = None,
) -> None:
    """If storage_class is provided, volume_name cannot be provided. If volume_name is provided, storage_class must equal '' (empty string) or it will be set to the default storage class by Kubernetes."""
    ...


# kubernetes.py -- object
def use_secret_as_env(
    task: PipelineTask,
    secret_env_source: V1SecretEnvSource,
) -> None:
    ...


def use_secret_as_volume(
    task: PipelineTask,
    volume=V1Volume,
    volume_mount=V1VolumeMount,
) -> None:
    ...


def mount_pvc(
    task: PipelineTask,
    volume=V1Volume,
    volume_mount=V1VolumeMount,
) -> None:
    """Mounts a PVC onto task's container. Dynamically provisions PVC if it does not exist. Else, uses the existing PVC."""
    ...
