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

mkdir -p /tmp/tmp.log

if [[ -z "$NS" ]]; then
    echo "Both --ns parameters are required."
    exit 1
fi

function check_namespace {
    if ! kubectl get namespace "$1" &>/dev/null; then
        echo "Namespace '$1' does not exist. Skipping log collection."
        return 1
    fi
    return 0
}

function display_pod_info {
    local NAMESPACE=$1

    kubectl get pods -n "${NAMESPACE}"

    local POD_NAMES

    POD_NAMES=$(kubectl get pods -n "${NAMESPACE}" -o custom-columns=":metadata.name" --no-headers)

    if [[ -z "${POD_NAMES}" ]]; then
        echo "No pods found in namespace '${NAMESPACE}'." | tee -a "$OUTPUT_FILE"
        return
    fi

    echo "Pod Information for Namespace: ${NAMESPACE}" > "$OUTPUT_FILE"

    for POD_NAME in ${POD_NAMES}; do
        {
            echo "===== Pod: ${POD_NAME} in ${NAMESPACE} ====="
            echo "----- EVENTS -----"
            kubectl describe pod "${POD_NAME}" -n "${NAMESPACE}" | grep -A 100 Events || echo "No events found for pod ${POD_NAME}."

            echo "----- LOGS -----"

            # Get all containers (init + regular) from the pod
            INIT_CONTAINERS=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.initContainers[*].name}' 2>/dev/null || echo "")
            CONTAINERS=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null || echo "")

            # Collect logs from init containers
            if [[ -n "${INIT_CONTAINERS}" ]]; then
                for CONTAINER in ${INIT_CONTAINERS}; do
                    echo "----- Init Container: ${CONTAINER} (current) -----"
                    kubectl logs "${POD_NAME}" -c "${CONTAINER}" -n "${NAMESPACE}" 2>&1 || echo "No current logs found for init container ${CONTAINER}."

                    echo "----- Init Container: ${CONTAINER} (previous) -----"
                    kubectl logs "${POD_NAME}" -c "${CONTAINER}" -n "${NAMESPACE}" --previous 2>&1 || echo "No previous logs found for init container ${CONTAINER}."
                done
            fi

            # Collect logs from regular containers
            if [[ -n "${CONTAINERS}" ]]; then
                for CONTAINER in ${CONTAINERS}; do
                    echo "----- Container: ${CONTAINER} (current) -----"
                    kubectl logs "${POD_NAME}" -c "${CONTAINER}" -n "${NAMESPACE}" 2>&1 || echo "No current logs found for container ${CONTAINER}."

                    echo "----- Container: ${CONTAINER} (previous) -----"
                    kubectl logs "${POD_NAME}" -c "${CONTAINER}" -n "${NAMESPACE}" --previous 2>&1 || echo "No previous logs found for container ${CONTAINER}."
                done
            else
                # Fallback: try to get logs without specifying container (for single-container pods)
                echo "----- Default Container -----"
                kubectl logs "${POD_NAME}" -n "${NAMESPACE}" 2>&1 || echo "No logs found for pod ${POD_NAME}."
            fi

            echo "==========================="
            echo ""
        } | tee -a "$OUTPUT_FILE"
    done

    echo "Pod information stored in $OUTPUT_FILE"
}

if check_namespace "$NS"; then
    display_pod_info "$NS"
else
    exit 0
fi
