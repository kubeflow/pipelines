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

TAG_NAME=$1
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
MANIFEST_DIR="${DIR}/.."

if [[ -z "$TAG_NAME" ]]; then
  echo "Usage: release.sh <release-tag>" >&2
  exit 1
fi

echo "This release script uses yq, it can be downloaded at https://github.com/mikefarah/yq/releases/tag/3.3.0"
kustomization_yamls_with_images=(
  "base/pipeline/kustomization.yaml"
)
for path in "${kustomization_yamls_with_images[@]}"
do
  yq w -i "${MANIFEST_DIR}/$path" images[*].newTag "$TAG_NAME"
done

yq w -i "${MANIFEST_DIR}/base/installs/generic/pipeline-install-config.yaml" data.appVersion "$TAG_NAME"

## The launcher image is added as an environment variable.
API_SERVER_MANIFEST="${MANIFEST_DIR}/base/pipeline/ml-pipeline-apiserver-deployment.yaml"

yq w -i ${API_SERVER_MANIFEST} \
  "spec.template.spec.containers.(name==ml-pipeline-api-server).env.(name==V2_LAUNCHER_IMAGE).value" \
  "ghcr.io/kubeflow/kfp-launcher:${TAG_NAME}"

# Argo parses data["sidecar.container"] as YAML, but Kustomize treats it as a
# string: images.newTag cannot update the driver image embedded inside it.
# Update that image separately in both base and TLS ConfigMaps so the driver
# uses the same release tag as the other KFP images.
DRIVER_PLUGIN_MANIFESTS=(
  "base/pipeline/ml-pipeline-driver-plugin-cm.yaml"
  "env/cert-manager/platform-agnostic-standalone-tls/patches/ml-pipeline-driver-plugin-cm.yaml"
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
