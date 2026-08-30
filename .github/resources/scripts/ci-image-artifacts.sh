#!/usr/bin/env bash
#
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

case "$(uname -m)" in
  x86_64)
    ARCH_NAME="amd64"
    ;;
  aarch64|arm64)
    ARCH_NAME="arm64"
    ;;
  *)
    echo "::error::Unsupported runner architecture: $(uname -m)" >&2
    exit 1
    ;;
esac

CONTROL_PLANE_IMAGE_ARTIFACTS=(
  "apiserver"
  "scheduledworkflow"
  "persistenceagent"
  "frontend"
  "metadata-writer"
  "viewer-crd-controller"
  "visualization-server"
  "cache-deployer"
  "cache-server"
  "metadata-envoy"
)

# metadata-writer and visualization-server are v1-deprecated and are not
# required by the KFP v2 ARM64 validation path. Both depend on ML Metadata /
# TFX Python packages that do not currently provide Linux ARM64 wheels.
# Keep them in the AMD64 path for existing coverage, but do not require or
# attempt to load them for ARM64.
if [[ "${ARCH_NAME}" == "arm64" ]]; then
  CONTROL_PLANE_IMAGE_ARTIFACTS=(
    "apiserver"
    "scheduledworkflow"
    "persistenceagent"
    "frontend"
    "viewer-crd-controller"
    "cache-deployer"
    "cache-server"
    "metadata-envoy"
  )
fi

RUNTIME_IMAGE_ARTIFACTS=("driver" "launcher")
ALL_CI_IMAGE_ARTIFACTS=(
  "${CONTROL_PLANE_IMAGE_ARTIFACTS[@]}"
  "${RUNTIME_IMAGE_ARTIFACTS[@]}"
  "runtime-base-images"
)
