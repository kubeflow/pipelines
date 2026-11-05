#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

# Exercise a fenced upgrade from the paired KFP and MLMD release dump.
set -euo pipefail

mode="${1:?expected cli-first or startup-only}"
case "$mode" in
  cli-first|startup-only) ;;
  *) echo "Unknown upgrade mode: $mode" >&2; exit 2 ;;
esac
image="${2:?expected Kind-loaded disposable API server image}"
binary="${3:?expected disposable migration CLI}"
production_image="${4:?expected hold-enabled API server image}"
fixture=backend/src/apiserver/migration/mlmd/testdata/release_218
root="$(mktemp -d)"
context="kind-kfp-mlmd-upgrade"
kubectl config view --minify --flatten --raw --context "$context" > "$root/kubeconfig"
export KUBECONFIG="$root/kubeconfig"
kube=(kubectl -n kubeflow)
forwards=()
cleanup() {
  for pid in "${forwards[@]}"; do kill "$pid" 2>/dev/null || true; done
  rm -f -- "$root/kubeconfig"
}
trap cleanup EXIT

"${kube[@]}" get deployment ml-pipeline -o json | python3 -c '
import json, sys
deployment = json.load(sys.stdin)
deployment.pop("status", None)
deployment["metadata"] = {"name": "ml-pipeline", "namespace": "kubeflow"}
deployment["spec"]["replicas"] = 1
json.dump(deployment, sys.stdout)
' > "$root/legacy-api.json"

scale_down() {
  local deployment="$1"
  local selector
  selector=$("${kube[@]}" get deployment "$deployment" -o json |
    jq -er '.spec.selector.matchLabels | to_entries | map(.key+"="+.value) | join(",")')
  "${kube[@]}" scale deployment "$deployment" --replicas=0
  "${kube[@]}" rollout status "deployment/$deployment" --timeout=120s
  if [[ -n "$("${kube[@]}" get pods -l "$selector" -o name)" ]]; then
    "${kube[@]}" wait --for=delete pods -l "$selector" --timeout=120s
  fi
}

port_forward() {
  local service="$1" mapping="$2"
  local log="$root/$service.forward"
  "${kube[@]}" port-forward "service/$service" "$mapping" > "$log" 2>&1 &
  local pid=$!
  forwards+=("$pid")
  for _ in {1..30}; do
    if grep -q 'Forwarding from 127.0.0.1:' "$log"; then return; fi
    if ! kill -0 "$pid" 2>/dev/null; then break; fi
    sleep 1
  done
  cat "$log" >&2
  return 1
}

mysql_pod=$("${kube[@]}" get pod -l app=mysql -o jsonpath='{.items[0].metadata.name}')
mysql_db() { "${kube[@]}" exec -i "$mysql_pod" -- mysql -uroot "$@"; }

scale_down ml-pipeline-scheduledworkflow
# Controllers must stay stopped while the captured Workflow resources are replayed.
scale_down workflow-controller
scale_down ml-pipeline-persistenceagent
scale_down metadata-writer
scale_down ml-pipeline
scale_down metadata-grpc-deployment
mysql_db < "$fixture/source.sql"
kubectl wait crd/workflows.argoproj.io --for=condition=Established --timeout=60s
jq -c '.items[] | {
  apiVersion, kind, metadata: {name: .metadata.name, labels: .metadata.labels},
  spec, status
}' "$fixture/workflows.json" | while IFS= read -r workflow; do
  name=$(jq -r '.metadata.name' <<< "$workflow")
  phase=$(jq -r '.status.phase' <<< "$workflow")
  "${kube[@]}" create -f - <<< "$workflow"
  actual=$("${kube[@]}" get workflow "$name" -o jsonpath='{.status.phase}')
  if [[ "$actual" != "$phase" ]]; then
    status=$(jq -c '{status}' <<< "$workflow")
    "${kube[@]}" patch workflow "$name" --subresource=status --type=merge -p "$status"
  fi
