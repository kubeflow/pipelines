#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

artifacts_path=${1:?Usage: build-runtime-base-images.sh OUTPUT_DIRECTORY}
source .github/resources/scripts/helper-functions.sh
mkdir -p "${artifacts_path}"

pull_and_save_runtime_base_images \
  .github/resources/runtime-base-images.txt \
  "${artifacts_path}/runtime-base-images.tar"

retry 3 30 env DOCKER_BUILDKIT=1 docker build \
  --file test_data/sdk_compiled_pipelines/valid/critical/modelcar/Dockerfile \
  --tag registry.domain.local/modelcar:test \
  .
docker save registry.domain.local/modelcar:test \
  --output "${artifacts_path}/modelcar.tar"
