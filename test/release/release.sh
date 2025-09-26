#!/bin/bash
#
# Copyright 2020 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

TAG=$1
BRANCH=$2
REPO=kubeflow/pipelines

if [[ -z "$BRANCH" || -z "$TAG" ]]; then
  echo "Usage: ./test/release/release.sh <release-tag> <release-branch>" >&2
  exit 1
fi

# Checking out the repo's release branch
clone_dir="$(mktemp -d)"
# Use GitHub CLI if found, otherwise use git clone.
if which gh; then
  gh repo clone github.com/${REPO} "${clone_dir}"
else
  git clone "git@github.com:${REPO}.git" "${clone_dir}"
fi
cd "$clone_dir"
git checkout "$BRANCH"

echo "Deriving release branch name from VERSION file (drop patch, prepend release-)."
# VERSION may be like 2.14.3 or 2.15.0-rc.1. We only care about MAJOR.MINOR.
VERSION_FILE_CONTENT="$(cat VERSION)"
VERSION_CORE="${VERSION_FILE_CONTENT%%-*}"
MAJOR_MINOR="$(echo "$VERSION_CORE" | awk -F. '{print $1"."$2}')"
RELEASE_BRANCH_FROM_VERSION="release-${MAJOR_MINOR}"

echo "Will update image references to tag: ${RELEASE_BRANCH_FROM_VERSION}"

# Update image tag references used by client generation and tooling to the release branch tag
sed -i -E "s#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\\1${RELEASE_BRANCH_FROM_VERSION}#" backend/api/Makefile
sed -i -E "s#^(RELEASE_IMAGE=ghcr.io/kubeflow/kfp-release:).*#\\1${RELEASE_BRANCH_FROM_VERSION}#" backend/api/Makefile
sed -i -E "s#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\\1${RELEASE_BRANCH_FROM_VERSION}#" api/Makefile
sed -i -E "s#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\\1${RELEASE_BRANCH_FROM_VERSION}#" sdk/Makefile
sed -i -E "s#^(PREBUILT_REMOTE_IMAGE=ghcr.io/kubeflow/kfp-api-generator:).*#\\1${RELEASE_BRANCH_FROM_VERSION}#" kubernetes_platform/Makefile
# Keep release tools Makefile consistent as well
sed -i -E "s#^(REMOTE=ghcr.io/kubeflow/kfp-release:).*#\\1${RELEASE_BRANCH_FROM_VERSION}#" test/release/Makefile

# Update the release tools Dockerfile base image tag to align with the branch
sed -i -E "s#^(FROM ghcr.io/kubeflow/kfp-api-generator:).*#\\1${RELEASE_BRANCH_FROM_VERSION}#" test/release/Dockerfile.release

# Update default RELEASE_IMAGE tag in bump-version-docker.sh to this branch
sed -i -E "s#^(RELEASE_IMAGE=\$\{RELEASE_IMAGE:-ghcr.io/kubeflow/kfp-release:).*#\\1${RELEASE_BRANCH_FROM_VERSION}}#" test/release/bump-version-docker.sh

# Ensure the release bump container uses the correct tag for this branch
export RELEASE_IMAGE="ghcr.io/kubeflow/kfp-release:${RELEASE_BRANCH_FROM_VERSION}"

echo "Preparing local git tags used by changelog generation."
# tags with "-" are pre-releases, e.g. 1.0.0-rc.1
if [[ "$TAG" =~ "-" ]]; then
  echo "Releasing a pre-release $TAG."
else
  echo "Releasing a stable release $TAG."
  echo "Deleting all local pre-release tags to generate changelog from the last stable release. See issue https://github.com/kubeflow/pipelines/issues/4248.".
  for tag in $(git tag | grep -)
  do
    git tag -d "$tag"
  done
fi

echo "Running the bump version script in cloned repo"
echo -n "$TAG" > ./VERSION

pushd ./test/release
RELEASE_IMAGE="$RELEASE_IMAGE" make release-in-place
popd

echo "Checking in the version bump changes"
git add --all
git commit --message "chore(release): bumped version to $TAG"
git tag -a "$TAG" -m "Kubeflow Pipelines $TAG release"

echo "Pushing the changes upstream"
read -p "Do you want to push the version change and tag $TAG tag to upstream? [y|n]"
if [ "$REPLY" != "y" ]; then
   exit
fi
git push --set-upstream origin "$BRANCH"
git push origin "$TAG"
