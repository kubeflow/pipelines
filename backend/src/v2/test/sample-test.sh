#!/bin/bash
#
# Copyright 2021 The Kubeflow Authors
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
source_root=$(pwd)

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
cd "${DIR}"
source "${DIR}/scripts/ci-env.sh"

# Install required packages from commit
python3 -m pip install --upgrade pip

# TODO: remove deprecated dependency
python3 -m pip install -r $source_root/sdk/python/requirements-deprecated.txt
python3 -m pip install $source_root/sdk/python

# Install KFP server API from commit.
cp -r $source_root/backend/api/v1beta1/python_http_client /python_http_client
python3 -m pip install /python_http_client
# Run sample test
ENV_PATH=kfp-ci.env make
