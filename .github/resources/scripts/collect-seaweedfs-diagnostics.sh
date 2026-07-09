#!/usr/bin/env bash
#
# Collects targeted SeaweedFS diagnostics after an E2E test failure.
#
# E2E lanes intermittently fail with many
#   dial tcp <seaweedfs-clusterip>:9000: i/o timeout
# errors while pipelines upload artifacts and logs. That signature (the TCP
# connection is never established) has three candidate causes, and this script
# captures the evidence that distinguishes them without needing metrics-server
# (which these Kind clusters do not run):
#
#   1. SeaweedFS crashed/restarted   -> pod restart count and last terminated
#                                        state (OOMKilled, etc.)
#   2. Node CPU/mem contention       -> node allocated requests vs allocatable
#                                        and node pressure conditions. The
#                                        SeaweedFS deployment sets a CPU request
#                                        (no limit), so scheduling shares only
#                                        matter when the node is contended.
#   3. Cluster dataplane failure     -> SeaweedFS is Running with no restarts
#                                        and the node is healthy, yet the
#                                        ClusterIP path times out (kube-proxy /
#                                        CNI). Captured by ruling out 1 and 2
#                                        plus kube-proxy pod state.
#
# All commands are best-effort; a missing object never fails the caller.

set -u

NS="kubeflow"
SELECTOR="app=seaweedfs"
OUTPUT_FILE=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        --ns) NS="${2:?--ns requires a value}"; shift ;;
        --selector) SELECTOR="${2:?--selector requires a value}"; shift ;;
        --output) OUTPUT_FILE="${2:?--output requires a value}"; shift ;;
        *) echo "Unknown parameter passed: $1"; exit 1 ;;
    esac
    shift
done

emit() {
    if [[ -n "$OUTPUT_FILE" ]]; then
        mkdir -p "$(dirname "$OUTPUT_FILE")"
        tee -a "$OUTPUT_FILE"
    else
        cat
    fi
}

{
    echo "================ SeaweedFS failure diagnostics ================"

    # Newest matching pod: with strategy Recreate there is normally one, but a
    # rollout can briefly leave a Terminating pod alongside the new one, and the
    # new pod is the one whose state matters.
    POD=$(kubectl get pod -n "$NS" -l "$SELECTOR" --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || true)
    if [[ -z "$POD" ]]; then
        echo "No SeaweedFS pod (selector '$SELECTOR') found in namespace '$NS'; skipping."
        echo "=============================================================="
        exit 0
    fi
    echo "SeaweedFS pod: $POD"

    echo
    echo "----- (1) restarts / last terminated state -----"
    # A non-zero restart count or an OOMKilled/Error last state means the dial
    # timeouts are the pod being down, and the CPU request is not the lever.
    kubectl get pod "$POD" -n "$NS" -o wide 2>/dev/null || true
    kubectl get pod "$POD" -n "$NS" -o jsonpath='restartCount={.status.containerStatuses[0].restartCount}{"\n"}lastState={.status.containerStatuses[0].lastState}{"\n"}qosClass={.status.qosClass}{"\n"}' 2>/dev/null || true

    NODE=$(kubectl get pod "$POD" -n "$NS" -o jsonpath='{.spec.nodeName}' 2>/dev/null || true)

    echo
    echo "----- (2) node contention on $NODE -----"
    # 'Allocated resources' shows summed CPU requests vs allocatable. If CPU
    # requests approach allocatable, SeaweedFS competes for shares under load
    # and a larger request helps; if the node is idle, contention is not the
    # cause and a request bump will not help.
    if [[ -n "$NODE" ]]; then
        kubectl describe node "$NODE" 2>/dev/null \
            | sed -n '/Conditions:/,/Addresses:/p;/Allocated resources:/,/Events:/p' || true
    else
        echo "Could not resolve SeaweedFS node."
    fi

    echo
    echo "----- (3) dataplane: kube-proxy / CNI -----"
    # If SeaweedFS is Running with no restarts and the node is healthy but the
    # ClusterIP still times out, the failure is in the service dataplane.
    kubectl get pod -n kube-system -l k8s-app=kube-proxy -o wide 2>/dev/null || true
    # Query Endpoints by service name: the SeaweedFS Service has no labels, so a
    # label selector matches nothing (and would still exit 0, masking the miss).
    # Empty ENDPOINTS here means the Service has no ready backends.
    kubectl get endpoints seaweedfs -n "$NS" -o wide 2>/dev/null || true

    echo
    echo "----- full describe (events, resource config) -----"
    kubectl describe pod "$POD" -n "$NS" 2>/dev/null || true

    echo "=============================================================="
} | emit
