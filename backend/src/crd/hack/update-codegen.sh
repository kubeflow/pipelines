#!/bin/bash
#
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

# This shell is used to auto generate some useful tools for k8s, such as lister,
# informer, deepcopy, defaulter and so on.

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})
echo "SCRIPT_ROOT is $SCRIPT_ROOT"
CODEGEN_PKG=${SCRIPT_ROOT}/../../../../vendor/k8s.io/code-generator
echo "CODEGEN_PKG is $CODEGEN_PKG"

${CODEGEN_PKG}/generate-groups.sh "deepcopy" \
  github.com/kubeflow/pipelines/backend/src/crd/pkg/client \
  github.com/kubeflow/pipelines/backend/src/crd/pkg/apis \
  "scheduledworkflow:v1beta1 viewer:v1beta1" \
  --go-header-file ${SCRIPT_ROOT}/custom-boilerplate.go.txt

${CODEGEN_PKG}/generate-groups.sh "client,informer,lister" \
  github.com/kubeflow/pipelines/backend/src/crd/pkg/client \
  github.com/kubeflow/pipelines/backend/src/crd/pkg/apis \
  scheduledworkflow:v1beta1 \
  --go-header-file ${SCRIPT_ROOT}/custom-boilerplate.go.txt
