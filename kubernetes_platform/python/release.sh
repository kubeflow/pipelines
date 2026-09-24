#!/bin/bash -ex
#
# Copyright 2023 The Kubeflow Authors
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

# run from within ./kubernetes_platform/python
# set environment variable KFP_KUBERNETES_VERSION
# ensure you are on the correct release branch, created by the kfpr docs branch step

PKG_ROOT=$(pwd)
REPO_ROOT=$(dirname $(dirname $PKG_ROOT))
echo $REPO_ROOT

# Read the version without importing SDK/protobuf dependencies or requiring GNU grep.
PACKAGE_VERSION=$(sed -n "s/^__version__ = ['\"]\([^'\"]*\)['\"].*/\1/p" kfp/kubernetes/__init__.py)

if [ -z "$KFP_KUBERNETES_VERSION" ]
then
    echo "Set \$KFP_KUBERNETES_VERSION to use this script. Got empty variable."
elif [[ "$KFP_KUBERNETES_VERSION" != "$PACKAGE_VERSION" ]]
then
    echo "\$KFP_KUBERNETES_VERSION '$KFP_KUBERNETES_VERSION' does not match version in __init__.py '$PACKAGE_VERSION'."
else
    echo "Got version $KFP_KUBERNETES_VERSION from env var \$KFP_KUBERNETES_VERSION"

    echo "Building package..."
    TARGET_TAR_FILE=kfp-kubernetes-$KFP_KUBERNETES_VERSION.tar.gz
    pushd "$(dirname "$0")"
    dist_dir=$(mktemp -d)
    python3 -m build --sdist --wheel --outdir "$dist_dir"
    cp "$dist_dir"/*.tar.gz $TARGET_TAR_FILE
    popd
    echo "Created package."

    echo "Testing install"
    test_venv=$(mktemp -d)/venv
    python3 -m venv "$test_venv"
    "$test_venv/bin/pip" install "$TARGET_TAR_FILE"
    INSTALLED_VERSION=$("$test_venv/bin/pip" list | grep kfp-kubernetes | awk '{print $2}')
    rm -rf "$test_venv"
    if [[ "$INSTALLED_VERSION" != "$KFP_KUBERNETES_VERSION" ]]
    then
        echo "Something went wrong! Expected version $KFP_KUBERNETES_VERSION but found version $INSTALLED_VERSION"
    else
        uvx --python 3.12 --from twine==7.0.0 twine check "$TARGET_TAR_FILE" &&
            uvx --python 3.12 --from twine==7.0.0 twine upload "$TARGET_TAR_FILE"
    fi
fi
