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

archive=${1:?image archive is required}
source_image=${2:?source image is required}
cluster=${3:?Kind cluster is required}
version_file=${4:?VERSION file is required}
version=$(cat "$version_file")
if [[ ! "$version" =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$ ]]; then
  echo "Invalid release image tag in $version_file; provide a valid VERSION file." >&2
  exit 1
fi
# Viewer pods use the default compiled into the candidate UI/controller, not
# the CI manifest's control-plane image aliases. Never pull a released viewer.
default_image="ghcr.io/kubeflow/kfp-tensorboard:${version}"
docker image load -i "$archive"
docker tag "$source_image" "$default_image"
kind --name "$cluster" load docker-image "$default_image"
docker image rm "$default_image" "$source_image"
