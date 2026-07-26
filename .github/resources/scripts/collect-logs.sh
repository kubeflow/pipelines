#!/usr/bin/env bash

set -e

NS=""
OUTPUT_FILE="/tmp/tmp.log/tmp_pod_log.txt"
KUBECTL_REQUEST_TIMEOUT="${KUBECTL_REQUEST_TIMEOUT:-20s}"

kubectl_with_timeout() {
    kubectl --request-timeout="$KUBECTL_REQUEST_TIMEOUT" "$@"
}

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
    if ! kubectl_with_timeout get namespace "$1" &>/dev/null; then
        echo "Namespace '$1' is absent or unavailable. Skipping log collection."
        return 1
    fi
    return 0
}

function display_pod_info {
    local NAMESPACE=$1

    kubectl_with_timeout get pods -n "${NAMESPACE}" || \
        echo "Unable to list pod status in namespace '${NAMESPACE}'."

    local POD_NAMES

    if ! POD_NAMES=$(kubectl_with_timeout get pods -n "${NAMESPACE}" -o custom-columns=":metadata.name" --no-headers); then
        echo "Unable to enumerate pods in namespace '${NAMESPACE}'." | tee -a "$OUTPUT_FILE"
        return
    fi

    if [[ -z "${POD_NAMES}" ]]; then
        echo "No pods found in namespace '${NAMESPACE}'." | tee -a "$OUTPUT_FILE"
        return
    fi

    echo "Pod Information for Namespace: ${NAMESPACE}" | tee -a "$OUTPUT_FILE"

    for POD_NAME in ${POD_NAMES}; do
        {
            echo "===== Pod: ${POD_NAME} in ${NAMESPACE} ====="
            echo "----- EVENTS -----"
            kubectl_with_timeout describe pod "${POD_NAME}" -n "${NAMESPACE}" | grep -A 100 Events || echo "No events found for pod ${POD_NAME}."

            echo "----- LOGS -----"
            if [[ "${POD_NAME}" == *-agent* ]]; then
                kubectl_with_timeout logs "${POD_NAME}" -n "${NAMESPACE}" -c driver-plugin || \
                    echo "No logs found for pod ${POD_NAME}."
            else
                kubectl_with_timeout logs "${POD_NAME}" -n "${NAMESPACE}" || \
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
else
    exit 0
fi
