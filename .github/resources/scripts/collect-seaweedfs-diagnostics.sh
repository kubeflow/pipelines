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
#                                        and node pressure conditions. These are
#                                        scheduling context, not actual CPU-use
#                                        measurements, so runner telemetry is
#                                        still needed to assess live contention.
#   3. Cluster dataplane failure     -> SeaweedFS is Running with no restarts
#                                        and the node is healthy, yet the
#                                        ClusterIP path times out (kube-proxy /
#                                        CNI). Narrowed by 1 and 2 being clean
#                                        plus kube-proxy pod state, and probed
#                                        directly in sections (4) and (5).
#
# The same connection failures show up on other in-cluster ClusterIPs in these
# lanes (for example the MLflow service VIP on :8443), so the dataplane failure
# is not SeaweedFS-specific. Sections (4) and (5) therefore inspect *every*
# ClusterIP that recorded a TCP or UDP timeout, refusal, or reset, not just
# SeaweedFS's.
# Pass the already-collected pod-log file via --connection-failure-log (the old
# --dial-timeout-log name remains an alias); the script extracts each failed VIP
# from it and always includes the SeaweedFS ClusterIP.
#
# Section (4) snapshots the matched Service, EndpointSlices, and backing-pod
# state. Section (5) then inspects every Kind node (a ClusterIP dial is DNAT'd and
# conntrack-tracked on the *client* pod's node, which need not be the service's
# node). One dataplane candidate is conntrack-table saturation: the
# nested-parallel lanes open a burst of simultaneous connections, and once a
# node conntrack table fills, new SYNs are dropped and every dial reports 'i/o
# timeout' while the backing pod's own liveness probe (kubelet -> pod IP) keeps
# passing, so the pod is never restarted. Per node it dumps the node-global
# conntrack state once -- the cumulative drop/insert_failed counters (conntrack
# -S, falling back to /proc/net/stat/nf_conntrack) and any 'nf_conntrack: table
# full' dmesg line -- then the counter-bearing iptables/ipvs program for each
# failed ClusterIP. The program is followed end to end (KUBE-SERVICES ->
# KUBE-SVC -> KUBE-SEP DNAT), including the mark/postrouting rules that show
# whether matching traffic is eligible for SNAT and whether MASQUERADE uses
# --random-fully. A live endpoint DNAT rules out a missing/stale service program.
#
# When the conntrack table and the service program are both clean (as first
# observed live: table ~0.6% full, no table-full event, KUBE-SEP DNAT present),
# another candidate is accept-queue saturation on the backing pod itself.
# Section (6) reads the SeaweedFS pod's listen sockets and cumulative
# ListenOverflows / ListenDrops from inside its netns to confirm or rule that
# out. Each probe distinguishes a failed/forbidden query from a genuinely empty
# result so a blank line never falsely rules out the signal.
#
# All commands are best-effort; a missing object never fails the caller.

set -u

NAMESPACE="kubeflow"
SELECTOR="app=seaweedfs"
OUTPUT_FILE=""
CONNECTION_FAILURE_LOG=""
SERVICE_INVENTORY=""
SERVICE_INVENTORY_AVAILABLE=false

while [[ "$#" -gt 0 ]]; do
    case "$1" in
        --namespace) NAMESPACE="${2:?--namespace requires a value}"; shift ;;
        --selector) SELECTOR="${2:?--selector requires a value}"; shift ;;
        --output) OUTPUT_FILE="${2:?--output requires a value}"; shift ;;
        # A log (typically the already-collected pod logs) to scan for network
        # failures. Keep --dial-timeout-log as a compatibility alias.
        --connection-failure-log|--dial-timeout-log)
            CONNECTION_FAILURE_LOG="${2:?$1 requires a value}"; shift ;;
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

format_targets() {
    sed -E 's/^(tcp|udp)\|/\1:\/\//g' | paste -sd' ' -
}

