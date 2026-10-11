#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

: "${GITHUB_REPOSITORY:?GITHUB_REPOSITORY must be set}"
: "${GITHUB_RUN_ID:?GITHUB_RUN_ID must be set}"
: "${GH_TOKEN:?GH_TOKEN must be set}"
: "${IMAGE_PATH:?IMAGE_PATH must be set}"
: "${CLUSTER_NAME:?CLUSTER_NAME must be set}"
: "${IMAGE_REGISTRY:?IMAGE_REGISTRY must be set}"
: "${IMAGE_TAG:?IMAGE_TAG must be set}"
LOAD_MODELCAR_FIXTURE="${LOAD_MODELCAR_FIXTURE:-false}"

script_directory=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
source "$script_directory/ci-image-artifacts.sh"
source "$script_directory/helper-functions.sh"
deploy_files=$(ci_artifact_files deploy-image-files "$LOAD_MODELCAR_FIXTURE")

worker=false
if [[ "${1:-}" == --worker ]]; then
  worker=true
  shift
fi
if (( $# == 0 )) || { [[ "$worker" == true ]] && (( $# != 1 )); }; then
  echo "Usage: $0 ARTIFACT [ARTIFACT ...]" >&2
  exit 2
fi
seen=()
for artifact in "$@"; do
  known=false
  for expected in "${ALL_CI_IMAGE_ARTIFACTS[@]}"; do
    if [[ "$artifact" == "$expected" ]]; then known=true; break; fi
  done
  if [[ "$known" != true ]]; then
    echo "::error::Unknown CI image artifact: $artifact" >&2
    exit 2
  fi
  for previous in "${seen[@]:-}"; do
    if [[ "$artifact" == "$previous" ]]; then
      echo "::error::Duplicate CI image artifact: $artifact" >&2
      exit 2
    fi
  done
  seen+=("$artifact")
done

# --foreground keeps commands in their worker's process group, allowing the
# supervisor to stop downloads and Docker/Kind descendants on cancellation.
kind() { timeout --foreground --kill-after=30s 10m kind "$@"; }
docker() { timeout --foreground --kill-after=30s 10m docker "$@"; }

download_artifact() {
  local artifact=$1 destination=$2 required_files=$3 attempt result
  for attempt in 1 2; do
    DOWNLOAD_PATH="$destination" REQUIRED_FILES="$required_files" \
      bash "$script_directory/artifact-files.sh" prepare
    echo "[$artifact] download started (attempt $attempt/2)"
    if timeout --foreground --kill-after=30s 10m gh run download "$GITHUB_RUN_ID" \
        --repo "$GITHUB_REPOSITORY" --name "$artifact" --dir "$destination"; then
      if DOWNLOAD_PATH="$destination" REQUIRED_FILES="$required_files" \
          bash "$script_directory/artifact-files.sh" verify; then
        echo "[$artifact] download completed"
        return 0
      else
        result=$?
        # Only absent/empty files are retryable. Unsafe paths and other
        # validation failures must stop before any subsequent download.
        if [[ "$result" != 10 ]]; then return "$result"; fi
      fi
    fi
    if [[ "$attempt" == 1 ]]; then
      echo "[$artifact] download failed or incomplete; retrying in 20 seconds"
      sleep 20
    fi
  done
  echo "::error::Artifact $artifact failed download or completeness checks after two attempts" >&2
  return 1
}

load_artifact() {
  local artifact=$1 destination="$IMAGE_PATH/$1" required_files="" file archive
  while IFS= read -r file; do
    if [[ "$file" == "$artifact/"* ]]; then
      required_files+="${file#"$artifact/"}"$'\n'
    fi
  done <<< "$deploy_files"
  download_artifact "$artifact" "$destination" "$required_files"
  archive="$destination/$artifact.tar"
  echo "[$artifact] import started"
  case "$artifact" in
    driver|launcher)
      docker image load -i "$archive"
      docker tag "$IMAGE_REGISTRY/$artifact:$IMAGE_TAG" "$IMAGE_REGISTRY/$artifact:ci"
      kind --name "$CLUSTER_NAME" load docker-image "$IMAGE_REGISTRY/$artifact:ci"
      rm -- "$archive"
      docker image rm "$IMAGE_REGISTRY/$artifact:$IMAGE_TAG"
      docker image rm "$IMAGE_REGISTRY/$artifact:ci"
      ;;
    runtime-base-images)
      retry 3 10 kind --name "$CLUSTER_NAME" load image-archive "$archive"
      verify_runtime_base_images_in_kind "$script_directory/../runtime-base-images.txt" "$CLUSTER_NAME"
      rm -- "$archive"
      if [[ "$LOAD_MODELCAR_FIXTURE" == true ]]; then
        retry 3 10 kind --name "$CLUSTER_NAME" load image-archive "$destination/modelcar.tar"
        rm -- "$destination/modelcar.tar"
      fi
      ;;
    *)
      timeout --foreground --kill-after=30s 10m \
        bash "$script_directory/load-kind-image-archives.sh" "$CLUSTER_NAME" 1 "$archive"
      ;;
  esac
  echo "[$artifact] import completed"
}

if [[ "$worker" == true ]]; then
  load_artifact "$1"
  exit 0
fi

# Each worker owns one artifact directory and one process group. Python's
# standard library lets cancellation reap both workers and their descendants,
# including a failing second worker while the first is still downloading.
exec python3 - "$script_directory/load-ready-image-artifacts.sh" "$@" <<'PY'
import os
import signal
import subprocess
import sys
import time

script, *artifacts = sys.argv[1:]
active = []
cancelled = 0


def cancel(signum, _frame):
    global cancelled
    cancelled = signum


def stop_workers():
    for process, _artifact in active:
        try:
            os.killpg(process.pid, signal.SIGTERM)
        except ProcessLookupError:
            pass
    for process, _artifact in active:
        try:
            process.wait(timeout=5)
        except subprocess.TimeoutExpired:
            pass
        # The shell may have exited while a command still owns its group.
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass
        process.wait()


def main():
    signal.signal(signal.SIGINT, cancel)
    signal.signal(signal.SIGTERM, cancel)
    try:
        while (artifacts or active) and not cancelled:
            while artifacts and len(active) < 2 and not cancelled:
                artifact = artifacts.pop(0)
                process = subprocess.Popen(
                    ['bash', script, '--worker', artifact], start_new_session=True)
                active.append((process, artifact))
            for process, artifact in list(active):
                result = process.poll()
                if result is None:
                    continue
                if result:
                    print(f'::error::Image artifact {artifact} failed; stopping other workers',
                          file=sys.stderr)
                    return result if result > 0 else 128 - result
                active.remove((process, artifact))
            if active and not cancelled:
                time.sleep(0.05)
        return 128 + cancelled if cancelled else 0
    finally:
        stop_workers()


sys.exit(main())
PY
