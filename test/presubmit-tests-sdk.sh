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

python3 -m pip install --upgrade pip
python3 -m pip install coverage==4.5.4 coveralls==1.9.2
python3 -m pip install mock # TODO: Remove when added to requirements
python3 -m pip install -r "$source_root/sdk/python/requirements-test.txt"
python3 -m pip install --upgrade protobuf

# Testing the component authoring library
pushd "${source_root}/sdk/python"
python3 -m pip install -e .
python3 -m unittest discover --verbose --pattern '*test*.py' --top-level-directory .
popd

# Installing Argo CLI to lint all compiled workflows
"${source_root}/test/install-argo-cli.sh"

pushd "$source_root/sdk/python"
python3 -m pip install -e .
popd # Changing the current directory to the repo root for correct coverall paths
coverage run --source=kfp --append -m unittest discover --verbose --start-dir sdk/python --pattern '*test*.py'

set +x
# export COVERALLS_REPO_TOKEN=$(gsutil cat gs://ml-pipeline-test-keys/coveralls_repo_token)
set -x
REPO_BASE="https://github.com/kubeflow/pipelines"
export COVERALLS_SERVICE_NAME="prow"
export COVERALLS_SERVICE_JOB_ID=$PROW_JOB_ID
export CI_PULL_REQUEST="$REPO_BASE/pull/$PULL_NUMBER"
# coveralls
