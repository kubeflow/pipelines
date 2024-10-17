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

# This script updates argo NOTICES folder.
# Usage: ./imp-update-notices.sh
# It can be run anywhere.

set -ex

# Get this bash script's dir.
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
WORK_DIR="$(mktemp -d)"
TAG="$(cat "${DIR}/VERSION")"

echo "Checking required tools"
echo "go"
which go >/dev/null || (echo "go not found in PATH" && exit 1)
go_path=$(go env GOPATH)
echo "$PATH" | grep "${go_path}/bin" > /dev/null || ( \
    echo "\$GOPATH/bin: ${go_path}/bin should be in PATH"; \
    echo "https://golang.org/cmd/go/#hdr-GOPATH_and_Modules"; \
    exit 1)
which go-licenses >/dev/null || (echo "go-licenses not found in PATH" && exit 1)

# Clean up generated files
rm -rf "${DIR}/NOTICES"

cd "${WORK_DIR}"
git clone https://github.com/argoproj/argo-workflows
cd argo-workflows
REPO="${WORK_DIR}/argo-workflows"
git checkout "${TAG}"
go mod download
make dist/workflow-controller dist/argoexec

# Copy manually maintained extra license lookup table to work dir.
mkdir -p "${DIR}/NOTICES/workflow-controller"
mkdir -p "${DIR}/NOTICES/argoexec"
echo "Temporary dir:"
echo "${WORK_DIR}"

go-licenses csv ./cmd/workflow-controller > licenses-workflow-controller.csv
cp licenses-workflow-controller.csv "${DIR}/licenses-workflow-controller.csv"
go-licenses csv ./cmd/argoexec > licenses-argoexec.csv
cp licenses-argoexec.csv "${DIR}/licenses-argoexec.csv"
go-licenses save ./cmd/workflow-controller --save_path "${DIR}/NOTICES/workflow-controller" --force
go-licenses save ./cmd/argoexec --save_path "${DIR}/NOTICES/argoexec" --force
