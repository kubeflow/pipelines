#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

# A native-to-native rollout in the disposable standalone MySQL API cluster.
set -euo pipefail

# Published native main API immediately preceding the online backfill.
source_sha=3708af87ce8d7ad94f21dd5d851b25f03a215e3f
source_image=ghcr.io/kubeflow/kfp-api-server@sha256:2871431efeedba31fc85184624906897bab9b6b21c89620ace28c8a8e1c0538c
k=(kubectl -n kubeflow)
old_api=http://127.0.0.1:18889/apis/v2
new_api=http://127.0.0.1:18888/apis/v2
ports=()
legacy=
writer=
sql() {
  "${k[@]}" exec -i deployment/mysql -c mysql -- mysql --protocol=TCP -h 127.0.0.1 -uroot --batch --skip-column-names mlpipeline "$@"
}
release_batch() {
  # The release session keeps its lock while DROP waits for the trigger to exit.
  sql -e "DO GET_LOCK('kfp-upgrade-test-release', 5); DROP TRIGGER IF EXISTS kfp_upgrade_test_pause"
}
cleanup() {
  set +e
  for pid in "${ports[@]}"; do kill "$pid" 2>/dev/null || true; done
  release_batch
  for run in "$legacy" "$writer"; do
    if [[ -n "$run" ]]; then
      "${k[@]}" delete workflows -l "pipeline/runid=$run" --ignore-not-found=true --wait=false
    fi
  done
  "${k[@]}" scale deployment/ml-pipeline-persistenceagent --replicas=1
  "${k[@]}" delete deployment backfill-source --ignore-not-found=true --wait=false
  "${k[@]}" scale deployment/ml-pipeline --replicas=1
}
trap cleanup EXIT

forward() {
  local deployment=$1 port=$2
  "${k[@]}" port-forward --address=127.0.0.1 "deployment/$deployment" "$port:8888" > "$RUNNER_TEMP/$deployment-backfill-forward.log" 2>&1 &
  local pid=$!
  ports+=("$pid")
  local i
  for ((i=0; i<30; i++)); do
    kill -0 "$pid" || return 1
    if curl --silent --fail --max-time 1 "http://127.0.0.1:$port/apis/v2/healthz" > /dev/null; then return; fi
    sleep 1
  done
  return 1
}
post() {
  curl --fail-with-body --show-error --silent --max-time 5 -H 'Content-Type: application/json' --data-binary @- "$old_api/$1"
}
uuid() {
  jq -er --arg key "$1" '.[$key] | select(test("^[a-fA-F0-9-]{36}$"))'
}
create_run() {
  jq -n --arg name "$1" --arg exp "$experiment" '{
    display_name:$name, experiment_id:$exp,
    pipeline_spec:{
      pipelineInfo:{name:"backfill-rollout"}, schemaVersion:"2.1.0",
      components:{"comp-hold":{executorLabel:"exec-hold"}},
      deploymentSpec:{executors:{"exec-hold":{container:{
        image:"ghcr.io/containerd/busybox:1.36",command:["sleep"],args:["600"]
      }}}},
      root:{dag:{tasks:{hold:{taskInfo:{name:"hold"},componentRef:{name:"comp-hold"}}}}}
    }
  }' | post runs | uuid run_id
}

# The checked-out image has already initialized the test DB. Retain an
# independently addressable source replica before removing the disposable
# completion marker, so its own startup cannot consume the pending fixture.
"${k[@]}" get deployment ml-pipeline -o json | jq --arg image "$source_image" '
  .metadata = {name:"backfill-source",namespace:"kubeflow"} | del(.status) |
  .spec.replicas=1 | .spec.selector.matchLabels={app:"backfill-source"} |
  .spec.template.metadata.labels.app="backfill-source" |
  (.spec.template.spec.containers[] | select(.name=="ml-pipeline-api-server") | .image)=$image
