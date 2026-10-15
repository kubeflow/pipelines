#!/usr/bin/env bash

# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

DOCKER_DAEMON_CONFIG="${DOCKER_DAEMON_CONFIG:-/etc/docker/daemon.json}"
DOCKER_HUB_MIRROR="${DOCKER_HUB_MIRROR:-https://mirror.gcr.io}"

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then
  C_DIR="$PWD"
fi
source "${C_DIR}/helper-functions.sh"

# Containerized runners may expose Docker without a local systemd service.
# Their daemon configuration belongs to the runner provider.
if ! command -v systemctl >/dev/null 2>&1 || ! systemctl is-active --quiet docker; then
  retry 10 2 docker info
  echo "Docker has no active local systemd service; retaining provider-managed registry configuration."
  exit 0
fi

privilege_command=()
if [[ "${EUID}" -ne 0 ]]; then
  privilege_command=(sudo)
fi

# Use the host interpreter: setup-python may require LD_LIBRARY_PATH, which
# sudo strips. This host configuration helper uses only the standard library.
configuration_status=$("${privilege_command[@]}" /usr/bin/python3 \
  "${C_DIR}/configure_docker_registry_mirror.py" \
  --config "$DOCKER_DAEMON_CONFIG" \
  --mirror "$DOCKER_HUB_MIRROR")

if [[ "$configuration_status" == "changed" ]]; then
  "${privilege_command[@]}" systemctl restart docker
elif [[ "$configuration_status" != "unchanged" ]]; then
  echo "Unexpected Docker mirror configuration status: $configuration_status" >&2
  exit 1
fi

docker_information=$(retry 10 2 docker info)
if ! grep -Fq "$DOCKER_HUB_MIRROR" <<< "$docker_information"; then
  echo "Docker did not report the configured mirror: $DOCKER_HUB_MIRROR" >&2
  exit 1
fi

echo "Docker Hub mirror configured: $DOCKER_HUB_MIRROR"
