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

if [ "$IMAGES_BUILDING" == true ]; then
  MAX_ATTEMPT=$(expr $TIMEOUT_SECONDS / 20)
  for i in $(seq 1 ${MAX_ATTEMPT})
  do
    (( success_count=0 )) || true
    for id in "${BUILD_IDS[@]}"
    do
      status=$(gcloud builds describe $id --format='value(status)') || status="FETCH_ERROR"
      case "$status" in
        "SUCCESS")
          (( ++success_count ))
        ;;
        "WORKING")
          # do nothing
        ;;
        "FETCH_ERROR")
          echo "Fetching cloud build status failed, retrying..."
        ;;
        *)
          echo "Cloud build with build id ${id} failed with status ${status}"
          exit 1
        ;;
      esac
    done
    if [ $success_count == 4 ]; then
      echo "All cloud builds succeeded."
      break
    fi

    echo "Cloud build in progress, waiting for 20 seconds..."
    sleep 20
  done
fi
