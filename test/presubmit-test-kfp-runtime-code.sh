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

set -ex
source_root=$(pwd)

pip install --upgrade pip
pip install pyyaml
pip install $(grep 'absl-py==' sdk/python/requirements-dev.txt)

# precautionarilty uninstall typing-extensions, in case any of the test libs
# installed require this dep. we want to test that the kfp sdk installs it, so
# it cannot be present in the environment prior to test execution.
# we'd rather tests fail to execute (false positive failure) because a test
# lib was missing its dependency on typing-extensions than get a false
# negative from the actual kfp sdk test because typing-extensions was already
# present in the environment.
pip uninstall typing-extensions -y

# run with unittest because pytest requires typing-extensions
python -m unittest discover -s sdk/runtime_tests -p '*_test.py'
