#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -u

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
PROBE_NAME="kfp-service-path-probe"
PROBE_IMAGE="registry.access.redhat.com/ubi9/python-311:latest"
KUBECTL_REQUEST_TIMEOUT="${NETWORK_OBSERVABILITY_KUBECTL_REQUEST_TIMEOUT:-10s}"

kubectl_bounded() {
  kubectl --request-timeout="$KUBECTL_REQUEST_TIMEOUT" "$@"
}

resolve_one_pair() {
  local output_directory="$1"
  local targets_file="$2"
  local pair="$3"
  local namespace="$4"
  local service_name="$5"
  local port_name="$6"
  local service_file="$output_directory/${pair}-service.json"
  local endpoint_slices_file="$output_directory/${pair}-endpointslices.json"

  if ! kubectl_bounded -n "$namespace" get service "$service_name" -o json \
    >"$service_file" 2>>"$output_directory/setup.stderr"; then
    echo "Could not resolve Service $namespace/$service_name for paired probing." \
      >>"$output_directory/setup.stderr"
    return
  fi
  if ! kubectl_bounded -n "$namespace" get endpointslice \
    -l "kubernetes.io/service-name=$service_name" -o json \
    >"$endpoint_slices_file" 2>>"$output_directory/setup.stderr"; then
    echo "Could not resolve EndpointSlices for $namespace/$service_name." \
      >>"$output_directory/setup.stderr"
    return
  fi
  python3 "$SCRIPT_DIR/service_path_probe.py" resolve \
    --pair "$pair" \
    --service "$service_file" \
    --endpoint-slices "$endpoint_slices_file" \
    --port-name "$port_name" >>"$targets_file" 2>>"$output_directory/setup.stderr" || true
}

resolve_targets() {
  local namespace="$1"
  local output_directory="$2"
  local targets_file="$output_directory/targets.tsv"
  mkdir -p "$output_directory"
  : >"$targets_file"
  : >"$output_directory/setup.stderr"

  resolve_one_pair "$output_directory" "$targets_file" \
    "seaweedfs-s3" "$namespace" "seaweedfs" "http-s3-compat"
  resolve_one_pair "$output_directory" "$targets_file" \
    "kubernetes-api" "default" "kubernetes" "https"
  # The Kubernetes Service endpoint is the Kind control-plane node. CoreDNS
  # gives us a stable, genuinely pod-backed control for distinguishing a
  # destination-specific SeaweedFS failure from a general pod-to-pod outage.
  resolve_one_pair "$output_directory" "$targets_file" \
    "kube-dns-tcp" "kube-system" "kube-dns" "dns-tcp"
}

start_probe() {
  local namespace="$1"
  local output_directory="$2"
  local targets_file="$output_directory/targets.tsv"
  if [[ ! -s "$targets_file" ]]; then
    echo "No complete Service VIP/Endpoint pairs were resolved." \
      >>"$output_directory/setup.stderr"
    return
  fi

  kubectl_bounded -n "$namespace" delete pod "$PROBE_NAME" --ignore-not-found \
    --wait=true --timeout=20s >/dev/null 2>&1 || true
  kubectl_bounded -n "$namespace" delete configmap "$PROBE_NAME" --ignore-not-found \
    >/dev/null 2>&1 || true
  if ! kubectl_bounded -n "$namespace" create configmap "$PROBE_NAME" \
    --from-file="service_path_probe.py=$SCRIPT_DIR/service_path_probe.py" \
    --from-file="targets.tsv=$targets_file" \
    >>"$output_directory/setup.stdout" 2>>"$output_directory/setup.stderr"; then
    return
  fi

  if ! kubectl_bounded create -f - >>"$output_directory/setup.stdout" \
    2>>"$output_directory/setup.stderr" <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: $PROBE_NAME
  namespace: $namespace
  labels:
    app: $PROBE_NAME
spec:
  automountServiceAccountToken: false
  restartPolicy: Never
  terminationGracePeriodSeconds: 1
  securityContext:
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: probe
    image: $PROBE_IMAGE
    imagePullPolicy: IfNotPresent
    command:
    - python3
    - /config/service_path_probe.py
    - sample
    - --targets
    - /config/targets.tsv
    - --interval
    - "5"
    - --timeout
    - "1"
    env:
    - name: POD_NAME
      valueFrom:
        fieldRef:
          fieldPath: metadata.name
    - name: POD_IP
      valueFrom:
        fieldRef:
          fieldPath: status.podIP
    - name: NODE_NAME
      valueFrom:
        fieldRef:
          fieldPath: spec.nodeName
    resources:
      requests:
        cpu: 5m
        memory: 32Mi
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
    volumeMounts:
    - name: config
      mountPath: /config
      readOnly: true
  volumes:
  - name: config
    configMap:
      name: $PROBE_NAME
EOF
  then
    return
  fi
  kubectl_bounded -n "$namespace" wait --for=condition=Ready "pod/$PROBE_NAME" \
    --timeout=45s >>"$output_directory/setup.stdout" \
    2>>"$output_directory/setup.stderr" || true
}

