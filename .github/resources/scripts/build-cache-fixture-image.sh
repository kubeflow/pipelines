#!/usr/bin/env bash
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
# Preinstall each released SDK on its original Python base for cache scenarios.
set -euo pipefail

cluster_name=${1:?Usage: build-cache-fixture-image.sh CLUSTER_NAME OUTPUT_DIRECTORY}
output_directory=${2:?Usage: build-cache-fixture-image.sh CLUSTER_NAME OUTPUT_DIRECTORY}
repo_root=$(git rev-parse --show-toplevel)
cd "$repo_root"

nodes=$(kind get nodes --name "$cluster_name")
if [[ -z "$nodes" ]]; then
  echo "No Kind nodes found for $cluster_name." >&2
  exit 1
fi
node=${nodes%%$'\n'*}
# Deployment already verified this exact preloaded base. Export it back to the
# host so preparing the fixture cannot introduce another registry dependency.
build_context=$(mktemp -d)
trap 'rm -rf "$build_context"' EXIT
cp backend/test/v2/resources/cache-image/Dockerfile "$build_context/Dockerfile"
# Prepare released wheels once here, never in executor pods. Version 2.13.0
# is published only as an sdist; pip builds that wheel without repository sources.
# Keep the fixture's --no-deps installation behavior and exact released version.
matrix=$(python .github/resources/scripts/prepare_cache_fixtures.py --image-matrix)
while read -r base version image; do
  archive="$build_context/base.tar"
  docker exec "$node" ctr --namespace=k8s.io images export --platform linux/amd64 - "docker.io/library/$base" > "$archive"
  docker load --input "$archive"
  rm "$archive"
  mkdir -p "$build_context/wheels"
  python -m pip wheel --disable-pip-version-check --retries 3 --timeout 30 \
    --no-deps --wheel-dir "$build_context/wheels" "kfp==$version"
  # Use the daemon's preloaded image; installation itself stays offline.
  DOCKER_BUILDKIT=0 docker build --network=none --pull=false \
    --build-arg "PYTHON_BASE=$base" --build-arg "KFP_VERSION=$version" \
    --file "$build_context/Dockerfile" --tag "$image" "$build_context"
  kind load docker-image --name "$cluster_name" "$image"
  rm -rf "$build_context/wheels"
done <<< "$matrix"
# Reuse the readiness helper interpreter: its Kubernetes client installs PyYAML.
python .github/resources/scripts/prepare_cache_fixtures.py "$output_directory"
