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

set -xe

TAG_NAME=$1
BRANCH=$2
REPO=kubeflow/pipelines

if [[ -z "$BRANCH" || -z "$TAG_NAME" ]]; then
  echo "Usage: release.sh <release-tag> <release-branch>" >&2
  exit 1
fi

# Checking out the repo's release branch
clone_dir=$(mktemp -d)
git clone "git@github.com:${REPO}.git" "$clone_dir"
cd "$clone_dir"
git checkout "$BRANCH"

echo -n "$TAG_NAME" > ./VERSION
# Run the release script in cloned repo
"hack/release-imp.sh" $TAG_NAME

# Checking-in the component changes
git add --all
git commit --message "chore(release): bumped version to $TAG_NAME"
git tag -a "$TAG_NAME" -m "Kubeflow Pipelines $TAG_NAME release"

# Pushing the changes upstream
read -p "Do you want to push the version change and tag $TAG_NAME tag to upstream? [y|n]"
if [ "$REPLY" != "y" ]; then
   exit
fi
git push --set-upstream origin "$BRANCH"
git push origin "$TAG_NAME"
