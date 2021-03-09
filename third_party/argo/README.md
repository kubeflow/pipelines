# Third Party dependency -- Argo

This folder holds resources for KFP to redistribute <https://argoproj.github.io/projects/argo>
container images.

There's no code change. KFP only makes sure the images comply with licenses of all its dependencies
and transitive dependencies by redistributing license notices and source code (when required by
license) inside a `/NOTICES` folder inside the container.

## Upgrade Argo image

Prerequisites:

* Be an admin to gcr.io/ml-pipeline.

Instructions:

1. Set version of argo you want to upgrade to, for example:

    ```bash
    ARGO_TAG=v2.12.9
    ```

1. Update `NOTICES` folder by checking argo repository of the desired version and run [github.com/Bobgy/go-mod-licenses](github.com/Bobgy/go-mod-licenses) on it. This step makes sure we comply with legal requirements for redistributing argo container image.

    ```bash
    # clone argo git repo (you can also use git clone to do this)
    gh repo clone github.com/argoproj/argo-workflows
    cd argo-workflows
    git checkout "${ARGO_TAG}"
    # The following is a rough idea, please check go-mod-licenses documentation on the exact workflow.
    go-mod-licenses csv
    go-mod-licenses save
    # Preserve generated license info here.
    cp NOTICES <here>
    cp license_dict.csv <here>
    cp license_info.csv <here>
    ```

1. ```bash
    echo "${ARGO_TAG}" > VERSION
    ./release.sh
    ```

    After that, `gcr.io/ml-pipeline/argoexec:${ARGO_TAG}-license-compliance` and
    `gcr.io/ml-pipeline/workflow-controller:${ARGO_TAG}-license-compliance` will be available.

1. Update [manifests](../../manifests) and other places in the code base that still uses the old argo image tag.

1. Commit these changes to a PR.

1. Fix any other issues caused by the upgrade.

## TODOs

Ideas to improve this process:

* Add a presubmit test that verifies `NOTICES` folder is updated when argo image version is updated.
* Write a script that auto updates all occurrences of old argo image
tag to the new one.
* Reduce occurrences of argo image tag version, and let them use `./VERSION` programmatically when possible.
