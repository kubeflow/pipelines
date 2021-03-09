#!/bin/bash
#
# Copyright 2021 Google LLC
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

# Clean up generated files
rm -f "${DIR}/license_info.csv"
rm -rf "${DIR}/NOTICES"

cd "$WORK_DIR"
gh repo clone argoproj/argo-workflows
cd argo-workflows
REPO="${WORK_DIR}/argo-workflows"
git checkout "${TAG}"
go mod download

# Copy manually maintained extra license lookup table to work dir.
cp "${DIR}/license_dict.csv" "${REPO}/"
go-mod-licenses csv
cp "${REPO}/license_info.csv" "${DIR}/"
go-mod-licenses save
cp -r "${REPO}/NOTICES" "${DIR}/"
