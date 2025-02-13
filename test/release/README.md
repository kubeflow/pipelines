# KFP Release tools

For full release documentation, please read [RELEASE.md](../../RELEASE.md).

To do a release:

TAG=`<TAG>` BRANCH=`<BRANCH>` make release

Example, release in release branch:

```bash
    TAG=1.4.0 BRANCH=release-1.4 make release
```

Example, release an RC (release candidate) in master branch:

```bash
    TAG=2.0.0-rc.1 BRANCH=master make release
```

## Dev guide

Take a look at the `./Makefile`, all the rules there are entrypoints for dev activities.

## Implementation Details

The script `./release.sh` is a wrapper

1. Clones github.com/kubeflow/pipelines repo to a temporary path.
1. Updates `../../VERSION` to `TAG`.
1. Checkout the release branch.
1. Runs `./bump-version-docker.sh`.
1. Runs git commit and tag.
1. After confirming with user input, pushes to upstream branch.

The script `./bump-version-docker.sh`

1. Runs `./bump-version-in-place.sh` in gcr.io/ml-pipeline-test/release:latest image.

The script `./bump-version-in-place.sh` does the following:

1. Generate `./CHANGELOG.md` using commit history.
1. Regenerate open api specs based on proto files.
1. Regenerate `kfp-server-api` python package.
1. Update all version refs in this repo to `./VERSION` by calling each of the
`./**/hack/release.sh` scripts. The individual scripts are responsible for updating
version refs to their own folder.
