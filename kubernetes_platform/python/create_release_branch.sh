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

PKG_ROOT=$(pwd)
REPO_ROOT=$(dirname $(dirname $PKG_ROOT))
echo $REPO_ROOT

echo "Generating Python protobuf code..."
pushd "$PKG_ROOT/.."
make clean-python python
popd

SETUPPY_VERSION=$(python -c 'from kfp.kubernetes.__init__ import __version__; print(__version__)')

if [ -z "$KFP_KUBERNETES_VERSION" ]
then
    echo "Set \$KFP_KUBERNETES_VERSION to use this script. Got empty variable."
elif [[ "$KFP_KUBERNETES_VERSION" != "$SETUPPY_VERSION" ]]
then
    echo "\$KFP_KUBERNETES_VERSION '$KFP_KUBERNETES_VERSION' does not match version in setup.py '$SETUPPY_VERSION'."
else
    echo "Got version $KFP_KUBERNETES_VERSION from env var \$KFP_KUBERNETES_VERSION"

    BRANCH_NAME=kfp-kubernetes-$KFP_KUBERNETES_VERSION
    echo "Creating release branch $BRANCH_NAME..."
    git checkout -b $BRANCH_NAME

    echo "Moving .readthedocs.yml to root..."
    # required for this branch because readthedocs only supports on docs build per repo
    # and the default is currently the KFP SDK
    # GCPC uses this pattern in this repo as well
    mv $PKG_ROOT/docs/.readthedocs.yml $REPO_ROOT/.readthedocs.yml
    rm $REPO_ROOT/kubernetes_platform/.gitignore

    git add $PKG_ROOT/docs/.readthedocs.yml
    git add $REPO_ROOT/.readthedocs.yml
    git add $REPO_ROOT/kubernetes_platform/.gitignore
    git add $REPO_ROOT/*_pb2.py

    echo "Next steps:"
    echo "1. Inspect and commit the modified files."
    echo "2. Push branch using 'git push --set-upstream upstream $BRANCH_NAME'"
fi
