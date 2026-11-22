#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail

if [[ $# != 1 || ! $1 =~ ^(source|target)$ ]]; then
  echo "Usage: $0 source|target" >&2
  exit 2
fi
phase=$1
fixture_dir="${RUNNER_TEMP:?}/profile-upgrade"
mkdir -p "$fixture_dir"
git rev-parse HEAD > "$fixture_dir/candidate-sha.txt"

export PROFILE_UI_URL=http://127.0.0.1:3000
export PROFILE_PIPELINE_FILE="$fixture_dir/pipeline.yaml"
export PROFILE_RUN_OUTPUT="$fixture_dir/$phase-runs.env"
export PROFILE_API_PREFIX=/apis/v2beta1
if [[ $phase == target ]]; then
  export PROFILE_API_PREFIX=/apis/v2
  kubectl -n kubeflow rollout status deployment/ml-pipeline --timeout=240s
  kubectl -n kubeflow rollout status deployment/ml-pipeline-ui --timeout=240s
fi

kubectl -n kubeflow port-forward service/ml-pipeline-ui 3000:80 > "$fixture_dir/$phase-forward.log" 2>&1 &
forward_pid=$!
trap 'kill "$forward_pid" 2>/dev/null || true; wait "$forward_pid" 2>/dev/null || true' EXIT
for attempt in {1..60}; do
  if curl --silent --fail --max-time 3 --output /dev/null "$PROFILE_UI_URL/"; then
    break
  fi
  if [[ $attempt == 60 ]]; then
    echo "Timed out waiting for the $phase fixture frontend" >&2
    exit 1
  fi
  sleep 2
done

if [[ $phase == source ]]; then
  GITHUB_OUTPUT="$fixture_dir/profiles.env" bash test/frontend-integration-test/prepare-profile-run.sh
fi
source "$fixture_dir/profiles.env"
export PROFILE_OWNER="$owner" PROFILE_FOREIGN="$foreign"
export PROFILE_OWNER_NAMESPACE="$owner_namespace" PROFILE_FOREIGN_NAMESPACE="$foreign_namespace"

if [[ $phase == source ]]; then
  node --test frontend/scripts/ui-smoke-test/profile-run-create.e2e.cjs
fi
source "$fixture_dir/source-runs.env"
export PROFILE_RUN_ID="$run_id" PROFILE_RUN_NAME="$run_name"
export PROFILE_COMPARISON_RUN_ID="$comparison_run_id" PROFILE_COMPARISON_RUN_NAME="$comparison_run_name"
export PROFILE_FOREIGN_RUN_ID="$foreign_run_id" PROFILE_FOREIGN_RUN_NAME="$foreign_run_name"
export PROFILE_FOREIGN_COMPARISON_RUN_ID="$foreign_comparison_run_id"
export PROFILE_FOREIGN_COMPARISON_RUN_NAME="$foreign_comparison_run_name"

if [[ $phase == source ]]; then
  bash test/frontend-integration-test/check-profile-upgrade-history.sh capture "$fixture_dir/history.json"
else
  bash test/frontend-integration-test/check-profile-upgrade-history.sh verify "$fixture_dir/history.json"
  bash test/frontend-integration-test/prepare-profile-run.sh --restore-fixture-network
  node --test frontend/scripts/ui-smoke-test/profile-artifact-catalog.e2e.cjs \
    frontend/scripts/ui-smoke-test/profile-run-comparison.e2e.cjs
  node --test frontend/scripts/ui-smoke-test/profile-run-create.e2e.cjs
fi
