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

# Run unit tests
cd "$source_root/components/google-cloud"
./tests/run_test.sh

# Verify package build correctly
python setup.py bdist_wheel clean

# Temporary work around for installing google.cloud.aiplatfrom from dev branch
pip3 install git+https://github.com/googleapis/python-aiplatform@dev

# Verify package can be installed and loaded correctly
WHEEL_FILE=$(find "$source_root/components/google-cloud/dist/" -name "google_cloud_pipeline_components*.whl")
pip3 install --upgrade $WHEEL_FILE
python -c "import google_cloud_pipeline_components"