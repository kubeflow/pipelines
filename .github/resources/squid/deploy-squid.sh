#!/bin/bash

set -e

C_DIR="${BASH_SOURCE%/*}"
NAMESPACE="squid"
CLUSTER_NAME="${CLUSTER_NAME:-kfp}"

function dump_squid_state {
    kubectl -n "${NAMESPACE}" get pods -l app=squid -o wide || true
    kubectl -n "${NAMESPACE}" describe deployment/squid || true
    for pod in $(kubectl -n "${NAMESPACE}" get pods -l app=squid -o name 2>/dev/null); do
        kubectl -n "${NAMESPACE}" describe "${pod}" || true
        kubectl -n "${NAMESPACE}" logs "${pod}" --all-containers=true --tail=200 || true
        kubectl -n "${NAMESPACE}" logs "${pod}" --all-containers=true --previous --tail=200 || true
    done
}

docker build --progress=plain -t "registry.domain.local/squid:test" -f "${C_DIR}/Containerfile" "${C_DIR}"
kind --name "${CLUSTER_NAME}" load docker-image registry.domain.local/squid:test

kubectl apply -k "${C_DIR}/manifests"

if ! kubectl -n "${NAMESPACE}" wait --for=condition=available deployment/squid --timeout=60s; then
    echo "Timeout occurred while waiting for the Squid deployment."
    dump_squid_state
    exit 1
fi

echo "Waiting for the Squid service to publish a ready endpoint."
end_time=$((SECONDS + 60))
while true; do
    endpoint_ip="$(kubectl -n "${NAMESPACE}" get endpoints squid -o jsonpath='{.subsets[0].addresses[0].ip}' 2>/dev/null || true)"
    if [[ -n "${endpoint_ip}" ]]; then
        echo "Squid endpoint is ready: ${endpoint_ip}"
        break
    fi

    if (( SECONDS >= end_time )); then
        echo "Timeout occurred while waiting for the Squid service endpoint."
        dump_squid_state
        kubectl -n "${NAMESPACE}" get endpoints squid -o yaml || true
        exit 1
    fi

    sleep 2
done
