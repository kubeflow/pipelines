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
}

start_probe() {
  local namespace="$1"
  local output_directory="$2"
  local targets_file="$output_directory/targets.tsv"
  resolve_targets "$namespace" "$output_directory"
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
  nohup python3 "$SCRIPT_DIR/network_window.py" sample \
    --output "$output_directory/network-window.jsonl" --interval 5 \
    >"$output_directory/sampler.stdout" 2>"$output_directory/sampler.stderr" &
  echo "$!" >"$output_directory/sampler.pid"
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
  local node node_pid capture_count=0
  while IFS= read -r node; do
    [[ -n "$node" ]] || continue
    node_pid=$(docker inspect --format '{{.State.Pid}}' "$node" 2>/dev/null || true)
    if [[ ! "$node_pid" =~ ^[0-9]+$ ]]; then
      echo "unavailable: could not resolve container pid for $node" >>"$status_file"
      continue
    fi
    sudo -n sh -c \
      'umask 022; echo $$ > "$1"; exec nsenter -t "$2" -n tcpdump -i any -nn -U -s 96 -C 5 -W 2 -w "$3" "$4"' \
      kfp-network-capture \
      "$output_directory/tcpdump-root-${node}.pid" \
      "$node_pid" \
      "$output_directory/connection-attempts-${node}.pcap" \
      "$capture_filter" \
      >>"$output_directory/tcpdump-${node}.log" 2>&1 &
    echo "$!" >"$output_directory/tcpdump-launcher-${node}.pid"
    echo "node $node container-pid $node_pid" >>"$status_file"
    capture_count=$((capture_count + 1))
  done < <(
    kubectl_bounded get nodes -o \
      'jsonpath={range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null || true
  )
  if ((capture_count == 0)); then
    echo "unavailable: no Kind node network namespaces were resolved" >>"$status_file"
  fi
}

start_observability() {
  local namespace="$1"
  local output_directory="$2"
  mkdir -p "$output_directory"
  start_probe "$namespace" "$output_directory"
  start_sampler "$output_directory"
  start_packet_capture "$output_directory" "$output_directory/targets.tsv"
}

stop_process() {
  local pid_file="$1"
  if [[ -r "$pid_file" ]]; then
    local process_id
    process_id=$(cat "$pid_file")
    kill "$process_id" 2>/dev/null || true
    # The sampler may be finishing a bounded docker exec call when SIGTERM
    # arrives. Give it enough time to flush its current JSONL sample.
    for _attempt in $(seq 1 100); do
      kill -0 "$process_id" 2>/dev/null || break
      sleep 0.2
    done
    kill -KILL "$process_id" 2>/dev/null || true
  fi
}

stop_packet_capture() {
  local output_directory="$1"
  local pid_file process_id
  if ! command -v sudo >/dev/null 2>&1; then
    return
  fi
  for pid_file in "$output_directory"/tcpdump-root-*.pid; do
    [[ -r "$pid_file" ]] || continue
    process_id=$(cat "$pid_file")
    sudo -n kill -INT "$process_id" >/dev/null 2>&1 || true
    for _attempt in $(seq 1 25); do
      sudo -n kill -0 "$process_id" >/dev/null 2>&1 || break
      sleep 0.2
    done
    sudo -n kill -TERM "$process_id" >/dev/null 2>&1 || true
  done
  sudo -n chmod -R a+rX "$output_directory" >/dev/null 2>&1 || true
}

collect_probe() {
  local namespace="$1"
  local output_directory="$2"
  kubectl_bounded -n "$namespace" get pod "$PROBE_NAME" -o wide \
    >"$output_directory/probe-pod.txt" 2>&1 || true
  kubectl_bounded -n "$namespace" logs "$PROBE_NAME" -c probe \
    >"$output_directory/service-path-probe.jsonl" \
    2>"$output_directory/probe-logs.stderr" || true
  kubectl_bounded -n "$namespace" delete pod "$PROBE_NAME" --ignore-not-found \
    --wait=false >/dev/null 2>&1 || true
  kubectl_bounded -n "$namespace" delete configmap "$PROBE_NAME" --ignore-not-found \
    >/dev/null 2>&1 || true
}

stop_observability() {
  local namespace="$1"
  local output_directory="$2"
  mkdir -p "$output_directory"
  stop_process "$output_directory/sampler.pid"
  stop_packet_capture "$output_directory"
  collect_probe "$namespace" "$output_directory"
  python3 "$SCRIPT_DIR/service_path_probe.py" report \
    --input "$output_directory/service-path-probe.jsonl" || true
  python3 "$SCRIPT_DIR/network_window.py" report \
    --input "$output_directory/network-window.jsonl" \
    --probe-input "$output_directory/service-path-probe.jsonl" || true
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
  *)
    echo "usage: $0 {start|stop|resolve-targets} <namespace> <output-directory>"
    exit 1
    ;;
esac
