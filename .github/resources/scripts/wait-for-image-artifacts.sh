#!/usr/bin/env bash
#
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must be set}"
: "${GITHUB_RUN_ID:?GITHUB_RUN_ID must be set}"

WAIT_ATTEMPTS="${WAIT_ATTEMPTS:-40}"
WAIT_INTERVAL_SECONDS="${WAIT_INTERVAL_SECONDS:-30}"
PUBLICATION_GRACE_ATTEMPTS="${PUBLICATION_GRACE_ATTEMPTS:-3}"
PRODUCER_STATE_UNAVAILABLE_EXTENSIONS="${PRODUCER_STATE_UNAVAILABLE_EXTENSIONS:-3}"

if ! [[ "$WAIT_ATTEMPTS" =~ ^[1-9][0-9]*$ ]]; then
  echo "WAIT_ATTEMPTS must be a positive integer, got: ${WAIT_ATTEMPTS}" >&2
  exit 2
fi
if ! [[ "$WAIT_INTERVAL_SECONDS" =~ ^[0-9]+$ ]]; then
  echo "WAIT_INTERVAL_SECONDS must be a non-negative integer, got: ${WAIT_INTERVAL_SECONDS}" >&2
  exit 2
fi
if ! [[ "$PUBLICATION_GRACE_ATTEMPTS" =~ ^[1-9][0-9]*$ ]]; then
  echo "PUBLICATION_GRACE_ATTEMPTS must be a positive integer, got: ${PUBLICATION_GRACE_ATTEMPTS}" >&2
  exit 2
fi
if ! [[ "$PRODUCER_STATE_UNAVAILABLE_EXTENSIONS" =~ ^[0-9]+$ ]]; then
  echo "PRODUCER_STATE_UNAVAILABLE_EXTENSIONS must be a non-negative integer, got: ${PRODUCER_STATE_UNAVAILABLE_EXTENSIONS}" >&2
  exit 2
fi

source "${BASH_SOURCE%/*}/ci-image-artifacts.sh"

case "$(uname -m)" in
  x86_64) ARCH_NAME="amd64" ;;
  aarch64|arm64) ARCH_NAME="arm64" ;;
  *) echo "::error::Unsupported runner architecture: $(uname -m)" >&2; exit 1 ;;
esac

EXPECTED_CI_IMAGE_ARTIFACTS=()
for artifact in "${ALL_CI_IMAGE_ARTIFACTS[@]}"; do
  EXPECTED_CI_IMAGE_ARTIFACTS+=("${artifact}-${ARCH_NAME}")
done

artifact_producer_pattern() {
  local artifact="${1%-${ARCH_NAME}}"
  if [[ "$artifact" == "runtime-base-images" ]]; then
    printf '%s\n' 'build / runtime-base-images'
  else
    printf '%s\n' "build / image-build (${artifact},"
  fi
}

producer_for_artifact() {
  local artifact="$1"
  local prefix
  prefix="$(artifact_producer_pattern "$artifact")"
  while IFS=$'\t' read -r job_name job_status job_conclusion; do
    [[ -z "$job_name" ]] && continue
    if [[ "$job_name" == "$prefix"* ]]; then
      printf '%s\t%s\t%s\n' "$job_name" "$job_status" "$job_conclusion"
      return 0
    fi
  done <<< "$PRODUCER_JOBS"
  return 1
}

artifact_is_available() {
  local artifact="$1"
  if grep -Fqx -- "$artifact" <<< "$artifact_names"; then
    return 0
  fi
  local legacy_name="${artifact%-${ARCH_NAME}}"
  grep -Fqx -- "$legacy_name" <<< "$artifact_names"
}

attempt=0
publication_grace_used=0
producer_state_unavailable_used=0
producer_extensions_used=0

