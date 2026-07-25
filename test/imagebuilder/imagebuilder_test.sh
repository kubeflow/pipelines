#!/bin/bash
#
# Copyright 2026 The Kubeflow Authors
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

set -euo pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
CONTAINER_ENGINE=${CONTAINER_ENGINE:-docker}
IMAGE_BUILDER_TEST_IMAGE=${IMAGE_BUILDER_TEST_IMAGE:-kfp-image-builder:test}

"${CONTAINER_ENGINE}" build \
    --platform linux/amd64 \
    --file "${REPO_ROOT}/test/imagebuilder/Dockerfile" \
    --tag "${IMAGE_BUILDER_TEST_IMAGE}" \
    "${REPO_ROOT}/test/imagebuilder"

"${CONTAINER_ENGINE}" run --rm \
    --entrypoint /bin/bash \
    "${IMAGE_BUILDER_TEST_IMAGE}" \
    -c 'set -euo pipefail
        gcloud --version
        gsutil version -l
        docker --version | grep -F "20.10.24"'

echo "image-builder tool smoke test passed"
