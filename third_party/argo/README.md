# Third Party dependency -- Argo

This folder holds resources for KFP to redistribute <https://argoproj.github.io/projects/argo>
container images.

There's no code change. At this time, the purpose of this directory is soley to contain
release-related scripts and metadata for the Argo Workflows dependency

## Upgrade Argo image

Instructions:

1. Set version of argo you want to upgrade to, for example:

    ```bash
    ARGO_TAG=v3.7.2
    ```

1. ```bash
    echo "${ARGO_TAG}" > VERSION
    ./release.sh
    ```

    NOTE: At this time, release.sh is a no-op included only for maintaining consistency with other third-party dependencies

1. Update the versions listed in the compatibility matrix in [README.md](../../README.md).

1. Consider bumping the minimum Argo version used in the GitHub workflows to match the prior version.
    * Add the previous version of argo to the list of versions found in the `argo\_version` array found in e2e workflow: `.github/workflows/e2e-test.yml`
    * Update the argo_versions for the various test matricies found in the samples workflow: `.github/workflows/kfp-samples.yml`
    * Also in the kfp-samples test matrix, add an entry to test the previous version, and consider removing older (ie n-2) legacy checks if they fails CI.  (Note: we should still have n-1 legacy checks pass)

1. Update [manifests](../../manifests) and other places in the code base that still uses the old argo image tag.
    * This can be performed by simply running `make update` from `manifests/kustomize/third-party/argo`
    * Verify all instances of the argo version have been updated.  A simple seach such as `grep "vX.Y.Z" * -R` usually is good enough
    * Verify backend images still build by running `make image\_all` from the `backend` directory

1. Commit these changes to a PR.

1. Fix any other issues caused by the upgrade.

## TODOs

Ideas to improve this process:

* Write a script that auto updates the GHAction workflow files alongside all the manifests when running `make update`
* Reduce occurrences of argo image tag version, and let them use `./VERSION` programmatically when possible.
