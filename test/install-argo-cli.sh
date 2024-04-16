#!/bin/bash
#
# Copyright 2021 The Kubeflow Authors
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

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
REPO_ROOT="${DIR}/.."
ARGO_VERSION="$(cat ${REPO_ROOT}/third_party/argo/VERSION)"
# ARGO_VERSION=v3.4.16
OS=${OS:-"linux-amd64"}

# if argo is not installed
if ! which argo; then
  echo "install argo"
  curl -sLO "https://github.com/argoproj/argo-workflows/releases/download/${ARGO_VERSION}/argo-${OS}.gz"
  gunzip "argo-${OS}.gz"
  chmod +x "argo-${OS}"
  mv "argo-${OS}" /usr/local/bin/argo
  argo version
fi
