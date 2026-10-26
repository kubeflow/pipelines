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

python3 "${DIR}/../../../hack/update_tensorboard_image.py" "$TAG_NAME"

echo "This release script uses yq, it can be downloaded at https://github.com/mikefarah/yq/releases/tag/v4.54.1"
kustomization_yamls_with_images=(
  "base/pipeline/kustomization.yaml"
)
for path in "${kustomization_yamls_with_images[@]}"
do
  yq eval -i '.images[].newTag = strenv(TAG_NAME)' "${MANIFEST_DIR}/$path"
done

yq eval -i '.data.appVersion = strenv(TAG_NAME)' "${MANIFEST_DIR}/base/installs/generic/pipeline-install-config.yaml"

## The launcher image is added as an environment variable.
API_SERVER_MANIFEST="${MANIFEST_DIR}/base/pipeline/ml-pipeline-apiserver-deployment.yaml"

yq eval -i '(.spec.template.spec.containers[] | select(.name == "ml-pipeline-api-server") | .env[] | select(.name == "V2_LAUNCHER_IMAGE").value) = "ghcr.io/kubeflow/kfp-launcher:" + strenv(TAG_NAME)' "${API_SERVER_MANIFEST}"

# Argo parses data["sidecar.container"] as YAML, but Kustomize treats it as a
# string: images.newTag cannot update the driver image embedded inside it.
# Update base, TLS and OpenShift plugin images together with other KFP images.
DRIVER_PLUGIN_MANIFESTS=(
  "base/pipeline/ml-pipeline-driver-plugin-cm.yaml"
  "env/cert-manager/platform-agnostic-standalone-tls/patches/ml-pipeline-driver-plugin-cm.yaml"
  "env/openshift/base/patches/driver-agent-plugin.yaml"
)
for path in "${DRIVER_PLUGIN_MANIFESTS[@]}"
do
  manifest="${MANIFEST_DIR}/$path"
  # Accept an existing release tag so release preparation can be run again.
  if ! contents="$(awk -v image="ghcr.io/kubeflow/kfp-driver:${TAG_NAME}" '
    /^[[:blank:]]*image:[[:blank:]]+ghcr[.]io\/kubeflow\/kfp-driver:[^[:space:]]+[[:blank:]]*$/ {
      sub(/ghcr[.]io\/kubeflow\/kfp-driver:[^[:space:]]+/, image)
      matches++
    }
    { print }
    END { if (matches != 1) exit 1 }
  ' "${manifest}")"; then
    echo "Expected exactly one tagged driver plugin image in ${manifest}; check sidecar.container" >&2
    exit 1
  fi
  printf '%s\n' "${contents}" > "${manifest}"
done
