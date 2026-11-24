#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

# Only for the disposable, standalone MySQL upgrade cluster.
set -euo pipefail
fixture="$RUNNER_TEMP/upgrade-backfill"
mkdir -p "$fixture"

mysql_sql() {
  kubectl -n kubeflow exec -i deployment/mysql -c mysql -- mysql --protocol=TCP -h 127.0.0.1 --batch --skip-column-names -uroot mlpipeline "$@"
}

create_run() {
  local port="$1" name="$2"
  jq -n --arg name "$name" --arg experiment "$(cat "$fixture/experiment")" '{
    name: $name,
    pipeline_spec: {workflow_manifest: ({
      apiVersion:"argoproj.io/v1alpha1", kind:"Workflow",
      metadata:{generateName:"upgrade-backfill-",labels:{"pipelines.kubeflow.org/backfill-fixture":"true"}},
      spec:{entrypoint:"hold", templates:[{name:"hold",suspend:{}}]}
    } | tojson)},
    resource_references:[{key:{type:"EXPERIMENT",id:$experiment},relationship:"OWNER"}]
  }' | curl --fail-with-body --silent --show-error --max-time 5 -H 'Content-Type: application/json' \
    --data-binary @- "http://127.0.0.1:$port/apis/v1beta1/runs" | jq -er '.run.id | select(test("^[a-fA-F0-9-]{36}$"))'
}

release_batch() {
  # Holding release while dropping the trigger also frees a batch already
  # inside it. Session locks are automatically released when mysql exits.
  mysql_sql -e "DO GET_LOCK('kfp-upgrade-test-release', 5); DROP TRIGGER IF EXISTS kfp_upgrade_test_pause"
}

case "$1" in
  source)
    jq -er '.expected[0]' "$RUNNER_TEMP/pagination-source.json" > "$fixture/experiment"
    create_run 8888 backfill-legacy > "$fixture/legacy"
    legacy=$(cat "$fixture/legacy")
    pipeline=$(mysql_sql -e "SELECT UUID FROM pipelines ORDER BY UUID LIMIT 1")
    [[ "$pipeline" =~ ^[a-fA-F0-9-]{36}$ ]]
    echo "$pipeline" > "$fixture/pipeline"

    # An inline run has no pipeline reference. Give it the reference-only
    # layout found on older installs and pause only its actual repair write.
    # The trigger survives a MySQL pod replacement during candidate deployment.
    mysql_sql <<SQL
UPDATE run_details SET PipelineId = '' WHERE UUID = '$legacy';
INSERT INTO resource_references (ResourceUUID, ResourceType, ReferenceUUID, ReferenceName, ReferenceType, Relationship, Payload)
VALUES ('$legacy', 'Run', '$pipeline', 'backfill', 'pipeline', 'Creator',
  JSON_OBJECT('ResourceUUID', '$legacy', 'ResourceType', 'Run', 'ReferenceUUID', '$pipeline', 'ReferenceName', 'backfill', 'ReferenceType', 'pipeline', 'Relationship', 'Creator'));
CREATE TABLE kfp_upgrade_test_reports (RunId varchar(191) PRIMARY KEY);
DELIMITER //
CREATE TRIGGER kfp_upgrade_test_pause BEFORE UPDATE ON run_details FOR EACH ROW
BEGIN
  DECLARE polls INT DEFAULT 0;
  -- Commit the old API report receipt with its UPDATE. A later report from
  -- the persistence agent can replace the stored manifest at any time.
  IF NEW.DisplayName = 'backfill-concurrent' AND JSON_VALID(NEW.WorkflowRuntimeManifest) THEN
    IF JSON_UNQUOTE(JSON_EXTRACT(NEW.WorkflowRuntimeManifest,
        '$.metadata.annotations."pipelines.kubeflow.org/backfill-report"')) = 'observed' THEN
      INSERT IGNORE INTO kfp_upgrade_test_reports VALUES (NEW.UUID);
    END IF;
  END IF;
  IF NEW.UUID = '$legacy' AND NEW.PipelineId = '$pipeline' AND OLD.PipelineId = '' THEN
    DO GET_LOCK('kfp-upgrade-test-inside', 0);
    WHILE IS_FREE_LOCK('kfp-upgrade-test-release') = 1 AND polls < 2400 DO
      DO SLEEP(0.25);
      SET polls = polls + 1;
    END WHILE;
    DO RELEASE_LOCK('kfp-upgrade-test-inside');
    IF polls = 2400 THEN
      SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'backfill upgrade fixture exceeded ten minutes';
    END IF;
  END IF;
END//
DELIMITER ;
SQL
    ;;

  verify)
    # Deployment rollout status excludes the retained 2.17.2 pod, which has a
    # different selector. The old API port is forwarded directly to that pod.
    kubectl -n kubeflow rollout status deployment/ml-pipeline --timeout=120s
    pending="SELECT IS_USED_LOCK('kfp-upgrade-test-inside') IS NOT NULL AND NOT EXISTS
      (SELECT 1 FROM migration_statuses WHERE Name = 'backfill_pipeline_refs')"
    for ((i=0; i<60; i++)); do
      [[ "$(mysql_sql -e "$pending")" == 1 ]] && break
      sleep 1
    done
    [[ "$(mysql_sql -e "$pending")" == 1 ]]
    curl --fail --silent --show-error --max-time 5 http://127.0.0.1:8888/apis/v2beta1/healthz > /dev/null

    writer=$(create_run 8889 backfill-concurrent)
    kubectl -n kubeflow wait workflows -l "pipeline/runid=$writer" \
      --for='jsonpath={.status.phase}=Running' --timeout=60s
    kubectl -n kubeflow get workflows -l "pipeline/runid=$writer" -o json |
      jq -ce 'select(.items|length == 1) | .items[0] |
        .metadata.annotations["pipelines.kubeflow.org/backfill-report"] = "observed" | tojson' > "$fixture/report.json"
    curl --fail-with-body --silent --show-error --max-time 5 -H 'Content-Type: application/json' \
      --data-binary "@$fixture/report.json" http://127.0.0.1:8889/apis/v2beta1/workflows
    [[ "$(mysql_sql -e "SELECT COUNT(*) FROM kfp_upgrade_test_reports WHERE RunId = '$writer'")" == 1 ]]
    [[ "$(mysql_sql -e "$pending")" == 1 ]]

    release_batch
    legacy=$(cat "$fixture/legacy")
    pipeline=$(cat "$fixture/pipeline")
    completed="SELECT EXISTS (SELECT 1 FROM migration_statuses WHERE Name = 'backfill_pipeline_refs')
      AND EXISTS (SELECT 1 FROM run_details WHERE UUID = '$legacy' AND PipelineId = '$pipeline')"
    for ((i=0; i<120; i++)); do
      [[ "$(mysql_sql -e "$completed")" == 1 ]] && exit 0
      sleep 1
    done
    echo 'Legacy row repair and final migration marker were not observed' >&2
    exit 1
    ;;

  cleanup)
    release_batch
    mysql_sql -e "DROP TABLE IF EXISTS kfp_upgrade_test_reports"
    kubectl -n kubeflow delete workflows -l pipelines.kubeflow.org/backfill-fixture=true --ignore-not-found=true --wait=true --timeout=120s
    ;;
  *) exit 2 ;;
esac
