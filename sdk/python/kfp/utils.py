from typing import List

def use_secret(secret_name:List=None, volume_name:List=None, secret_volume_mount_path:List=None):

    """An operator that configures the container to use a secret.
       
       This assumes that the secret is created and availabel in the k8s cluster
       the user are expected to set the volume names in order to allow for multiple
       volumes to be mounted.
    """
    params = [secret_name, volume_name, secret_volume_mount_path]
    param_names = ["secret_name", "volume_name", "secret_volume_mount_path"]
    for param, param_name in params, param_names:
        if param is None:
            raise ValueError("'{}' needs to be specified, is: {}".format(param_name, param))
        if type(param) is not list:
            raise ValueError("Parameter {} needs to be a list".format(param_name))
    
    def _use_secret(task):
        from kubernetes import client as k8s_client
        task = task.add_volume(
            k8s_client.V1Volume(
                name=volume_name,
                secret=k8s_client.V1SecretVolumeSource(
                    secret_name=secret_name,
                )
            )
        )
        task.container \
            .add_volume_mount(
                    k8s_client.V1VolumeMount(
                        name=volume_name,
                        mount_path=secret_volume_mount_path,
                    )
                )
        return task
    
    return use_secret