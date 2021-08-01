from typing import Dict
from kfp import dsl


def mount_pvc(
    pvc_name='pipeline-claim',
    volume_name='pipeline',
    volume_mount_path='/mnt/pipeline'
):
    """Modifier function to apply to a Container Op to simplify volume, volume mount addition and
    enable better reuse of volumes, volume claims across container ops.

    Example:
        ::

            train = train_op(...)
            train.apply(mount_pvc('claim-name', 'pipeline', '/mnt/pipeline'))
    """

    def _mount_pvc(task):
        from kubernetes import client as k8s_client
        # there can be other ops in a pipeline (e.g. ResourceOp, VolumeOp)
        # refer to #3906
        if not hasattr(task, "add_volume") or not hasattr(task,
                                                          "add_volume_mount"):
            return task
        local_pvc = k8s_client.V1PersistentVolumeClaimVolumeSource(
            claim_name=pvc_name
        )
        return (
            task.add_volume(
                k8s_client.V1Volume(
                    name=volume_name, persistent_volume_claim=local_pvc
                )
            ).add_volume_mount(
                k8s_client.V1VolumeMount(
                    mount_path=volume_mount_path, name=volume_name
                )
            )
        )

    return _mount_pvc


def use_k8s_secret(
    secret_name: str = 'k8s-secret', k8s_secret_key_to_env: Dict = {}
):
    """An operator that configures the container to use k8s credentials.

    k8s_secret_key_to_env specifies a mapping from the name of the keys in the k8s secret to the name of the
    environment variables where the values will be added.

    The secret needs to be deployed manually a priori.

    Example:
        ::

            train = train_op(...)
            train.apply(use_k8s_secret(secret_name='s3-secret',
            k8s_secret_key_to_env={'secret_key': 'AWS_SECRET_ACCESS_KEY'}))

        This will load the value in secret 's3-secret' at key 'secret_key' and source it as the environment variable
        'AWS_SECRET_ACCESS_KEY'. I.e. it will produce the following section on the pod:
        env:
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: s3-secret
              key: secret_key
    """

    def _use_k8s_secret(task):
        from kubernetes import client as k8s_client
        for secret_key, env_var in k8s_secret_key_to_env.items():
            task.container \
                .add_env_variable(
                k8s_client.V1EnvVar(
                    name=env_var,
                    value_from=k8s_client.V1EnvVarSource(
                        secret_key_ref=k8s_client.V1SecretKeySelector(
                            name=secret_name,
                            key=secret_key
                        )
                    )
                )
            )
        return task

    return _use_k8s_secret


def add_default_resource_spec(
    memory_request: str = '512Mi',
    cpu_request: str = '0.5',
    memory_limit: str = None,
    cpu_limit: str = None
):
    """Add default resource requests & limits.

    Args:
      memory_request: memory request, defaults to 500Mi. Format can be 512Mi, 2Gi etc.
      cpu_request: cpu request, defaults to 0.5 vCPUs.
      memory_limit: optional, defaults to memory request.
      cpu_limit: optional, defaults to cpu request.
    """
    if not memory_limit:
        memory_limit = memory_request
    if not cpu_limit:
        cpu_limit = cpu_request

    def _add_default_resource_spec(task: dsl.ContainerOp):
        # Skip tasks which are not container ops.
        if not isinstance(task, dsl.ContainerOp):
            return task
        if not task.container.get_resource_request('cpu'):
            task.container.add_resource_request('cpu', cpu_request)
        if not task.container.get_resource_request('memory'):
            task.container.add_resource_request('memory', memory_request)
        if not task.container.get_resource_limit('cpu'):
            task.container.add_resource_limit('cpu', cpu_limit)
        if not task.container.get_resource_limit('memory'):
            task.container.add_resource_limit('memory', memory_limit)

    return _add_default_resource_spec