done
"${kube[@]}" scale deployment metadata-grpc-deployment --replicas=1
"${kube[@]}" rollout status deployment/metadata-grpc-deployment --timeout=180s
"${kube[@]}" scale deployment ml-pipeline --replicas=1
"${kube[@]}" rollout status deployment/ml-pipeline --timeout=180s
port_forward mysql 13306:3306
port_forward metadata-grpc-service 18080:8080
port_forward ml-pipeline 18888:8888
python3 .github/resources/scripts/verify-mlmd-upgrade.py http://127.0.0.1:18888 "$fixture" --legacy
# A Service port-forward binds one Pod. Restart it after every API replacement.
kill "${forwards[-1]}"

export KFP_MLMD_MYSQL_DSN='root@tcp(127.0.0.1:13306)/mlpipeline?parseTime=true'
export KFP_MLMD_ADDRESS='127.0.0.1:18080'
"$binary" validate
if "$binary" --gate-only > "$root/old-api.log" 2>&1; then
  echo 'The writer fence accepted a running old API' >&2
  exit 1
fi
grep -q 'must be scaled to zero' "$root/old-api.log"
# Use a terminal source row without its final report to exercise read-only refusal.
run_id=$(jq -er '.[0].run_id' "$fixture/runs.json")
finish_time=$(mysql_db -N mlpipeline -e "SELECT FinishedAtInSec FROM run_details WHERE UUID='$run_id'")
[[ "$finish_time" =~ ^[1-9][0-9]*$ ]]
mysql_db mlpipeline -e "UPDATE run_details SET FinishedAtInSec=NULL WHERE UUID='$run_id'"
if "$binary" validate > "$root/blocked.json"; then
  echo 'Preflight accepted a run without its final report' >&2
  exit 1
fi
jq -e --arg run "$run_id" '.status == "blocked" and any(.blockers[]; contains($run))' "$root/blocked.json"
mysql_db mlpipeline -e "UPDATE run_details SET FinishedAtInSec=$finish_time WHERE UUID='$run_id'"
"$binary" validate
scale_down ml-pipeline
"$binary" --gate-only

"${kube[@]}" exec "$mysql_pod" -- mysqldump -uroot --single-transaction --set-gtid-purged=OFF mlpipeline > "$root/kfp-backup.sql"
manifest_checksums() {
  mysql_db -N mlpipeline -e 'SELECT UUID, SHA2(WorkflowRuntimeManifest,256) FROM run_details ORDER BY UUID'
}
manifest_checksums > "$root/manifests-before"
mlmd_checksum() {
  "${kube[@]}" exec "$mysql_pod" -- mysqldump -uroot --single-transaction --skip-comments --order-by-primary --set-gtid-purged=OFF metadb |
    sha256sum
}
mlmd_checksum > "$root/mlmd-before"
# Pin the fixture's migration clock so cache coverage survives as the capture ages.
mysql_db mlpipeline <<'SQL'
CREATE TABLE mlmd_migration_state (ID tinyint NOT NULL PRIMARY KEY, CacheCutoffInSec bigint NOT NULL) ENGINE=InnoDB;
INSERT INTO mlmd_migration_state VALUES (1, 1795060000);
SQL

# Apply migration RBAC and the target deployment without replacing the legacy DB,
# MLMD endpoint, or old runtime. Keep the target at zero until setup is complete.
"${kube[@]}" apply -f manifests/kustomize/base/pipeline/ml-pipeline-apiserver-role.yaml
"${kube[@]}" apply -f manifests/kustomize/base/pipeline/ml-pipeline-migration-workflow-role.yaml
kubectl kustomize --load-restrictor LoadRestrictionsNone .github/resources/manifests/standalone/default |
  python3 .github/resources/scripts/mlmd_upgrade_deployment.py "$image" "$mode" > "$root/native-api.yaml"
"${kube[@]}" apply -f "$root/native-api.yaml"
"${kube[@]}" set image deployment/ml-pipeline "ml-pipeline-api-server=$production_image"
"${kube[@]}" scale deployment ml-pipeline --replicas=1
for _ in {1..60}; do
  if "${kube[@]}" logs deployment/ml-pipeline > "$root/startup-hold.log" 2>&1 &&
     grep -q 'production MLMD migration is disabled' "$root/startup-hold.log"; then
    break
  fi
  sleep 2
