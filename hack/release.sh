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

TAG_NAME=$1
BRANCH=$2
REPO=kubeflow/pipelines

if [[ -z "$BRANCH" || -z "$TAG_NAME" ]]; then
  echo "Usage: ./hack/release.sh <release-tag> <release-branch>" >&2
  exit 1
fi

echo "Preparing local git tags used by changelog generation."
# tags with "-" are pre-releases, e.g. 1.0.0-rc.1
if [[ "$TAG_NAME" =~ "-" ]]; then
  echo "Releasing a pre-release $TAG_NAME."
else
  echo "Releasing a stable release $TAG_NAME."
  echo "Deleting all local pre-release tags to generate changelog from the last stable release. See issue https://github.com/kubeflow/pipelines/issues/4248.".
  for tag in $(git tag | grep -)
  do
    git tag -d "$tag"
  done
fi

echo "Running the ./hack/release-imp.sh script in cloned repo"
echo -n "$TAG_NAME" > ./VERSION
"hack/release-imp.sh"

echo "Checking in the version bump changes"
git add --all
git commit --message "chore(release): bumped version to $TAG_NAME"
git tag -a "$TAG_NAME" -m "Kubeflow Pipelines $TAG_NAME release"

echo "Pushing the changes upstream"
read -p "Do you want to push the version change and tag $TAG_NAME tag to upstream? [y|n]"
if [ "$REPLY" != "y" ]; then
   exit
fi
git push --set-upstream origin "$BRANCH"
git push origin "$TAG_NAME"