start_sampler() {
  local output_directory="$1"
  local sampler_pid
  nohup python3 "$SCRIPT_DIR/network_window.py" sample \
    --output "$output_directory/network-window.jsonl" --interval 5 \
    >"$output_directory/sampler.stdout" 2>"$output_directory/sampler.stderr" &
  sampler_pid=$!
  record_process_identity "$sampler_pid" "$output_directory/sampler.pid"

  # The first metric sample is the baseline for failures in probe cycle zero.
  # Avoid starting the probe before that baseline has actually been flushed.
  for _attempt in $(seq 1 120); do
    grep -q '"type": "metric"' "$output_directory/network-window.jsonl" \
      2>/dev/null && return
    kill -0 "$sampler_pid" 2>/dev/null || break
    sleep 0.1
  done
  echo "unavailable: sampler did not establish a metric baseline before probing" \
    >"$output_directory/sampler.status"
}

record_process_identity() {
  local process_id="$1"
  local identity_file="$2"
  local start_time=""
  if [[ -r "/proc/$process_id/stat" ]]; then
    start_time=$(awk '{print $22}' "/proc/$process_id/stat" 2>/dev/null || true)
  else
    start_time=$(ps -o lstart= -p "$process_id" 2>/dev/null | tr -d ' ' || true)
  fi
  printf '%s\t%s\n' "$process_id" "$start_time" >"$identity_file"
}

process_identity_matches() {
  local identity_file="$1"
  local expected_argument="$2"
  local process_id expected_start_time actual_start_time command_line
  IFS=$'\t' read -r process_id expected_start_time <"$identity_file" || return 1
  [[ "$process_id" =~ ^[0-9]+$ && -n "$expected_start_time" ]] || return 1
  if [[ -r "/proc/$process_id/stat" && -r "/proc/$process_id/cmdline" ]]; then
    actual_start_time=$(awk '{print $22}' "/proc/$process_id/stat" 2>/dev/null || true)
    command_line=$(tr '\0' ' ' <"/proc/$process_id/cmdline" 2>/dev/null || true)
  else
    actual_start_time=$(ps -o lstart= -p "$process_id" 2>/dev/null \
      | tr -d ' ' || true)
    command_line=$(ps -o command= -p "$process_id" 2>/dev/null || true)
  fi
  [[ "$actual_start_time" == "$expected_start_time" ]] || return 1
  [[ "$command_line" == *"$expected_argument"* ]]
}

build_capture_filter() {
  local targets_file="$1"
  python3 - "$targets_file" <<'PY'
import ipaddress
from pathlib import Path
import sys

terms = set()
try:
    lines = Path(sys.argv[1]).read_text(encoding="utf-8").splitlines()
except OSError:
    lines = []
for line in lines:
    fields = line.split("\t")
    if len(fields) != 4:
        continue
    try:
        address = str(ipaddress.ip_address(fields[2]))
        port = int(fields[3])
    except (ValueError, TypeError):
        continue
    if 0 < port <= 65535:
        terms.add(f"(host {address} and port {port})")
if terms:
    print(
        "tcp and ("
        + " or ".join(sorted(terms))
        + ") and (tcp[tcpflags] & (tcp-syn|tcp-rst) != 0)"
    )
PY
}

