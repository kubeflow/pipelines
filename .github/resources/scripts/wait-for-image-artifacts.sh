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

# Many matrix jobs can reach this barrier together. Poll slowly enough to keep
# their shared GITHUB_TOKEN comfortably below the repository API rate limit.
# Forty attempts at the default interval form a roughly 20-minute wait window.
# If that window expires while a missing artifact still has an active producer
# job, start another window instead of racing the producer upload.
WAIT_ATTEMPTS="${WAIT_ATTEMPTS:-40}"
WAIT_INTERVAL_SECONDS="${WAIT_INTERVAL_SECONDS:-30}"
PUBLICATION_GRACE_ATTEMPTS="${PUBLICATION_GRACE_ATTEMPTS:-3}"
PRODUCER_STATE_UNAVAILABLE_EXTENSIONS="${PRODUCER_STATE_UNAVAILABLE_EXTENSIONS:-1}"
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

producer_jobs_endpoint="repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/jobs?filter=latest&per_page=100"
if [[ -n "${GITHUB_RUN_ATTEMPT:-}" ]]; then
  if ! [[ "$GITHUB_RUN_ATTEMPT" =~ ^[1-9][0-9]*$ ]]; then
    echo "GITHUB_RUN_ATTEMPT must be a positive integer" >&2
    exit 2
  fi
  producer_jobs_endpoint="repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/attempts/${GITHUB_RUN_ATTEMPT}/jobs?per_page=100"
fi

active_missing_producers=()
failed_missing_producers=()
successful_missing_producers=()
unknown_missing_producers=()
assess_artifacts() {
  ARTIFACT_RECORDS="$artifact_records" PRODUCER_JOBS="$producer_jobs" \
    python3 - "${GITHUB_RUN_ATTEMPT:-}" "${ALL_CI_IMAGE_ARTIFACTS[@]}" <<'PY'
from datetime import datetime
import json
import os
import sys


def positive_integer(value):
    if type(value) is not int or value <= 0:
        raise ValueError('Expected a positive artifact/job ID or run attempt')
    return value


def timestamp(value):
    if not isinstance(value, str):
        raise ValueError('Missing artifact or producer timestamp')
    result = datetime.fromisoformat(value.replace('Z', '+00:00'))
    if result.tzinfo is None:
        raise ValueError('Artifact and producer timestamps need a timezone')
    return result


try:
    artifacts = [json.loads(line) for line in os.environ['ARTIFACT_RECORDS'].splitlines()]
    jobs = [json.loads(line) for line in os.environ['PRODUCER_JOBS'].splitlines()]
    attempt = int(sys.argv[1]) if sys.argv[1] else None
    decisions = []
    for name in sys.argv[2:]:
        prefix = ('build / runtime-base-images' if name == 'runtime-base-images'
                  else f'build / image-build ({name},')
        producers = [job for job in jobs if job['name'].startswith(prefix)]
        for job in producers:
            positive_integer(job['id'])
            positive_integer(job['run_attempt'])
        if not producers:
            decisions.append(('unknown', name, '-'))
            continue
        producer = max(producers, key=lambda job: (job['run_attempt'], job['id']))
        if attempt is not None and producer['run_attempt'] != attempt:
            decisions.append(('unknown', name, '-'))
            continue
        if producer['status'] in ('queued', 'in_progress', 'waiting', 'pending', 'requested'):
            if producer['conclusion'] is not None:
                raise ValueError('Active producer has a terminal conclusion')
            decisions.append(('active', name, '-'))
            continue
        if producer['status'] != 'completed':
            raise ValueError('Unknown image producer status')
        if producer['conclusion'] != 'success':
            decisions.append(('failed', name, producer['conclusion'] or 'unknown'))
            continue
        started = timestamp(producer['started_at'])
        candidates = []
        for artifact in artifacts:
            if artifact['name'] != name:
                continue
            if type(artifact['expired']) is not bool:
                raise ValueError('Missing artifact expiration state')
            if artifact['expired']:
                continue
            artifact_id = positive_integer(artifact['id'])
            created = timestamp(artifact['created_at'])
            if created >= started:
                candidates.append((created, artifact_id))
        if candidates:
            decisions.append(('ready', name, str(max(candidates)[1])))
        else:
            decisions.append(('missing', name, '-'))
    # Emit only after the complete response is validated; never accept a
    # partial ready batch when a later record is malformed.
    for decision in decisions:
        print('\t'.join(decision))
except (KeyError, TypeError, ValueError, AttributeError) as error:
    print(f'Cannot safely select image artifacts: {error}', file=sys.stderr)
    sys.exit(1)
PY
}

