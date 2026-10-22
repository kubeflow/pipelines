#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

# Exercise rebuilt images against a disposable source copy as a non-root user.
set -euo pipefail
if [[ $# != 2 || $1 != amd64 ]]; then
  echo "Usage: $0 amd64 <output-directory>" >&2
  exit 1
fi
root=$(git rev-parse --show-toplevel)
mkdir -p "$2"
output=$(cd "$2" && pwd)
source_dir=$(mktemp -d)
trap 'rm -rf -- "$source_dir"' EXIT
git -C "$root" archive HEAD | tar -x -C "$source_dir"
chmod -R a+rwX "$source_dir"
index=0
for image in kfp-api-generator:ci kfp-release:ci; do
  test "$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$image")" = linux/amd64
  docker run --rm --interactive --user 65532:65532 \
    --env HOME=/tmp/kfp-smoke-home --env LC_ALL=C.UTF-8 --env TZ=UTC \
    --mount "type=bind,source=$source_dir,target=/go/src/github.com/kubeflow/pipelines" \
    --workdir /go/src/github.com/kubeflow/pipelines "$image" bash -se <<'CONTAINER'
set -euo pipefail
mkdir -p "$HOME"
test "$(id -u)" != 0
go version
uv --version
python3 -c 'import setuptools'
for version in v1beta1 v2beta1; do
  export API_VERSION="$version"
  backend/api/hack/generator.sh
  backend/api/build_kfp_server_api_python_package.sh
  test -s "backend/api/$version/go_client/run.pb.go"
  test -s "backend/api/$version/swagger/kfp_api_single_file.swagger.json"
  uv build --out-dir "backend/api/$version/python_http_client/dist" "backend/api/$version/python_http_client"
done
make -C api generate
make -C kubernetes_platform generate-go-in-container
python3 api/v2alpha1/python/generate_proto.py
python3 kubernetes_platform/python/generate_proto.py
for package in api/v2alpha1/python kubernetes_platform/python sdk/python; do
  uv build --out-dir "$package/dist" "$package"
  compgen -G "$package/dist/*.whl" >/dev/null
  compgen -G "$package/dist/*.tar.gz" >/dev/null
done
CONTAINER
  python3 - "$source_dir" > "$output/api-sources-$index.sha256" <<'PY'
import hashlib
from pathlib import Path
import sys
root = Path(sys.argv[1])
for version in ('v1beta1', 'v2beta1'):
    for directory in ('go_client', 'go_http_client', 'swagger'):
        for path in sorted((root / 'backend/api' / version / directory).rglob('*')):
            if path.is_file():
                print(hashlib.sha256(path.read_bytes()).hexdigest(), path.relative_to(root))
PY
  index=$((index + 1))
done
diff -u "$output/api-sources-0.sha256" "$output/api-sources-1.sha256"
docker run --rm --interactive --user 65532:65532 \
  --env HOME=/tmp/kfp-smoke-home --env LC_ALL=C.UTF-8 \
  --mount "type=bind,source=$source_dir,target=/go/src/github.com/kubeflow/pipelines" \
  --workdir /go/src/github.com/kubeflow/pipelines kfp-release:ci bash -se <<'CONTAINER'
set -euo pipefail
mkdir -p "$HOME"
node --version
npm --version
mkdir /tmp/npm-smoke
cd /tmp/npm-smoke
printf '{"name":"kfp-tooling-smoke","version":"1.0.0","private":true}\n' > package.json
npm install --ignore-scripts --no-audit --no-fund ip-address@10.7.4
node -e 'const {Address4} = require("ip-address"); if (!Address4.isValid("127.0.0.1")) process.exit(1)'
npm pack --ignore-scripts
test -s kfp-tooling-smoke-1.0.0.tgz
cd /go/src/github.com/kubeflow/pipelines
python3 manifests/kustomize/hack/yq_release_test.py
manifests/kustomize/hack/release.sh 2.18.0-smoke
test "$(yq eval '.data.appVersion' manifests/kustomize/base/installs/generic/pipeline-install-config.yaml)" = 2.18.0-smoke
git-cliff --version
CONTAINER
