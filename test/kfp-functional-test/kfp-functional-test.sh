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

if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  # activating the service account
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi
source_root=$(pwd)

# TODO(#4853) Skipping pip 20.3 due to a bad version resolution logic.
python3 -m pip install --upgrade pip!=20.3.*
python3 -m pip install -r "$source_root/test/kfp-functional-test/requirements.txt"

TEST_RESULT_BUCKET=ml-pipeline-test
TEST_RESULT_FOLDER=kfp-functional-e2e-test
TEST_RESULTS_GCS_DIR=gs://${TEST_RESULT_BUCKET}/${TEST_RESULT_FOLDER}
HOST=https://5e682a68692ffad5-dot-datalab-vm-staging.googleusercontent.com

mkdir -p "$source_root"/${TEST_RESULT_FOLDER}

cd "$source_root"/test/kfp-functional-test

python3 run_kfp_functional_test.py --result_dir "$source_root"/${TEST_RESULT_FOLDER} --host ${HOST} --gcs_dir ${TEST_RESULTS_GCS_DIR}

rm -rf "${source_root:?}"/${TEST_RESULT_FOLDER}