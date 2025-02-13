#!/bin/bash
#
# Copyright 2021 The Kubeflow Authors
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

# Fail the entire script when any command fails.
set -ex
NAMESPACE="${NAMESPACE:-kubeflow}"
TEST_CLUSTER="${TEST_CLUSTER:-kfp-standalone-1}"
REGION="${REGION:-us-central1}"
PROJECT="${PROJECT:-kfp-ci}"
# The current directory is /home/prow/go/src/github.com/kubeflow/pipelines
# 1. install go in /home/prow/go1.20.4
cd /home/prow
mkdir go1.21.4
cd go1.21.4
curl -LO https://dl.google.com/go/go1.20.4.linux-amd64.tar.gz
tar -xf go1.21.4.linux-amd64.tar.gz
export PATH="/home/prow/go1.20.4/go/bin:${PATH}"
cd /home/prow/go/src/github.com/kubeflow/pipelines/backend/src/v2
# 2. Check go modules are tidy
# Reference: https://github.com/golang/go/issues/27005#issuecomment-564892876
go mod download
go mod tidy
git diff --exit-code -- go.mod go.sum || (echo "go modules are not tidy, run 'go mod tidy'." && exit 1)

# Note, for tests that use metadata grpc api, port-forward it locally in a separate terminal by:
if [[ ! -z "${GOOGLE_APPLICATION_CREDENTIALS}" ]]; then
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"
fi
gcloud container clusters get-credentials "$TEST_CLUSTER" --region "$REGION" --project "$PROJECT"

function cleanup() {
  echo "killing kubectl port forward before exit"
  kill "$PORT_FORWARD_PID"
}
trap cleanup EXIT
kubectl port-forward svc/metadata-grpc-service 8080:8080 -n "$NAMESPACE" & PORT_FORWARD_PID=$!
# wait for kubectl port forward
sleep 10
go test -v -cover ./...

# TODO(zijianjoy): re-enable license check for v2 images
# verify licenses are up-to-date,  because all license updates must be reviewed by a human.
# ../hack/install-go-licenses.sh
# make license-launcher
# git diff --exit-code -- third_party/licenses/launcher.csv || (echo "v2/third_party/licenses/launcher.csv is outdated, refer to https://github.com/kubeflow/pipelines/tree/master/v2#update-licenses for update instructions.")
