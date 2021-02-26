#!/bin/bash
#
# Copyright 2020 Google LLC
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
clone_dir=$(mktemp -d)
git clone "git@github.com:${REPO}.git" "$clone_dir"
cd "$clone_dir"
git checkout "$BRANCH"

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
# TODO(Bobgy): pin image tag
PREBUILT_REMOTE_IMAGE=gcr.io/ml-pipeline-test/release:latest
pushd ./test/release
TAG="${TAG}" BRANCH="${BRANCH}" make release
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
# git push --set-upstream origin "$BRANCH"
# git push origin "$TAG"
