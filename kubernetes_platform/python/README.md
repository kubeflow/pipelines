# Kubernetes Platform-specific Features

The `kfp-kubernetes` Python library enables authoring [Kubeflow pipelines](https://www.kubeflow.org/docs/components/pipelines/v2/) with Kubernetes-specific features. These features are supported by the [default KFP open source BE](https://github.com/kubeflow/pipelines/tree/master/backend). Specifically, the `kfp-kubernetes` library supports authoring pipelines that use:

* [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)
* [PersistentVolumeClaims](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims)
* [ImagePullPolicy](https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy)

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
    print(os.environ['SECRET_VAR'])

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

### Secret: As optional source for a mounted volume
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
                                    mount_path='/mnt/my_vol'
                                    optional=True)
```

### ConfigMap: As environment variable
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def print_config_map():
    import os
    print(os.environ['CM_VAR'])

@dsl.pipeline
def pipeline():
    task = print_config_map()
    kubernetes.use_config_map_as_env(task,
                                 config_map_name='my-cm',
                                 config_map_key_to_env={'foo': 'CM_VAR'})
```

### ConfigMap: As mounted volume
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def print_config_map():
    with open('/mnt/my_vol') as f:
        print(f.read())

@dsl.pipeline
def pipeline():
    task = print_config_map()
    kubernetes.use_config_map_as_volume(task,
                                       config_map_name='my-cm',
                                       mount_path='/mnt/my_vol')
```

### ConfigMap: As optional source for a mounted volume
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def print_config_map():
    with open('/mnt/my_vol') as f:
        print(f.read())

@dsl.pipeline
def pipeline():
    task = print_config_map()
    kubernetes.use_config_map_as_volume(task,
                                       config_map_name='my-cm',
                                       mount_path='/mnt/my_vol',
				       optional=True)
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

### PersistentVolumeClaim: Create PVC on-the-fly tied to your pod's lifecycle
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def make_data():
    with open('/data/file.txt', 'w') as f:
        f.write('my data')

@dsl.pipeline
def my_pipeline():
    task1 = make_data()
    # note that the created pvc will be autoamatically cleaned up once pod disappeared and cannot be shared between pods
    kubernetes.add_ephemeral_volume(
        task1,
        volume_name="my-pvc",
        mount_path="/data",
        access_modes=['ReadWriteOnce'],
        size='5Gi',
    )
```

### Pod Metadata: Add pod labels and annotations to the container pod's definition
```python
from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass


@dsl.pipeline
def my_pipeline():
    task = comp()
    kubernetes.add_pod_label(
        task,
        label_key='kubeflow.com/kfp',
        label_value='pipeline-node',
    )
    kubernetes.add_pod_annotation(
        task,
        annotation_key='run_id',
        annotation_value='123456',
    )
```

### Kubernetes Field: Use Kubernetes Field Path as enviornment variable
```python
from kfp import dsl
from kfp import kubernetes


@dsl.component
def comp():
    pass


@dsl.pipeline
def my_pipeline():
    task = comp()
    kubernetes.use_field_path_as_env(
        task,
        env_name='KFP_RUN_NAME',
        field_path="metadata.annotations['pipelines.kubeflow.org/run_name']"
    )
```

### Timeout: Set timeout in seconds defined as pod spec's activeDeadlineSeconds
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def comp():
    pass

@dsl.pipeline
def my_pipeline():
    task = comp()
    kubernetes.set_timeout(task, 20)
```

### ImagePullPolicy: One of "Always" "Never", "IfNotPresent".
```python
from kfp import dsl
from kfp import kubernetes

@dsl.component
def simple_task():
    print("hello-world")

@dsl.pipeline
def pipeline():
    task = simple_task()
    kubernetes.set_image_pull_policy(task, "Always")
```
