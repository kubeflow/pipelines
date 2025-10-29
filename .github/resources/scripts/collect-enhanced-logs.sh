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

    # 7. Collect logs from recently failed/completed workflow pods
    echo "===== WORKFLOW/PIPELINE POD LOGS =====" >> "$OUTPUT_FILE"

    # Get pods that look like pipeline execution pods (containing workflow-like names)
    local WORKFLOW_PODS
    WORKFLOW_PODS=$(kubectl get pods -n "${NAMESPACE}" -o jsonpath='{range .items[?(@.metadata.labels.workflows\.argoproj\.io/workflow)]}{.metadata.name}{" "}{.status.phase}{" "}{.metadata.labels.workflows\.argoproj\.io/workflow}{"\n"}{end}' 2>/dev/null || echo "")

    if [[ -n "$WORKFLOW_PODS" ]]; then
        echo "Found Argo Workflow pods:" >> "$OUTPUT_FILE"
        echo "$WORKFLOW_PODS" >> "$OUTPUT_FILE"
        echo "" >> "$OUTPUT_FILE"

        # Collect logs from workflow pods
        while IFS= read -r line; do
            if [[ -n "$line" ]]; then
                local pod_name=$(echo "$line" | awk '{print $1}')
                local pod_phase=$(echo "$line" | awk '{print $2}')
                local workflow_name=$(echo "$line" | awk '{print $3}')

                echo "--- Workflow Pod: $pod_name (Phase: $pod_phase, Workflow: $workflow_name) ---" >> "$OUTPUT_FILE"
                kubectl logs "$pod_name" -n "${NAMESPACE}" --previous=false >> "$OUTPUT_FILE" 2>&1 || echo "No current logs for $pod_name" >> "$OUTPUT_FILE"

                # Also try to get previous logs if pod restarted
                kubectl logs "$pod_name" -n "${NAMESPACE}" --previous=true >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs for $pod_name" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"
            fi
        done <<< "$WORKFLOW_PODS"
    else
        echo "No Argo Workflow pods found with workflow labels" >> "$OUTPUT_FILE"

        # Fallback: look for pods with workflow-like naming patterns
        echo "Searching for pods with workflow-like names..." >> "$OUTPUT_FILE"
        local PATTERN_PODS
        PATTERN_PODS=$(kubectl get pods -n "${NAMESPACE}" -o name 2>/dev/null | grep -E "(pipeline|workflow|producer|consumer)" || echo "")

        if [[ -n "$PATTERN_PODS" ]]; then
            echo "Found workflow-pattern pods:" >> "$OUTPUT_FILE"
            for pod_name in $PATTERN_PODS; do
                pod_name=$(echo "$pod_name" | sed 's|pod/||')
                echo "--- Pattern Pod: $pod_name ---" >> "$OUTPUT_FILE"
                kubectl logs "$pod_name" -n "${NAMESPACE}" --tail=100 >> "$OUTPUT_FILE" 2>&1 || echo "No logs for $pod_name" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"
            done
        else
            echo "No workflow-pattern pods found" >> "$OUTPUT_FILE"
        fi
    fi
    echo "" >> "$OUTPUT_FILE"

    # 8. Collect critical KFP infrastructure logs for workflow debugging
    echo "===== CRITICAL KFP INFRASTRUCTURE LOGS =====" >> "$OUTPUT_FILE"

    # Workflow Controller - Critical for understanding why workflows aren't created
    echo "--- WORKFLOW CONTROLLER LOGS (ALL LOGS) ---" >> "$OUTPUT_FILE"
    local WF_CONTROLLER_POD
    WF_CONTROLLER_POD=$(kubectl get pods -n "${NAMESPACE}" -l app=workflow-controller -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [[ -n "$WF_CONTROLLER_POD" ]]; then
        echo "Workflow Controller Pod: $WF_CONTROLLER_POD" >> "$OUTPUT_FILE"
        echo "Getting ALL logs from container startup..." >> "$OUTPUT_FILE"
        kubectl logs "$WF_CONTROLLER_POD" -n "${NAMESPACE}" >> "$OUTPUT_FILE" 2>&1 || echo "No logs for workflow controller" >> "$OUTPUT_FILE"

        # Also get previous logs if the pod restarted
        echo "--- WORKFLOW CONTROLLER PREVIOUS LOGS (if restarted) ---" >> "$OUTPUT_FILE"
        kubectl logs "$WF_CONTROLLER_POD" -n "${NAMESPACE}" --previous >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs (pod did not restart)" >> "$OUTPUT_FILE"
    else
        echo "No workflow controller pod found" >> "$OUTPUT_FILE"
    fi
    echo "" >> "$OUTPUT_FILE"

    # Persistence Agent - Handles workflow submission
    echo "--- PERSISTENCE AGENT LOGS (ALL LOGS) ---" >> "$OUTPUT_FILE"
    local PERSISTENCE_POD
    PERSISTENCE_POD=$(kubectl get pods -n "${NAMESPACE}" -l app=ml-pipeline-persistenceagent -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [[ -n "$PERSISTENCE_POD" ]]; then
        echo "Persistence Agent Pod: $PERSISTENCE_POD" >> "$OUTPUT_FILE"
        echo "Getting ALL logs from container startup..." >> "$OUTPUT_FILE"
        kubectl logs "$PERSISTENCE_POD" -n "${NAMESPACE}" >> "$OUTPUT_FILE" 2>&1 || echo "No logs for persistence agent" >> "$OUTPUT_FILE"

        # Also get previous logs if the pod restarted
        echo "--- PERSISTENCE AGENT PREVIOUS LOGS (if restarted) ---" >> "$OUTPUT_FILE"
        kubectl logs "$PERSISTENCE_POD" -n "${NAMESPACE}" --previous >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs (pod did not restart)" >> "$OUTPUT_FILE"
    else
        echo "No persistence agent pod found" >> "$OUTPUT_FILE"
    fi
    echo "" >> "$OUTPUT_FILE"

    # Scheduled Workflow Controller - Handles scheduled runs
    echo "--- SCHEDULED WORKFLOW CONTROLLER LOGS (ALL LOGS) ---" >> "$OUTPUT_FILE"
    local SWF_POD
    SWF_POD=$(kubectl get pods -n "${NAMESPACE}" -l app=ml-pipeline-scheduledworkflow -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [[ -n "$SWF_POD" ]]; then
        echo "Scheduled Workflow Controller Pod: $SWF_POD" >> "$OUTPUT_FILE"
        echo "Getting ALL logs from container startup..." >> "$OUTPUT_FILE"
        kubectl logs "$SWF_POD" -n "${NAMESPACE}" >> "$OUTPUT_FILE" 2>&1 || echo "No logs for scheduled workflow controller" >> "$OUTPUT_FILE"

        # Also get previous logs if the pod restarted
        echo "--- SCHEDULED WORKFLOW CONTROLLER PREVIOUS LOGS (if restarted) ---" >> "$OUTPUT_FILE"
        kubectl logs "$SWF_POD" -n "${NAMESPACE}" --previous >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs (pod did not restart)" >> "$OUTPUT_FILE"
    else
        echo "No scheduled workflow controller pod found" >> "$OUTPUT_FILE"
    fi
    echo "" >> "$OUTPUT_FILE"

    # API Server logs - Complete logs for full context
    echo "--- API SERVER LOGS (ALL LOGS) ---" >> "$OUTPUT_FILE"
    local API_SERVER_POD
    API_SERVER_POD=$(kubectl get pods -n "${NAMESPACE}" -l app=ml-pipeline -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
    if [[ -n "$API_SERVER_POD" ]]; then
        echo "API Server Pod: $API_SERVER_POD" >> "$OUTPUT_FILE"
        echo "Getting ALL logs from container startup..." >> "$OUTPUT_FILE"
        kubectl logs "$API_SERVER_POD" -n "${NAMESPACE}" >> "$OUTPUT_FILE" 2>&1 || echo "No logs for API server" >> "$OUTPUT_FILE"

        # Also get previous logs if the pod restarted
        echo "--- API SERVER PREVIOUS LOGS (if restarted) ---" >> "$OUTPUT_FILE"
        kubectl logs "$API_SERVER_POD" -n "${NAMESPACE}" --previous >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs (pod did not restart)" >> "$OUTPUT_FILE"
    else
        echo "No API server pod found" >> "$OUTPUT_FILE"
    fi
    echo "" >> "$OUTPUT_FILE"

    # Check for any workflow-related Custom Resources
    echo "--- WORKFLOW CUSTOM RESOURCES ---" >> "$OUTPUT_FILE"
    kubectl get workflows -n "${NAMESPACE}" -o yaml >> "$OUTPUT_FILE" 2>&1 || echo "No workflows found or CRD not available" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # Check for WorkflowTemplate resources
    echo "--- WORKFLOW TEMPLATES ---" >> "$OUTPUT_FILE"
    kubectl get workflowtemplates -n "${NAMESPACE}" >> "$OUTPUT_FILE" 2>&1 || echo "No workflow templates found or CRD not available" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # Check Argo Workflow Controller configuration
    echo "--- ARGO WORKFLOW CONTROLLER CONFIG ---" >> "$OUTPUT_FILE"
    kubectl get configmap -n "${NAMESPACE}" | grep -E "(workflow|argo)" >> "$OUTPUT_FILE" 2>&1 || echo "No Argo-related ConfigMaps found" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 9. Collect all pod logs from user namespace if different and in multi-user mode
    if [[ -n "$TEST_CONTEXT" && "$TEST_CONTEXT" == *"MultiUser"* ]]; then
        echo "===== CHECKING USER NAMESPACE PODS =====" >> "$OUTPUT_FILE"
        # Common user namespace patterns
        for user_ns in "kubeflow-user-example-com" "kubeflow-user-test" "default"; do
            if kubectl get namespace "$user_ns" &>/dev/null && [[ "$user_ns" != "$NAMESPACE" ]]; then
                echo "Found user namespace: $user_ns" >> "$OUTPUT_FILE"
                kubectl get pods -n "$user_ns" -o wide --show-labels >> "$OUTPUT_FILE" 2>&1 || true

                # Get ALL user namespace pods (not just pattern matches)
                echo "=== ALL USER NAMESPACE PODS WITH STATUS ===" >> "$OUTPUT_FILE"
                kubectl get pods -n "$user_ns" -o wide >> "$OUTPUT_FILE" 2>&1 || echo "Failed to get pods in $user_ns" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"

                # Get user namespace workflow pods (broader patterns)
                local USER_WORKFLOW_PODS
                USER_WORKFLOW_PODS=$(kubectl get pods -n "$user_ns" -o name 2>/dev/null | grep -E "(pipeline|workflow|producer|consumer|dag-driver|system)" || echo "")

                if [[ -n "$USER_WORKFLOW_PODS" ]]; then
                    echo "User namespace workflow/execution pods:" >> "$OUTPUT_FILE"
                    for pod_name in $USER_WORKFLOW_PODS; do
                        pod_name=$(echo "$pod_name" | sed 's|pod/||')
                        echo "--- User NS Pod: $pod_name ---" >> "$OUTPUT_FILE"

                        # Get pod status first
                        kubectl describe pod "$pod_name" -n "$user_ns" >> "$OUTPUT_FILE" 2>&1 || echo "Failed to describe $pod_name" >> "$OUTPUT_FILE"
                        echo "" >> "$OUTPUT_FILE"

                        # Get all logs
                        echo "Pod logs for $pod_name:" >> "$OUTPUT_FILE"
                        kubectl logs "$pod_name" -n "$user_ns" >> "$OUTPUT_FILE" 2>&1 || echo "No logs for $pod_name" >> "$OUTPUT_FILE"

                        # Get previous logs if pod restarted
                        echo "Previous logs for $pod_name (if restarted):" >> "$OUTPUT_FILE"
                        kubectl logs "$pod_name" -n "$user_ns" --previous >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs for $pod_name" >> "$OUTPUT_FILE"
                        echo "" >> "$OUTPUT_FILE"
                    done
                fi

                # Specifically look for failed/pending/error pods
                echo "=== FAILED/PENDING/ERROR PODS IN USER NAMESPACE ===" >> "$OUTPUT_FILE"
                local FAILED_PODS
                FAILED_PODS=$(kubectl get pods -n "$user_ns" --field-selector=status.phase!=Running,status.phase!=Succeeded -o name 2>/dev/null || echo "")
                if [[ -n "$FAILED_PODS" ]]; then
                    echo "Found non-running pods:" >> "$OUTPUT_FILE"
                    for pod_name in $FAILED_PODS; do
                        pod_name=$(echo "$pod_name" | sed 's|pod/||')
                        echo "--- Failed/Pending Pod: $pod_name ---" >> "$OUTPUT_FILE"

                        # Detailed pod information
                        kubectl describe pod "$pod_name" -n "$user_ns" >> "$OUTPUT_FILE" 2>&1 || echo "Failed to describe $pod_name" >> "$OUTPUT_FILE"
                        echo "" >> "$OUTPUT_FILE"

                        # All container logs
                        echo "Logs for failed pod $pod_name:" >> "$OUTPUT_FILE"
                        kubectl logs "$pod_name" -n "$user_ns" --all-containers=true >> "$OUTPUT_FILE" 2>&1 || echo "No logs for $pod_name" >> "$OUTPUT_FILE"
                        echo "" >> "$OUTPUT_FILE"
                    done
                else
                    echo "No failed/pending pods found in user namespace" >> "$OUTPUT_FILE"
                fi
                echo "" >> "$OUTPUT_FILE"

                # Check resource quotas and limits
                echo "=== USER NAMESPACE RESOURCE QUOTAS ===" >> "$OUTPUT_FILE"
                kubectl get resourcequota -n "$user_ns" -o yaml >> "$OUTPUT_FILE" 2>&1 || echo "No resource quotas in $user_ns" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"

                echo "=== USER NAMESPACE LIMIT RANGES ===" >> "$OUTPUT_FILE"
                kubectl get limitrange -n "$user_ns" -o yaml >> "$OUTPUT_FILE" 2>&1 || echo "No limit ranges in $user_ns" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"

                # Check events in user namespace
                echo "=== USER NAMESPACE EVENTS (last 30 events) ===" >> "$OUTPUT_FILE"
                kubectl get events -n "$user_ns" --sort-by='.lastTimestamp' | tail -30 >> "$OUTPUT_FILE" 2>&1 || echo "No events in $user_ns" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"

                # Also check for workflows in user namespace
                echo "--- USER NAMESPACE WORKFLOWS ---" >> "$OUTPUT_FILE"
                kubectl get workflows -n "$user_ns" >> "$OUTPUT_FILE" 2>&1 || echo "No workflows found in user namespace $user_ns" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"
            fi
        done
    fi

    echo "Enhanced log collection completed. Output saved to: $OUTPUT_FILE"
}

check_namespace "$NS"
collect_comprehensive_logs "$NS"