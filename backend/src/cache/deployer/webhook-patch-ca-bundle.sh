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

# This file will patch CA bundle and namespace to MutatingWebhookConfiguration

ROOT=$(cd $(dirname $0)/../../; pwd)

set -o errexit
set -o nounset
set -o pipefail
set -ex

while [[ $# -gt 0 ]]; do
    case ${1} in
        --cert_input_path)
            cert_input_path="$2"
            shift
            ;;
    esac
    shift
done

[ -z ${cert_input_path} ] && cert_input_path=${CA_FILE}

export CA_BUNDLE=$(cat ${cert_input_path})

if command -v envsubst >/dev/null 2>&1; then
    envsubst
else
    sed -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g" -e "s|\${NAMESPACE}|${NAMESPACE}|g"
fi
