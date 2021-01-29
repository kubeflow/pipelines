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

# TODO(#5051) Unpin pip version once we figure out how to make the new dependency resolver in pip 20.3+ work in our case.
python3 -m pip install --upgrade pip==20.2.3
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
# Compile and setup bazel for compiling the protos
# Instruction from https://docs.bazel.build/versions/master/install-ubuntu.html
curl -sSL https://github.com/bazelbuild/bazel/releases/download/3.4.1/bazel-3.4.1-installer-linux-x86_64.sh -o bazel_installer.sh
chmod +x bazel_installer.sh
./bazel_installer.sh

# Install TFX from head
cd $source_root
git clone --depth 1 https://github.com/tensorflow/tfx.git
cd $source_root/tfx
python3 -m pip install . --upgrade --use-feature=2020-resolver \
  --extra-index-url https://pypi-nightly.tensorflow.org/simple

# Three KFP-related unittests
cd $source_root/tfx/tfx/orchestration/kubeflow
python3 kubeflow_dag_runner_test.py
python3 base_component_test.py
cd $source_root/tfx/tfx/orchestration/kubeflow/v2
python3 compiler_utils_test.py
python3 kubeflow_v2_dag_runner_test.py
python3 parameter_utils_test.py
python3 pipeline_builder_test.py
python3 step_builder_test.py
cd $source_root/tfx/tfx/orchestration/kubeflow/v2/container
python3 kubeflow_v2_entrypoint_utils_test.py
python3 kubeflow_v2_run_executor_test.py
cd $source_root/tfx/tfx/orchestration/kubeflow/v2/file_based_example_gen
python3 driver_test.py
cd $source_root/tfx/tfx/examples/chicago_taxi_pipeline
python3 taxi_pipeline_kubeflow_gcp_test.py
python3 taxi_pipeline_kubeflow_local_test.py
