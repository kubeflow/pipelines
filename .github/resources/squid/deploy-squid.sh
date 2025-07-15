#!/bin/bash

set -e

C_DIR="${BASH_SOURCE%/*}"
NAMESPACE="squid"

docker build --progress=plain -t "registry.domain.local/squid:test" -f ${C_DIR}/Containerfile ${C_DIR}
kind --name kfp load docker-image registry.domain.local/squid:test

kubectl apply -k ${C_DIR}/manifests

if ! kubectl -n ${NAMESPACE} wait --for=condition=available deployment/squid --timeout=60s; then
    echo "Timeout occurred while waiting for the Squid deployment."
    exit 1
fi