start_packet_capture() {
  local output_directory="$1"
  local targets_file="$2"
  local status_file="$output_directory/tcpdump.status"
  if ! command -v tcpdump >/dev/null 2>&1; then
    echo "unavailable: tcpdump is not installed" >"$status_file"
    return
  fi
  if ! command -v sudo >/dev/null 2>&1 || ! sudo -n true >/dev/null 2>&1; then
    echo "unavailable: passwordless sudo is not available" >"$status_file"
    return
  fi
  if ! command -v nsenter >/dev/null 2>&1; then
    echo "unavailable: nsenter is not installed" >"$status_file"
    return
  fi
  local capture_filter
  capture_filter=$(build_capture_filter "$targets_file" 2>/dev/null || true)
  if [[ -z "$capture_filter" ]]; then
    echo "unavailable: no valid resolved capture targets" >"$status_file"
    return
  fi

  echo "starting bounded SYN/RST capture in each Kind node netns (two 5 MB ring files per node, 96-byte snaplen)" \
    >"$status_file"
  local node node_pid capture_count=0 capture_user
  capture_user=$(id -un)
  while IFS= read -r node; do
    [[ -n "$node" ]] || continue
    node_pid=$(docker inspect --format '{{.State.Pid}}' "$node" 2>/dev/null || true)
    if [[ ! "$node_pid" =~ ^[0-9]+$ ]]; then
      echo "unavailable: could not resolve container pid for $node" >>"$status_file"
      continue
    fi
    sudo -n sh -c '
      umask 022
      start_time=$(
        awk '\''{print $22}'\'' /proc/$$/stat 2>/dev/null \
          || ps -o lstart= -p $$ | tr -d " "
      )
      printf "%s\t%s\n" "$$" "$start_time" > "$1"
      exec nsenter -t "$2" -n tcpdump -i any -nn -U -s 96 \
        -C 5 -W 2 -Z "$5" -w "$3" "$4"
    ' \
      kfp-network-capture \
      "$output_directory/tcpdump-root-${node}.pid" \
      "$node_pid" \
      "$output_directory/connection-attempts-${node}.pcap" \
      "$capture_filter" \
      "$capture_user" \
      >>"$output_directory/tcpdump-${node}.log" 2>&1 &
    echo "$!" >"$output_directory/tcpdump-launcher-${node}.pid"

    # tcpdump opens the first ring file immediately. Validate both the process
    # identity and output before claiming capture is active; this catches
    # privilege-drop/rotation errors while setup is still running.
    local capture_ready=false
    for _attempt in $(seq 1 20); do
      if [[ -s "$output_directory/tcpdump-root-${node}.pid" ]] \
        && sudo_process_identity_matches \
          "$output_directory/tcpdump-root-${node}.pid" \
          "$output_directory/connection-attempts-${node}.pcap" \
        && compgen -G \
          "$output_directory/connection-attempts-${node}.pcap*" >/dev/null; then
        capture_ready=true
        break
      fi
      sleep 0.1
    done
    if [[ "$capture_ready" == "true" ]]; then
      echo "active: node $node container-pid $node_pid, owner $capture_user" \
        >>"$status_file"
      capture_count=$((capture_count + 1))
    else
      echo "unavailable: capture exited before producing a pcap for node $node" \
        >>"$status_file"
      if [[ -s "$output_directory/tcpdump-${node}.log" ]]; then
        tail -5 "$output_directory/tcpdump-${node}.log" \
          | sed 's/^/capture log: /' >>"$status_file"
      fi
    fi
  done < <(
    kubectl_bounded get nodes -o \
      'jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true
  )
  if ((capture_count == 0)); then
    echo "unavailable: no active Kind-node captures were established" >>"$status_file"
  fi
}

