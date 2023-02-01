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
    """If volume_name is provided, a pre-existing PV will be used."""
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
    ...
    """Mounts a PVC onto task's container. Dynamically provisions PVC if it does not exist. Else, uses the existing PVC."""
    ...


# -------- V1 -----
# volume
import kfp.dsl as dsl


@dsl.pipeline(name='volume-pipeline')
def volume_pipeline():
    # create PVC and dynamically provision volume
    create_pvc_task = dsl.VolumeOp(
        name='create_volume',
        generate_unique_name=True,
        resource_name='pvc1',
        size='1Gi',
        storage_class='standard',
        modes=dsl.VOLUME_MODE_RWM,
    )

    # use PVC; write to file
    task_a = dsl.ContainerOp(
        name='step1_ingest',
        image='alpine',
        command=['sh', '-c'],
        arguments=[
            'mkdir /data/step1 && '
            'echo hello > /data/step1/file1.txt'
        ],
        pvolumes={'/data': create_pvc_task.volume},
    )

    # create a PVC from a pre-existing volume
    create_pvc_from_existing_vol = dsl.VolumeOp(
        name='pre_existing_volume',
        generate_unique_name=True,
        resource_name='pvc2',
        size='5Gi',
        storage_class='standard',
        volume_name='my-pre-existing-volume',
        modes=dsl.VOLUME_MODE_RWM,
    )

    # use the previous task's PVC and the newly created PVC
    task_b = dsl.ContainerOp(
        name='step2_gunzip',
        image='library/bash:4.4.23',
        command=['sh', '-c'],
        arguments=['cat /data/step1/file.txt'],
        pvolumes={
            '/data': task_a.pvolume,
            '/other_data': create_pvc_from_existing_vol.volume
        },
    )


# if __name__ == '__main__':
#     import datetime
#     import warnings
#     import webbrowser

#     from kfp import compiler

#     warnings.filterwarnings('ignore')
#     ir_file = __file__.replace('.py', '.yaml')
#     compiler.Compiler().compile(pipeline_func=volume_pipeline,
#                                 package_path=ir_file)
#     # pipeline_name = __file__.split('/')[-1].replace('_',
#     #                                                 '-').replace('.py', '')
#     # display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
#     # job_id = f'{pipeline_name}-{display_name}'
#     # endpoint = 'https://75167a6cffcb723c-dot-us-central1.pipelines.googleusercontent.com'
#     # kfp_client = Client(host=endpoint)
#     # run = kfp_client.create_run_from_pipeline_package(ir_file)
#     # url = f'{endpoint}/#/runs/details/{run.run_id}'
#     # print(url)
#     # webbrowser.open_new_tab(url)