' | "${k[@]}" apply -f -
"${k[@]}" rollout status deployment/backfill-source --timeout=180s
forward backfill-source 18889
curl --fail --silent "$old_api/healthz" | jq -e --arg sha "$source_sha" '.commit_sha == $sha'
experiment=$(echo '{"display_name":"backfill-rollout"}' | post experiments | uuid experiment_id)
pipeline=$(echo '{"display_name":"backfill-rollout","name":"backfill-rollout"}' | post pipelines | uuid pipeline_id)
legacy=$(create_run backfill-legacy)
# Only this test may submit the witnessed report. Main validates the live
# workflow and persists that resource, including its current annotations.
"${k[@]}" scale deployment/ml-pipeline-persistenceagent --replicas=0
"${k[@]}" wait --for=delete pod -l app=ml-pipeline-persistenceagent --timeout=90s
"${k[@]}" scale deployment/ml-pipeline --replicas=0
"${k[@]}" wait --for=delete pod -l app=ml-pipeline --timeout=90s

sql <<SQL
UPDATE run_details SET PipelineId = '' WHERE UUID = '$legacy';
INSERT INTO resource_references (ResourceUUID, ResourceType, ReferenceUUID, ReferenceName, ReferenceType, Relationship, Payload)
VALUES ('$legacy','Run','$pipeline','backfill','pipeline','Creator',
  JSON_OBJECT('ResourceUUID','$legacy','ResourceType','Run','ReferenceUUID','$pipeline','ReferenceType','pipeline','Relationship','Creator'));
DELETE FROM migration_statuses WHERE Name = 'backfill_pipeline_refs';
DELIMITER //
CREATE TRIGGER kfp_upgrade_test_pause BEFORE UPDATE ON run_details FOR EACH ROW
BEGIN
  DECLARE polls INT DEFAULT 0;
  IF NEW.UUID = '$legacy' AND NEW.PipelineId = '$pipeline' AND OLD.PipelineId = '' THEN
    DO GET_LOCK('kfp-upgrade-test-inside',0);
    WHILE IS_FREE_LOCK('kfp-upgrade-test-release') = 1 AND polls < 1200 DO
      DO SLEEP(0.25);
      SET polls = polls + 1;
    END WHILE;
    DO RELEASE_LOCK('kfp-upgrade-test-inside');
    IF polls = 1200 THEN SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='fixture release timed out'; END IF;
  END IF;
END//
DELIMITER ;
SQL

"${k[@]}" scale deployment/ml-pipeline --replicas=1
"${k[@]}" rollout status deployment/ml-pipeline --timeout=120s
forward ml-pipeline 18888
pending="SELECT IS_USED_LOCK('kfp-upgrade-test-inside') IS NOT NULL AND NOT EXISTS
  (SELECT 1 FROM migration_statuses WHERE Name='backfill_pipeline_refs')"
for ((i=0; i<60; i++)); do
  [[ "$(sql -e "$pending")" == 1 ]] && break
  sleep 1
done
[[ "$(sql -e "$pending")" == 1 ]]
curl --fail --silent --max-time 5 "$new_api/healthz" > /dev/null
writer=$(create_run backfill-concurrent)
"${k[@]}" wait workflows -l "pipeline/runid=$writer" --for='jsonpath={.status.phase}=Running' --timeout=60s
workflow=$("${k[@]}" get workflows -l "pipeline/runid=$writer" -o json | jq -er 'select(.items|length==1) | .items[0].metadata.name')
"${k[@]}" annotate workflow "$workflow" pipelines.kubeflow.org/backfill-report=observed
receipt="SELECT COUNT(*) FROM run_details WHERE UUID='$writer' AND CASE WHEN JSON_VALID(WorkflowRuntimeManifest)
  THEN JSON_UNQUOTE(JSON_EXTRACT(WorkflowRuntimeManifest,'$.metadata.annotations.\"pipelines.kubeflow.org/backfill-report\"'))='observed' ELSE 0 END"
[[ "$(sql -e "$receipt")" == 0 ]]
"${k[@]}" get workflow "$workflow" -o json | jq -c 'tojson' | post workflows
[[ "$(sql -e "$receipt")" == 1 ]]
[[ "$(sql -e "$pending")" == 1 ]]
release_batch
for ((i=0; i<120; i++)); do
  if [[ "$(sql -e "SELECT EXISTS (SELECT 1 FROM migration_statuses WHERE Name='backfill_pipeline_refs')
    AND EXISTS (SELECT 1 FROM run_details WHERE UUID='$legacy' AND PipelineId='$pipeline')")" == 1 ]]; then
    echo "Source API $source_sha created run $writer and persisted its report while the candidate was Ready and repairing $legacy."
    exit 0
  fi
  sleep 1
done
echo 'Expected legacy row repair and migration marker after releasing the worker' >&2
exit 1
