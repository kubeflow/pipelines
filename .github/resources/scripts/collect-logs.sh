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

mkdir -p "$(dirname "$OUTPUT_FILE")"

if [[ -z "$NS" ]]; then
    echo "Namespace (--ns) parameter is required."
    exit 1
fi

# Verify namespace exists
check_namespace() {
    if ! kubectl get namespace "$1" &>/dev/null; then
        echo "Error: Namespace '$1' does not exist."
        exit 1
    fi
}

# Main log collection function
collect_comprehensive_logs() {
    local NAMESPACE=$1

    echo "===== ENHANCED LOG COLLECTION REPORT =====" > "$OUTPUT_FILE"
    echo "Collection Time: $(date)" >> "$OUTPUT_FILE"
    echo "Namespace: ${NAMESPACE}" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 1. Pod overview with labels
    echo "===== POD OVERVIEW WITH LABELS =====" >> "$OUTPUT_FILE"
    kubectl get pods -n "${NAMESPACE}" -o wide --show-labels >> "$OUTPUT_FILE" 2>&1 || echo "Failed to get pod overview" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 2. Argo Workflows
    echo "===== ARGO WORKFLOWS =====" >> "$OUTPUT_FILE"
    kubectl get workflows -n "${NAMESPACE}" -o wide --show-labels >> "$OUTPUT_FILE" 2>&1 || echo "No workflows found" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 3. Recent events
    echo "===== RECENT EVENTS =====" >> "$OUTPUT_FILE"
    kubectl get events -n "${NAMESPACE}" --sort-by='.lastTimestamp' >> "$OUTPUT_FILE" 2>&1 || echo "Failed to get events" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    # 4. Detailed pod logs
    collect_pod_logs "${NAMESPACE}"

    # 5. KFP infrastructure logs
    collect_infrastructure_logs "${NAMESPACE}"

    # 6. Workflow-specific resources
    collect_workflow_resources "${NAMESPACE}"

    # 7. Multi-user namespace analysis (auto-detect)
    collect_user_namespace_logs

    echo "Enhanced log collection completed. Output saved to: $OUTPUT_FILE"
}

# Collect logs from all pods in namespace
collect_pod_logs() {
    local NAMESPACE=$1
    local POD_NAMES

    POD_NAMES=$(kubectl get pods -n "${NAMESPACE}" -o custom-columns=":metadata.name" --no-headers)

    if [[ -z "${POD_NAMES}" ]]; then
        echo "No pods found in namespace '${NAMESPACE}'." >> "$OUTPUT_FILE"
        return
    fi

    # Collect detailed information for each pod
    for POD_NAME in ${POD_NAMES}; do
        {
            echo "=========================================="
            echo "POD: ${POD_NAME}"
            echo "=========================================="

            echo "----- POD DESCRIPTION -----"
            kubectl describe pod "${POD_NAME}" -n "${NAMESPACE}" || echo "Failed to describe pod ${POD_NAME}"

            echo ""
            echo "----- POD LOGS -----"
            kubectl logs "${POD_NAME}" -n "${NAMESPACE}" --all-containers=true || echo "No logs found for pod ${POD_NAME}"

            # Get previous logs if pod restarted
            echo ""
            echo "----- PREVIOUS LOGS (if restarted) -----"
            kubectl logs "${POD_NAME}" -n "${NAMESPACE}" --all-containers=true --previous || echo "No previous logs for pod ${POD_NAME}"

            echo ""
            echo "=========================================="
            echo ""
        } >> "$OUTPUT_FILE"
    done
}

# Collect logs from critical KFP infrastructure components
collect_infrastructure_logs() {
    local NAMESPACE=$1

    echo "===== CRITICAL KFP INFRASTRUCTURE LOGS =====" >> "$OUTPUT_FILE"

    # Define infrastructure components
    local components=(
        "workflow-controller:app=workflow-controller"
        "persistence-agent:app=ml-pipeline-persistenceagent"
        "scheduled-workflow:app=ml-pipeline-scheduledworkflow"
        "api-server:app=ml-pipeline"
    )

    for component in "${components[@]}"; do
        local name="${component%%:*}"
        local selector="${component##*:}"

        echo "--- ${name^^} LOGS (ALL LOGS) ---" >> "$OUTPUT_FILE"
        local pod=$(kubectl get pods -n "${NAMESPACE}" -l "${selector}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")

        if [[ -n "$pod" ]]; then
            echo "${name^} Pod: $pod" >> "$OUTPUT_FILE"
            kubectl logs "$pod" -n "${NAMESPACE}" >> "$OUTPUT_FILE" 2>&1 || echo "No logs for $name" >> "$OUTPUT_FILE"

            # Previous logs if restarted
            echo "--- ${name^^} PREVIOUS LOGS (if restarted) ---" >> "$OUTPUT_FILE"
            kubectl logs "$pod" -n "${NAMESPACE}" --previous >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs" >> "$OUTPUT_FILE"
        else
            echo "No $name pod found" >> "$OUTPUT_FILE"
        fi
        echo "" >> "$OUTPUT_FILE"
    done
}

