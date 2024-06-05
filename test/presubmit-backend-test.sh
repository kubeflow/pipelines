#!/bin/bash
#
# Copyright 2020 The Kubeflow Authors
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

# Add installed go binaries to PATH.
export PATH="${PATH}:$(go env GOPATH)/bin"

# 1. Check go modules are tidy
# Reference: https://github.com/golang/go/issues/27005#issuecomment-564892876
go mod download
go mod tidy
git diff --exit-code -- go.mod go.sum || (echo "go modules are not tidy, run 'go mod tidy'." && exit 1)

# 2. Run tests in the backend directory
go test -v -cover ./backend/...

# 3. Check for forbidden go licenses
./hack/install-go-licenses.sh
go-licenses check ./backend/src/apiserver ./backend/src/cache ./backend/src/agent/persistence ./backend/src/crd/controller/scheduledworkflow ./backend/src/crd/controller/viewer
