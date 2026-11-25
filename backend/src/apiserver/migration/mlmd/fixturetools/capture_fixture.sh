#!/usr/bin/env bash
set -euo pipefail

KFP218_KUBECONFIG=${KFP218_KUBECONFIG:?set KFP218_KUBECONFIG}
KFP218_CONTEXT=${KFP218_CONTEXT:?set KFP218_CONTEXT}
KFP218_OUTPUT=${KFP218_OUTPUT:?set KFP218_OUTPUT}

mkdir -p "$KFP218_OUTPUT"
kubectl_release=(kubectl --kubeconfig "$KFP218_KUBECONFIG" --context "$KFP218_CONTEXT" -n kubeflow)
mysql_pod=$(
  "${kubectl_release[@]}" get pods -l app=mysql \
    -o jsonpath='{range .items[?(@.status.phase=="Running")]}{.metadata.name}{"\n"}{end}' \
    | head -n 1
)
test -n "$mysql_pod"

"${kubectl_release[@]}" exec "$mysql_pod" -- \
  mysqldump -uroot --single-transaction --skip-lock-tables \
  --set-gtid-purged=OFF --skip-comments --skip-dump-date \
  --skip-extended-insert --databases mlpipeline metadb \
  > "$KFP218_OUTPUT/source.sql"

"${kubectl_release[@]}" get workflows.argoproj.io -o json \
  > "$KFP218_OUTPUT/workflows.json"
"${kubectl_release[@]}" get configmap pipeline-install-config -o json \
  > "$KFP218_OUTPUT/install-config.json"
"${kubectl_release[@]}" get deployments -o json \
  | jq '[.items[] | {name: .metadata.name, containers: [.spec.template.spec.containers[] | {name, image}]}]' \
  > "$KFP218_OUTPUT/deployment-images.json"

"${kubectl_release[@]}" exec "$mysql_pod" -- mysql -uroot --batch --skip-column-names -e \
  'SELECT "run_details", COUNT(*) FROM mlpipeline.run_details; SELECT "tasks", COUNT(*) FROM mlpipeline.tasks; SELECT "contexts", COUNT(*) FROM metadb.Context; SELECT "executions", COUNT(*) FROM metadb.Execution; SELECT "artifacts", COUNT(*) FROM metadb.Artifact; SELECT "events", COUNT(*) FROM metadb.Event;' \
  > "$KFP218_OUTPUT/table-counts.tsv"

"${kubectl_release[@]}" exec "$mysql_pod" -- mysql -uroot --batch --skip-column-names -e \
  'SELECT "task-with-mlmd-id",COUNT(*) FROM mlpipeline.tasks WHERE MLMDExecutionID <> ""; SELECT "task-with-fingerprint",COUNT(*) FROM mlpipeline.tasks WHERE Fingerprint <> ""; SELECT "skipped-task",COUNT(*) FROM mlpipeline.tasks WHERE State="SKIPPED"; SELECT "execution-id-range",MIN(id),MAX(id),COUNT(*) FROM metadb.Execution; SELECT "artifact-id-range",MIN(id),MAX(id),COUNT(*) FROM metadb.Artifact; SELECT "event-id-range",MIN(id),MAX(id),COUNT(*) FROM metadb.Event; SELECT "metric-artifacts",COUNT(*) FROM metadb.Artifact a JOIN metadb.Type t ON a.type_id=t.id WHERE t.name="system.Metrics";' \
  > "$KFP218_OUTPUT/coverage.tsv"
