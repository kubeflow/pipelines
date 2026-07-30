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
# Forty attempts at the default interval allow roughly 20 minutes for image
# builders delayed by GitHub-hosted runner availability.
WAIT_ATTEMPTS="${WAIT_ATTEMPTS:-40}"
WAIT_INTERVAL_SECONDS="${WAIT_INTERVAL_SECONDS:-30}"
if ! [[ "$WAIT_ATTEMPTS" =~ ^[1-9][0-9]*$ ]]; then
  echo "WAIT_ATTEMPTS must be a positive integer, got: ${WAIT_ATTEMPTS}" >&2
  exit 2
fi
if ! [[ "$WAIT_INTERVAL_SECONDS" =~ ^[0-9]+$ ]]; then
  echo "WAIT_INTERVAL_SECONDS must be a non-negative integer, got: ${WAIT_INTERVAL_SECONDS}" >&2
  exit 2
fi

source "${BASH_SOURCE%/*}/ci-image-artifacts.sh"

missing_artifacts=()
for attempt in $(seq 1 "$WAIT_ATTEMPTS"); do
  artifact_names=""
  if ! artifact_names=$(gh api --paginate \
      "repos/${GITHUB_REPOSITORY}/actions/runs/${GITHUB_RUN_ID}/artifacts?per_page=100" \
      --jq '.artifacts[].name'); then
    echo "::warning::Could not list image artifacts on attempt ${attempt}/${WAIT_ATTEMPTS}."
  fi

  missing_artifacts=()
  for artifact in "${ALL_CI_IMAGE_ARTIFACTS[@]}"; do
    if ! grep -Fqx -- "$artifact" <<< "$artifact_names"; then
      missing_artifacts+=("$artifact")
    fi
  done

  if (( ${#missing_artifacts[@]} == 0 )); then
    echo "All ${#ALL_CI_IMAGE_ARTIFACTS[@]} branch image artifacts are available."
    exit 0
  fi

  if (( attempt < WAIT_ATTEMPTS )); then
    echo "Waiting for branch image artifacts (${attempt}/${WAIT_ATTEMPTS}); missing: ${missing_artifacts[*]}"
    sleep "$WAIT_INTERVAL_SECONDS"
  fi
done

echo "Missing branch image artifacts after ${WAIT_ATTEMPTS} attempts: ${missing_artifacts[*]}" >&2
exit 1
