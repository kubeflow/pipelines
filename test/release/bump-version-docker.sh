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

echo "Usage: update kubeflow/pipelines/VERSION to new version tag by"
echo '`echo -n "\$VERSION" > VERSION` first, then run this script.'
echo "Please use the above command to make sure the file doesn't have extra"
echo "line endings."

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
REPO_ROOT="$DIR/../.."
TAG_NAME="$(cat $REPO_ROOT/VERSION)"

if [[ -z "$TAG_NAME" ]]; then
  echo "ERROR: $REPO_ROOT/VERSION is empty" >&2
  exit 1
fi

pushd "${REPO_ROOT}"
RELEASE_IMAGE=${RELEASE_IMAGE:-gcr.io/ml-pipeline-test/release@sha256:b96a0d2af1b10ab19883ecbd4df4aadb37ea5afd71e55e946d3eacb719a940dc}
docker run -it --rm \
  --user $(id -u):$(id -g) \
  --mount type=bind,source="$(pwd)",target=/go/src/github.com/kubeflow/pipelines \
  ${RELEASE_IMAGE} /go/src/github.com/kubeflow/pipelines/test/release/bump-version-in-place.sh
popd
