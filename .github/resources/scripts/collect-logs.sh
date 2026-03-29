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

function describe_argo_workflows {
    local NAMESPACE=$1
    echo "===== Argo Workflows list ====="
    kubectl describe wf  -n "${NAMESPACE}"
    echo "===== Argo Workflows data ====="
    kubectl get events -n "${NAMESPACE}" --field-selector involvedObject.kind=Workflow --sort-by='.metadata.creationTimestamp'
    echo "==============================="
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
            if [[ "${POD_NAME}" == *-agent* ]]; then
                kubectl logs "${POD_NAME}" -n "${NAMESPACE}" -c driver-plugin || \
                    echo "No logs found for pod ${POD_NAME}."
            else
                kubectl logs "${POD_NAME}" -n "${NAMESPACE}" || \
                    echo "No logs found for pod ${POD_NAME}."
            fi

            echo "==========================="
            echo ""
        } | tee -a "$OUTPUT_FILE"
    done

    echo "Pod information stored in $OUTPUT_FILE"
}

if check_namespace "$NS"; then
    display_pod_info "$NS"
    describe_argo_workflows "$NS"
else
    exit 0
fi
