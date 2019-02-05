
def use_local_volume(pvc_name='pipeline-claim', volume_name='pipeline', volume_mount_path='/mnt/pipeline'):
    """
        Modifier function to apply to a Container Op to simplify volume, volume mount addition and 
        enable better reuse of volumes, volume claims across container ops.
        Usage:
            train = train_op(...)
            train.apply(use_local_volume('claim-name', 'pipeline', '/mnt/pipeline'))
    """
    def _use_local_volume(task):
        from kubernetes import client as k8s_client
        local_pvc = k8s_client.V1PersistentVolumeClaimVolumeSource(claim_name=pvc_name)
        return (
            task
                .add_volume(
                    k8s_client.V1Volume(name=volume_name, persistent_volume_claim=local_pvc)
                )
                .add_volume_mount(
                    k8s_client.V1VolumeMount(mount_path=volume_mount_path, name=volume_name)
                )
        )
    return _use_local_volume
