#!/usr/bin/env bash
# Wait until a node advertises allocatable nvidia.com/gpu.
# Env: NAMESPACE (FGO namespace, for diagnostics)

set -euo pipefail

# print output on success; warn on failure without aborting.
run_diagnostic() {
  local label=$1
  shift
  local out
  if ! out=$("$@" 2>&1); then
    echo "WARNING: ${label} failed: ${out}" >&2
  else
    printf '%s\n' "${out}"
  fi
}

echo "Waiting for nodes to advertise nvidia.com/gpu..."
deadline=$((SECONDS + 300))
while (( SECONDS < deadline )); do
  gpu_lines=()
  if ! out=$(kubectl get nodes -o jsonpath='{range .items[*]}{.metadata.name}{"="}{.status.allocatable.nvidia\.com/gpu}{"\n"}{end}' 2>&1); then
    echo "WARNING: kubectl get nodes (GPU poll) failed: ${out}" >&2
  else
    mapfile -t gpu_lines <<< "${out}"
  fi
  printf '%s\n' "${gpu_lines[@]}"
  for line in "${gpu_lines[@]}"; do
    gpu_qty="${line#*=}"
    if [[ -n "${gpu_qty}" && "${gpu_qty}" != "0" && "${gpu_qty}" != "<none>" ]]; then
      echo "Allocatable nvidia.com/gpu is ready (${line})."
      run_diagnostic "kubectl get nodes" \
        kubectl get nodes -o custom-columns='NAME:.metadata.name,GPU:.status.allocatable.nvidia\.com/gpu,POOL:.metadata.labels.run\.ai/simulated-gpu-node-pool'
      run_diagnostic "kubectl get pods" \
        kubectl get pods -n "${NAMESPACE}" -o wide
      exit 0
    fi
  done
  sleep 5
done

echo "Timed out waiting for nvidia.com/gpu allocatable."
run_diagnostic "kubectl get nodes" kubectl get nodes -o yaml
run_diagnostic "kubectl get pods" kubectl get pods -n "${NAMESPACE}" -o wide
run_diagnostic "kubectl describe nodes" kubectl describe nodes
exit 1
