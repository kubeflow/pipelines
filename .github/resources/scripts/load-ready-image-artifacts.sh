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
RUNTIME_DOWNLOAD_ONLY="${RUNTIME_DOWNLOAD_ONLY:-false}"

script_directory=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd -P)
source "$script_directory/ci-image-artifacts.sh"
source "$script_directory/helper-functions.sh"
# Download logs are replayed after the cross-step join. Record phase timestamps
# here so replay does not turn their GitHub log timestamps into transfer times.
log_phase() {
  printf '%s %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"
}
deploy_files=$(ci_artifact_files deploy-image-files "$LOAD_MODELCAR_FIXTURE")

worker=false
if [[ "${1:-}" == --worker ]]; then
  worker=true
  shift
fi
# The runtime-only prefetch has one worker and inherits its supervisor's
# process group, so cancellation owns every downloader descendant.
if [[ "$RUNTIME_DOWNLOAD_ONLY" == true ]]; then worker=true; fi
if (( $# == 0 )) || { [[ "$worker" == true ]] && (( $# != 1 )); }; then
  echo "Usage: $0 NAME:ARTIFACT_ID [NAME:ARTIFACT_ID ...]" >&2
  exit 2
fi
seen=()
for selection in "$@"; do
  artifact=${selection%%:*}
  artifact_id=${selection#*:}
  if [[ "$selection" != *:* || ! "$artifact_id" =~ ^[1-9][0-9]*$ ]]; then
    echo "::error::Expected an artifact name and positive ID: $selection" >&2
    exit 2
  fi
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

# Extract only declared files; validate every ZIP entry before writing any file.
# The shared artifact-files helper validates destination paths before this runs.
extract_artifact() {
  timeout --foreground --kill-after=30s 10m python3 - "$artifact_zip" "$1" "$2" "${3:-false}" <<'PYZIP'
from pathlib import Path
import shutil
import stat
import sys
import zipfile
import zlib

archive_path, destination, manifest, allow_reference = sys.argv[1:]
required = set(manifest.splitlines())
root = Path(destination)
try:
    with zipfile.ZipFile(archive_path) as archive:
        if allow_reference == 'true' and 'runtime-reference.json' in archive.namelist():
            if any(name in archive.namelist() for name in required):
                raise ValueError('Runtime artifact mixes a reference with image archives')
            required = {'runtime-reference.json'}
            reference = archive.getinfo('runtime-reference.json')
            if reference.file_size > 1024:
                raise ValueError('Runtime reference exceeds 1024 bytes')
        selected = {}
        for entry in archive.infolist():
            name = entry.filename
            parts = name.rstrip('/').split('/')
            mode = stat.S_IFMT(entry.external_attr >> 16)
            if ('\\' in name or '\x00' in entry.orig_filename or
                    any(part in ('', '.', '..') for part in parts) or
                    mode not in (0, stat.S_IFREG, stat.S_IFDIR)):
                raise ValueError(f'Unsafe ZIP member: {name!r}')
            if name not in required:
                continue
            if entry.is_dir() or mode == stat.S_IFDIR or name in selected:
                raise ValueError(f'Invalid or duplicate required ZIP member: {name!r}')
            selected[name] = entry
        for name, entry in selected.items():
            target = root / name
            # Required paths were checked before download. Recheck existing
            # parents, and create the file exclusively, never following links.
            parent = root
            for component in Path(name).parts[:-1]:
                parent /= component
                if parent.is_symlink():
                    raise ValueError(f'Symlink in extraction path: {name!r}')
                parent.mkdir(exist_ok=True)
            with archive.open(entry) as source, target.open('xb') as output:
                shutil.copyfileobj(source, output)
except (zipfile.BadZipFile, EOFError, zlib.error) as error:
    print(f'::warning::Incomplete or corrupt artifact ZIP: {error}', file=sys.stderr)
    sys.exit(10)
except (OSError, RuntimeError, ValueError) as error:
    print(f'::error::Cannot safely extract artifact ZIP: {error}', file=sys.stderr)
    sys.exit(2)
PYZIP
}

load_runtime_bundles() {
  local reference=$1 destination=$2 selections bundle shared_id metadata inventory
  selections=$(python3 "$script_directory/runtime_base_image_artifacts.py" reference-bundles "$reference" \
    --argo-version "${RUNTIME_ARGO_VERSION:-}" --db-type "${RUNTIME_DB_TYPE:-}" \
    --modelcar "$LOAD_MODELCAR_FIXTURE")
  # Validate every selected source before importing anything. The reference is
  # only an ID hint; GitHub metadata establishes source, fingerprint and expiry.
  while read -r bundle shared_id; do
    metadata=$(timeout --foreground --kill-after=30s 10m gh api \
      "repos/$GITHUB_REPOSITORY/actions/artifacts/$shared_id")
    python3 "$script_directory/runtime_base_image_artifacts.py" validate-bundle \
      --reference "$reference" --bundle "$bundle" --source-sha "${SOURCE_SHA:?}" \
      --repository-id "${GITHUB_REPOSITORY_ID:?}" <<< "$metadata" > /dev/null
  done <<< "$selections"
  while read -r bundle shared_id; do
    log_phase "[runtime-base-images] selected bundle $bundle, validated shared artifact ID $shared_id"
    download_artifact "runtime-$bundle" "$shared_id" "$destination/$bundle" "$bundle.tar"
    if [[ "$RUNTIME_DOWNLOAD_ONLY" == true ]]; then continue; fi
    log_phase "[runtime-$bundle] Kind archive load started"
    retry 3 10 kind --name "$CLUSTER_NAME" load image-archive "$destination/$bundle/$bundle.tar"
    log_phase "[runtime-$bundle] Kind archive load completed; CRI verification started"
    if [[ "$bundle" != modelcar ]]; then
      inventory="$destination/$bundle/images.txt"
      python3 "$script_directory/runtime_image_bundles.py" "$bundle" --inventory "$script_directory/../runtime-base-images.txt" > "$inventory"
      verify_runtime_base_images_in_kind "$inventory" "$CLUSTER_NAME"
    fi
    log_phase "[runtime-$bundle] CRI verification completed"
    rm -- "$destination/$bundle/$bundle.tar"
  done <<< "$selections"
  if [[ "$RUNTIME_DOWNLOAD_ONLY" == true ]]; then
    log_phase "[runtime-base-images] prefetch completed (downloads only)"
    return
  fi
  rm -- "$reference"
  log_phase "[runtime-base-images] import completed"
}

download_artifact() {
  local artifact=$1 artifact_id=$2 destination=$3 required_files=$4 attempt result
  local allow_reference=${5:-false} verified_files
  if [[ "$artifact" == runtime-* && "$RUNTIME_DOWNLOAD_ONLY" != true ]]; then
    if verified_files=$(python3 "$script_directory/runtime_prefetch_files.py" verify \
        "$destination" "$artifact_id" "$required_files" "$allow_reference"); then
      if DOWNLOAD_PATH="$destination" REQUIRED_FILES="$verified_files" \
          bash "$script_directory/artifact-files.sh" verify; then
        log_phase "[$artifact] reusing verified prefetched artifact ID $artifact_id"
        return
      fi
    fi
  fi
  for attempt in 1 2; do
    DOWNLOAD_PATH="$destination" REQUIRED_FILES="$required_files" \
      bash "$script_directory/artifact-files.sh" prepare
    rm -f -- "$destination/.runtime-prefetch.json"
    if [[ "$allow_reference" == true ]]; then
      rm -f -- "$destination/runtime-reference.json"
    fi
    log_phase "[$artifact] download started (artifact ID $artifact_id, attempt $attempt/2)"
    if timeout --foreground --kill-after=30s 10m gh api \
        "repos/$GITHUB_REPOSITORY/actions/artifacts/$artifact_id/zip" > "$artifact_zip"; then
      log_phase "[$artifact] ZIP transfer completed; extraction started"
      if extract_artifact "$destination" "$required_files" "$allow_reference"; then
        log_phase "[$artifact] ZIP extraction completed; completeness verification started"
        verified_files=$required_files
        if [[ "$allow_reference" == true && -f "$destination/runtime-reference.json" ]]; then
          verified_files=runtime-reference.json
        fi
        if DOWNLOAD_PATH="$destination" REQUIRED_FILES="$verified_files" \
            bash "$script_directory/artifact-files.sh" verify; then
          rm -- "$artifact_zip"
          log_phase "[$artifact] download completed"
          if [[ "$RUNTIME_DOWNLOAD_ONLY" == true ]]; then
            python3 "$script_directory/runtime_prefetch_files.py" save \
              "$destination" "$artifact_id" "$required_files" "$allow_reference"
          fi
          return 0
        else
          result=$?
        fi
      else
        result=$?
      fi
      # Only incomplete downloads are retryable. Unsafe paths and other
      # validation failures must stop before any subsequent download.
      if [[ "$result" != 10 ]]; then return "$result"; fi
    fi
    if [[ "$attempt" == 1 ]]; then
      log_phase "[$artifact] download failed or incomplete; retrying in 20 seconds"
      sleep 20
    fi
  done
  echo "::error::Artifact $artifact failed download or completeness checks after two attempts" >&2
  return 1
}

load_artifact() {
  local artifact=${1%%:*} artifact_id=${1#*:} required_files="" file archive
  local destination="$IMAGE_PATH/$artifact"
  while IFS= read -r file; do
    if [[ "$file" == "$artifact/"* ]]; then
      required_files+="${file#"$artifact/"}"$'\n'
    fi
  done <<< "$deploy_files"
  if [[ "$artifact" == runtime-base-images ]]; then
    download_artifact "$artifact" "$artifact_id" "$destination" "$required_files" true
    if [[ -f "$destination/runtime-reference.json" ]]; then
      local reference="$destination/runtime-reference.json" metadata shared_id
      if python3 - "$reference" <<'PYBUNDLES'
import json
from pathlib import Path
import sys
sys.exit(0 if 'bundles' in json.loads(Path(sys.argv[1]).read_text()) else 1)
PYBUNDLES
      then
        load_runtime_bundles "$reference" "$destination"
        return
      fi
      # Legacy references retain the normal single-archive path at deployment.
      if [[ "$RUNTIME_DOWNLOAD_ONLY" == true ]]; then return; fi
      shared_id=$(python3 "$script_directory/runtime_base_image_artifacts.py" reference-id "$reference")
      metadata=$(timeout --foreground --kill-after=30s 10m gh api \
        "repos/$GITHUB_REPOSITORY/actions/artifacts/$shared_id")
      shared_id=$(python3 "$script_directory/runtime_base_image_artifacts.py" validate-reference \
        --reference "$reference" --source-sha "${SOURCE_SHA:?}" \
        --repository-id "${GITHUB_REPOSITORY_ID:?}" <<< "$metadata")
      log_phase "[$artifact] validated shared artifact ID $shared_id; downloading directly"
      if [[ "$RUNTIME_DOWNLOAD_ONLY" != true ]]; then rm -- "$reference"; fi
      download_artifact "$artifact" "$shared_id" "$destination" "$required_files"
    fi
  else
    download_artifact "$artifact" "$artifact_id" "$destination" "$required_files"
  fi
  if [[ "$RUNTIME_DOWNLOAD_ONLY" == true ]]; then
    log_phase "[$artifact] prefetch completed (downloads only)"
    return
  fi
  archive="$destination/$artifact.tar"
  log_phase "[$artifact] import started"
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
      log_phase "[$artifact] Kind archive load started"
      retry 3 10 kind --name "$CLUSTER_NAME" load image-archive "$archive"
      log_phase "[$artifact] Kind archive load completed; CRI verification started"
      verify_runtime_base_images_in_kind "$script_directory/../runtime-base-images.txt" "$CLUSTER_NAME"
      log_phase "[$artifact] CRI verification completed"
      rm -- "$archive"
      if [[ "$LOAD_MODELCAR_FIXTURE" == true ]]; then
        log_phase "[$artifact] Modelcar import started"
        retry 3 10 kind --name "$CLUSTER_NAME" load image-archive "$destination/modelcar.tar"
        log_phase "[$artifact] Modelcar import completed"
        rm -- "$destination/modelcar.tar"
      fi
      ;;
    *)
      timeout --foreground --kill-after=30s 10m \
        bash "$script_directory/load-kind-image-archives.sh" "$CLUSTER_NAME" 1 "$archive"
      ;;
  esac
  log_phase "[$artifact] import completed"
}

if [[ "$worker" == true ]]; then
  artifact_zip=$(mktemp)
  trap 'rm -f -- "$artifact_zip"' EXIT
  trap 'exit 143' TERM
  trap 'exit 130' INT
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
