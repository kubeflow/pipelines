#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

set -euo pipefail
source "$(dirname -- "${BASH_SOURCE[0]}")/ci-image-artifacts.sh"

API_SOURCE_DIRECTORIES=(
  backend/api/v2/go_client
  backend/api/v2/go_http_client
  backend/api/v2/swagger
  backend/api/v2/python_http_client
  sdk/python/kfp/server_api
)

# Compare source bytes, not archives (whose packaging metadata can differ).
# Keep generator metadata in the comparison; exclude only build products.
snapshot_sources() {
  python3 - "$@" <<'PY'
import hashlib
from pathlib import Path
import sys

root = Path(sys.argv[1])
paths = []
for directory in sys.argv[2:]:
    files = [path for path in (root / directory).rglob('*') if path.is_file()]
    if not files:
        raise SystemExit(f'No generated files in {directory}')
    for path in files:
        relative = path.relative_to(root)
        if any(part in ('dist', '__pycache__') or part.endswith('.egg-info')
               for part in relative.parts):
            continue
        paths.append((relative.as_posix(), path))
for relative, path in sorted(paths):
    print(f'{hashlib.sha256(path.read_bytes()).hexdigest()}  {relative}')
PY
}

validate_native_images() {
  local architecture=$1 machine
  case "$architecture" in
    amd64) machine=x86_64 ;;
    arm64) machine=aarch64 ;;
    *) echo "Expected amd64 or arm64, got: $architecture" >&2; return 1 ;;
  esac
  if [[ $(uname -s) != Linux || $(uname -m) != "$machine" ]]; then
    echo "This smoke requires a native Linux $architecture runner." >&2
    return 1
  fi
  local image
  for image in "${TOOL_IMAGE_ARTIFACTS[@]}"; do
    if [[ $(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$image:ci") != "linux/$architecture" ]]; then
      echo "$image:ci does not match native Linux $architecture." >&2
      return 1
    fi
  done
}

main() {
  if [[ $# != 2 ]]; then
    echo "Usage: $0 <amd64|arm64> <output-directory>" >&2
    return 1
  fi
  validate_native_images "$1"
  local output root source_dir image
  root=$(git rev-parse --show-toplevel)
  mkdir -p "$2"
  output=$(cd "$2" && pwd)
  source_dir=$(mktemp -d "${TMPDIR:-/tmp}/kfp-maintainer-smoke.XXXXXX")
  # This directory is exclusively owned by this run; never modify the checkout.
  trap 'rm -rf -- "$source_dir"' EXIT
  git -C "$root" archive HEAD | tar -x -C "$source_dir"

  for image in "${TOOL_IMAGE_ARTIFACTS[@]}"; do
    docker run --rm --interactive --user "$(id -u):$(id -g)" \
      --env HOME=/tmp/kfp-smoke-home --env API_VERSION=v2 \
      --env LC_ALL=C.UTF-8 --env TZ=UTC \
      --mount "type=bind,source=$source_dir,target=/go/src/github.com/kubeflow/pipelines" \
      --workdir /go/src/github.com/kubeflow/pipelines "$image:ci" bash -se <<'CONTAINER'
set -euo pipefail
mkdir -p "$HOME"
test "$(protoc --version)" = "libprotoc ${PROTOC_VERSION}"
swagger_version=$(cd /tmp/api-generator-tools && go mod edit -json | jq -er '.Require[] | select(.Path == "github.com/go-swagger/go-swagger") | .Version')
swagger version | grep -F "${swagger_version#v}"
go version
java -version
python3 -c 'import setuptools'
backend/api/hack/generator.sh
backend/api/build_kfp_server_api_python_package.sh
test -s backend/api/v2/go_client/run.pb.go
test -s backend/api/v2/swagger/kfp_api_single_file.swagger.json
test -s sdk/python/kfp/server_api/api_client.py
python3 -m pip wheel --no-deps ./sdk/python --wheel-dir /tmp/kfp-smoke-dist
compgen -G '/tmp/kfp-smoke-dist/kfp-*.whl' >/dev/null
CONTAINER
    snapshot_sources "$source_dir" "${API_SOURCE_DIRECTORIES[@]}" \
      > "$output/$image$TOOL_API_OUTPUT_SUFFIX"
  done
  diff -u "$output/$TOOL_GENERATOR_IMAGE$TOOL_API_OUTPUT_SUFFIX" \
    "$output/$TOOL_RELEASE_IMAGE$TOOL_API_OUTPUT_SUFFIX"

  docker run --rm --interactive --user "$(id -u):$(id -g)" \
    --env HOME=/tmp/kfp-smoke-home --env LC_ALL=C.UTF-8 --env TZ=UTC \
    --mount "type=bind,source=$source_dir,target=/go/src/github.com/kubeflow/pipelines" \
    --workdir /go/src/github.com/kubeflow/pipelines "$TOOL_RELEASE_IMAGE:ci" bash -se <<'CONTAINER'
set -euo pipefail
mkdir -p "$HOME"
test "$(node --version)" = "v$(tr -d 'v\r\n' < frontend/.nvmrc)"
npm --version
yq --version | grep -F '3.4.1'
test "$(git-cliff --version)" = "git-cliff ${GIT_CLIFF_VERSION}"
manifests/kustomize/hack/release.sh 3.0.0-smoke
test "$(yq r manifests/kustomize/base/installs/generic/pipeline-install-config.yaml data.appVersion)" = 3.0.0-smoke
# Synthetic history exercises the project's real changelog configuration without
# needing credentials, tags, or a writable clone of the contributor's repository.
git init -q
git config user.name 'KFP tooling smoke fixture'
git config user.email 'smoke@example.invalid'
git -c commit.gpgsign=false commit --allow-empty -qm 'chore: fixture baseline'
previous=$(git rev-parse HEAD)
git -c commit.gpgsign=false commit --allow-empty -qm 'feat: native maintainer tooling smoke'
git-cliff -c cliff.toml --tag 3.0.0-smoke --prepend CHANGELOG.md "$previous..HEAD"
grep -F 'native maintainer tooling smoke' CHANGELOG.md
CONTAINER
  snapshot_sources "$source_dir" manifests/kustomize > "$output/$TOOL_MANIFESTS_OUTPUT"
  # Only the synthetic release heading contains a wall-clock date. Preserve
  # every other byte, including the project's existing historical changelog.
  python3 - "$source_dir/CHANGELOG.md" "$output/$TOOL_CHANGELOG_OUTPUT" <<'PY'
from pathlib import Path
import re
import sys

text, count = re.subn(r'^(## 3\.0\.0-smoke) \(\d{4}-\d{2}-\d{2}\)$',
                      r'\1 (<release-date>)', Path(sys.argv[1]).read_text(),
                      count=1, flags=re.MULTILINE)
if count != 1:
    raise SystemExit('Expected one synthetic release heading')
Path(sys.argv[2]).write_text(text)
PY
  rm -rf -- "$source_dir"
  trap - EXIT
}

if [[ ${BASH_SOURCE[0]} == "$0" ]]; then
  main "$@"
fi
