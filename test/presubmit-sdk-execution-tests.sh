#!/bin/bash -ex
# Copyright 2022 Kubeflow Pipelines contributors
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
python3 -m pip install $source_root/sdk/python
apt-get update && apt-get install -y protobuf-compiler
pushd api
make clean python
popd
python3 -m pip install api/v2alpha1/python
python3 -m pip install -r $source_root/test/sdk-execution-tests/requirements.txt


export KFP_ENDPOINT="https://$(curl https://raw.githubusercontent.com/kubeflow/testing/master/test-infra/kfp/endpoint)"
export TIMEOUT_SECONDS=2700
pytest $source_root/test/sdk-execution-tests/sdk_execution_tests.py --asyncio-task-timeout $TIMEOUT_SECONDS