done
grep -q 'production MLMD migration is disabled' "$root/startup-hold.log"
mysql_db -N mlpipeline -e "SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA='mlpipeline' AND TABLE_NAME IN ('tasks_migration_tmp','tasks_pre_migration','artifacts','artifact_tasks')" |
  grep -qx 0
scale_down ml-pipeline
"${kube[@]}" set image deployment/ml-pipeline "ml-pipeline-api-server=$image"
"${kube[@]}" rollout status deployment/ml-pipeline --timeout=120s

if [[ "$mode" == cli-first ]]; then
  # Hold the same named lock as API startup and verify CLI contention first.
  mysql_db -N mlpipeline -e "SELECT GET_LOCK('kfp_mlmd_migration_lock', 0), SLEEP(30)" > "$root/lock" &
  lock_process=$!
  forwards+=("$lock_process")
  for _ in {1..30}; do
    owner=$(mysql_db -N -e "SELECT IS_USED_LOCK('kfp_mlmd_migration_lock')")
    if [[ "$owner" =~ ^[0-9]+$ ]]; then break; fi
    sleep 1
  done
  [[ "$owner" =~ ^[0-9]+$ ]]
  if "$binary" --available-disk-bytes=1073741824 --mlmd-database-bytes=1048576 > "$root/contended.log" 2>&1; then
    echo 'The CLI migrated while another connection owned the lock' >&2
    exit 1
  fi
  grep -q 'MLMD migration already in progress' "$root/contended.log"
  mysql_db -e "KILL CONNECTION $owner"
  wait "$lock_process" || true
  "$binary" --available-disk-bytes=1073741824 --mlmd-database-bytes=1048576 | tee "$root/migration.json"
  jq -e '.status == "complete" and .tasks == 34 and .artifacts == 26 and .links == 26 and (.skipped | length > 0)' "$root/migration.json"
fi

"${kube[@]}" scale deployment ml-pipeline --replicas=1
"${kube[@]}" rollout status deployment/ml-pipeline --timeout=300s
port_forward ml-pipeline 18888:8888
python3 .github/resources/scripts/verify-mlmd-upgrade.py http://127.0.0.1:18888 "$fixture"
if [[ "$mode" == startup-only ]]; then
  "${kube[@]}" logs deployment/ml-pipeline > "$root/startup.log"
  grep -q 'MLMD startup migration complete' "$root/startup.log"
  grep -q 'MLMD migration skipped' "$root/startup.log"
fi
manifest_checksums > "$root/manifests-after"
diff -u "$root/manifests-before" "$root/manifests-after"
kill "${forwards[-1]}"

scale_down metadata-grpc-deployment
mlmd_checksum > "$root/mlmd-after"
diff -u "$root/mlmd-before" "$root/mlmd-after"
"${kube[@]}" set env deployment/ml-pipeline METADATA_GRPC_SERVICE_HOST- METADATA_GRPC_SERVICE_PORT-
"${kube[@]}" rollout restart deployment/ml-pipeline
"${kube[@]}" rollout status deployment/ml-pipeline --timeout=180s
port_forward ml-pipeline 18888:8888
python3 .github/resources/scripts/verify-mlmd-upgrade.py http://127.0.0.1:18888 "$fixture"
kill "${forwards[-1]}"

# Roll back only the KFP database and API. The paired MLMD schema was never replaced.
scale_down ml-pipeline
mysql_db -e 'DROP DATABASE mlpipeline; CREATE DATABASE mlpipeline'
mysql_db mlpipeline < "$root/kfp-backup.sql"
"${kube[@]}" scale deployment metadata-grpc-deployment --replicas=1
"${kube[@]}" rollout status deployment/metadata-grpc-deployment --timeout=180s
"${kube[@]}" apply -f "$root/legacy-api.json"
"${kube[@]}" rollout status deployment/ml-pipeline --timeout=180s
port_forward ml-pipeline 18888:8888
python3 .github/resources/scripts/verify-mlmd-upgrade.py http://127.0.0.1:18888 "$fixture" --legacy
