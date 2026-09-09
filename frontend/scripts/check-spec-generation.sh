#!/bin/bash

# Run from frontend after installing its dependencies and protoc.
set -euo pipefail

# Keep generated imports beneath frontend so TypeScript resolves node_modules.
generated_dir=$(mktemp -d ./spec-generation.XXXXXX)
trap 'rm -rf -- "$generated_dir"' EXIT

PROTO_OUT_DIR="$generated_dir/pipeline" npm run build:pipeline-spec
PROTO_OUT_DIR="$generated_dir/platform" npm run build:platform-spec:kubernetes-platform

./node_modules/.bin/tsc --ignoreConfig --noEmit --strict --skipLibCheck \
  --target es2015 --lib es2020,dom --module commonjs --esModuleInterop \
  "$generated_dir/pipeline/pipeline_spec.ts" \
  "$generated_dir/platform/kubernetes_executor_config.ts"
