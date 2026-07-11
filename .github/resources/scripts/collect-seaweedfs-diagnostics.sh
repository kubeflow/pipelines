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
#   2. Node CPU/memory contention    -> node allocated requests vs allocatable
#                                        and node pressure conditions. The
#                                        SeaweedFS deployment sets a CPU request
#                                        (no limit), so scheduling shares only
#                                        matter when the node is contended.
#   3. Cluster dataplane failure     -> SeaweedFS is Running with no restarts
#                                        and the node is healthy, yet the
#                                        ClusterIP path times out (kube-proxy /
#                                        CNI). Ruled in by 1 and 2 being clean
#                                        plus kube-proxy pod state, and probed
#                                        directly in section (4).
#
# Section (4) inspects every Kind node's netfilter state for the failing
# SeaweedFS ClusterIP (a ClusterIP dial is DNAT'd and conntrack-tracked on the
# client pod's node, which need not be the SeaweedFS node). The leading
# dataplane candidate is conntrack-table saturation: the nested-parallel lanes
# open a burst of simultaneous connections, and once a node conntrack table
# fills, new SYNs are dropped and every dial reports 'i/o timeout' while the
# pod's own liveness probe (kubelet -> pod IP) keeps passing, so the pod is
# never restarted. The durable evidence is the cumulative insert_failed/drop
# counters in /proc/net/stat/nf_conntrack and any 'nf_conntrack: table full'
# dmesg line; the presence or absence of iptables/ipvs rules for the ClusterIP
# separates conntrack exhaustion from a missing/stale service program. Each
# probe distinguishes a failed/forbidden query from a genuinely empty result
# so a blank line never falsely rules out the signal.
#
# All commands are best-effort; a missing object never fails the caller.

set -u

NAMESPACE="kubeflow"
SELECTOR="app=seaweedfs"
OUTPUT_FILE=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        --namespace) NAMESPACE="${2:?--namespace requires a value}"; shift ;;
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
    POD=$(kubectl get pod -n "$NAMESPACE" -l "$SELECTOR" --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || true)
    if [[ -z "$POD" ]]; then
        echo "No SeaweedFS pod (selector '$SELECTOR') found in namespace '$NAMESPACE'; skipping."
        echo "=============================================================="
        exit 0
    fi
    echo "SeaweedFS pod: $POD"

    echo
    echo "----- (1) restarts / last terminated state -----"
    # A non-zero restart count or an OOMKilled/Error last state means the dial
    # timeouts are the pod being down, and the CPU request is not the lever.
    kubectl get pod "$POD" -n "$NAMESPACE" -o wide 2>/dev/null || true
    kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='restartCount={.status.containerStatuses[0].restartCount}{"\n"}lastState={.status.containerStatuses[0].lastState}{"\n"}qosClass={.status.qosClass}{"\n"}' 2>/dev/null || true

    NODE=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.spec.nodeName}' 2>/dev/null || true)

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
    kubectl get endpoints seaweedfs -n "$NAMESPACE" -o wide 2>/dev/null || true

    # The ClusterIP is the address the failing dials target; section (4) probes
    # the node's netfilter state for exactly this VIP.
    CLUSTER_IP=$(kubectl get svc seaweedfs -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}' 2>/dev/null || true)
    echo "SeaweedFS ClusterIP: ${CLUSTER_IP:-<unresolved>}"

    echo
    echo "----- (4) node netfilter state for ClusterIP ${CLUSTER_IP:-<unresolved>} -----"
    # Kind runs each node as a Docker container named after the Kubernetes node,
    # so 'docker exec <node>' reaches the node's network namespace where
    # kube-proxy programs the ClusterIP and the kernel tracks connections. A
    # ClusterIP dial is DNAT'd and conntrack-tracked on the *client* pod's node,
    # which need not be the SeaweedFS node, so probe every Kind node rather than
    # only the destination. Best-effort: nodes that are not Docker containers
    # (non-Kind / remote runtime) or an unreachable daemon skip individually.
    if ! command -v docker >/dev/null 2>&1; then
        echo "docker not available from this runner; skipping node-level dataplane probe."
    else
        NODES=$(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true)
        if [[ -z "$NODES" ]]; then
            echo "Could not list cluster nodes; skipping node-level dataplane probe."
        fi
        for node in $NODES; do
            echo "===== node: $node ====="
            if ! docker inspect "$node" >/dev/null 2>&1; then
                echo "Node '$node' is not a Docker container (non-Kind or remote runtime); skipping."
                continue
            fi

            echo "conntrack in-use / max:"
            # Point-in-time gauge; may have drained by the time diagnostics run.
            docker exec "$node" sh -c 'cat /proc/sys/net/netfilter/nf_conntrack_count /proc/sys/net/netfilter/nf_conntrack_max 2>/dev/null | paste -sd/ -' 2>/dev/null || echo "(unavailable)"

            echo "conntrack stat header + totals (durable; nonzero insert_failed/drop == table pressure):"
            # /proc/net/stat/nf_conntrack has one row per CPU; the insert_failed
            # and drop columns are cumulative since boot and survive the burst
            # draining.
            docker exec "$node" sh -c 'cat /proc/net/stat/nf_conntrack 2>/dev/null' 2>/dev/null || echo "(unavailable)"

            echo "kernel 'nf_conntrack: table full' events:"
            # Distinguish a forbidden/failed dmesg from a successful empty query:
            # a blank line must not read as "no event" and falsely rule out the
            # main signal this section captures.
            docker exec "$node" sh -c '
                out=$(dmesg 2>/dev/null) || { echo "(dmesg unavailable — cannot confirm table-full events)"; exit 0; }
                if printf "%s\n" "$out" | grep -qi "nf_conntrack: table full"; then
                    printf "%s\n" "$out" | grep -i "nf_conntrack: table full" | tail -5
                else
                    echo "(no table-full event logged)"
                fi' 2>/dev/null || echo "(unavailable)"

            echo "service program for ${CLUSTER_IP:-<unresolved>} (iptables, then ipvs):"
            # Rules present + conntrack pressure == exhaustion; rules absent ==
            # kube-proxy never (re)programmed the VIP. Track whether any ruleset
            # was actually read so a missing iptables/ipvsadm binary is reported
            # as "cannot confirm", never as confirmed rule absence.
            if [[ -z "$CLUSTER_IP" ]]; then
                echo "ClusterIP unresolved; cannot query service program."
                continue
            fi
            docker exec "$node" sh -c '
                ip=$1
                read_any=0
                match=""
                if ipt=$(iptables-save 2>/dev/null); then
                    read_any=1
                    match=$(printf "%s\n" "$ipt" | grep -F "$ip")
                fi
                if [ -z "$match" ] && ipvs=$(ipvsadm -Ln 2>/dev/null); then
                    read_any=1
                    match=$(printf "%s\n" "$ipvs" | grep -A4 "$ip")
                fi
                if [ -n "$match" ]; then
                    printf "%s\n" "$match"
                elif [ "$read_any" = 1 ]; then
                    echo "(no iptables/ipvs rule references $ip)"
                else
                    echo "(iptables-save and ipvsadm both unavailable — cannot confirm rule presence)"
                fi' _ "$CLUSTER_IP" 2>/dev/null || echo "(unavailable)"
        done
    fi

    echo
    echo "----- full describe (events, resource config) -----"
    kubectl describe pod "$POD" -n "$NAMESPACE" 2>/dev/null || true

    echo "=============================================================="
} | emit
