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

if ! command -v gh >/dev/null 2>&1; then
  echo "GitHub CLI is required; run setup-github-cli.sh before waiting for image artifacts." >&2
  exit 2
fi

# Many matrix jobs reach this barrier together. Conditional artifact reads avoid
# charging unchanged inventories against their shared primary API rate limit.
# Forty attempts at the default interval form a roughly 20-minute wait window.
# If that window expires while a missing artifact still has an active producer
# job, start another window instead of racing the producer upload.
WAIT_ATTEMPTS="${WAIT_ATTEMPTS:-40}"
WAIT_INTERVAL_SECONDS="${WAIT_INTERVAL_SECONDS:-30}"
PUBLICATION_GRACE_ATTEMPTS="${PUBLICATION_GRACE_ATTEMPTS:-3}"
PRODUCER_STATE_UNAVAILABLE_EXTENSIONS="${PRODUCER_STATE_UNAVAILABLE_EXTENSIONS:-1}"
ARTIFACT_API_TIMEOUT_SECONDS="${ARTIFACT_API_TIMEOUT_SECONDS:-1200}"
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

if ! [[ "$ARTIFACT_API_TIMEOUT_SECONDS" =~ ^[1-9][0-9]{0,5}$ ]]; then
  echo "ARTIFACT_API_TIMEOUT_SECONDS must be an integer from 1 to 999999, got: ${ARTIFACT_API_TIMEOUT_SECONDS}" >&2
  exit 2
fi

source "${BASH_SOURCE%/*}/ci-image-artifacts.sh"
source "${BASH_SOURCE%/*}/list-image-artifacts.sh"
artifact_cache_dir=$(mktemp -d)
trap 'rm -rf "$artifact_cache_dir"' EXIT

# Most deployments never start a viewer. Only viewer acceptance callers need
# the optional runtime; do not delay every backend lane on its publication.
required_artifacts=()
for artifact in "${ALL_CI_IMAGE_ARTIFACTS[@]}"; do
  if [[ "$artifact" != "tensorboard" || "${LOAD_TENSORBOARD_IMAGE:-false}" == "true" ]]; then
    required_artifacts+=("$artifact")
  fi
done

active_missing_producers=()
failed_missing_producers=()
successful_missing_producers=()
unknown_missing_producers=()
classify_missing_producers() {
  local producer_jobs=""
  if ! producer_jobs=$(gh api --paginate \
      "repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/jobs?filter=latest&per_page=100" \
      --jq '.jobs[] | [.name, .status, (.conclusion // "")] | @tsv'); then
    echo "::warning::Could not list image producer jobs." >&2
    return 1
  fi

  active_missing_producers=()
  failed_missing_producers=()
  successful_missing_producers=()
  unknown_missing_producers=()
  local artifact conclusion job_name producer_found producer_prefix status
  for artifact in "${missing_artifacts[@]}"; do
    if [[ "$artifact" == "runtime-base-images" ]]; then
      producer_prefix="build / runtime-base-images"
    else
      producer_prefix="build / image-build (${artifact},"
    fi

    producer_found=false
    while IFS=$'\t' read -r job_name status conclusion; do
      if [[ "$job_name" == "$producer_prefix"* ]]; then
        producer_found=true
        if [[ "$status" != "completed" ]]; then
          active_missing_producers+=("$artifact")
        elif [[ "$conclusion" == "success" ]]; then
          successful_missing_producers+=("$artifact")
        else
          failed_missing_producers+=("${artifact}:${conclusion:-unknown}")
        fi
        break
      fi
    done <<< "$producer_jobs"
    if [[ "$producer_found" == "false" ]]; then
      unknown_missing_producers+=("$artifact")
    fi
  done
}

