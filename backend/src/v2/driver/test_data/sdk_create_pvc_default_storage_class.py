"""Pipeline compiled into sdk_create_pvc_default_storage_class.yaml.

Mirrors kubernetes.CreatePVC called without storage_class_name, as the SDK
compiles it: the component declares storage_class_name as optional with no
default value, so the driver must not receive the input at all.
"""

from kfp import compiler, dsl, kubernetes


@dsl.pipeline(name='sdk-create-pvc-default-storage-class')
def pipeline():
    kubernetes.CreatePVC(
        pvc_name='sdk-pvc',
        access_modes=['ReadWriteOnce'],
        size='5Mi',
    )


if __name__ == '__main__':
    compiler.Compiler().compile(pipeline, __file__.replace('.py', '.yaml'))
