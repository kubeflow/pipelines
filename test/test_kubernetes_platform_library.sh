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

pip install --upgrade pip
pip install wheel

pushd "$source_root/kubernetes_platform"
make clean python
popd
## remove
pushd "$source_root/api"
make clean python
pip install "$source_root/api/v2alpha1/python"
popd
##
pip install -e "$source_root/sdk/python"
pip install -e "$source_root/kubernetes_platform/python[dev]"
pytest "$source_root/kubernetes_platform/python/kfp/test"
