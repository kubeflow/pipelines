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

# This is a wrapper script on top of test.sh, it installs required dependencies.

set -ex

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

# Install Kustomize
KUSTOMIZE_VERSION=3.10.0
# Reference: https://kubectl.docs.kubernetes.io/installation/kustomize/binaries/
curl -s -O "https://raw.githubusercontent.com/\
kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
chmod +x install_kustomize.sh
./install_kustomize.sh "${KUSTOMIZE_VERSION}" /usr/local/bin/

# kpt and kubectl should already be installed in gcr.io/google.com/cloudsdktool/cloud-sdk:latest
# so we do not need to install them here

# trigger real unit tests
${DIR}/test.sh