missing_artifacts=()
attempt=0
publication_grace_remaining=0
producer_state_unavailable_extensions=0
artifact_api_deadline=0
artifact_api_retry_delay=60
while true; do
  artifact_names=""
  if artifact_names=$(list_image_artifacts "$artifact_cache_dir"); then
    artifact_api_deadline=0
    artifact_api_retry_delay=60
  else
    api_status=$?
    if (( api_status == 2 )); then exit 1; fi
    now=$(date +%s)
    if (( artifact_api_deadline == 0 )); then
      artifact_api_deadline=$((now + ARTIFACT_API_TIMEOUT_SECONDS))
    fi
    retry_delay=$artifact_api_retry_delay
    if [[ -f "$artifact_cache_dir/retry-after" ]]; then
      server_delay=$(<"$artifact_cache_dir/retry-after")
      if (( server_delay > retry_delay )); then retry_delay=$server_delay; fi
    fi
    if (( now + retry_delay >= artifact_api_deadline )); then
      echo "Artifact inventory remains unavailable; retry delay ${retry_delay}s exceeds the remaining API wait budget. Retry the workflow after API access recovers." >&2
      exit 1
    fi
    echo "::warning::Artifact inventory unavailable; retrying in ${retry_delay}s without consuming publication grace."
    sleep "$retry_delay"
    artifact_api_retry_delay=$((artifact_api_retry_delay * 2))
    if (( artifact_api_retry_delay > 300 )); then artifact_api_retry_delay=300; fi
    continue
  fi
  attempt=$((attempt + 1))
  window_attempt=$((((attempt - 1) % WAIT_ATTEMPTS) + 1))

  missing_artifacts=()
  for artifact in "${required_artifacts[@]}"; do
    if ! grep -Fqx -- "$artifact" <<< "$artifact_names"; then
      missing_artifacts+=("$artifact")
    fi
  done

  if (( ${#missing_artifacts[@]} == 0 )); then
    echo "All ${#required_artifacts[@]} branch image artifacts are available."
    exit 0
  fi

  if (( publication_grace_remaining > 0 )); then
    publication_grace_remaining=$((publication_grace_remaining - 1))
    if (( publication_grace_remaining == 0 )); then
      echo "Missing branch image artifacts after producer completion grace: ${missing_artifacts[*]}" >&2
      exit 1
    fi
    echo "Waiting for completed producers' artifacts to become visible;" \
      "remaining grace attempts: ${publication_grace_remaining}"
    sleep "$WAIT_INTERVAL_SECONDS"
    continue
  fi

  if (( window_attempt == WAIT_ATTEMPTS )); then
    if classify_missing_producers; then
      if (( ${#failed_missing_producers[@]} > 0 )); then
        echo "Image producers completed without publishing required artifacts: ${failed_missing_producers[*]}" >&2
        exit 1
      fi
      if (( ${#active_missing_producers[@]} > 0 )); then
        echo "Extending image artifact wait; active producers: ${active_missing_producers[*]}"
      else
        publication_grace_remaining=$PUBLICATION_GRACE_ATTEMPTS
        echo "Allowing ${PUBLICATION_GRACE_ATTEMPTS} publication grace attempts;" \
          "successful producers: ${successful_missing_producers[*]:-none};" \
          "unknown producers: ${unknown_missing_producers[*]:-none}"
      fi
    else
      if (( producer_state_unavailable_extensions >= PRODUCER_STATE_UNAVAILABLE_EXTENSIONS )); then
        echo "Missing branch image artifacts and producer state remains unavailable: ${missing_artifacts[*]}" >&2
        exit 1
      fi
      producer_state_unavailable_extensions=$((producer_state_unavailable_extensions + 1))
      echo "Extending image artifact wait because producer state is unavailable" \
        "(${producer_state_unavailable_extensions}/${PRODUCER_STATE_UNAVAILABLE_EXTENSIONS})."
    fi
  fi

  echo "Waiting for branch image artifacts (${window_attempt}/${WAIT_ATTEMPTS}); missing: ${missing_artifacts[*]}"
  sleep "$WAIT_INTERVAL_SECONDS"
done
