#!/bin/bash
#
# Copyright 2020-2021 The Kubeflow Authors
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

# When running this in MacOS, we can run `OS=darwin-amd64 ./presubmit.sh`
OS="${OS:-linux-amd64}"

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null && pwd)"

if ! which helm; then
    HELM_VERSION=v2.17.0
    if ! which curl; then
        apk --no-cache add curl
    fi
    curl -sLO "https://get.helm.sh/helm-${HELM_VERSION}-${OS}.tar.gz"
    tar xvf "helm-${HELM_VERSION}-${OS}.tar.gz"
    mv ./${OS}/helm /usr/local/bin/helm
    helm version
fi

if ! which bash; then
    apk --no-cache add bash
fi
bash "${DIR}/snapshots.sh"


if ! which git; then
    apk --no-cache add git
fi
# exit 1 if snapshots are not up-to-date.
git diff --exit-code -- "${DIR}" || \
    (echo "gcp_marketplace test snapshots are not tidy, run 'manifests/gcp_marketplace/test/snapshots.sh' to update them." \
    && exit 1)
