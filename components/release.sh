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
# To run it, find a good release candidate commit SHA from ml-pipeline-staging project,
# and provide a full github COMMIT SHA to the script. E.g.
# ./release.sh 2118baf752d3d30a8e43141165e13573b20d85b8
# The script copies the images from staging to prod, and update the local code.
# You can then send a PR using your local branch.

set -xe

images=(
  "ml-pipeline-dataflow-tf-predict"
  "ml-pipeline-dataflow-tfdv"
  "ml-pipeline-dataflow-tft"
  "ml-pipeline-dataflow-tfma"
  "ml-pipeline-kubeflow-deployer"
  "ml-pipeline-kubeflow-tf-trainer"
  "ml-pipeline-kubeflow-tf"
  "ml-pipeline-dataproc-analyze"
  "ml-pipeline-dataproc-create-cluster"
  "ml-pipeline-dataproc-delete-cluster"
  "ml-pipeline-dataproc-predict"
  "ml-pipeline-dataproc-transform"
  "ml-pipeline-dataproc-train"
  "resnet-deploy"
  "resnet-preprocess"
  "resnet-train"
  "ml-pipeline-local-confusion-matrix"
  "ml-pipeline-local-roc"
)

COMMIT_SHA=$1
FROM_GCR_PREFIX='gcr.io/ml-pipeline-staging/'
TO_GCR_PREFIX='gcr.io/ml-pipeline/'
PARENT_PATH=$( cd "$(dirname "${BASH_SOURCE[0]}")" ; pwd -P )

for image in "${images[@]}"
do
  TARGET_IMAGE_BASE=${TO_GCR_PREFIX}${image}
  TARGET_IMAGE=${TARGET_IMAGE_BASE}:${COMMIT_SHA}

  # Move image from staging to prod GCR
  gcloud container images add-tag --quiet \
  ${FROM_GCR_PREFIX}${image}:${COMMIT_SHA} ${TARGET_IMAGE}

  # Update the code
  find "${PARENT_PATH}/../samples" -type f | while read file; do sed -i -e "s|${TARGET_IMAGE_BASE}:\([a-zA-Z0-9_.-]\)\+|${TARGET_IMAGE}|g" "$file"; done
  find "${PARENT_PATH}" -type f | while read file; do sed -i -e "s|${TARGET_IMAGE_BASE}:\([a-zA-Z0-9_.-]\)\+|${TARGET_IMAGE}|g" "$file"; done
done
