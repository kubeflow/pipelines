# Use Platform-Specific Features

## Overview

One of the benefits of KFP is cross-platform portability.
The KFP SDK compiles pipeline definitions to [IR YAML][ir-yaml] which can be read and executed by different backends, including the Kubeflow Pipelines [open source backend][oss-be] and [Vertex AI Pipelines](https://cloud.google.com/vertex-ai/docs/pipelines/introduction).

For cases where features are not portable across platforms, users may author pipelines with platform-specific functionality via KFP SDK platform-specific plugin libraries.
In general, platform-specific plugin libraries provide functions that act on tasks similarly to [task-level configuration methods][task-level-config-methods] provided by the KFP SDK directly.

<!-- TODO: add docs on how to create a platform-specific authoring library -->

## kfp-kubernetes

The `kfp.kubernetes` module is included in the unified `kfp` SDK. It is supported by the Kubeflow Pipelines [open source backend][oss-be] and enables direct access to some Kubernetes resources and functionality. Older SDK releases distributed these helpers separately as `kfp-kubernetes`.

For more information, see the [`kfp.kubernetes` API reference][kfp-kubernetes-docs].

### **Kubernetes PersistentVolumeClaims**

In this example we will use `kfp.kubernetes` to create a [PersistentVolumeClaim (PVC)][persistent-volume], use the PVC to pass data between tasks, and then delete the PVC.

We will assume you have basic familiarity with `PersistentVolume` and `PersistentVolumeClaim` resources in Kubernetes, in addition to [authoring components][authoring-components], and [authoring pipelines][authoring-pipelines] in KFP.

#### **Step 1:** Install the SDK

In a fresh environment, install `kfp`:

```sh
python -m pip install kfp
```

When migrating from the split packages, follow the [SDK upgrade instructions](https://github.com/kubeflow/pipelines/blob/master/sdk/python/README.md#migrating-from-the-split-packages) first. The `kfp[kubernetes]` extra remains accepted for compatibility.

#### **Step 2:** Create components that read/write to the mount path

Create two simple components that read and write to a file in the `/data` directory.

In a later step, we will mount a PVC volume to the `/data` directory.

```python
from kfp import dsl

@dsl.component
def producer() -> str:
    with open('/data/file.txt', 'w') as file:
        file.write('Hello world')
    with open('/data/file.txt', 'r') as file:
        content = file.read()
    print(content)
    return content

@dsl.component
def consumer() -> str:
    with open('/data/file.txt', 'r') as file:
        content = file.read()
    print(content)
    return content
```

#### **Step 3:** Dynamically provision a PVC using CreatePVC

Now that we have our components, we can begin constructing a pipeline.

We need a PVC to mount, so we will create one using the `kubernetes.CreatePVC` pre-baked component:

```python
from kfp import kubernetes

@dsl.pipeline
def my_pipeline():
    pvc1 = kubernetes.CreatePVC(
        # can also use pvc_name instead of pvc_name_suffix to use a pre-existing PVC
        pvc_name_suffix='-my-pvc',
        access_modes=['ReadWriteMany'],
        size='5Gi',
        storage_class_name='standard',
    )
```

This component provisions a 5GB PVC from the [StorageClass][storage-class] `'standard'` with the `ReadWriteMany` [access mode][access-mode].
The PVC will be named after the underlying Argo workflow that creates it, concatenated with the suffix `-my-pvc`. The `CreatePVC` component returns this name as the output `'name'`.

> **Note:** [KFP 2.15](https://github.com/kubeflow/pipelines/releases/tag/2.15.0) introduces an upgrade to Argo Workflows v3.7.3

#### **Step 4:** Read and write data to the PVC

Next, we'll use the `mount_pvc` task modifier with the `producer` and `consumer` components.

We schedule `task2` to run after `task1` so the components don't read and write to the PVC at the same time.

```python
    # write to the PVC
    task1 = producer()
    kubernetes.mount_pvc(
        task1,
        pvc_name=pvc1.outputs['name'],
        mount_path='/data',
    )

    # read to the PVC
    task2 = consumer()
    kubernetes.mount_pvc(
        task2,
        pvc_name=pvc1.outputs['name'],
        mount_path='/reused_data',
    )
    task2.after(task1)
```

#### **Step 5:** Delete the PVC

Finally, we can schedule deletion of the PVC after `task2` finishes to clean up the Kubernetes resources we created.

```python
    delete_pvc1 = kubernetes.DeletePVC(
        pvc_name=pvc1.outputs['name']
    ).after(task2)
```

For the full pipeline and more information, see a [similar example][full-example] in the [`kfp-kubernetes` documentation][kfp-kubernetes-docs].


[ir-yaml]: ../../concepts/ir-yaml.md
[oss-be]: ../../operator-guides/installation.md
[task-level-config-methods]: ../components/compose-components-into-pipelines.md#task-configurations
[kfp-kubernetes-docs]: https://kubeflow-pipelines.readthedocs.io/en/latest/source/kubernetes.html
[persistent-volume]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/
[storage-class]: https://kubernetes.io/docs/concepts/storage/storage-classes/
[access-mode]: https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes
[full-example]: https://kfp-kubernetes.readthedocs.io/en/kfp-kubernetes-0.0.1/#persistentvolumeclaim-dynamically-create-pvc-mount-then-delete
[authoring-components]: ../components/index.md
[authoring-pipelines]: ../index.md
