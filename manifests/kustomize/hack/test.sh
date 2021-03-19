#!/bin/bash
#
# Copyright 2021 Google LLC
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

# This script quickly tests whether major entrances of the manifests folder can
# be hydrated. Maybe we can improve on it to provide snapshot diff.

set -ex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
MANIFESTS_DIR="${DIR}/.."

# These kustomization.yaml folders expect using kubectl kustomize (kustomize v2).
kustomization_yamls=(
  "base/installs/generic"
  "env/dev"
  "env/gcp"
  "env/platform-agnostic"
  "env/aws"
  "env/azure"
)
for path in "${kustomization_yamls[@]}"
do
  kubectl kustomize "${MANIFESTS_DIR}/${path}" >/dev/null
done

# These kustomization.yaml folders expect using kustomize v3+.
kustomization_yamls_v3=(
  "base/installs/multi-user"
  "env/platform-agnostic-multi-user"
)
for path in "${kustomization_yamls_v3[@]}"
do
  kustomize build --load_restrictor none "${MANIFESTS_DIR}/${path}" >/dev/null
done
