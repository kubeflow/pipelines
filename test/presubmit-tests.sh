#!/bin/bash

# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

echo presubmit test starts...

ls $(dirname $0)

gcloud container clusters get-credentials spinner --zone us-west1-a --project ml-pipeline-test
kubectl config set-context $(kubectl config current-context) --namespace=default

echo submitting argo job...
argo submit $(dirname $0)/integration_test_gke.yaml -p commit-sha="${PULL_PULL_SHA}"
echo argo job submitted successfully
