from kfp import dsl
from kfp import kubernetes
from kubernetes import client as k8s


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
    )

    task2 = comp().after(task1)
    kubernetes.use_pvc(task2, pvc_name=pvc_name)


# secret -- object
import kubernetes.client as k8s


@dsl.pipeline
def pipeline():
    task = comp()
    kubernetes.use_secret_as_env(task,
                                 secret_env_source=k8s.V1SecretEnvSource(
                                     name='my-secret', optional=False))

    kubernetes.use_secret_as_volume(
        task,
        volume=k8s.V1Volume(
            name='my-volume',
            secret=k8s.V1SecretVolumeSource(secret_name='my-secret')),
        volume_mount=k8s.V1VolumeMount(name='my-volume',
                                       mount_path='/mnt/my_vol'))


# volume -- object
from kubernetes.client import V1PersistentVolumeClaim, V1ObjectMeta, V1PersistentVolumeClaimSpec


@dsl.pipeline
def pipeline():
    task1 = comp()
    pvc = kubernetes.use_pvc(
        task1,
        pvc=V1PersistentVolumeClaim(
            api_version='v1',
            kind='PersistentVolumeClaim',
            metadata=V1ObjectMeta(generate_name='my-pvc-prefix-'),
            spec=V1PersistentVolumeClaimSpec(
                access_modes=['ReadWriteMany'],
                resources='1Gi',
                storage_class_name='standard',
            ),
        ))

    task2 = comp().after(task1)
    kubernetes.use_pvc(task2, pvc=pvc)


# kubernetes.py -- object
def use_secret_as_env(task: PipelineTask,
                      secret_env_source: V1SecretEnvSource) -> None:
    ...


def use_secret_as_volume(task: PipelineTask,
                         volume=V1Volume,
                         volume_mount=V1VolumeMount) -> None:
    ...


def use_pvc(task: PipelineTask,
            pvc=V1PersistentVolumeClaim) -> V1PersistentVolumeClaim:
    """Mounts a PVC onto task's container. Dynamically provisions PVC if it does not exist. Else, uses the existing PVC."""
    ...
