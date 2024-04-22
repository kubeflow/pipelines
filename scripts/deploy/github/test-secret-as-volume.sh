#!/bin/bash
#
# Copyright 2023 kubeflow.org
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

KUBEFLOW_NS="${KUBEFLOW_NS:-kubeflow}"

C_DIR="${BASH_SOURCE%/*}"
if [[ ! -d "$C_DIR" ]]; then C_DIR="$PWD"; fi
source "${C_DIR}/test-pipeline.sh"

# need kfp-kubernetes for this test case
# unfortunately, we can't install it from kubernetes_platform/python
pip install kfp-kubernetes

# create the secret
kubectl create secret -n "$KUBEFLOW_NS" generic "user-gcp-sa" --from-literal="type=service_account" || true

RESULT=0
run_test_case "secret-volume" "samples/v2/pipeline_with_secret_as_volume.py" "SUCCEEDED" 5 || RESULT=$?

# remove secret after the test finishes
kubectl delete secret -n "$KUBEFLOW_NS" "user-gcp-sa"

STATUS_MSG=PASSED
if [[ "$RESULT" -ne 0 ]]; then
  STATUS_MSG=FAILED
fi
