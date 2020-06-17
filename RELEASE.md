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
1. View related cloudbuild jobs' statuses by clicking the latest commit's status icon
in the release branch. The page will look like https://github.com/kubeflow/pipelines/runs/775788343.
1. Wait and make sure the `build-each-commit` cloudbuild job that builds all images
in gcr.io/ml-pipeline-test succeeded. If it fails, please click "View more details on Google Cloud Build" and then "Retry".
1. Select the `release-on-tag` cloudbuild job that copies built images and artifacts to
public image registry and gcs bucket. This job should have already failed because
artifacts haven't been built. Now, please click "View more details on Google Cloud Build"
and then "Retry", because after waiting for previous step, artifacts are now ready.

    TODO: we should have an automation KFP cluster, and the waiting and submiting
    `release-on-tag` cloudbuild task should be automatically waited.
1. Search "PyPI" in internal doc for getting password of kubeflow-pipelines user.
1. Release `kfp-server-api` python packages to PyPI.
    ```bash
    git checkout $BRANCH
    git pull upstream
    cd backend/api/python_http_client
    python3 setup.py --quiet sdist
    python3 -m twine upload --username kubeflow-pipelines dist/*
    ```
1. Release `kfp` python packages to PyPI.
    ```bash
    export TAG_NAME=0.2.2
    pip3 install twine --user
    gsutil cp gs://ml-pipeline/release/$TAG_NAME/kfp.tar.gz kfp-$TAG_NAME.tar.gz
    python3 -m twine upload --username kubeflow-pipelines kfp-$TAG_NAME.tar.gz
    ```

    !!! The file name must contain the version (you might need to rename the file). See https://github.com/kubeflow/pipelines/issues/1292

    The username is "kubeflow-pipelines"

1. Create a github release using `$VERSION` git tag and title `Version $VERSION`,
fill in the description.

Use this template for public releases
<pre>
To deploy Kubeflow Pipelines in an existing cluster, follow the instruction in [here](https://www.kubeflow.org/docs/pipelines/standalone-deployment-gcp/) or via UI [here](https://console.cloud.google.com/ai-platform/pipelines)

Install python SDK (python 3.5 above) by running:
```
python3 -m pip install kfp kfp-server-api --upgrade
```

See the [Change Log](https://github.com/kubeflow/pipelines/blob/master/CHANGELOG.md)
</pre>

Use this template for prereleases (release candidates) and please check the
`This is a prerelease` checkbox in the Github release UI.
<pre>
To deploy Kubeflow Pipelines in an existing cluster, follow the instruction in [here](https://www.kubeflow.org/docs/pipelines/standalone-deployment-gcp/).

Install python SDK (python 3.5 above) by running:
```
python3 -m pip install kfp kfp-server-api --pre --upgrade
```

See the [Change Log](https://github.com/kubeflow/pipelines/blob/master/CHANGELOG.md)
</pre>

## Cherry picking PRs to release branch

### Option - git cherry-pick
* Find the commit you want to cherry pick on master as $COMMIT_SHA.
* Find the active release branch name $BRANCH, e.g. release-1.0
*
    ```bash
    git co $BRANCH
    git co -b <cherry-pick-pr-branch-name>
    git cherry-pick $COMMIT_SHA
    ```
* Resolve merge conflicts if any
* `git push origin HEAD`
* create a PR and remember to update PR's destination branch to `$BRANCH`
* Ask the same OWNERS that would normally need to approve this PR

### Option - Kubeflow cherry_pick_pull.sh helper
Kubeflow has a cherry pick helper script: https://github.com/kubeflow/kubeflow/blob/master/hack/cherry-picks.md

It automates the process using `hub` CLI tool and bash, so it takes some one off efforts to set up for the first time.

After that, this is convenient to do a lot of cherry picks, because for each PR you'd only need to specify
release branch and PR number.

Known caveats:
* It may produce PR title that is duplicative, you can edit the title after cherry picking.
