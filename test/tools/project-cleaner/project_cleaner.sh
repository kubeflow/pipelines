#!/bin/bash
#
# Copyright 2018 Google LLC
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

SCRIPT_DIR=$(dirname "${BASH_SOURCE}")
pushd "${SCRIPT_DIR}"

# Required as this script runs in Prow using kubekins-e2e image. That image uses go-1.12 version
# as a result below packages specified in go.mod file are not discovered.
go get -u "google.golang.org/api/container/v1"
go get -u "gopkg.in/yaml.v2"

echo "Building project cleaner tool"
go build .

echo "Executing project cleaner"
./project-cleaner --resource_spec resource_spec.yaml
popd