# Optional executable receives name:ID tokens for each newly verified batch.
# Artifact IDs pin downloads across retries and distinguish rerun uploads.
# Its success means the entire batch is ready for deployment; failure must stop
# this barrier.
processed_artifacts=()
missing_artifacts=()
attempt=0
publication_grace_remaining=0
producer_state_unavailable_extensions=0
while true; do
  attempt=$((attempt + 1))
  window_attempt=$((((attempt - 1) % WAIT_ATTEMPTS) + 1))
  artifact_records=""
  producer_jobs=""
  metadata_available=true
  if ! artifact_records=$(gh api --paginate \
      "repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/artifacts?per_page=100" \
      --jq '.artifacts[] | {id, name, created_at, expired} | @json'); then
    echo "::warning::Could not list image artifacts on attempt ${window_attempt}/${WAIT_ATTEMPTS}."
    metadata_available=false
  fi
  if ! producer_jobs=$(gh api --paginate "$producer_jobs_endpoint" \
      --jq '.jobs[] | {id, name, run_attempt, status, conclusion, started_at} | @json'); then
    echo "::warning::Could not list image producer jobs." >&2
    metadata_available=false
  fi

  missing_artifacts=()
  ready_artifacts=()
  active_missing_producers=()
  failed_missing_producers=()
  successful_missing_producers=()
  unknown_missing_producers=()
  assessment=""
  if [[ "$metadata_available" == true ]] && ! assessment=$(assess_artifacts); then
    metadata_available=false
  fi
  if [[ "$metadata_available" == true ]]; then
    while IFS=$'\t' read -r state artifact detail; do
      if [[ "$state" == ready ]]; then
        token="$artifact:$detail"
        if ! printf '%s\n' "${processed_artifacts[@]:-}" | grep -Fqx -- "$token"; then
          ready_artifacts+=("$token")
        fi
        continue
      fi
      missing_artifacts+=("$artifact")
      case "$state" in
        active) active_missing_producers+=("$artifact") ;;
        failed) failed_missing_producers+=("$artifact:$detail") ;;
        missing) successful_missing_producers+=("$artifact") ;;
        unknown) unknown_missing_producers+=("$artifact") ;;
      esac
    done <<< "$assessment"
  else
    missing_artifacts=("${ALL_CI_IMAGE_ARTIFACTS[@]}")
  fi

  if (( $# > 0 )); then
    if (( ${#ready_artifacts[@]} > 0 )); then
      echo "Preparing available branch image artifacts: ${ready_artifacts[*]}"
      "$@" "${ready_artifacts[@]}"
      processed_artifacts+=("${ready_artifacts[@]}")
    fi
  fi

  if (( ${#missing_artifacts[@]} == 0 )); then
    echo "All ${#ALL_CI_IMAGE_ARTIFACTS[@]} branch image artifacts are available."
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
    if [[ "$metadata_available" == true ]]; then
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

  # Importing a batch can take longer than the polling interval. Check again
  # immediately afterward so artifacts published during that work are not held
  # for another interval. Empty polls retain the shared-token rate limit.
  if (( $# > 0 )) && (( ${#ready_artifacts[@]} > 0 )); then
    continue
  fi

  echo "Waiting for branch image artifacts (${window_attempt}/${WAIT_ATTEMPTS}); missing: ${missing_artifacts[*]}"
  sleep "$WAIT_INTERVAL_SECONDS"
done
