# Manifests for Argo workflows

Kubeflow Pipelines uses [Argo Workflows](https://argoproj.github.io/argo-workflows/) as the underlying workflow execution engine.

This folder contains:

* `upstream/manifests` a mirror of argo workflows manifests upstream. It should never be edited here. Run `make update` to update it.
* `installs` a folder with preconfigured argo workflows installations used in Kubeflow Pipelines distributions.

  Major differences from upstream argo manifests:

  * Argo server is not included.
  * Argo workflow controller configmap is preconfigured to integrate with KFP.
  * Images are configured to use KFP redistributed ones which comply with open source licenses.
  * A default artifact repository config is added for in-cluster minio service.

## Upgrade argo

Refer to [third_party/argo/README.md](../../../../third_party/argo/README.md).

### Upgrade argo manifests

As one step of above, we need to upgrade argo manifests in this folder.

1. Run:

    ```bash
    make update
    ```

    Note, argo version is pulled from [third_party/argo/VERSION](../../../../third_party/argo/VERSION). Edit the VERSION file first.

2. Manually edit [YAML patches](upstream/manifests/namespace-install/overlays) to JSON patches, because YAML patches confuses kpt, refer to [#5368](https://github.com/kubeflow/pipelines/issues/5368).