sudo_process_identity_matches() {
  local identity_file="$1"
  local expected_argument="$2"
  [[ -r "$identity_file" ]] || return 1
  local process_id expected_start_time
  IFS=$'\t' read -r process_id expected_start_time <"$identity_file" || return 1
  [[ "$process_id" =~ ^[0-9]+$ && -n "$expected_start_time" ]] || return 1
  sudo -n sh -c '
    process_id="$1"
    expected_start_time="$2"
    expected_argument="$3"
    if test -r "/proc/$process_id/stat" && test -r "/proc/$process_id/cmdline"; then
      actual_start_time=$(awk '\''{print $22}'\'' "/proc/$process_id/stat") || exit 1
      command_line=$(tr "\0" " " < "/proc/$process_id/cmdline") || exit 1
    else
      actual_start_time=$(ps -o lstart= -p "$process_id" | tr -d " ") || exit 1
      command_line=$(ps -o command= -p "$process_id") || exit 1
    fi
    test "$actual_start_time" = "$expected_start_time" || exit 1
    case "$command_line" in
      *tcpdump*"$expected_argument"*) exit 0 ;;
      *) exit 1 ;;
    esac
  ' kfp-network-capture-check "$process_id" "$expected_start_time" \
    "$expected_argument" >/dev/null 2>&1
}

start_observability() {
  local namespace="$1"
  local output_directory="$2"
  mkdir -p "$output_directory"
  resolve_targets "$namespace" "$output_directory"
  # Establish the host/node baseline and packet capture before the first probe
  # SYN. Several settled failures began in probe cycle zero, which made the old
  # probe-first ordering unable to attribute their onset.
  start_sampler "$output_directory"
  start_packet_capture "$output_directory" "$output_directory/targets.tsv"
  capture_target_network_state "$namespace" "$output_directory" "baseline"
  start_probe "$namespace" "$output_directory"
}

stop_process() {
  local pid_file="$1"
  local expected_argument="$2"
  if [[ -r "$pid_file" ]]; then
    local process_id
    IFS=$'\t' read -r process_id _start_time <"$pid_file" || return
    if ! process_identity_matches "$pid_file" "$expected_argument"; then
      echo "not signaling stale or unexpected process identity from $pid_file" >&2
      return
    fi
    kill "$process_id" 2>/dev/null || true
    # The sampler may be finishing a bounded docker exec call when SIGTERM
    # arrives. Give it enough time to flush its current JSONL sample.
    for _attempt in $(seq 1 100); do
      kill -0 "$process_id" 2>/dev/null || break
      sleep 0.2
    done
    if process_identity_matches "$pid_file" "$expected_argument"; then
      kill -KILL "$process_id" 2>/dev/null || true
    fi
  fi
}

capture_target_network_state() {
  local namespace="$1"
  local output_directory="$2"
  local phase="$3"
  local output_file="$output_directory/seaweedfs-target-${phase}.txt"
  local pod_name
  pod_name=$(python3 "$SCRIPT_DIR/service_path_probe.py" resolve-pod \
    --endpoint-slices "$output_directory/seaweedfs-s3-endpointslices.json" \
    --port-name http-s3-compat 2>/dev/null || true)
  if [[ -z "$pod_name" ]]; then
    echo "unavailable: no ready SeaweedFS EndpointSlice Pod target resolved" \
      >"$output_file"
    return
  fi

  {
    echo "__POD__"
    echo "$pod_name"
    kubectl_bounded -n "$namespace" exec "$pod_name" -c seaweedfs -- sh -c '
      echo __NETNS__
      readlink /proc/self/ns/net 2>/dev/null || true
      echo __SOCKSTAT__
      cat /proc/net/sockstat 2>/dev/null || true
      echo __NETSTAT__
      cat /proc/net/netstat 2>/dev/null || true
      echo __SNMP__
      cat /proc/net/snmp 2>/dev/null || true
      echo __TCP__
      cat /proc/net/tcp 2>/dev/null || true
      echo __TCP6__
      cat /proc/net/tcp6 2>/dev/null || true
    '
  } >"$output_file" 2>"$output_directory/seaweedfs-target-${phase}.stderr" || {
    echo "unavailable: could not read SeaweedFS target network namespace" \
      >>"$output_file"
  }
}

