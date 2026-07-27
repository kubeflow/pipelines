#!/bin/bash

set -e

C_DIR="${BASH_SOURCE%/*}"
NAMESPACE="tinyproxy"
CLUSTER_NAME="${CLUSTER_NAME:-kfp}"

function dump_tinyproxy_state {
    kubectl -n "${NAMESPACE}" get pods -l app=tinyproxy -o wide || true
    kubectl -n "${NAMESPACE}" describe deployment/tinyproxy || true
    for pod in $(kubectl -n "${NAMESPACE}" get pods -l app=tinyproxy -o name 2>/dev/null); do
        kubectl -n "${NAMESPACE}" describe "${pod}" || true
        kubectl -n "${NAMESPACE}" logs "${pod}" --all-containers=true --tail=200 || true
        kubectl -n "${NAMESPACE}" logs "${pod}" --all-containers=true --previous --tail=200 || true
    done
}

docker build --progress=plain -t "registry.domain.local/tinyproxy:test" -f "${C_DIR}/Containerfile" "${C_DIR}"
kind --name "${CLUSTER_NAME}" load docker-image registry.domain.local/tinyproxy:test

kubectl apply -k "${C_DIR}/manifests"

if ! kubectl -n "${NAMESPACE}" wait --for=condition=available deployment/tinyproxy --timeout=60s; then
    echo "Timeout occurred while waiting for the Tinyproxy deployment."
    dump_tinyproxy_state
    exit 1
fi

echo "Waiting for the Tinyproxy service to publish a ready endpoint."
end_time=$((SECONDS + 60))
while true; do
    endpoint_ip="$(kubectl -n "${NAMESPACE}" get endpoints tinyproxy -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null || true)"
    if [[ -n "${endpoint_ip}" ]]; then
        echo "Tinyproxy endpoint is ready: ${endpoint_ip}"
        break
    fi

    if (( SECONDS >= end_time )); then
        echo "Timeout occurred while waiting for the Tinyproxy service endpoint."
        dump_tinyproxy_state
        kubectl -n "${NAMESPACE}" get endpoints tinyproxy -o yaml || true
        exit 1
    fi

    sleep 2
done
