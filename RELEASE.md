# Releasing Kubeflow Pipelines

WIP: this document is still incomplete.

## Common Prerequisites

* OS: Linux (MacOS not supported yet due to different behavior of sed)
* Permissions needed
    * Can create a branch in github.com/kubeflow/pipelines
* Tools that should be in your `$PATH`
    * jq 1.6 https://stedolan.github.io/jq/download/
    * yq https://github.com/mikefarah/yq/releases/tag/3.3.0
    * jdk 8
* Preparations
    1. Clone github.com/kubeflow/pipelines repo into `$KFP_REPO`
    2. `cd $KFP_REPO`

## Cutting a release branch

1. Choose a good commit on master branch with commit hash as `$COMMIT_SHA`
2. Choose the next release branch's `$MINOR_VERSION` in format `x.y`, e.g. `1.0`, `1.1`...
2. Make a release branch of format `release-$MINOR_VERSION`, e.g. `release-1.0`, `release-1.1`. Branch from the commit and push to kubeflow pipelines upstream repo.
    ```bash
    git checkout $COMMIT_SHA
    git checkout -b release-$MINOR_VERSION
    git push upstream HEAD
    ```

## Releasing from release branch

1. Choose the release's complete `$VERSION` following semantic versioning, e.g.
    * `1.0.0-rc.1`
    * `1.0.0-rc.2`
    * `1.0.0`
    * `1.0.1`
    * ...
1. Update all version refs in release branch by
    ```bash
    ./hack/release.sh $VERSION release-$MINOR_VERSION
    ```
    It will prompt you whether to push it to release branch. Press `y` and hit `Enter`.

    Note, the script will clone kubeflow/pipelines repo into a temporary location on your computer, make those changes and attempt to push to upstream, so that it won't interfere with your current git repo.

    TODO: this script should also regenerate
    * changelog
    * python api client
1. Wait and make sure the cloudbuild job that builds all images in gcr.io/ml-pipeline-test succeeded for above commit. Then submit the second cloudbuild job that copies these images to gcr.io/ml-pipeline.

    TODO: we should have an automation KFP cluster, and the waiting and submiting second cloudbuild task should be automated.
1. Release `kfp-server-api` and `kfp` python packages on pypi.
1. Create a github release using `$VERSION` git tag, fill in the description.
