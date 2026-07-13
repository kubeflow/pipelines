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
# node). One dataplane candidate is conntrack-table saturation: the
# nested-parallel lanes open a burst of simultaneous connections, and once a
# node conntrack table fills, new SYNs are dropped and every dial reports 'i/o
# timeout' while the backing pod's own liveness probe (kubelet -> pod IP) keeps
# passing, so the pod is never restarted. Per node it dumps the node-global
# conntrack state once -- the durable drop/insert_failed counters (conntrack -S,
# falling back to /proc/net/stat/nf_conntrack) and any 'nf_conntrack: table
# full' dmesg line -- then the iptables/ipvs program for each timed-out
# ClusterIP, followed end to end (KUBE-SERVICES -> KUBE-SVC -> KUBE-SEP DNAT) so
# a live endpoint DNAT rules out a missing/stale service program.
#
# When the conntrack table and the service program are both clean (as first
# observed live: table ~0.6% full, no table-full event, KUBE-SEP DNAT present),
# the remaining candidate is accept-queue saturation on the backing pod itself.
# Section (5) reads the SeaweedFS pod's listen sockets and cumulative
# ListenOverflows / ListenDrops from inside its netns to confirm or rule that
# out. Each probe distinguishes a failed/forbidden query from a genuinely empty
# result so a blank line never falsely rules out the signal.
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

# Dump the iptables (or ipvs fallback) program for one ClusterIP:port on one
# Kind node, following the chain end to end: the KUBE-SERVICES match, the
# KUBE-SVC chain for the port, and the KUBE-SEP DNAT to the backing pod. A live
# KUBE-SEP DNAT proves kube-proxy programmed a real endpoint (so a dial timeout
# is not a missing/stale rule); an empty KUBE-SVC chain means no ready backend.
# Tracks whether a ruleset was actually read so a missing iptables-save /
# ipvsadm binary reports "cannot confirm", never a false "no rule".
# Usage: probe_service_rules <node> <ip> <port>
probe_service_rules() {
    docker exec "$1" sh -c '
        ip=$1; port=$2
        rules=$(iptables-save 2>/dev/null)
        if [ -z "$rules" ]; then
            if ipvs=$(ipvsadm -Ln 2>/dev/null); then
                printf "%s\n" "$ipvs" | grep -A6 "$ip:$port" \
                    || echo "(no ipvs entry for $ip:$port)"
            else
                echo "(iptables-save and ipvsadm both unavailable — cannot confirm rule presence)"
            fi
            exit 0
        fi
        svc_rules=$(printf "%s\n" "$rules" | grep -F "$ip" | grep -E "KUBE-SERVICES|KUBE-MARK-MASQ")
        if [ -n "$svc_rules" ]; then
            printf "%s\n" "$svc_rules"
        else
            echo "(no KUBE-SERVICES rule references $ip)"
        fi
        # Resolve the KUBE-SVC chain for this specific port, then its KUBE-SEP
        # endpoint chain(s) and their DNAT target (the pod the VIP forwards to).
        svc=$(printf "%s\n" "$rules" | grep -F "$ip" | grep -E -- "--dport $port -j KUBE-SVC-" \
            | grep -oE "KUBE-SVC-[A-Z0-9]+" | head -1)
        if [ -z "$svc" ]; then
            echo "  (no KUBE-SVC chain for $ip:$port — port not programmed)"
            exit 0
        fi
        echo "  endpoint chain $svc ->"
        printf "%s\n" "$rules" | grep -E -- "-A $svc " | sed "s/^/    /" \
            || echo "    (chain $svc empty — no ready endpoints)"
        seps=$(printf "%s\n" "$rules" | grep -E -- "-A $svc .*-j KUBE-SEP-" \
            | grep -oE "KUBE-SEP-[A-Z0-9]+" | sort -u)
        if [ -z "$seps" ]; then
            echo "    (no KUBE-SEP endpoint jump — VIP has no live backend)"
        else
            for sep in $seps; do
                printf "%s\n" "$rules" | grep -E -- "-A $sep .*(DNAT|to-destination)" | sed "s/^/    /" \
                    || echo "    ($sep: no DNAT rule)"
            done
        fi' _ "$2" "$3" 2>/dev/null || echo "(unavailable)"
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

            echo "conntrack drop / insert_failed counters (durable; nonzero == table pressure):"
            # Cumulative-since-boot counters survive the burst draining, unlike
            # the in-use gauge above. 'conntrack -S' is the reliable source;
            # /proc/net/stat/nf_conntrack is a fallback but was observed absent
            # on some Kind node kernels, so try the CLI first.
            docker exec "$node" sh -c '
                if command -v conntrack >/dev/null 2>&1 && conntrack -S 2>/dev/null; then
                    :
                elif [ -r /proc/net/stat/nf_conntrack ]; then
                    cat /proc/net/stat/nf_conntrack
                else
                    echo "(conntrack -S and /proc/net/stat/nf_conntrack both unavailable)"
                fi' 2>/dev/null || echo "(unavailable)"

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

            # Per-VIP service program end to end (KUBE-SERVICES -> KUBE-SVC ->
            # KUBE-SEP DNAT): a live DNAT to the pod means the rule is not the
            # problem; an empty chain means no ready backend.
            if [[ -z "$TARGET_VIPS" ]]; then
                echo "service program: no timed-out ClusterIPs to correlate."
            else
                for vip in $TARGET_VIPS; do
                    echo "service program for $vip (iptables, then ipvs):"
                    probe_service_rules "$node" "${vip%%:*}" "${vip##*:}"
                done
            fi
        done
    fi

    echo
    echo "----- (5) SeaweedFS pod socket / listen-queue state -----"
    # Once the node conntrack table and the service program are clean, the
    # leading remaining cause of the ClusterIP dial timeouts is accept-queue
    # saturation: under the parallel burst the S3 server's listen backlog
    # overflows and new SYNs are dropped inside the pod netns (the client sees a
    # dial timeout) while an already-established liveness connection keeps
    # passing, so the pod is never restarted. ListenOverflows / ListenDrops in
    # the pod's /proc/net/netstat are cumulative since pod start, so they
    # survive the burst; 'ss -lnt' shows the live backlog (Recv-Q = pending
    # connections, Send-Q = backlog limit) when iproute2 is present. Read from
    # inside the pod netns via kubectl exec; best-effort if the container lacks
    # a shell or the tools.
    echo "listen sockets (Recv-Q=pending backlog / Send-Q=backlog limit):"
    kubectl exec "$POD" -n "$NAMESPACE" -c seaweedfs -- sh -c 'ss -lnt 2>/dev/null || netstat -lnt 2>/dev/null || echo "(ss/netstat not present in container)"' 2>/dev/null || echo "(kubectl exec unavailable)"
    echo "TcpExt counters (nonzero ListenOverflows / ListenDrops == accept-queue saturation):"
    kubectl exec "$POD" -n "$NAMESPACE" -c seaweedfs -- sh -c 'grep "^TcpExt" /proc/net/netstat 2>/dev/null || echo "(/proc/net/netstat unavailable)"' 2>/dev/null || echo "(kubectl exec unavailable)"

    echo
    echo "----- full describe (events, resource config) -----"
    kubectl describe pod "$POD" -n "$NAMESPACE" 2>/dev/null || true

    echo "=============================================================="
} | emit
