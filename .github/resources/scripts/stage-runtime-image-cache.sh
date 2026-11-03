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

artifact_files="${BASH_SOURCE%/*}/artifact-files.sh"
export REQUIRED_FILES=$'runtime-base-images.tar\nmodelcar.tar'
export DOWNLOAD_PATH="${CACHE_DIRECTORY:?Expected CACHE_DIRECTORY}"
status=0
bash "$artifact_files" verify || status=$?
# Missing bytes are a cache miss; invalid paths and operational failures remain fatal.
if (( status != 0 && status != 10 )); then
  exit "$status"
fi
if [[ "${CACHE_OUTCOME:-}" != success || "${CACHE_HIT:-}" != true ]] || (( status != 0 )); then
  bash "$artifact_files" prepare
  echo 'ready=false' >> "${GITHUB_OUTPUT:?Expected GITHUB_OUTPUT}"
  exit 0
fi

DOWNLOAD_PATH="${IMAGE_DIRECTORY:?Expected IMAGE_DIRECTORY}" bash "$artifact_files" prepare
for file in runtime-base-images.tar modelcar.tar; do
  mv -- "$CACHE_DIRECTORY/$file" "$IMAGE_DIRECTORY/$file"
done
echo 'ready=true' >> "${GITHUB_OUTPUT:?Expected GITHUB_OUTPUT}"