# Dump the iptables (or ipvs fallback) program for one ClusterIP:port on one
# Kind node, following the chain end to end: the KUBE-SERVICES match, the
# KUBE-SVC chain for the port, and each KUBE-SEP rule for the backing pod. With
# iptables-save -c, every printed rule carries its cumulative packet/byte
# counters. A live KUBE-SEP DNAT proves kube-proxy programmed a real endpoint
# (so a dial timeout is not a missing/stale rule); an empty KUBE-SVC chain means
# no ready backend. Node-wide KUBE-MARK-MASQ and KUBE-POSTROUTING rules are
# printed once separately rather than repeated for every failed VIP.
# Tracks whether a ruleset was actually read so a missing iptables-save /
# ipvsadm binary reports "cannot confirm", never a false "no rule".
# Usage: probe_service_rules <node> <protocol> <ip> <port>
probe_service_rules() {
    docker exec "$1" sh -c '
        protocol=$1; ip=$2; port=$3
        if rules=$(iptables-save -c 2>/dev/null) && [ -n "$rules" ]; then
            counter_note="iptables counters: cumulative [packets:bytes] values"
        else
            rules=$(iptables-save 2>/dev/null)
            counter_note="iptables packet counters unavailable"
        fi
        if [ -z "$rules" ]; then
            if ipvs=$(ipvsadm -Ln --stats 2>/dev/null || ipvsadm -Ln 2>/dev/null); then
                protocol_upper=$(printf "%s" "$protocol" | tr "[:lower:]" "[:upper:]")
                entry=$(printf "%s\n" "$ipvs" \
                    | awk -v protocol="$protocol_upper" -v target="$ip:$port" "
                        \$1 ~ /^(TCP|UDP)$/ {
                            if (capturing) exit
                            if (\$1 == protocol && \$2 == target) {
                                capturing=1
                                print
                            }
                            next
                        }
                        capturing {print}
                    ")
                if [ -n "$entry" ]; then
                    printf "%s\n" "$entry"
                else
                    echo "(no $protocol ipvs entry for $ip:$port)"
                fi
            else
                echo "(iptables-save and ipvsadm both unavailable — cannot confirm rule presence)"
            fi
            exit 0
        fi
        echo "  $counter_note"
        svc_rules=$(printf "%s\n" "$rules" | grep -F "$ip" \
            | grep -E -- "-p $protocol( |$)" | grep -E -- "--dport $port( |$)" \
            | grep -E -- "-A KUBE-SERVICES ")
        if [ -n "$svc_rules" ]; then
            printf "%s\n" "$svc_rules"
        else
            echo "(no $protocol KUBE-SERVICES rule references $ip:$port)"
        fi
        # Resolve the KUBE-SVC chain for this specific port, then its KUBE-SEP
        # endpoint chain(s) and their DNAT target (the pod the VIP forwards to).
        svc=$(printf "%s\n" "$rules" | grep -F "$ip" | grep -E -- "-p $protocol( |$)" \
            | grep -E -- "--dport $port -j KUBE-SVC-" \
            | grep -oE "KUBE-SVC-[A-Z0-9]+" | head -1)
        if [ -z "$svc" ]; then
            echo "  (no $protocol KUBE-SVC chain for $ip:$port — port not programmed)"
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
                sep_rules=$(printf "%s\n" "$rules" | grep -E -- "-A $sep ")
                if [ -n "$sep_rules" ]; then
                    printf "%s\n" "$sep_rules" | sed "s/^/    /"
                else
                    echo "    ($sep: no endpoint rules)"
                fi
            done
        fi' _ "$2" "$3" "$4" 2>/dev/null || echo "(unavailable)"
}

# Dump the node-wide SNAT plumbing once. Per-VIP KUBE-SVC/KUBE-SEP counters
# above show whether a matching flow jumped to KUBE-MARK-MASQ; these global
# rules only show what happens after that mark is set.
# Usage: probe_masquerade_rules <node>
probe_masquerade_rules() {
    docker exec "$1" sh -c '
        if rules=$(iptables-save -c 2>/dev/null) && [ -n "$rules" ]; then
            echo "iptables counters: cumulative [packets:bytes] values"
        else
            rules=$(iptables-save 2>/dev/null)
            echo "iptables packet counters unavailable"
        fi
        if [ -z "$rules" ]; then
            echo "(iptables-save unavailable — cannot inspect masquerade rules)"
            exit 0
        fi

        mark_rules=$(printf "%s\n" "$rules" | grep -E -- "-A KUBE-MARK-MASQ ")
        if [ -n "$mark_rules" ]; then
            printf "%s\n" "$mark_rules"
        else
            echo "(KUBE-MARK-MASQ rule unavailable)"
        fi

        postrouting_rules=$(printf "%s\n" "$rules" \
            | grep -E -- "-A KUBE-POSTROUTING .*MASQUERADE")
        if [ -n "$postrouting_rules" ]; then
            printf "%s\n" "$postrouting_rules"
            if printf "%s\n" "$postrouting_rules" | grep -q -- "--random-fully"; then
                echo "MASQUERADE --random-fully: enabled"
            else
                echo "MASQUERADE --random-fully: not present"
            fi
        else
            echo "(KUBE-POSTROUTING MASQUERADE rule unavailable)"
            echo "MASQUERADE --random-fully: cannot determine"
        fi' 2>/dev/null || echo "(unavailable)"
}

# Snapshot the Kubernetes objects behind one failed ClusterIP. Unlike the
# node-level service program below, this exposes a temporarily missing endpoint,
# restarting backend, or last termination reason. The snapshot is post-failure,
# so recovered state is evidence but not proof that the endpoint was ready when
# the connection failed.
# Usage: probe_service_backends <ip> <port>
probe_service_backends() {
    local ip="$1"
    local port="$2"
    if [[ "$SERVICE_INVENTORY_AVAILABLE" != "true" ]]; then
        echo "Service lookup unavailable for $ip:$port."
        return
    fi

    local services
    services=$(printf '%s\n' "$SERVICE_INVENTORY" \
        | awk -F '\t' -v ip="$ip" '$3 == ip {print $1 "\t" $2}' || true)

    if [[ -z "$services" ]]; then
        echo "No Kubernetes Service currently owns $ip:$port."
        return
    fi

    while IFS=$'\t' read -r service_namespace service_name; do
        [[ -n "$service_namespace" && -n "$service_name" ]] || continue
        echo "Service: $service_namespace/$service_name (failed VIP $ip:$port)"
        kubectl get service "$service_name" -n "$service_namespace" -o wide 2>/dev/null \
            || echo "(service snapshot unavailable)"

        echo "EndpointSlices:"
        kubectl get endpointslice -n "$service_namespace" \
            -l "kubernetes.io/service-name=$service_name" -o wide 2>/dev/null \
            || echo "(EndpointSlice snapshot unavailable)"

        local targets
        if ! targets=$(kubectl get endpointslice -n "$service_namespace" \
            -l "kubernetes.io/service-name=$service_name" \
            -o jsonpath='{range .items[*].endpoints[*]}{.addresses[0]}{"|"}{.conditions.ready}{"|"}{.conditions.serving}{"|"}{.conditions.terminating}{"|"}{.targetRef.kind}{"|"}{.targetRef.namespace}{"|"}{.targetRef.name}{"\n"}{end}' \
            2>/dev/null); then
            echo "(EndpointSlice backend lookup unavailable)"
            continue
        fi
        if [[ -z "$targets" ]]; then
            echo "(no EndpointSlice backends found)"
            continue
        fi

        echo "Backing endpoints (address, ready, serving, terminating, target):"
        printf '%s\n' "$targets"
        while IFS='|' read -r address ready serving terminating target_kind target_namespace target_name; do
            [[ "$target_kind" == "Pod" && -n "$target_name" ]] || continue
            target_namespace="${target_namespace:-$service_namespace}"
            echo "Backing pod: $target_namespace/$target_name ($address; ready=$ready serving=$serving terminating=$terminating)"
            kubectl get pod "$target_name" -n "$target_namespace" -o wide 2>/dev/null \
                || echo "(backing pod snapshot unavailable)"
            kubectl get pod "$target_name" -n "$target_namespace" \
                -o jsonpath='{range .status.containerStatuses[*]}container={.name} ready={.ready} restarts={.restartCount} state={.state} lastState={.lastState}{"\n"}{end}' \
                2>/dev/null || echo "(backing pod status unavailable)"
            echo "Backing pod events:"
            local pod_description
            if pod_description=$(kubectl describe pod "$target_name" -n "$target_namespace" 2>/dev/null); then
                printf '%s\n' "$pod_description" | sed -n '/Events:/,$p'
            else
                echo "(backing pod events unavailable)"
            fi
        done <<< "$targets"
    done <<< "$services"
}

# ClusterIP targets (protocol|ip:port) that recorded a timeout, refusal, or
# reset in the supplied log. For established connections and UDP DNS queries,
# select the destination after '->' rather than the client address. Retaining
# the protocol is essential for Services such as CoreDNS that expose both TCP
# and UDP on port 53.
# Read before the diagnostics block so appended output cannot perturb the scan.
LOG_TARGETS=""
LOG_SERVICE_TARGETS=""
if [[ -n "$CONNECTION_FAILURE_LOG" && -r "$CONNECTION_FAILURE_LOG" ]]; then
    LOG_TARGETS=$(sed -nE \
        -e 's/.*dial (tcp|udp) ([0-9.]+:[0-9]+): (i\/o timeout|connect: connection refused|connect: connection reset by peer).*/\1|\2/p' \
        -e 's/.*(read|write) (tcp|udp) [0-9.]+:[0-9]+->([0-9.]+:[0-9]+): .*(i\/o timeout|connection refused|connection reset by peer).*/\2|\3/p' \
        "$CONNECTION_FAILURE_LOG" 2>/dev/null | sort -u || true)
    # HTTP clients report the Service DNS name rather than its numeric
    # ClusterIP when they connect successfully but time out awaiting headers.
    # Preserve namespace/name/port until the one-shot Service inventory below
    # can resolve it without adding another API call per failure.
    LOG_SERVICE_TARGETS=$(sed -nE \
        -e 's#.*https?://([[:alnum:]-]+)[.]([[:alnum:]-]+)[.]svc([.]cluster[.]local)?:([0-9]+)[^ ]*.*(context deadline exceeded|Client[.]Timeout exceeded).*#tcp|\2|\1|\4#p' \
        "$CONNECTION_FAILURE_LOG" 2>/dev/null | sort -u || true)
fi

{
    echo "================ SeaweedFS failure diagnostics ================"

    # Newest matching pod: with strategy Recreate there is normally one, but a
    # rollout can briefly leave a Terminating pod alongside the new one, and the
    # new pod is the one whose state matters.
    POD=$(kubectl get pod -n "$NAMESPACE" -l "$SELECTOR" --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}' 2>/dev/null || true)
    if [[ -z "$POD" ]]; then
        # Pod-independent probes (Service inventory, sections 4-5) must still
        # run: other Services' logged connection failures (e.g. the MLflow
        # VIP) deserve their snapshots even when SeaweedFS itself is absent.
        echo "No SeaweedFS pod (selector '$SELECTOR') found in namespace '$NAMESPACE'; skipping pod-scoped sections (1, 2, 6)."
    else
        echo "SeaweedFS pod: $POD"
    fi

    NODE=""
    if [[ -n "$POD" ]]; then
        echo
        echo "----- (1) restarts / last terminated state -----"
        # A non-zero restart count or an OOMKilled/Error last state means the
        # dial timeouts are the pod being down, and the CPU request is not the
        # lever.
        kubectl get pod "$POD" -n "$NAMESPACE" -o wide 2>/dev/null || true
        kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='restartCount={.status.containerStatuses[0].restartCount}{"\n"}lastState={.status.containerStatuses[0].lastState}{"\n"}qosClass={.status.qosClass}{"\n"}' 2>/dev/null || true

        NODE=$(kubectl get pod "$POD" -n "$NAMESPACE" -o jsonpath='{.spec.nodeName}' 2>/dev/null || true)

        echo
        echo "----- (2) node contention on $NODE -----"
        # 'Allocated resources' shows summed CPU requests vs allocatable. If
        # CPU requests approach allocatable, SeaweedFS competes for shares
        # under load and a larger request helps; if the node is idle,
        # contention is not the cause and a request bump will not help.
        if [[ -n "$NODE" ]]; then
            kubectl describe node "$NODE" 2>/dev/null \
                | sed -n '/Conditions:/,/Addresses:/p;/Allocated resources:/,/Events:/p' || true
        else
            echo "Could not resolve SeaweedFS node."
        fi
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

    # Snapshot Services once so all later sections use one consistent mapping,
    # and so historical startup-probe failures to Pod IPs in describe output do
    # not get mislabeled and probed as ClusterIP failures.
    if SERVICE_INVENTORY=$(kubectl get service -A \
        -o jsonpath='{range .items[*]}{.metadata.namespace}{"\t"}{.metadata.name}{"\t"}{.spec.clusterIP}{"\n"}{end}' \
        2>/dev/null); then
        SERVICE_INVENTORY_AVAILABLE=true
    else
        SERVICE_INVENTORY_AVAILABLE=false
    fi
    FAILED_SERVICE_TARGETS=""
    UNOWNED_CONNECTION_TARGETS=""
    UNRESOLVED_SERVICE_NAMES=""
    if [[ "$SERVICE_INVENTORY_AVAILABLE" == "true" ]]; then
        for named_target in $LOG_SERVICE_TARGETS; do
            IFS='|' read -r protocol service_namespace service_name service_port <<< "$named_target"
            service_ip=$(printf '%s\n' "$SERVICE_INVENTORY" \
                | awk -F '\t' -v namespace="$service_namespace" -v name="$service_name" \
                    '$1 == namespace && $2 == name {print $3; exit}')
            if [[ "$service_ip" =~ ^[0-9.]+$ ]]; then
                LOG_TARGETS+=$'\n'"${protocol}|${service_ip}:${service_port}"
            else
                UNRESOLVED_SERVICE_NAMES+="${service_name}.${service_namespace}.svc:${service_port}"$'\n'
            fi
        done
        LOG_TARGETS=$(printf '%s\n' "$LOG_TARGETS" | grep -E '^(tcp|udp)\|' | sort -u || true)
        for target in $LOG_TARGETS; do
            vip="${target#*|}"
            if printf '%s\n' "$SERVICE_INVENTORY" \
                | awk -F '\t' -v ip="${vip%%:*}" '$3 == ip {found=1} END {exit !found}'; then
                FAILED_SERVICE_TARGETS+="${target}"$'\n'
            else
                UNOWNED_CONNECTION_TARGETS+="${target}"$'\n'
            fi
        done
    else
        # Preserve node-level evidence when the API is unavailable; do not
        # misclassify lookup failure as proof that these Services do not exist.
        FAILED_SERVICE_TARGETS="$LOG_TARGETS"
    fi

    # Sections (4) and (5) target the SeaweedFS VIP plus every other ClusterIP
    # that logged a timeout, refusal, or reset (e.g. the MLflow service VIP), so
    # the dataplane evidence covers whichever service the workflow pods could
    # not reach, not only SeaweedFS.
    TARGETS=$(
        { [[ -n "$CLUSTER_IP" ]] && echo "tcp|${CLUSTER_IP}:9000"; printf '%s' "$FAILED_SERVICE_TARGETS"; } \
            | grep -E '^(tcp|udp)\|[0-9.]+:[0-9]+$' | sort -u
    )
    BACKEND_VIPS=$(
        printf '%s\n' "$TARGETS" | sed -E 's/^(tcp|udp)\|//' \
            | grep -E '^[0-9.]+:[0-9]+$' | sort -u
    )

    echo
    echo "----- (4) service and backend state for failed ClusterIPs -----"
    echo "ClusterIPs correlated: $(printf '%s\n' "${TARGETS:-<none>}" | format_targets)"
    if [[ "$SERVICE_INVENTORY_AVAILABLE" != "true" ]]; then
        echo "Service inventory unavailable; ClusterIP ownership cannot be confirmed."
    fi
    if [[ -n "$UNOWNED_CONNECTION_TARGETS" ]]; then
        # A Service deleted, recreated, or re-IP'd after the failure leaves its
        # old VIP absent from the current inventory — exactly the stale-Service
        # scenario the node probe exists to catch. Skip only the backend-object
        # lookups for these; section (5) still probes their node programs.
        echo "Targets not in the current Service inventory (backend lookup skipped; node program probed in section 5): $(printf '%s\n' "$UNOWNED_CONNECTION_TARGETS" | format_targets)"
    fi
    if [[ -n "$UNRESOLVED_SERVICE_NAMES" ]]; then
        echo "Service DNS timeout targets not present in the current inventory: $(printf '%s\n' "$UNRESOLVED_SERVICE_NAMES" | paste -sd' ' -)"
    fi
    if [[ -z "$BACKEND_VIPS" ]]; then
        echo "No failed ClusterIPs to correlate."
    else
        for vip in $BACKEND_VIPS; do
            probe_service_backends "${vip%%:*}" "${vip##*:}"
        done
    fi

    echo
    echo "----- runner-kernel softirq backlog (shared by every Kind node) -----"
    # veth traffic is delivered through the per-CPU softirq backlog queue;
    # when ksoftirqd is starved on a CPU-saturated runner the queue overflows
    # and packets are dropped silently — upstream of every counter this script
    # checks (conntrack, accept queue). Kind nodes are containers sharing the
    # runner kernel and /proc/net/softnet_stat is kernel-global per-CPU state,
    # so it is read once here on the runner host: reading it inside each node
    # container would print identical counters under different node headings.
    # Counters are cumulative, and runner VMs are ephemeral, so they are
    # naturally scoped to this job. Current limits are recorded so the failure
    # states what the backlog/budget were at the time.
    if [[ -r /proc/net/softnet_stat ]]; then
        row=0
        while read -r -a fields; do
            # Newer kernels omit offline CPUs from the file and expose the
            # actual CPU id in field 13; the row ordinal is only a fallback
            # for older 11-column kernels.
            if [[ ${#fields[@]} -ge 13 ]]; then
                cpu=$((16#${fields[12]}))
            else
                cpu=$row
            fi
            printf 'cpu%d processed=%d dropped=%d time_squeeze=%d\n' \
                "$cpu" "$((16#${fields[0]}))" "$((16#${fields[1]}))" "$((16#${fields[2]}))"
            row=$((row + 1))
        done < /proc/net/softnet_stat
        printf 'netdev_max_backlog=%s netdev_budget=%s netdev_budget_usecs=%s\n' \
            "$(cat /proc/sys/net/core/netdev_max_backlog 2>/dev/null || echo '?')" \
            "$(cat /proc/sys/net/core/netdev_budget 2>/dev/null || echo '?')" \
            "$(cat /proc/sys/net/core/netdev_budget_usecs 2>/dev/null || echo '?')"
    else
        echo "(/proc/net/softnet_stat unavailable)"
    fi

    # Node-level targets include logged addresses missing from the current
    # Service inventory: a stale/deleted Service's old VIP must still get its
    # iptables/ipvs snapshot even though there is no backend object to query.
    NODE_TARGETS=$(
        { printf '%s\n' "$TARGETS"; printf '%s' "$UNOWNED_CONNECTION_TARGETS"; } \
            | grep -E '^(tcp|udp)\|[0-9.]+:[0-9]+$' | sort -u
    )

    echo
    echo "----- (5) node netfilter state for failed ClusterIPs -----"
    echo "ClusterIPs correlated: $(printf '%s\n' "${NODE_TARGETS:-<none>}" | format_targets)"
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

            echo "conntrack insertion/drop counters (cumulative node-wide; nonzero requires correlation):"
            # Cumulative-since-boot counters survive the burst draining, unlike
            # the in-use gauge above, but are not attributed to a VIP or time
            # window. insert_failed/drop can indicate unresolved tuple clashes
            # or other insertion failures; table pressure requires supporting
            # early_drop/table-full evidence. 'conntrack -S' is the reliable
            # source; /proc/net/stat/nf_conntrack is a fallback but was observed
            # absent on some Kind node kernels, so try the CLI first.
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

            echo "node-wide SNAT / masquerade plumbing:"
            echo "(only flows whose KUBE-SVC/KUBE-SEP rule jumps to KUBE-MARK-MASQ use this path)"
            probe_masquerade_rules "$node"

            # Per-VIP service program end to end (KUBE-SERVICES -> KUBE-SVC ->
            # KUBE-SEP DNAT): a live DNAT to the pod means the rule is not the
            # problem; an empty chain means no ready backend.
            if [[ -z "$NODE_TARGETS" ]]; then
                echo "service program: no failed ClusterIPs to correlate."
            else
                for target in $NODE_TARGETS; do
                    protocol="${target%%|*}"
                    vip="${target#*|}"
                    echo "service program for $protocol://$vip (iptables, then ipvs):"
                    probe_service_rules "$node" "$protocol" "${vip%%:*}" "${vip##*:}"
                done
            fi
        done
    fi

    echo
    echo "----- (6) SeaweedFS pod socket / listen-queue state -----"
    if [[ -z "$POD" ]]; then
        echo "No SeaweedFS pod; skipping socket state."
    else
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
    fi

    if [[ -n "$POD" ]]; then
        echo
        echo "----- full describe (events, resource config) -----"
        kubectl describe pod "$POD" -n "$NAMESPACE" 2>/dev/null || true
    fi

    echo "=============================================================="
} | emit
