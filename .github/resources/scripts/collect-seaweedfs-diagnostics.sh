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
# The same 'dial tcp <clusterip>: i/o timeout' signature shows up on other
# in-cluster ClusterIPs in these lanes (for example the MLflow service VIP on
# :8443), so the dataplane failure is not SeaweedFS-specific. Section (4)
# therefore probes the node netfilter state for *every* ClusterIP that recorded
# a dial timeout, not just SeaweedFS's. Pass the already-collected pod-log file
# via --dial-timeout-log; the script extracts each timed-out VIP from it and
# always includes the SeaweedFS ClusterIP.
#
# Section (4) inspects every Kind node (a ClusterIP dial is DNAT'd and
# conntrack-tracked on the *client* pod's node, which need not be the service's
# node). The leading dataplane candidate is conntrack-table saturation: the
# nested-parallel lanes open a burst of simultaneous connections, and once a
# node conntrack table fills, new SYNs are dropped and every dial reports 'i/o
# timeout' while the backing pod's own liveness probe (kubelet -> pod IP) keeps
# passing, so the pod is never restarted. Per node it dumps the node-global
# conntrack state once -- the cumulative insert_failed/drop counters in
# /proc/net/stat/nf_conntrack and any 'nf_conntrack: table full' dmesg line are
# the durable exhaustion evidence -- then the iptables/ipvs program for each
# timed-out ClusterIP, whose presence or absence separates conntrack exhaustion
# from a missing/stale service program. Each probe distinguishes a
# failed/forbidden query from a genuinely empty result so a blank line never
# falsely rules out the signal.
#
# All commands are best-effort; a missing object never fails the caller.

set -u

NAMESPACE="kubeflow"
SELECTOR="app=seaweedfs"
OUTPUT_FILE=""
DIAL_TIMEOUT_LOG=""

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        --namespace) NAMESPACE="${2:?--namespace requires a value}"; shift ;;
        --selector) SELECTOR="${2:?--selector requires a value}"; shift ;;
        --output) OUTPUT_FILE="${2:?--output requires a value}"; shift ;;
        # A log (typically the already-collected pod logs) to scan for
        # 'dial tcp <ip>:<port>: i/o timeout'; section (4) probes each such VIP.
        --dial-timeout-log) DIAL_TIMEOUT_LOG="${2:?--dial-timeout-log requires a value}"; shift ;;
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

# Dump the iptables (or ipvs fallback) program for one ClusterIP on one Kind
# node. Tracks whether a ruleset was actually read so a missing iptables-save /
# ipvsadm binary reports "cannot confirm", never a false "no rule".
# Usage: probe_service_rules <node> <ip>
probe_service_rules() {
    docker exec "$1" sh -c '
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
        fi' _ "$2" 2>/dev/null || echo "(unavailable)"
}

# ClusterIP VIPs (ip:port) that recorded a dial timeout in the supplied log.
# Read before the diagnostics block so appended output cannot perturb the scan.
LOG_VIPS=""
if [[ -n "$DIAL_TIMEOUT_LOG" && -r "$DIAL_TIMEOUT_LOG" ]]; then
    LOG_VIPS=$(grep -oE 'dial tcp [0-9.]+:[0-9]+: i/o timeout' "$DIAL_TIMEOUT_LOG" 2>/dev/null \
        | grep -oE '[0-9.]+:[0-9]+' | sort -u || true)
fi

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

    # The SeaweedFS ClusterIP is always probed; the service listens on :9000
    # (DNAT'd to the pod's S3 port), which is the address the failing dials
    # target.
    CLUSTER_IP=$(kubectl get svc seaweedfs -n "$NAMESPACE" -o jsonpath='{.spec.clusterIP}' 2>/dev/null || true)
    echo "SeaweedFS ClusterIP: ${CLUSTER_IP:-<unresolved>}"

    # Section (4) targets the SeaweedFS VIP plus every other ClusterIP that a
    # dial timeout was logged against (e.g. the MLflow service VIP), so the
    # dataplane evidence covers whichever service the workflow pods could not
    # reach, not only SeaweedFS.
    TARGET_VIPS=$(
        { [[ -n "$CLUSTER_IP" ]] && echo "${CLUSTER_IP}:9000"; printf '%s\n' "$LOG_VIPS"; } \
            | grep -E '^[0-9.]+:[0-9]+$' | sort -u
    )

    echo
    echo "----- (4) node netfilter state for timed-out ClusterIPs -----"
    echo "ClusterIPs correlated: $(printf '%s ' ${TARGET_VIPS:-<none>})"
    # Kind runs each node as a Docker container named after the Kubernetes node,
    # so 'docker exec <node>' reaches the node's network namespace where
    # kube-proxy programs the ClusterIPs and the kernel tracks connections. A
    # ClusterIP dial is DNAT'd and conntrack-tracked on the *client* pod's node,
    # which need not be the service's node, so probe every Kind node rather than
    # only one. Best-effort: nodes that are not Docker containers (non-Kind /
    # remote runtime) or an unreachable daemon skip individually.
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

            # Node-global conntrack state: independent of any single VIP, so it
            # is dumped once per node and answers "is this node's conntrack
            # table exhausted?" for every timed-out ClusterIP at once.
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

            # Per-VIP service program: rules present + conntrack pressure ==
            # exhaustion; rules absent == kube-proxy never (re)programmed the VIP.
            if [[ -z "$TARGET_VIPS" ]]; then
                echo "service program: no timed-out ClusterIPs to correlate."
            else
                for vip in $TARGET_VIPS; do
                    echo "service program for $vip (iptables, then ipvs):"
                    probe_service_rules "$node" "${vip%%:*}"
                done
            fi
        done
    fi

    echo
    echo "----- full describe (events, resource config) -----"
    kubectl describe pod "$POD" -n "$NAMESPACE" 2>/dev/null || true

    echo "=============================================================="
} | emit
