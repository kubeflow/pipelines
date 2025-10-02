# Upgrading Argo Workflows

Kubeflow Pipelines regularly upgrades the version of Argo Workflows provided.  See below for 
documentation on the steps required to perform this upgrade

## Upgrade Argo Workflows

Instructions:

1. Set version of argo you want to upgrade to, for example:

    ```bash
    ARGO_TAG=v3.7.2
    ```

1. Run the `update` target of the [Makefile](./Makefile) in this directory
    ```bash
    cd ./third_party/argo  # From repo root, if not already there
    echo "${ARGO_TAG}" > VERSION
    make update
    ```
1. Update the versions listed in the compatibility matrix in [README.md](../../README.md).

1. Consider bumping the minimum Argo version used in the GitHub workflows to match the prior version.
    * Add the previous version of argo to the list of versions found in the `argo_version` array found in [e2e workflow](../../.github/workflows/e2e-test.yml)
    * Update the argo\_versions for the various test matricies found in the [API check workflow](../../.github/workflows/kfp-samples.yml)
    * Also in the test matrix for the [API check workflow](../../.github/workflows/kfp-samples.yml), add an entry to test the previous version, and consider removing older (ie n-2) legacy checks if they fails CI.  (Note: we should still have n-1 legacy checks pass)

1. Verify all instances of the argo version have been updated.  
    * A simple seach such as `grep "vX.Y.Z" * -R` from repo root usually is good enough

1. Verify backend images still build by running `make image_all` from the `backend` directory
    * It would be a good idea to tag and push these images to a dev repo for use in testing

1. Validate the changes
    * Deploy KFP to a test environment, update the `image` used for the APIServer to the one built in the step above
    * Verify the API Server and WorkflowController pods come up, can run Pipelines, etc.
    * Fix any other issues caused by the upgrade.

1. Commit these changes to a PR.

NOTE: At this time, release.sh is a no-op included only for maintaining consistency with other third-party dependencies

## Upgrade Argo Workflows

### Upgade All Refrences of Argo Workflows

To upgrade to a new Argo version, including manifests, code dependencies, tests, and documentation:

1. Update the version in [VERSION](./VERSION)
2. Run `make update` to automatically update all references of the Argo Workflows dependency
3. Test the new configuration with your KFP deployment

** NOTE: this does not include the [GitHub CI workflows](../../.github/), which need to be manually updated as describe above **

### Update Argo Workflows Manifests

To upgrade just the manifests used for Argo Workflows:

1. Update the version in [VERSION](./VERSION)
2. Run `make update_manifests` to automatically update all remote Git references to the new version
3. Test the new configuration with your KFP deployment


### Update Argo Workflows in Backend Code

To upgrade just the Backend code and package references to a new Argo version:

1. Update the version in [VERSION](./VERSION)
2. Run `make update_backend` to automatically update all remote Git references to the new version
3. Test the new configuration with your KFP deployment


### Update Argo Workflows in Tests

To upgrade just the test references to a new Argo version:

1. Update the version in [VERSION](./VERSION)
2. Run `make update_tests` to automatically update all remote Git references to the new version
3. Test the new configuration with your KFP deployment


### Update Argo Workflows in Backend Code

To upgrade just the doc references to a new Argo version:

1. Update the version in [VERSION](./VERSION)
2. Run `make update_docs` to automatically update all remote Git references to the new version
3. Test the new configuration with your KFP deployment

## TODOs

Ideas to improve this process:

* Automate updating/adding new argo versions to the [GitHub Workflows](../../.github)

