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

source_root=$(pwd)

python3 -m pip install --upgrade pip
python3 -m pip install $source_root/sdk/python
apt-get update && apt-get install -y protobuf-compiler
# install kfp-pipeline-spec from source
pushd api
make clean python
popd
python3 -m pip install api/v2alpha1/python

# generate kfp-kubernetes proto files from source
pushd "$source_root/kubernetes_platform"
make clean python
popd

# install kfp-kubernetes from source
# rust needed for transitive deps in dev extras on Python:3.12
apt-get install rustc -y
pip install -e $source_root/kubernetes_platform/python[dev]

pip install -r $source_root/test/kfp-kubernetes-execution-tests/requirements.txt

export KFP_ENDPOINT="https://$(curl https://raw.githubusercontent.com/kubeflow/testing/master/test-infra/kfp/endpoint)"
export TIMEOUT_SECONDS=2700
pytest $source_root/test/kfp-kubernetes-execution-tests/sdk_execution_tests.py --asyncio-task-timeout $TIMEOUT_SECONDS
