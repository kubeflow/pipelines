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

# This script automated the process to update the component images.
# To run it, find a good release candidate commit SHA from ml-pipeline-test project,
# and provide a full github COMMIT SHA to the script. E.g.
# ./update-for-release.sh 2118baf752d3d30a8e43141165e13573b20d85b8
# The script copies the images from test to prod, and update the local code.

set -xe

function update_for_release {
  local images=(
    "ml-pipeline-kubeflow-deployer"
    "ml-pipeline-kubeflow-tf-trainer"
    "ml-pipeline-kubeflow-tf-trainer-gpu"
    "ml-pipeline-kubeflow-tfjob"
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

  local TAG_NAME=$1
  local FROM_GCR_PREFIX='gcr.io/ml-pipeline-test/'
  local TO_GCR_PREFIX='gcr.io/ml-pipeline/'

  if [ -z "$TAG_NAME" ]; then
    echo "Usage: update_for_release <tag-name>" >&2
    return 1
  fi

  # Update setup.py VERSION
  sed -i.bak -e "s|VERSION =.\+'|VERSION = '${TAG_NAME}'|g" "components/gcp/container/component_sdk/python/setup.py"

  # Updating components and samples.
  for image in "${images[@]}"
  do
    local TARGET_IMAGE_BASE=${TO_GCR_PREFIX}${image}
    local TARGET_IMAGE=${TARGET_IMAGE_BASE}:${TAG_NAME}

    # Update the code
    find components samples -type f | while read file; do
      sed -i -e "s|${TARGET_IMAGE_BASE}:\([a-zA-Z0-9_.-]\)\+|${TARGET_IMAGE}|g" "$file"
    done
  done

  # Updating the samples to use the updated components
  git diff --name-only | while read component_file; do
      echo $component_file
      find components samples -type f | while read file; do
        sed -i -E "s|(https://raw.githubusercontent.com/kubeflow/pipelines/)[^/]+(/$component_file)|\1${TAG_NAME}\2|g" "$file";
      done
  done

  # Checking-in the component changes
  git add --all
  git commit --message "Updated components images and refs to version $TAG_NAME"
}
