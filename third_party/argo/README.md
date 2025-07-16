# Third Party dependency -- Argo

This folder holds resources for KFP to redistribute <https://argoproj.github.io/projects/argo>
container images.

There's no code change. At this time, the purpose of this directory is soley to contain
release-related scripts and metadata for the Argo Workflows dependency

## Upgrade Argo image

Instructions:

1. Set version of argo you want to upgrade to, for example:

   ```bash
   ARGO_TAG=v3.5.14
   ```

1. ```bash
   echo "${ARGO_TAG}" > VERSION
   ./release.sh
   ```

   NOTE: At this time, release.sh is a no-op included only for maintaining consistency with other third-party dependencies

1. Update [manifests](../../manifests) and other places in the code base that still uses the old argo image tag.
   - Upgrade [Argo upstream manifests](https://github.com/kubeflow/pipelines/blob/master/manifests/kustomize/third-party/argo/README.md#upgrade-argo).
   - Search for the old argo versions in the repo and update them to new versions based on the reference.

1. Commit these changes to a PR.

1. Fix any other issues caused by the upgrade.

## TODOs

Ideas to improve this process:

- Write a script that auto updates all occurrences of old argo image
  tag to the new one.
- Reduce occurrences of argo image tag version, and let them use `./VERSION` programmatically when possible.
