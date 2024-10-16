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

# Create a virtual environment and activate it
python3 -m venv venv
source venv/bin/activate

python3 -m pip install --upgrade pip
python3 -m pip install coveralls==1.9.2
python3 -m pip install $(grep 'absl-py==' sdk/python/requirements-dev.txt)
python3 -m pip install $(grep 'docker==' sdk/python/requirements-dev.txt)
python3 -m pip install $(grep 'pytest==' sdk/python/requirements-dev.txt)
python3 -m pip install $(grep 'pytest-xdist==' sdk/python/requirements-dev.txt)
python3 -m pip install $(grep 'pytest-cov==' sdk/python/requirements-dev.txt)
python3 -m pip install --upgrade protobuf

python3 -m pip install sdk/python

# TODO: remove deprecated dependency; then remove --ignore arg
pytest sdk/python/kfp --ignore=sdk/python/kfp/deprecated --cov=kfp

set +x
# export COVERALLS_REPO_TOKEN=$(gsutil cat gs://ml-pipeline-test-keys/coveralls_repo_token)
set -x
REPO_BASE="https://github.com/kubeflow/pipelines"
export COVERALLS_SERVICE_NAME="prow"
export COVERALLS_SERVICE_JOB_ID=$PROW_JOB_ID
export CI_PULL_REQUEST="$REPO_BASE/pull/$PULL_NUMBER"
# coveralls

# Deactivate the virtual environment
deactivate
