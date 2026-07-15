#!/usr/bin/env bash

set -euo pipefail

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then
  C_DIR="$PWD"
fi
source "${C_DIR}/helper-functions.sh"

retry 3 20 docker buildx build "$@"
