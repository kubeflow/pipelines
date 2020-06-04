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
echo "jq>=1.6"
which jq || (echo "jq not found in PATH" && exit 1)
echo "yq>=3.3"
which yq || (echo "yq not found in PATH" && exit 1)
yq -V | grep 3. || (echo "yq version 3.x should be used" && exit 1)
echo "java>=8"
which java || (echo "java not found in PATH" && exit 1)

