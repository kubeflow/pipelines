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
FORK_REMOTE=$3
UPSTREAM_REPO=kubeflow/pipelines

if [[ -z "$BRANCH" || -z "$TAG" || -z "$FORK_REMOTE" ]]; then
  echo "Usage: ./test/release/release.sh <release-tag> <release-branch> <fork-remote>" >&2
  echo "Example: ./test/release/release.sh 2.0.0 release-2.0 git@github.com:myuser/pipelines.git" >&2
  exit 1
fi

# Validate that fork is not pointing to kubeflow/pipelines org
if [[ "$FORK_REMOTE" =~ (github\.com[:/]kubeflow/pipelines|github\.com[:/]kubeflow/kfp) ]]; then
  echo "ERROR: FORK_REMOTE must NOT point to kubeflow/pipelines or kubeflow/kfp organization." >&2
  echo "       Please use your own fork instead." >&2
  echo "       Received: $FORK_REMOTE" >&2
  exit 1
fi

echo "Using fork: $FORK_REMOTE"
echo "Upstream: $UPSTREAM_REPO"

# Clone from the fork
clone_dir="$(mktemp -d)"
echo "Cloning fork into ${clone_dir}..."
git clone "$FORK_REMOTE" "${clone_dir}"
cd "$clone_dir"

# Add upstream remote
echo "Adding upstream remote..."
git remote add upstream "https://github.com/${UPSTREAM_REPO}.git"
git fetch upstream

# Checkout the release branch from upstream
echo "Checking out release branch from upstream..."
git checkout -b "$BRANCH" "upstream/$BRANCH"

echo "Deriving release branch name from VERSION file (drop patch, prepend release-)."
# Release Branch should be of the format release-x.y
RELEASE_BRANCH_FROM_VERSION="${BRANCH}"

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

echo ""
echo "==================== NEXT STEPS ===================="
echo "The release branch has been prepared locally in: ${clone_dir}"
echo ""
echo "To push to your fork and create a PR:"
echo "  1. Review changes: cd ${clone_dir} && git log --oneline -5"
echo "  2. Push to fork: git push --set-upstream origin $BRANCH"
echo "  3. Push tag: git push origin $TAG"
echo "  4. Create a PR from your fork's $BRANCH branch to upstream $UPSTREAM_REPO $BRANCH"
echo ""
read -p "Do you want to push the branch and tag to your fork now? [y|n] "
if [ "$REPLY" != "y" ]; then
   echo "Skipping push. You can manually push later from: ${clone_dir}"
   exit
fi

echo "Pushing branch and tag to fork..."
git push --set-upstream origin "$BRANCH"
git push origin "$TAG"

echo ""
echo "✓ Successfully pushed to your fork!"
echo ""
echo "Next: Create a Pull Request from your fork's '$BRANCH' branch"
echo "      to upstream kubeflow/pipelines '$BRANCH' branch"
echo ""
