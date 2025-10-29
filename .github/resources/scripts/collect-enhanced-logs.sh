#!/usr/bin/env bash

set -e

NS=""
OUTPUT_FILE="/tmp/enhanced_pod_logs.txt"
TEST_CONTEXT=""
START_TIME=""

while [[ "$#" -gt 0 ]]; do
    case $1 in
        --ns) NS="$2"; shift ;;
        --output) OUTPUT_FILE="$2"; shift ;;
        --test-context) TEST_CONTEXT="$2"; shift ;;
        --start-time) START_TIME="$2"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

mkdir -p /tmp/enhanced.log

if [[ -z "$NS" ]]; then
    echo "Namespace (--ns) parameter is required."
    exit 1
fi

function check_namespace {
    if ! kubectl get namespace "$1" &>/dev/null; then
        echo "Namespace '$1' does not exist."
        exit 1
    fi
}

function collect_comprehensive_logs {
    local NAMESPACE=$1

    echo "===== ENHANCED LOG COLLECTION REPORT =====" > "$OUTPUT_FILE"
    echo "Collection Time: $(date)" >> "$OUTPUT_FILE"
    echo "Test Context: ${TEST_CONTEXT:-'Not specified'}" >> "$OUTPUT_FILE"
    echo "Test Start Time: ${START_TIME:-'Not specified'}" >> "$OUTPUT_FILE"
    echo "Namespace: ${NAMESPACE}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 1. Collect all pod information with labels and annotations
    echo "===== POD OVERVIEW WITH LABELS =====" >> "$OUTPUT_FILE"
    kubectl get pods -n "${NAMESPACE}" -o wide --show-labels >> "$OUTPUT_FILE" 2>&1 || echo "Failed to get pod overview" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 2. Collect Argo Workflow information
    echo "===== ARGO WORKFLOWS =====" >> "$OUTPUT_FILE"
    kubectl get workflows -n "${NAMESPACE}" -o wide --show-labels >> "$OUTPUT_FILE" 2>&1 || echo "No workflows found or failed to get workflows" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 3. Collect recent events (last 30 minutes)
    echo "===== RECENT EVENTS =====" >> "$OUTPUT_FILE"
    kubectl get events -n "${NAMESPACE}" --sort-by='.lastTimestamp' >> "$OUTPUT_FILE" 2>&1 || echo "Failed to get events" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 4. Filter pods created after test start time (if provided)
    local POD_NAMES
    if [[ -n "$START_TIME" ]]; then
        echo "===== PODS CREATED DURING TEST (after $START_TIME) =====" >> "$OUTPUT_FILE"
        POD_NAMES=$(kubectl get pods -n "${NAMESPACE}" -o jsonpath='{range .items[*]}{.metadata.name}{" "}{.metadata.creationTimestamp}{"\n"}{end}' | awk -v start_time="$START_TIME" '$2 >= start_time {print $1}')
        if [[ -n "$POD_NAMES" ]]; then
            echo "Test-related pods: $POD_NAMES" >> "$OUTPUT_FILE"
        else
            echo "No pods found created after $START_TIME" >> "$OUTPUT_FILE"
            # Fall back to all pods
            POD_NAMES=$(kubectl get pods -n "${NAMESPACE}" -o custom-columns=":metadata.name" --no-headers)
        fi
    else
        POD_NAMES=$(kubectl get pods -n "${NAMESPACE}" -o custom-columns=":metadata.name" --no-headers)
    fi
    echo "" >> "$OUTPUT_FILE"

    if [[ -z "${POD_NAMES}" ]]; then
        echo "No pods found in namespace '${NAMESPACE}'." >> "$OUTPUT_FILE"
        return
    fi

    # 5. Detailed pod information with logs
    for POD_NAME in ${POD_NAMES}; do
        {
            echo "=========================================="
            echo "POD: ${POD_NAME}"
            echo "=========================================="

            echo "----- POD METADATA -----"
            kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o yaml | grep -E "(name:|namespace:|labels:|annotations:|creationTimestamp:|phase:|conditions:)" || echo "Failed to get pod metadata"

            echo ""
            echo "----- POD DESCRIPTION -----"
            kubectl describe pod "${POD_NAME}" -n "${NAMESPACE}" || echo "Failed to describe pod ${POD_NAME}"

            echo ""
            echo "----- POD LOGS (last 1000 lines) -----"
            kubectl logs "${POD_NAME}" -n "${NAMESPACE}" --tail=1000 || echo "No logs found for pod ${POD_NAME}"

            # Check for multiple containers
            local CONTAINERS
            CONTAINERS=$(kubectl get pod "${POD_NAME}" -n "${NAMESPACE}" -o jsonpath='{.spec.containers[*].name}' 2>/dev/null)
            if [[ $(echo "$CONTAINERS" | wc -w) -gt 1 ]]; then
                echo ""
                echo "----- CONTAINER LOGS -----"
                for CONTAINER in $CONTAINERS; do
                    echo "--- Container: $CONTAINER ---"
                    kubectl logs "${POD_NAME}" -c "$CONTAINER" -n "${NAMESPACE}" --tail=500 || echo "No logs for container $CONTAINER"
                done
            fi

            echo ""
            echo "=========================================="
            echo ""
        } >> "$OUTPUT_FILE"
    done

    # 6. Collect pipeline run information if available
    echo "===== PIPELINE RUNS (if available) =====" >> "$OUTPUT_FILE"
    kubectl get runs -n "${NAMESPACE}" -o wide --show-labels >> "$OUTPUT_FILE" 2>&1 || echo "No pipeline runs found or CRD not available" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    echo "Enhanced log collection completed. Output saved to: $OUTPUT_FILE"
}

check_namespace "$NS"
collect_comprehensive_logs "$NS"