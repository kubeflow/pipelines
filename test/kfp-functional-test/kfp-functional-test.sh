#!/bin/bash -ex
# Copyright 2020 Google LLC
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
source_root="$(pwd)"

# TODO(#4853) Skipping pip 20.3 due to a bad version resolution logic.
python3 -m pip install --upgrade pip!=20.3.*
python3 -m pip install -r "${source_root}/test/kfp-functional-test/requirements.txt"

HOST=https://5e682a68692ffad5-dot-datalab-vm-staging.googleusercontent.com

python3 "${source_root}/test/kfp-functional-test/run_kfp_functional_test.py" --host "${HOST}"
