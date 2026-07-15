#!/usr/bin/env bash

set -euo pipefail

usage() {
  echo "Usage: $0 <primary-image> [fallback-image ...]"
  exit 1
}

if [ $# -lt 1 ]; then
  usage
fi

PRIMARY_IMAGE="$1"
shift
FALLBACK_IMAGES=("$@")

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then
  C_DIR="$PWD"
fi
source "${C_DIR}/helper-functions.sh"

pull_image() {
  local image="$1"

  echo "Pulling ${image} with retries..."
  retry 3 20 docker pull "$image"
}

if pull_image "$PRIMARY_IMAGE"; then
  exit 0
fi

for fallback_image in "${FALLBACK_IMAGES[@]}"; do
  echo "Falling back to ${fallback_image} for ${PRIMARY_IMAGE}"
  if pull_image "$fallback_image"; then
    if [ "$fallback_image" != "$PRIMARY_IMAGE" ]; then
      docker tag "$fallback_image" "$PRIMARY_IMAGE"
    fi
    exit 0
  fi
done

echo "ERROR: Failed to pull ${PRIMARY_IMAGE} and all configured fallbacks."
exit 1
