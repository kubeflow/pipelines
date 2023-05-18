# Kubernetes Platform-specific Features

The `kfp-kubernetes` Python library enables authoring [Kubeflow pipelines](https://www.kubeflow.org/docs/components/pipelines/v2/) with Kubernetes-specific features. These features are supported by the [default KFP open source BE](https://github.com/kubeflow/pipelines/tree/master/backend). Specifically, the `kfp-kubernetes` library supports authoring pipelines that use:

* [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
* [PersistentVolumeClaims](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)

See the [`kfp-kubernetes` reference documentation](https://kfp-kubernetes.readthedocs.io/).

## Installation
The `kfp-kubernetes` package can be installed as a `kfp` SDK extra dependency with `kfp==2.x.x`:
<!-- TODO: remove --pre when kfp v2 goes to GA -->
```sh
pip install kfp[kubernetes] --pre
```

Or installed independently:
```sh
pip install kfp-kubernetes
```

## Example usage
<!-- TODO: test these examples once the BE implementation exists -->
### Secret: As environment variable
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def print_secret():
    import os
    print(os.environ['my-secret'])

@dsl.pipeline
def pipeline():
    task = print_secret()
    kubernetes.use_secret_as_env(task,
                                 secret_name='my-secret',
                                 secret_key_to_env={'password': 'SECRET_VAR'})
```

### Secret: As mounted volume
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def print_secret():
    with open('/mnt/my_vol') as f:
        print(f.read())

@dsl.pipeline
def pipeline():
    task = print_secret()
    kubernetes.use_secret_as_volume(task,
                                    secret_name='my-secret',
                                    mount_path='/mnt/my_vol')
```

### PersistentVolumeClaim: Dynamically create PVC, mount, then delete
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def make_data():
    with open('/data/file.txt', 'w') as f:
        f.write('my data')

@dsl.component
def read_data():
    with open('/reused_data/file.txt') as f:
        print(f.read())

@dsl.pipeline
def my_pipeline():
    pvc1 = kubernetes.CreatePVC(
        # can also use pvc_name instead of pvc_name_suffix to use a pre-existing PVC
        pvc_name_suffix='-my-pvc',
        access_modes=['ReadWriteOnce'],
        size='5Gi',
        storage_class_name='standard',
    )

    task1 = make_data()
    # normally task sequencing is handled by data exchange via component inputs/outputs
    # but since data is exchanged via volume, we need to call .after explicitly to sequence tasks
    task2 = read_data().after(task1)

    kubernetes.mount_pvc(
        task1,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )
    kubernetes.mount_pvc(
        task2,
        pvc_name=pvc1.outputs['name'],
        mount_path='/reused_data',
    )

    # wait to delete the PVC until after task2 completes
    delete_pvc1 = kubernetes.DeletePVC(
        pvc_name=pvc1.outputs['name']).after(task2)
```
