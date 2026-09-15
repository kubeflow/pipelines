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

if (( $# < 3 )); then
  echo "Usage: $0 CLUSTER_NAME MAX_PARALLEL ARCHIVE [ARCHIVE ...]" >&2
  exit 2
fi

cluster_name="$1"
max_parallel="$2"
shift 2

if ! [[ "$max_parallel" =~ ^[1-9][0-9]*$ ]]; then
  echo "MAX_PARALLEL must be a positive integer, got: ${max_parallel}" >&2
  exit 2
fi

for archive in "$@"; do
  if [[ ! -f "$archive" ]]; then
    echo "Image archive does not exist: ${archive}" >&2
    exit 1
  fi
done

batch_pids=()
batch_archives=()

wait_for_batch() {
  local failed=0
  local index
  for index in "${!batch_pids[@]}"; do
    if ! wait "${batch_pids[$index]}"; then
      echo "Failed to load Kind image archive: ${batch_archives[$index]}" >&2
      failed=1
    fi
  done
  batch_pids=()
  batch_archives=()
  return "$failed"
}

for archive in "$@"; do
  (
    kind --name "$cluster_name" load image-archive "$archive"
    rm "$archive"
  ) &
  batch_pids+=("$!")
  batch_archives+=("$archive")

  if (( ${#batch_pids[@]} >= max_parallel )); then
    wait_for_batch
  fi
done

if (( ${#batch_pids[@]} > 0 )); then
  wait_for_batch
fi
