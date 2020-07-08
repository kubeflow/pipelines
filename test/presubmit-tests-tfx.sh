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
python3 -m pip install -r "$source_root/sdk/python/requirements.txt"
# Additional dependencies
#pip3 install coverage==4.5.4 coveralls==1.9.2 six>=1.13.0
# Sample test infra dependencies
pip3 install minio
pip3 install junit_xml
# Using Argo to lint all compiled workflows
export LOCAL_BIN="${HOME}/.local/bin"
mkdir -p "$LOCAL_BIN"
export PATH="${PATH}:$LOCAL_BIN" # Unnecessary - Travis already has it in PATH
wget --quiet -O "${LOCAL_BIN}/argo" https://github.com/argoproj/argo/releases/download/v2.4.3/argo-linux-amd64 && chmod +x "${LOCAL_BIN}/argo"

pushd $source_root/sdk/python
python3 -m pip install -e .
popd # Changing the current directory to the repo root for correct coverall paths

# Test against TFX
# Compile and setup protobuf
PROTOC_ZIP=protoc-3.7.1-linux-x86_64.zip
curl -OL -sS https://github.com/protocolbuffers/protobuf/releases/download/v3.7.1/$PROTOC_ZIP
unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
rm -f $PROTOC_ZIP

# Install TFX from head
cd $source_root
git clone https://github.com/tensorflow/tfx.git
cd $source_root/tfx
#pip3 install --upgrade pip
#pip3 install --upgrade 'numpy>=1.16,<1.17'
#set -x
#set -e
#python3 setup.py bdist_wheel
#WHEEL_PATH=$(find dist -name "tfx-*.whl")
#python3 -m pip install "${WHEEL_PATH}" --upgrade
python3 -m pip install . --upgrade
#set +e
#set +x

# Three KFP-related unittests
cd $source_root/tfx/tfx/orchestration/kubeflow
python3 kubeflow_dag_runner_test.py
cd $source_root/tfx/tfx/examples/chicago_taxi_pipeline
python3 taxi_pipeline_kubeflow_gcp_test.py
python3 taxi_pipeline_kubeflow_local_test.py
