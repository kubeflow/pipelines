#!/usr/bin/env bash

set -e

NS=""
OUTPUT_FILE="/tmp/tmp.log/tmp_pod_log.txt"

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --ns) NS="$2"; shift ;;
        --output) OUTPUT_FILE="$2"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

if [[ -z "$NS" ]]; then
    echo "--ns parameter is required."
    exit 1
fi

if [[ -z "$OUTPUT_FILE" ]]; then
    echo "--output parameter is required."
    exit 1
fi

mkdir -p "$(dirname "$OUTPUT_FILE")"
touch "$OUTPUT_FILE"

function check_namespace {
    if ! kubectl get namespace "$1" &>/dev/null; then
        echo "Namespace '$1' does not exist. Skipping log collection."
        return 1
    fi
    return 0
}

function display_namespace_state {
    local NAMESPACE=$1

    {
        echo "Namespace State for Namespace: ${NAMESPACE}"
        echo "----- SERVICES -----"
        kubectl get svc -n "${NAMESPACE}" -o wide || echo "No services found in namespace '${NAMESPACE}'."

        echo "----- ENDPOINTS -----"
        kubectl get endpoints -n "${NAMESPACE}" -o wide || echo "No endpoints found in namespace '${NAMESPACE}'."

        echo "----- ENDPOINT SLICES -----"
        kubectl get endpointslices.discovery.k8s.io -n "${NAMESPACE}" -o wide || echo "No endpoint slices found in namespace '${NAMESPACE}'."

        echo "----- PODS -----"
        kubectl get pods -n "${NAMESPACE}" -o wide || echo "No pods found in namespace '${NAMESPACE}'."

        echo "----- DEPLOYMENTS -----"
        kubectl get deployments -n "${NAMESPACE}" -o wide || echo "No deployments found in namespace '${NAMESPACE}'."

        echo "----- STATEFULSETS -----"
        kubectl get statefulsets -n "${NAMESPACE}" -o wide || echo "No statefulsets found in namespace '${NAMESPACE}'."

        echo "----- EVENTS -----"
        kubectl get events -n "${NAMESPACE}" --sort-by=.lastTimestamp || echo "No events found in namespace '${NAMESPACE}'."
        echo ""
    } | tee -a "$OUTPUT_FILE"
}

function display_pod_info {
    local NAMESPACE=$1

    local POD_NAMES

    POD_NAMES=$(kubectl get pods -n "${NAMESPACE}" -o custom-columns=":metadata.name" --no-headers)

    if [[ -z "${POD_NAMES}" ]]; then
        echo "No pods found in namespace '${NAMESPACE}'." | tee -a "$OUTPUT_FILE"
        return
    fi

    echo "Pod Information for Namespace: ${NAMESPACE}" | tee -a "$OUTPUT_FILE"

    for POD_NAME in ${POD_NAMES}; do
        {
            echo "===== Pod: ${POD_NAME} in ${NAMESPACE} ====="
            echo "----- EVENTS -----"
            kubectl describe pod "${POD_NAME}" -n "${NAMESPACE}" | grep -A 100 Events || echo "No events found for pod ${POD_NAME}."

            echo "----- LOGS -----"
            kubectl logs "${POD_NAME}" -n "${NAMESPACE}" || echo "No logs found for pod ${POD_NAME}."

            echo "==========================="
            echo ""
        } | tee -a "$OUTPUT_FILE"
    done

    echo "Pod information stored in $OUTPUT_FILE"
}

if check_namespace "$NS"; then
    display_namespace_state "$NS"
    display_pod_info "$NS"
else
    exit 0
fi
