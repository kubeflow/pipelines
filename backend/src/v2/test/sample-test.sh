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

REPO_NAME="${REPO_NAME:-example-test-organization/pipelines}"

if [[ -n "${PULL_NUMBER}" ]]; then
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}@refs/pull/${PULL_NUMBER}/merge#egg=kfp&subdirectory=sdk/python"

else
  export KFP_PACKAGE_PATH="git+https://github.com/${REPO_NAME}#egg=kfp&subdirectory=sdk/python"
fi

python3 -m pip install --upgrade pip
python3 -m pip install ${KFP_PACKAGE_PATH}

# The -u flag makes python output unbuffered, so that we can see real time log.
# Reference: https://stackoverflow.com/a/107717
python3 -u ./samples/v2/sample_test.py
