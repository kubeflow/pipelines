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

if [ $(id -u) = 0 ]; then
  apt-get update && apt-get install -y protobuf-compiler
  # rust needed for transitive deps in dev extras on Python:3.12
  apt-get install rustc -y
else
  sudo apt-get update && sudo apt-get install -y protobuf-compiler
  # rust needed for transitive deps in dev extras on Python:3.12
  sudo apt-get install rustc -y
fi

pip install --upgrade pip
pip install wheel
pip install sdk/python

pushd api
make clean python
popd
python3 -m pip install api/v2alpha1/python
pip install api/v2alpha1/python

# generate Python proto code from source
pushd "$source_root/kubernetes_platform"
make clean python
popd

pip install -e "$source_root/kubernetes_platform/python[dev]"
pytest "$source_root/kubernetes_platform/python/test" -n auto
