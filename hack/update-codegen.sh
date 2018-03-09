#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname ${BASH_SOURCE})/..
echo "SCRIPT_ROOT is $SCRIPT_ROOT"
CODEGEN_PKG=${CODEGEN_PKG:-$(cd ${SCRIPT_ROOT}; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}
echo "CODEGEN_PKG is $CODEGEN_PKG"

${CODEGEN_PKG}/generate-groups.sh "deepcopy,client,informer,lister" \
  ml/pkg/client ml/pkg/apis \
  pipeline:v1alpha1 \
  --go-header-file ${SCRIPT_ROOT}/hack/custom-boilerplate.go.txt
