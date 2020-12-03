#!/bin/bash
#
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

set -e

echo "The following tools are needed when releasing KFP:"
echo "node==12"
which node >/dev/null || (echo "node not found in PATH, recommend install via https://github.com/nvm-sh/nvm#installing-and-updating" && exit 1)
node -v | grep v12 || (echo "node not v12.x version" && exit 1)
echo "jq>=1.6"
which jq >/dev/null || (echo "jq not found in PATH" && exit 1)
echo "yq>=3.3"
which yq >/dev/null || (echo "yq not found in PATH" && exit 1)
yq -V | grep 3. || (echo "yq version 3.x should be used" && exit 1)
echo "java>=8"
which java >/dev/null || (echo "java not found in PATH" && exit 1)
echo "bazel==0.24.0"
which bazel >/dev/null || (echo "bazel not found in PATH" && exit 1)
bazel version | grep 0.24.0 || (echo "bazel not 0.24.0 version" && exit 1)
echo "python>3"
which python >/dev/null || (echo "python not found in PATH" && exit 1)
python -c "import setuptools" || (echo "setuptools should be installed in python" && exit 1)
echo "go"
which go >/dev/null || (echo "go not found in PATH" && exit 1)
go_path=$(go env GOPATH)
echo "$PATH" | grep "${go_path}/bin" > /dev/null || ( \
    echo "\$GOPATH/bin: ${go_path}/bin should be in PATH"; \
    echo "https://golang.org/cmd/go/#hdr-GOPATH_and_Modules"; \
    exit 1)

echo "All tools installed"
echo "Please add another needed tools if above list is not complete"
