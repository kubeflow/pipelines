#!/bin/bash
#
# Copyright 2019 The Kubeflow Authors
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

# This script builds argo images and pushes them to KFP's gcr container registry.
# Usage: ./release.sh
# It can be run anywhere.

set -ex

# Get this bash script's dir.
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# Update NOTICES of the specified argo version.
"${DIR}/imp-1-update-notices.sh"
# Build and push license compliant argo images to gcr.io/ml-pipeline.
"${DIR}/imp-2-build-push-images.sh"
