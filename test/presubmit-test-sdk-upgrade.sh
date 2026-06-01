#!/bin/bash -ex
# Copyright 2023 Kubeflow Pipelines contributors
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

set -ex

# This test verifies that upgrading from the latest PyPI release to HEAD works correctly.
# We intentionally use pip (not uv) for the initial install and final upgrade to simulate
# the real-world upgrade path that users would experience. uv is only used for building
# the packages from source.

# Install the latest released version of KFP from PyPI
python3 -m pip install --upgrade pip
python3 -m pip install kfp
LATEST_KFP_SDK_RELEASE=$(python3 -m pip show kfp | grep "Version:" | awk '{print $2}' | awk '{$1=$1};1')
echo "Installed latest KFP SDK version: $LATEST_KFP_SDK_RELEASE"

# Build and install workspace packages from source using uv
# Generate proto files (requires Docker)
pushd api
make python
popd

# Build all workspace packages
uv build --package kfp-pipeline-spec
uv build --package kfp-server-api
uv build --package kfp

# Install the built packages (simulates upgrade from PyPI to HEAD)
python3 -m pip install dist/kfp_pipeline_spec-*.whl --force-reinstall
python3 -m pip install dist/kfp_server_api-*.whl --force-reinstall
python3 -m pip install dist/kfp-*.whl --force-reinstall

# HEAD will only be different than latest for a release PR
HEAD_KFP_SDK_VERSION=$(python3 -m pip show kfp | grep "Version:" | awk '{print $2}')
echo "Successfully upgraded to KFP SDK version @ HEAD: $HEAD_KFP_SDK_VERSION"

python3 -c 'import kfp'
echo "Successfully ran 'import kfp' @ HEAD: $HEAD_KFP_SDK_VERSION"
