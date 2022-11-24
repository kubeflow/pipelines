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
# Using Argo to lint all compiled workflows
"${source_root}/test/install-argo-cli.sh"

pushd $source_root/sdk/python
python3 -m pip install -e .
popd # Changing the current directory to the repo root for correct coverall paths

# Install TFX 
python3 -m pip install tfx[kfp]==1.11.0

cd $source_root
git clone --branch r1.11.0 --depth 1 https://github.com/tensorflow/tfx.git

# KFP-related tests
python3 $source_root/tfx/tfx/orchestration/kubeflow/kubeflow_dag_runner_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/base_component_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/compiler_utils_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/kubeflow_v2_dag_runner_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/parameter_utils_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/pipeline_builder_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/step_builder_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/container/kubeflow_v2_entrypoint_utils_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/container/kubeflow_v2_run_executor_test.py
python3 $source_root/tfx/tfx/orchestration/kubeflow/v2/file_based_example_gen/driver_test.py
python3 $source_root/tfx/tfx/examples/penguin/penguin_pipeline_kubeflow_test.py
