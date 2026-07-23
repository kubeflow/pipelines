#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "Usage: $0 <context> <dockerfile> <tag> <output-tar> [build-arg ...]"
  exit 1
}

if [ $# -lt 4 ]; then
  usage
fi

CONTEXT="$1"
DOCKERFILE="$2"
IMAGE_TAG="$3"
OUTPUT_TAR="$4"
shift 4

BUILD_ARGS=("$@")

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then
  C_DIR="$PWD"
fi
source "${C_DIR}/helper-functions.sh"

build_image() {
  local build_command=(env DOCKER_BUILDKIT=1 docker build --file "$DOCKERFILE" --tag "$IMAGE_TAG")

  for build_arg in "${BUILD_ARGS[@]}"; do
    build_command+=(--build-arg "$build_arg")
  done

  build_command+=("$CONTEXT")
  "${build_command[@]}"
}

save_image() {
  rm -f "$OUTPUT_TAR"
  docker save "$IMAGE_TAG" -o "$OUTPUT_TAR"
  test -s "$OUTPUT_TAR"
}

retry 3 20 build_image
retry 3 20 save_image
