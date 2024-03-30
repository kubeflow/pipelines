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
# ensure you are on the correct release branch, created by create_release_branch.sh

PKG_ROOT=$(pwd)
REPO_ROOT=$(dirname $(dirname $PKG_ROOT))
echo $REPO_ROOT

SETUPPY_VERSION=$(python -c 'from kfp.kubernetes.__init__ import __version__; print(__version__)')

if [ -z "$KFP_KUBERNETES_VERSION" ]
then
    echo "Set \$KFP_KUBERNETES_VERSION to use this script. Got empty variable."
elif [[ "$KFP_KUBERNETES_VERSION" != "$SETUPPY_VERSION" ]]
then
    echo "\$KFP_KUBERNETES_VERSION '$KFP_KUBERNETES_VERSION' does not match version in setup.py '$SETUPPY_VERSION'."
else
    echo "Got version $KFP_KUBERNETES_VERSION from env var \$KFP_KUBERNETES_VERSION"

    echo "Building package..."
    TARGET_TAR_FILE=kfp-kubernetes-$KFP_KUBERNETES_VERSION.tar.gz
    pushd "$(dirname "$0")"
    dist_dir=$(mktemp -d)
    python3 setup.py sdist --format=gztar --dist-dir "$dist_dir"
    cp "$dist_dir"/*.tar.gz $TARGET_TAR_FILE
    popd
    echo "Created package."

    echo "Testing install"
    pip install $TARGET_TAR_FILE
    INSTALLED_VERSION=$(pip list | grep kfp-kubernetes | awk '{print $2}')
    if [[ "$INSTALLED_VERSION" != "$KFP_KUBERNETES_VERSION" ]]
    then
        echo "Something went wrong! Expected version $KFP_KUBERNETES_VERSION but found version $INSTALLED_VERSION"
    else
        python -m twine upload $TARGET_TAR_FILE
    fi
fi
