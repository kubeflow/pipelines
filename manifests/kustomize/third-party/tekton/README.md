# Tekton manifests

## Upgrade the Tekton Manifest Release

To upgrade the Tekton pipeline or Tekton dashboard manifest to the latest release, run the following commands in this directory

```shell
curl -L https://storage.googleapis.com/tekton-releases/pipeline/latest/release.yaml --output upstream/manifests/base/tektoncd-install/tekton-release.yaml
curl -L https://storage.googleapis.com/tekton-releases/dashboard/latest/release.yaml --output upstream/manifests/base/tektoncd-dashboard/tekton-dashboard-release.yaml
```
