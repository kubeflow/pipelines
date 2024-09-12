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

# This is a wrapper script on top of test.sh, it installs required dependencies.

set -ex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"
TMP="$(mktemp -d)"

# Add TMP to PATH
PATH="$TMP:$PATH"

pushd "${TMP}"

# Install kustomize
KUSTOMIZE_VERSION=5.2.1
# Reference: https://github.com/kubernetes-sigs/kustomize/releases/tag/kustomize%2Fv5.2.1
curl -s -LO "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz"
tar -xzf kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz
chmod +x kustomize

# Install yq
# Reference: https://github.com/mikefarah/yq/releases/tag/3.4.1
curl -s -LO "https://github.com/mikefarah/yq/releases/download/3.4.1/yq_linux_amd64"
chmod +x yq_linux_amd64
mv yq_linux_amd64 yq

# Install kpt
KPT_VERSION=1.0.0-beta.54
# Reference: https://github.com/kptdev/kpt/releases/tag/v1.0.0-beta.54
curl -s -LO "https://github.com/kptdev/kpt/releases/download/v${KPT_VERSION}/kpt_linux_amd64"
chmod +x kpt_linux_amd64
mv kpt_linux_amd64 kpt
popd

# trigger real unit tests
${DIR}/test.sh
# verify release script runs properly

${DIR}/release.sh v1.2.3-dummy
# --no-pager sends output to stdout
# Show git diff, so people can manually verify results of the release script
git --no-pager diff
