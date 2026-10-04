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

# The CI workspace already contains HEAD, so test upgrades in a clean environment.
UPGRADE_TEST_DIR=$(mktemp -d)
trap 'rm -rf "$UPGRADE_TEST_DIR"' EXIT
python3 -m venv "$UPGRADE_TEST_DIR/venv"
UPGRADE_PYTHON="$UPGRADE_TEST_DIR/venv/bin/python"

# Install the latest released version of KFP from PyPI
"$UPGRADE_PYTHON" -m pip install --upgrade pip
"$UPGRADE_PYTHON" -m pip install kfp
LATEST_KFP_SDK_RELEASE=$("$UPGRADE_PYTHON" -m pip show kfp | grep "Version:" | awk '{print $2}' | awk '{$1=$1};1')
echo "Installed latest KFP SDK version: $LATEST_KFP_SDK_RELEASE"

# Build and install workspace packages from source using uv
# Generate proto files (requires Docker)
pushd api
make python
popd

# Build all workspace packages
uv build --package kfp-pipeline-spec --out-dir "$UPGRADE_TEST_DIR/dist"
uv build --package kfp-server-api --out-dir "$UPGRADE_TEST_DIR/dist"
uv build --package kfp --out-dir "$UPGRADE_TEST_DIR/dist"

# Resolve all local wheels together so unpublished versions never require PyPI.
"$UPGRADE_PYTHON" -m pip install "$UPGRADE_TEST_DIR"/dist/*.whl --force-reinstall

# HEAD will only be different than latest for a release PR
HEAD_KFP_SDK_VERSION=$("$UPGRADE_PYTHON" -m pip show kfp | grep "Version:" | awk '{print $2}')
echo "Successfully upgraded to KFP SDK version @ HEAD: $HEAD_KFP_SDK_VERSION"

"$UPGRADE_PYTHON" -c 'import kfp'
echo "Successfully ran 'import kfp' @ HEAD: $HEAD_KFP_SDK_VERSION"
