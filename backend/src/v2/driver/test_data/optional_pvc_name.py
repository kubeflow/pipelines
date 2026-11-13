from typing import Optional

from kfp import compiler, dsl, kubernetes


@dsl.component(base_image='python:3.11')
def noop():
    print('ok')


@dsl.pipeline
def nested(child_pvc_name: Optional[str] = None):
    task = noop()
    kubernetes.mount_pvc(task, pvc_name=child_pvc_name, mount_path='/mnt/pvc')


@dsl.pipeline
def optional_pvc_name(
    required_pvc_name: str,
    optional_pvc_name: Optional[str] = None,
    optional_secret_name: Optional[str] = None,
):
    # Caching enabled: Container() resolves PVC names for the cache fingerprint first.
    cached = noop()
    kubernetes.mount_pvc(
        cached, pvc_name=optional_pvc_name, mount_path='/mnt/pvc'
    )

    # Caching disabled: only the pod spec patch resolves the PVC name.
    uncached = noop().set_caching_options(False)
    kubernetes.mount_pvc(
        uncached, pvc_name=optional_pvc_name, mount_path='/mnt/pvc'
    )

    required = noop()
    kubernetes.mount_pvc(
        required, pvc_name=required_pvc_name, mount_path='/mnt/pvc'
    )

    # Existing behaviour for comparison.
    secret = noop().set_caching_options(False)
    kubernetes.use_secret_as_volume(
        secret,
        secret_name=optional_secret_name,
        mount_path='/mnt/secret',
        optional=True,
    )

    # A null optional passed into a nested pipeline is dropped by the DAG driver,
    # so the PVC task sees no value for child_pvc_name at all.
    nested(child_pvc_name=optional_pvc_name)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=optional_pvc_name,
        package_path=__file__.replace('.py', '.yaml'),
    )
