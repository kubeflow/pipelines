#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
# Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.
set -euo pipefail
context=kind-kfp-custom-roles
[[ $(kubectl config current-context) == "$context" ]]
state=${RUNNER_TEMP:?}/custom-roles
mkdir -p "$state"
kube() { kubectl --context "$context" --request-timeout=30s "$@"; }
# Trusted-header authentication is confined to the disposable cluster and
# localhost forwards. This tests authorization, not ingress authentication.
kube -n kubeflow set env deployment/ml-pipeline MULTIUSER=true \
  KUBEFLOW_USERID_HEADER=kubeflow-userid KUBEFLOW_USERID_PREFIX= \
  MULTIUSER_SHARED_READ=false REQUIRE_NAMESPACE_FOR_PIPELINES=false \
  DEFAULTPIPELINERUNNERSERVICEACCOUNT=pipeline-runner \
  COMPILED_PIPELINE_SPEC_PATCH='{}'
kube -n kubeflow set env deployment/ml-pipeline-ui ENABLE_AUTHZ=true \
  KUBEFLOW_USERID_HEADER=kubeflow-userid KUBEFLOW_USERID_PREFIX=
kube -n kubeflow rollout status deployment/ml-pipeline --timeout=300s
kube -n kubeflow rollout status deployment/ml-pipeline-ui --timeout=300s
pids=()
cleanup() {
  for pid in "${pids[@]}"; do kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; done
}
trap cleanup EXIT
for spec in 'ml-pipeline 8888:8888' 'ml-pipeline-ui 3000:80'; do
  read -r service mapping <<< "$spec"
  (exec kubectl --context "$context" -n kubeflow port-forward --address=127.0.0.1 "service/$service" "$mapping") >"$state/$service-forward.log" 2>&1 &
  pids+=("$!")
  ready=false
  for attempt in {1..30}; do
    kill -0 "${pids[-1]}"
    if grep -q '^Forwarding from 127\.0\.0\.1:' "$state/$service-forward.log"; then ready=true; break; fi
    sleep 1
  done
  [[ "$ready" == true ]]
done
python3 .github/resources/scripts/custom-role-acceptance.py \
  --allow-test-cluster-mutations --output "$state/report.json"
