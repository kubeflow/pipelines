#!/usr/bin/env bash

set -euo pipefail

BUILDER_NAME="${1:-kfp-buildx-${GITHUB_RUN_ID:-local}-${GITHUB_JOB:-job}-${RANDOM}}"

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then
  C_DIR="$PWD"
fi
source "${C_DIR}/helper-functions.sh"

setup_builder() {
  docker buildx rm "$BUILDER_NAME" >/dev/null 2>&1 || true
  docker buildx create --name "$BUILDER_NAME" --driver docker-container --use
  docker buildx inspect "$BUILDER_NAME" --bootstrap >/dev/null
}

retry 3 20 setup_builder

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "builder_name=$BUILDER_NAME" >> "$GITHUB_OUTPUT"
fi

echo "Using buildx builder $BUILDER_NAME"