stop_packet_capture() {
  local output_directory="$1"
  local pid_file process_id expected_start_time node capture_path
  if ! command -v sudo >/dev/null 2>&1; then
    return
  fi
  for pid_file in "$output_directory"/tcpdump-root-*.pid; do
    [[ -r "$pid_file" ]] || continue
    IFS=$'\t' read -r process_id expected_start_time <"$pid_file" || continue
    node=${pid_file##*/tcpdump-root-}
    node=${node%.pid}
    capture_path="$output_directory/connection-attempts-${node}.pcap"
    if ! sudo_process_identity_matches "$pid_file" "$capture_path"; then
      echo "not signaling stale or unexpected tcpdump identity for node $node" \
        >>"$output_directory/tcpdump.status"
      continue
    fi
    sudo -n kill -INT "$process_id" >/dev/null 2>&1 || true
    for _attempt in $(seq 1 25); do
      sudo -n kill -0 "$process_id" >/dev/null 2>&1 || break
      sleep 0.2
    done
    if sudo_process_identity_matches "$pid_file" "$capture_path"; then
      sudo -n kill -TERM "$process_id" >/dev/null 2>&1 || true
    fi
  done
  sudo -n chmod -R a+rX "$output_directory" >/dev/null 2>&1 || true
}

collect_probe() {
  local namespace="$1"
  local output_directory="$2"
  if [[ ! -s "$output_directory/probe-pod.txt" ]]; then
    kubectl_bounded -n "$namespace" get pod "$PROBE_NAME" -o wide \
      >"$output_directory/probe-pod.txt" 2>&1 || true
  fi
  if [[ ! -s "$output_directory/service-path-probe.jsonl" ]]; then
    kubectl_bounded -n "$namespace" logs "$PROBE_NAME" -c probe \
      >"$output_directory/service-path-probe.jsonl" \
      2>"$output_directory/probe-logs.stderr" || true
  fi
}

quiesce_probe() {
  local namespace="$1"
  local output_directory="$2"
  # Python is PID 1 in this restartNever pod. Its SIGTERM handler stops new
  # cycles and lets in-flight one-second dials finish, leaving complete logs
  # while node counters and packet capture are still active.
  kubectl_bounded -n "$namespace" exec "$PROBE_NAME" -c probe -- \
    kill -TERM 1 >>"$output_directory/probe-stop.stdout" \
    2>>"$output_directory/probe-stop.stderr" || true
  if kubectl_bounded -n "$namespace" wait \
    --for=jsonpath='{.status.phase}'=Succeeded "pod/$PROBE_NAME" \
    --timeout=20s >>"$output_directory/probe-stop.stdout" \
    2>>"$output_directory/probe-stop.stderr"; then
    echo "confirmed: probe reached Succeeded" \
      >"$output_directory/probe-stop.status"
    return 0
  fi

  # Preserve the best log available before deleting a probe that ignored or
  # could not receive SIGTERM. Keep observers active until deletion completes.
  echo "graceful stop unconfirmed; collecting logs before deletion" \
    >"$output_directory/probe-stop.status"
  collect_probe "$namespace" "$output_directory"
  if kubectl_bounded -n "$namespace" delete pod "$PROBE_NAME" \
    --ignore-not-found --wait=true --timeout=20s \
    >>"$output_directory/probe-stop.stdout" \
    2>>"$output_directory/probe-stop.stderr"; then
    echo "confirmed: probe deleted after graceful stop failed" \
      >>"$output_directory/probe-stop.status"
    return 0
  fi
  echo "unavailable: probe termination was not confirmed before observer stop" \
    >>"$output_directory/probe-stop.status"
  return 1
}

cleanup_probe() {
  local namespace="$1"
  kubectl_bounded -n "$namespace" delete pod "$PROBE_NAME" --ignore-not-found \
    --wait=true --timeout=20s >/dev/null 2>&1 || true
  kubectl_bounded -n "$namespace" delete configmap "$PROBE_NAME" --ignore-not-found \
    >/dev/null 2>&1 || true
}

stop_observability() {
  local namespace="$1"
  local output_directory="$2"
  mkdir -p "$output_directory"
  quiesce_probe "$namespace" "$output_directory" || true
  collect_probe "$namespace" "$output_directory"
  capture_target_network_state "$namespace" "$output_directory" "final"
  python3 "$SCRIPT_DIR/network_window.py" wait-for-following-sample \
    --input "$output_directory/network-window.jsonl" \
    --probe-input "$output_directory/service-path-probe.jsonl" \
    --timeout 8 >"$output_directory/sampler-final.status" 2>&1 || true
  stop_process "$output_directory/sampler.pid" "$output_directory/network-window.jsonl"
  stop_packet_capture "$output_directory"
  cleanup_probe "$namespace"
  report_observer_finalization "$output_directory"
  report_packet_capture "$output_directory"
  python3 "$SCRIPT_DIR/service_path_probe.py" report \
    --input "$output_directory/service-path-probe.jsonl" || true
  python3 "$SCRIPT_DIR/network_window.py" report \
    --input "$output_directory/network-window.jsonl" \
    --probe-input "$output_directory/service-path-probe.jsonl" || true
  python3 "$SCRIPT_DIR/network_window.py" target-report \
    --baseline "$output_directory/seaweedfs-target-baseline.txt" \
    --end "$output_directory/seaweedfs-target-final.txt" \
    --port 8333 --label SeaweedFS || true
}

report_observer_finalization() {
  local output_directory="$1"
  local report_file="$output_directory/observer-finalization-report.md"
  {
    echo
    echo "## Network observer finalization"
    echo
    if [[ -r "$output_directory/probe-stop.status" ]]; then
      sed 's/^/- probe: /' "$output_directory/probe-stop.status"
    else
      echo "- probe: unavailable: stop status was not recorded"
    fi
    if [[ -r "$output_directory/sampler-final.status" ]]; then
      sed 's/^/- sampler: /' "$output_directory/sampler-final.status"
    else
      echo "- sampler: unavailable: final-sample status was not recorded"
    fi
    echo
  } | tee "$report_file"
  if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    cat "$report_file" >>"$GITHUB_STEP_SUMMARY"
  fi
}

report_packet_capture() {
  local output_directory="$1"
  local report_file="$output_directory/tcpdump-report.md"
  {
    echo
    echo "## Bounded SYN/RST packet capture"
    echo
    if [[ -r "$output_directory/tcpdump.status" ]]; then
      sed 's/^/- /' "$output_directory/tcpdump.status"
    else
      echo "- unavailable: capture status was not recorded"
    fi
    local capture_file size capture_files=0
    for capture_file in "$output_directory"/connection-attempts-*.pcap*; do
      [[ -f "$capture_file" ]] || continue
      size=$(wc -c <"$capture_file" | tr -d ' ')
      echo "- $(basename "$capture_file"): ${size} bytes"
      capture_files=$((capture_files + 1))
    done
    if ((capture_files == 0)); then
      echo "- no packet-capture files were produced"
    fi
    echo
  } | tee "$report_file"
  if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
    cat "$report_file" >>"$GITHUB_STEP_SUMMARY"
  fi
}

case "${1:-}" in
  start)
    start_observability "${2:?namespace required}" "${3:?output directory required}"
    ;;
  stop)
    stop_observability "${2:?namespace required}" "${3:?output directory required}"
    ;;
  resolve-targets)
    resolve_targets "${2:?namespace required}" "${3:?output directory required}"
    ;;
  capture-filter)
    build_capture_filter "${2:?targets file required}"
    ;;
  start-capture)
    start_packet_capture "${2:?output directory required}" \
      "${3:?targets file required}"
    ;;
  stop-capture)
    stop_packet_capture "${2:?output directory required}"
    ;;
  *)
    echo "usage: $0 {start|stop|resolve-targets} <namespace> <output-directory>"
    exit 1
    ;;
esac
