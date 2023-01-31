from kubernetes.client import V1PersistentVolumeClaimSpec, V1PersistentVolumeClaim, V1VolumeMount, V1Volume, V1VolumeMount, V1ObjectMeta

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
    name="osm-config",
    persistent_volume_claim=pvc,
)
volume_mount = V1VolumeMount(
    name='osm-config',
    mount_path='/data',
    sub_path=None,
    read_only=False,
)

volume_mount = volume_mount
volume = volume


@dsl.component
def comp():
    pass


from kfp import kubernetes


# passing k8s objects
# declarative...
@dsl.pipeline
def p():
    t1 = comp()
    kubernetes.add_pvc(t1, pvc=pvc)
    kubernetes.take_snapshot_after(t1, pvc=pvc)

    # I think this would be sufficient for task/snapshot scheduling
    t2 = comp().after(t1)
    kubernetes.add_pvc(t2, pvc=pvc)
    kubernetes.take_snapshot_after(t1, pvc=pvc)


# passing arguments
@dsl.pipeline
def p():
    t1 = comp()
    pvc = kubernetes.add_pvc(task=t1, **kwargs)
    kubernetes.take_snapshot_after(task=t1, pvc=pvc)

    # I think this would be sufficient for task/snapshot scheduling
    t2 = comp().after(t1)
    kubernetes.add_pvc(task=t2, pvc=pvc)


# Operator approach
# imperative
@dsl.pipeline
def p():
    pvc_task.output = kubernetes.create_pvc()

    t1 = comp()
    pvc = kubernetes.add_pvc(t1, volume=pvc_task.output)
    kubernetes.take_snapshot_after(t1, pvc=pvc_task.output)

    # I think this would be sufficient for task/snapshot scheduling
    t2 = comp().after(t1)
    kubernetes.add_pvc(t2, pvc=pvc)


# --- secret ---
# convenience
@dsl.pipeline
def p():
    t1 = comp()
    kubernetes.use_k8s_secret(
        t1,
        secret_name='s3-secret',
        secret_key_to_env_var={'secret_key': 'AWS_SECRET_ACCESS_KEY'})


# fully parameterized
@dsl.pipeline
def p():
    t1 = comp()
    kubernetes.secret_as_env_var(
        t1,
        secret_name='secret1',
        secret_key_to_env_var={'secret_key': 'AWS_SECRET_ACCESS_KEY'})


# k8s object
@dsl.pipeline
def p():
    t1 = comp()
    kubernetes.add_env_var(
        t1,
        env_var=k8s_client.V1EnvVar(
            name='MY_SECRET',
            value_from=k8s_client.V1EnvVarSource(
                secret_key_ref=k8s_client.V1SecretKeySelector(
                    name='secret1', key='secret_key'))))


# In most cases, three options
# - convenience
# - fully parameterized
# - k8s object

# in volume case, there is actioning and sequencing associated
# - create ops?
# - declarative