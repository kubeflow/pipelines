#!/bin/bash

ROOT=$(cd $(dirname $0)/../../; pwd)

set -o errexit
set -o nounset
set -o pipefail
set -ex

export CA_BUNDLE=$(cat ${CA_FILE})

if command -v envsubst >/dev/null 2>&1; then
    envsubst
else
    sed -e "s|\${CA_BUNDLE}|${CA_BUNDLE}|g" -e "s|\${NAMESPACE}|${NAMESPACE}|g"
fi
