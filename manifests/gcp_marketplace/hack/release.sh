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

set -ex

TAG_NAME=$1
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

if [[ -z "$TAG_NAME" ]]; then
  echo "Usage: release.sh <release-tag>" >&2
  exit 1
fi

echo "This release script uses yq, it can be downloaded at https://github.com/mikefarah/yq/releases/tag/3.3.0"
yq w -i "$DIR/../schema.yaml" "x-google-marketplace.publishedVersion" "$TAG_NAME"
yq w -i "$DIR/../schema.yaml" "x-google-marketplace.publishedVersionMetadata.releaseNote" "Based on $TAG_NAME version."
yq w -i "$DIR/../chart/kubeflow-pipelines/templates/application.yaml" "spec.descriptor.version" "$TAG_NAME"
