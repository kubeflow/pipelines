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
  "env/gcp/inverse-proxy/kustomization.yaml"
)
for path in "${kustomization_yamls_with_images[@]}"
do
  yq w -i "${MANIFEST_DIR}/$path" images[*].newTag "$TAG_NAME"
done

yq w -i "${MANIFEST_DIR}/base/installs/generic/pipeline-install-config.yaml" data.appVersion "$TAG_NAME"

## Driver & Launcher images are added as environment variables
API_SERVER_MANIFEST="${MANIFEST_DIR}/base/pipeline/ml-pipeline-apiserver-deployment.yaml"

yq w -i ${API_SERVER_MANIFEST} \
  "spec.template.spec.containers.(name==ml-pipeline-api-server).env.(name==V2_LAUNCHER_IMAGE).value" \
  "ghcr.io/kubeflow/kfp-launcher:${TAG_NAME}"

yq w -i ${API_SERVER_MANIFEST} \
  "spec.template.spec.containers.(name==ml-pipeline-api-server).env.(name==V2_DRIVER_IMAGE).value" \
  "ghcr.io/kubeflow/kfp-driver:${TAG_NAME}"

# The executor-plugin container is stored as embedded YAML in ConfigMap data,
# so Kustomize's image transformer and the yq image updates above cannot see it.
# Keep both base and TLS plugin definitions aligned with the release tag.
DRIVER_PLUGIN_MANIFESTS=(
  "base/pipeline/ml-pipeline-driver-plugin-cm.yaml"
  "env/cert-manager/platform-agnostic-standalone-tls/patches/ml-pipeline-driver-plugin-cm.yaml"
)
for path in "${DRIVER_PLUGIN_MANIFESTS[@]}"
do
  manifest="${MANIFEST_DIR}/$path"
  contents="$(<"${manifest}")"
  placeholder="image: ghcr.io/kubeflow/kfp-driver:dummy"
  replacement="image: ghcr.io/kubeflow/kfp-driver:${TAG_NAME}"
  if [[ "${contents}" != *"${placeholder}"* ]]; then
    echo "Driver plugin image placeholder not found in ${manifest}" >&2
    exit 1
  fi
  printf '%s\n' "${contents//${placeholder}/${replacement}}" > "${manifest}"
done
