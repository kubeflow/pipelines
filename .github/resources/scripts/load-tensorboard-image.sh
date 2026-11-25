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
default_image=${4:?Compiled default image is required}
if [[ ! "$default_image" =~ ^ghcr.io/kubeflow/kfp-tensorboard:[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$ ]]; then
  echo "Invalid TensorBoard default; use the controller/UI image printed by hack/update_tensorboard_image.py --print-default." >&2
  exit 1
fi
# Load the candidate under its compiled default; never pull a released viewer.
docker image load -i "$archive"
docker tag "$source_image" "$default_image"
kind --name "$cluster" load docker-image "$default_image"
docker image rm "$default_image" "$source_image"
