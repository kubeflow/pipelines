#!/bin/bash
#
# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -ex

# Usage: wait_for_builds "${BUILD_IDS[@]}"
# returns true if success, otherwise false.
function wait_for_builds {
  local pending_ids=("$@") # copy pending build ids
  local max_attempts=$(expr $TIMEOUT_SECONDS / 20)
  for i in $(seq 1 ${max_attempts})
  do
    local new_pending_ids=()
    for id in "${pending_ids[@]}"
    do
      status=$(gcloud builds describe $id --format='value(status)') || status="FETCH_ERROR"
      case "$status" in
        "SUCCESS")
          echo "Build with id ${id} has succeeded."
        ;;
        "WORKING" | "QUEUED")
          new_pending_ids+=( "$id" )
        ;;
        "FETCH_ERROR")
          echo "Fetching cloud build status failed, retrying..."
          new_pending_ids+=( "$id" )
        ;;
        *)
          echo "Fetching build log for build $id"
          gcloud builds log "$id" || echo "Fetching build log failed"
          # Intentionally placed this status echo at the end, because when a
          # developer is looking at logs, sth at the end is most discoverable.
          echo "Cloud build with build id ${id} failed with status ${status}"
          return 1
        ;;
      esac
    done
    pending_ids=("${new_pending_ids[@]}")
    if [ 0 == "${#pending_ids[@]}" ]; then
      echo "All cloud builds succeeded."
      return 0
    fi
    echo "Cloud build in progress, waiting for 20 seconds..."
    sleep 20
  done
  echo "Waiting for cloud build ids ${pending_ids[@]} timed out."
  return 2
}

function wait_and_retry {
  local max_attempts=3
  local i
  for i in $(seq 1 ${max_attempts})
  do
    if [ $i != 1 ]; then
      echo "Retry #${i}, build images again..."
      source "${DIR}/build-images.sh"
    fi
    if [ "$IMAGES_BUILDING" != true ]; then
      echo "Images already built"
      return 0
    fi
    if wait_for_builds "${BUILD_IDS[@]}"; then
      echo "Images built successfully"
      return 0
    fi
  done
  echo "Images failed to build"
  return 1
}

wait_and_retry
