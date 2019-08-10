#!/bin/bash
#
# Copyright 2018 Google LLC
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

# This script automated the process to release the component images.
# To run it, find a good release candidate commit SHA from ml-pipeline-test project,
# and provide a full github COMMIT SHA to the script. E.g.
# ./release.sh 2118baf752d3d30a8e43141165e13573b20d85b8
# The script copies the images from test to prod, and update the local code.
# You can then send a PR using your local branch.

set -xe

images=(
  "ml-pipeline-dataflow-tf-predict"
  "ml-pipeline-dataflow-tfdv"
  "ml-pipeline-dataflow-tft"
  "ml-pipeline-dataflow-tfma"
  "ml-pipeline-kubeflow-deployer"
  "ml-pipeline-kubeflow-tf-trainer"
  "ml-pipeline-kubeflow-tf-trainer-gpu"
  "ml-pipeline-kubeflow-tf"
  "ml-pipeline-dataproc-analyze"
  "ml-pipeline-dataproc-create-cluster"
  "ml-pipeline-dataproc-delete-cluster"
  "ml-pipeline-dataproc-predict"
  "ml-pipeline-dataproc-transform"
  "ml-pipeline-dataproc-train"
  "ml-pipeline-local-confusion-matrix"
  "ml-pipeline-local-roc"
  "ml-pipeline-gcp"
)

COMMIT_SHA=$1
FROM_GCR_PREFIX='gcr.io/ml-pipeline-test/'
TO_GCR_PREFIX='gcr.io/ml-pipeline/'
REPO=kubeflow/pipelines

if [ -z "$COMMIT_SHA" ]; then
  echo "Usage: release.sh <commit-SHA>" >&2
  exit 1
fi

function sed_in_place() {
  # works for both OSX and linux
  sed -i '.bak' -E "$1" "$2"
  rm "$2.bak"
}

# Checking out the repo
clone_dir=$(mktemp -d)
git clone "git@github.com:${REPO}.git" "$clone_dir"
cd "$clone_dir"
branch="release-$COMMIT_SHA"
# Currently the release is based on master
release_head=master
git checkout "$release_head" -b "$branch"

# Releasing the container images to public. Updating components and samples.
for image in "${images[@]}"
do
  TARGET_IMAGE_BASE=${TO_GCR_PREFIX}${image}
  TARGET_IMAGE=${TARGET_IMAGE_BASE}:${COMMIT_SHA}

  # Move image from test to prod GCR
  gcloud container images add-tag --quiet \
  ${FROM_GCR_PREFIX}${image}:${COMMIT_SHA} ${TARGET_IMAGE}

  # Update the code
  find components samples -type f ! -iname "*.png" ! -iname "*.jpg" | while read file; do
    sed_in_place "s|${TARGET_IMAGE_BASE}:[a-zA-Z0-9_.-]+|${TARGET_IMAGE}|g" "$file";
  done
done

# Checking-in the container image changes
git add --all
git commit --message "Updated component images to version $COMMIT_SHA"
image_update_commit_sha=$(git rev-parse HEAD)

# Updating the samples to use the updated components
#git diff HEAD~1 HEAD --name-only | while read component_file; do
find components samples -type f ! -iname "*.png" ! -iname "*.jpg" | while read file; do
    sed_in_place "s|(https://raw.githubusercontent.com/kubeflow/pipelines/)[^/]+/|\1${image_update_commit_sha}/|g" "$file"
done
echo "clone dir: $clone_dir"

# Checking-in the component changes
git add --all
git commit --message "Updated components to version git l$image_update_commit_sha"
component_update_commit_sha=$(git rev-parse HEAD)
# Pushing the changes upstream
read -p "Do you want to push the new branch to upstream to create a PR? [y|n]"
if [ "$REPLY" != "y" ]; then
   exit
fi
git push --set-upstream origin "$branch"
echo "clone dir: $clone_dir"

sensible-browser "https://github.com/${REPO}/compare/master...$branch"
