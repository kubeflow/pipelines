#!/bin/bash -ex
# Copyright 2020 Kubeflow Pipelines contributors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

source_root=$(pwd)
SETUP_ENV="${SETUP_ENV:-true}"

if [ "${SETUP_ENV}" = "true" ]; then
  # Sync all dependencies using uv
  uv sync --extra ci

  # Install protobuf-compiler for proto generation
  apt-get update && apt-get install -y protobuf-compiler

  # Generate proto files
  pushd api
  make clean python
  popd

  # Install workspace packages in editable mode
  uv pip install -e sdk/python -e api/v2alpha1/python -e kubernetes_platform/python -e backend/api/v2beta1/python_http_client
fi

# Test loading all component.yaml definitions
"$source_root/components/test_load_all_components.sh"