while :; do
  attempt=$((attempt + 1))

  artifact_names=""
  if ! artifact_names=$(gh api --paginate \
      "repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/artifacts?per_page=100" \
      --jq '.artifacts[].name'); then
    echo "::warning::Could not list image artifacts on attempt ${attempt}."
  fi

  missing_artifacts=()
  for artifact in "${EXPECTED_CI_IMAGE_ARTIFACTS[@]}"; do
    if ! artifact_is_available "$artifact"; then
      missing_artifacts+=("$artifact")
    fi
  done

  if (( ${#missing_artifacts[@]} == 0 )); then
    echo "All ${#EXPECTED_CI_IMAGE_ARTIFACTS[@]} ${ARCH_NAME} branch image artifacts are available."
    exit 0
  fi

  if (( attempt < WAIT_ATTEMPTS )); then
    echo "Waiting for branch image artifacts (${attempt}/${WAIT_ATTEMPTS}); missing: ${missing_artifacts[*]}"
    sleep "$WAIT_INTERVAL_SECONDS"
    continue
  fi

  producer_state_ok=true
  PRODUCER_JOBS=""
  if ! PRODUCER_JOBS=$(gh api --paginate \
      "repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/jobs?per_page=100" \
      --jq '.jobs[] | [.name, .status, (.conclusion // "")] | @tsv'); then
    producer_state_ok=false
  fi

  if [[ "$producer_state_ok" != true ]]; then
    if (( producer_state_unavailable_used < PRODUCER_STATE_UNAVAILABLE_EXTENSIONS )); then
      producer_state_unavailable_used=$((producer_state_unavailable_used + 1))
      echo "Extending image artifact wait because producer state is unavailable (${producer_state_unavailable_used}/${PRODUCER_STATE_UNAVAILABLE_EXTENSIONS})"
      sleep "$WAIT_INTERVAL_SECONDS"
      continue
    fi
    echo "producer state remains unavailable; missing: ${missing_artifacts[*]}" >&2
    exit 1
  fi

  active_producers=()
  failed_producers=()
  successful_producers=()
  for artifact in "${missing_artifacts[@]}"; do
    if ! producer_info=$(producer_for_artifact "$artifact"); then
      continue
    fi
    IFS=$'\t' read -r job_name job_status job_conclusion <<< "$producer_info"
    case "$job_status" in
      queued|in_progress|waiting|requested|pending)
        active_producers+=("${artifact%-${ARCH_NAME}}") ;;
      completed)
        case "$job_conclusion" in
          success) successful_producers+=("${artifact%-${ARCH_NAME}}") ;;
          failure|cancelled|timed_out|action_required|stale|startup_failure)
            failed_producers+=("${artifact%-${ARCH_NAME}}:${job_conclusion}") ;;
        esac ;;
    esac
  done

  if (( ${#failed_producers[@]} > 0 )); then
    echo "Branch image producer failed: ${failed_producers[*]}" >&2
    exit 1
  fi

  if (( ${#active_producers[@]} > 0 && producer_extensions_used < WAIT_ATTEMPTS )); then
    producer_extensions_used=$((producer_extensions_used + 1))
    echo "Extending image artifact wait; active producers: ${active_producers[*]}"
    sleep "$WAIT_INTERVAL_SECONDS"
    continue
  fi

  if (( publication_grace_used < PUBLICATION_GRACE_ATTEMPTS )); then
    publication_grace_used=$((publication_grace_used + 1))
    if (( ${#successful_producers[@]} > 0 )); then
      echo "Allowing ${PUBLICATION_GRACE_ATTEMPTS} publication grace attempts (${publication_grace_used}/${PUBLICATION_GRACE_ATTEMPTS}); completed producers: ${successful_producers[*]}"
    else
      echo "Allowing ${PUBLICATION_GRACE_ATTEMPTS} publication grace attempts (${publication_grace_used}/${PUBLICATION_GRACE_ATTEMPTS})"
    fi
    sleep "$WAIT_INTERVAL_SECONDS"
    continue
  fi

  echo "Missing branch image artifacts after producer completion grace: ${missing_artifacts[*]}" >&2
  exit 1
done
