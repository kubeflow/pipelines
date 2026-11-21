#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0

# Run against a disposable two-profile cluster before and after an in-place upgrade.
# The CI frontend accepts the user identity header directly in this fixture.
set -euo pipefail

if [[ $# != 2 || ! $1 =~ ^(capture|verify)$ ]]; then
  echo "Usage: $0 capture|verify SNAPSHOT.json" >&2
  exit 2
fi

: "${PROFILE_UI_URL:?set PROFILE_UI_URL to the fixture frontend address}"
: "${PROFILE_OWNER:?set PROFILE_OWNER}"
: "${PROFILE_FOREIGN:?set PROFILE_FOREIGN}"
: "${PROFILE_RUN_ID:?set PROFILE_RUN_ID to the first pre-upgrade run}"
: "${PROFILE_FOREIGN_RUN_ID:?set PROFILE_FOREIGN_RUN_ID to the second pre-upgrade run}"
: "${PROFILE_OWNER_NAMESPACE:?set PROFILE_OWNER_NAMESPACE}"
: "${PROFILE_FOREIGN_NAMESPACE:?set PROFILE_FOREIGN_NAMESPACE}"

if [[ $PROFILE_OWNER == "$PROFILE_FOREIGN" ||
      $PROFILE_OWNER_NAMESPACE == "$PROFILE_FOREIGN_NAMESPACE" ||
      $PROFILE_RUN_ID == "$PROFILE_FOREIGN_RUN_ID" ]]; then
  echo "Expected two distinct profile identities, namespaces, and runs" >&2
  exit 2
fi

api="${PROFILE_UI_URL%/}/apis/v2beta1/runs"

read_history() {
  local run_id=$1 namespace=$2 owner=$3 other=$4 record status
  record=$(curl --fail-with-body --silent --show-error     --header "kubeflow-userid: $owner" "$api/$run_id")
  status=$(curl --silent --show-error --output /dev/null --write-out '%{http_code}'     --header "kubeflow-userid: $other" "$api/$run_id")
  case "$status" in
    403|404) ;;
    *) echo "Expected foreign run access to be denied for $run_id, got HTTP $status" >&2; return 1 ;;
  esac
  jq -e --arg run_id "$run_id" --arg namespace "$namespace" '
    select(.run_id == $run_id and .namespace == $namespace and .state == "SUCCEEDED")
    | select((.created_at | type) == "string" and (.finished_at | type) == "string")
    | {run_id, namespace, display_name, created_at, finished_at, state}
  ' <<< "$record"
}

first=$(read_history "$PROFILE_RUN_ID" "$PROFILE_OWNER_NAMESPACE" "$PROFILE_OWNER" "$PROFILE_FOREIGN")
second=$(read_history "$PROFILE_FOREIGN_RUN_ID" "$PROFILE_FOREIGN_NAMESPACE" "$PROFILE_FOREIGN" "$PROFILE_OWNER")
history=$(jq -Sn --argjson first "$first" --argjson second "$second" '[$first, $second]')

if [[ $1 == capture ]]; then
  printf '%s\n' "$history" > "$2"
else
  diff -u "$2" <(printf '%s\n' "$history")
fi