# Collect workflow-related Kubernetes resources
collect_workflow_resources() {
    local NAMESPACE=$1

    echo "--- WORKFLOW CUSTOM RESOURCES ---" >> "$OUTPUT_FILE"
    kubectl get workflows -n "${NAMESPACE}" -o yaml >> "$OUTPUT_FILE" 2>&1 || echo "No workflows found" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    echo "--- WORKFLOW TEMPLATES ---" >> "$OUTPUT_FILE"
    kubectl get workflowtemplates -n "${NAMESPACE}" >> "$OUTPUT_FILE" 2>&1 || echo "No workflow templates found" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    echo "--- PIPELINE RUNS ---" >> "$OUTPUT_FILE"
    kubectl get runs -n "${NAMESPACE}" -o wide --show-labels >> "$OUTPUT_FILE" 2>&1 || echo "No pipeline runs found" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    echo "--- ARGO WORKFLOW CONTROLLER CONFIG ---" >> "$OUTPUT_FILE"
    kubectl get configmap -n "${NAMESPACE}" | grep -E "(workflow|argo)" >> "$OUTPUT_FILE" 2>&1 || echo "No Argo-related ConfigMaps found" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
}

# Collect logs from user namespaces (multi-user mode)
collect_user_namespace_logs() {
    echo "===== USER NAMESPACE ANALYSIS =====" >> "$OUTPUT_FILE"

    # Common user namespace patterns
    for user_ns in "kubeflow-user-example-com" "kubeflow-user-test" "default"; do
        if kubectl get namespace "$user_ns" &>/dev/null && [[ "$user_ns" != "$NS" ]]; then
            echo "Found user namespace: $user_ns" >> "$OUTPUT_FILE"

            # All pods in user namespace
            echo "=== ALL USER NAMESPACE PODS ===" >> "$OUTPUT_FILE"
            kubectl get pods -n "$user_ns" -o wide >> "$OUTPUT_FILE" 2>&1 || echo "Failed to get pods in $user_ns" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"

            # Workflow/execution pods
            collect_user_workflow_pods "$user_ns"

            # Failed/pending pods
            collect_failed_pods "$user_ns"

            # Resource constraints
            collect_resource_info "$user_ns"

            # Recent events
            echo "=== USER NAMESPACE EVENTS ===" >> "$OUTPUT_FILE"
            kubectl get events -n "$user_ns" --sort-by='.lastTimestamp' | tail -30 >> "$OUTPUT_FILE" 2>&1 || echo "No events in $user_ns" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"

            # Workflows in user namespace
            echo "--- USER NAMESPACE WORKFLOWS ---" >> "$OUTPUT_FILE"
            kubectl get workflows -n "$user_ns" >> "$OUTPUT_FILE" 2>&1 || echo "No workflows found in $user_ns" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"
        fi
    done
}

# Collect workflow/execution pods in user namespace
collect_user_workflow_pods() {
    local user_ns=$1
    local workflow_pods

    workflow_pods=$(kubectl get pods -n "$user_ns" -o name 2>/dev/null | grep -E "(pipeline|workflow|producer|consumer|dag-driver|system)" || echo "")

    if [[ -n "$workflow_pods" ]]; then
        echo "=== USER NAMESPACE WORKFLOW PODS ===" >> "$OUTPUT_FILE"
        for pod_name in $workflow_pods; do
            pod_name=$(echo "$pod_name" | sed 's|pod/||')
            echo "--- User NS Pod: $pod_name ---" >> "$OUTPUT_FILE"

            kubectl describe pod "$pod_name" -n "$user_ns" >> "$OUTPUT_FILE" 2>&1 || echo "Failed to describe $pod_name" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"

            echo "Pod logs for $pod_name:" >> "$OUTPUT_FILE"
            kubectl logs "$pod_name" -n "$user_ns" --all-containers=true >> "$OUTPUT_FILE" 2>&1 || echo "No logs for $pod_name" >> "$OUTPUT_FILE"

            echo "Previous logs for $pod_name:" >> "$OUTPUT_FILE"
            kubectl logs "$pod_name" -n "$user_ns" --all-containers=true --previous >> "$OUTPUT_FILE" 2>&1 || echo "No previous logs for $pod_name" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"
        done
    fi
}

# Collect failed/pending pods
collect_failed_pods() {
    local user_ns=$1
    local failed_pods

    echo "=== FAILED/PENDING PODS ===" >> "$OUTPUT_FILE"
    failed_pods=$(kubectl get pods -n "$user_ns" --field-selector=status.phase!=Running,status.phase!=Succeeded -o name 2>/dev/null || echo "")

    if [[ -n "$failed_pods" ]]; then
        echo "Found non-running pods:" >> "$OUTPUT_FILE"
        for pod_name in $failed_pods; do
            pod_name=$(echo "$pod_name" | sed 's|pod/||')
            echo "--- Failed/Pending Pod: $pod_name ---" >> "$OUTPUT_FILE"

            kubectl describe pod "$pod_name" -n "$user_ns" >> "$OUTPUT_FILE" 2>&1 || echo "Failed to describe $pod_name" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"

            kubectl logs "$pod_name" -n "$user_ns" --all-containers=true >> "$OUTPUT_FILE" 2>&1 || echo "No logs for $pod_name" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"
        done
    else
        echo "No failed/pending pods found" >> "$OUTPUT_FILE"
    fi
    echo "" >> "$OUTPUT_FILE"
}

# Collect resource quotas and limits
collect_resource_info() {
    local user_ns=$1

    echo "=== RESOURCE QUOTAS ===" >> "$OUTPUT_FILE"
    kubectl get resourcequota -n "$user_ns" -o yaml >> "$OUTPUT_FILE" 2>&1 || echo "No resource quotas in $user_ns" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"

    echo "=== LIMIT RANGES ===" >> "$OUTPUT_FILE"
    kubectl get limitrange -n "$user_ns" -o yaml >> "$OUTPUT_FILE" 2>&1 || echo "No limit ranges in $user_ns" >> "$OUTPUT_FILE"
    echo "" >> "$OUTPUT_FILE"
}

# Main execution
check_namespace "$NS"
collect_comprehensive_logs "$NS"
