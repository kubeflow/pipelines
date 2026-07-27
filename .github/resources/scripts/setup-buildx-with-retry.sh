#!/usr/bin/env bash

set -euo pipefail

BUILDER_NAME="${1:-kfp-buildx-${GITHUB_RUN_ID:-local}-${GITHUB_JOB:-job}-${RANDOM}}"
BUILDKIT_IMAGE="${BUILDKIT_IMAGE:-moby/buildkit:buildx-stable-1}"
BUILDKIT_MIRROR_IMAGE="${BUILDKIT_MIRROR_IMAGE:-mirror.gcr.io/moby/buildkit:buildx-stable-1}"

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then
  C_DIR="$PWD"
fi
source "${C_DIR}/helper-functions.sh"
BUILDKIT_CONFIG_DEFAULT="$(cd "${C_DIR}/.." && pwd)/buildkitd.toml"
BUILDKIT_CONFIG="${BUILDKIT_CONFIG:-$BUILDKIT_CONFIG_DEFAULT}"

pull_buildkit_image() {
  if [[ "$BUILDKIT_MIRROR_IMAGE" != "$BUILDKIT_IMAGE" ]] && \
      docker pull "$BUILDKIT_MIRROR_IMAGE" && \
      docker tag "$BUILDKIT_MIRROR_IMAGE" "$BUILDKIT_IMAGE"; then
    echo "Pulled BuildKit from $BUILDKIT_MIRROR_IMAGE"
    return 0
  fi

  echo "BuildKit mirror unavailable; trying $BUILDKIT_IMAGE"
  docker pull "$BUILDKIT_IMAGE"
}

pull_buildkit_with_backoff() {
  local max_attempts=5
  local attempt=1

  while [[ "$attempt" -le "$max_attempts" ]]; do
    if pull_buildkit_image; then
      return 0
    fi

    if [[ "$attempt" -eq "$max_attempts" ]]; then
      return 1
    fi

    local sleep_seconds=$((attempt * 20))
    echo "Retrying BuildKit mirror and Docker Hub in ${sleep_seconds}s..."
    sleep "$sleep_seconds"
    attempt=$((attempt+1))
  done
}

setup_builder() {
  docker buildx rm "$BUILDER_NAME" >/dev/null 2>&1 || true
  docker buildx create --name "$BUILDER_NAME" --driver docker-container \
    --driver-opt "image=$BUILDKIT_IMAGE" \
    --buildkitd-config "$BUILDKIT_CONFIG" --use
  docker buildx inspect "$BUILDER_NAME" --bootstrap >/dev/null
}

# Pull and locally tag the explicit mirror path first. Docker's daemon-level
# mirror can silently fall through to Docker Hub, defeating path diversity when
# a runner cannot reach registry-1.docker.io.
pull_buildkit_with_backoff
retry 3 20 setup_builder

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "builder_name=$BUILDER_NAME" >> "$GITHUB_OUTPUT"
fi

echo "Using buildx builder $BUILDER_NAME"
