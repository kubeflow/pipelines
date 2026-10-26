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

set -ex

export TAG_NAME=$1
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
MANIFEST_DIR="${DIR}/.."

if [[ -z "$TAG_NAME" ]]; then
  echo "Usage: release.sh <release-tag>" >&2
  exit 1
fi

# Validate and stamp runtime defaults before changing manifests.
python3 "${DIR}/../../../hack/update_tensorboard_image.py" "$TAG_NAME"

echo "This release script uses yq, it can be downloaded at https://github.com/mikefarah/yq/releases/tag/v4.54.1"
kustomization_yamls_with_images=(
  "base/cache-deployer/kustomization.yaml"
  "base/cache/kustomization.yaml"
  "base/metadata/base/kustomization.yaml"
  "base/pipeline/metadata-writer/kustomization.yaml"
  "base/pipeline/kustomization.yaml"
  "env/gcp/inverse-proxy/kustomization.yaml"
)
for path in "${kustomization_yamls_with_images[@]}"
do
  yq eval -i '.images[].newTag = strenv(TAG_NAME)' "${MANIFEST_DIR}/$path"
done

yq eval -i '.data.appVersion = strenv(TAG_NAME)' "${MANIFEST_DIR}/base/installs/generic/pipeline-install-config.yaml"

## Driver & Launcher images are added as environment variables
API_SERVER_MANIFEST="${MANIFEST_DIR}/base/pipeline/ml-pipeline-apiserver-deployment.yaml"

yq eval -i '(.spec.template.spec.containers[] | select(.name == "ml-pipeline-api-server") | .env[] | select(.name == "V2_LAUNCHER_IMAGE").value) = "ghcr.io/kubeflow/kfp-launcher:" + strenv(TAG_NAME)' "${API_SERVER_MANIFEST}"

yq eval -i '(.spec.template.spec.containers[] | select(.name == "ml-pipeline-api-server") | .env[] | select(.name == "V2_DRIVER_IMAGE").value) = "ghcr.io/kubeflow/kfp-driver:" + strenv(TAG_NAME)' "${API_SERVER_MANIFEST}"
