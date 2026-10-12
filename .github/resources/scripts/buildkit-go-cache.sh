#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
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

# The GHA layer exporter does not persist RUN cache mounts. Transfer the Go
# compiler cache separately using the same pinned builder image and mount ID.
set -euo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
  echo 'Usage: buildkit-go-cache.sh restore|save DOCKERFILE BUILDER [CACHE_ID]' >&2
  exit 2
fi
mode="$1"
dockerfile="$2"
builder="$3"
cache_id="${4:-kfp-go-build}"
case "$mode" in restore|save) ;; *) echo 'Expected restore or save' >&2; exit 2 ;; esac
case "$cache_id" in kfp-go-build|kfp-go-build-retry) ;; *) echo 'Unexpected Go cache ID' >&2; exit 2 ;; esac
case "${TARGET_ARCH:?}" in amd64|arm64) ;; *) echo 'Unsupported architecture' >&2; exit 2 ;; esac
if [[ "${RUNNER_TEMP:?}" != /* || "$RUNNER_TEMP" == / ]]; then
  echo 'RUNNER_TEMP must be an absolute temporary directory' >&2
  exit 2
fi

cache_directory="$RUNNER_TEMP/go-build-cache"
if [[ -L "$cache_directory" ]]; then
  echo 'Go cache directory must not be a symbolic link' >&2
  exit 2
fi
base_image=$(awk 'toupper($1) == "FROM" {print $2; exit}' "$dockerfile")
if [[ ! "$base_image" =~ ^golang:[a-zA-Z0-9._-]+@sha256:[a-f0-9]{64}$ ]]; then
  echo 'Expected a digest-pinned Go builder as the first Dockerfile stage' >&2
  exit 2
fi

if [[ "$mode" == restore ]] && \
    { [[ ! -d "$cache_directory" ]] || [[ -z "$(ls -A "$cache_directory")" ]]; }; then
  echo 'No saved Go compiler cache; building with an empty cache mount.'
  exit 0
fi

temporary=$(mktemp -d "$RUNNER_TEMP/go-build-cache-transfer.XXXXXX")
trap 'rm -rf -- "$temporary"' EXIT
trap 'exit 143' TERM
trap 'exit 130' INT
# --no-cache clears referenced cache mounts in BuildKit. A unique argument
# reruns only the transfer instruction while preserving the compiler mount.
mount="type=cache,id=$cache_id,target=/root/.cache/go-build,sharing=locked"
build=(timeout --foreground --kill-after=10s 120s docker buildx build
  --builder "$builder" --platform "linux/$TARGET_ARCH"
  --build-arg "CACHE_TRANSFER_NONCE=${temporary##*/}" --file -)

if [[ "$mode" == restore ]]; then
  "${build[@]}" --output type=cacheonly "$cache_directory" <<EOF
FROM $base_image
COPY . /restore/
ARG CACHE_TRANSFER_NONCE
RUN --mount=$mount test -n "\$CACHE_TRANSFER_NONCE" && cp -a /restore/. /root/.cache/go-build/
EOF
else
  "${build[@]}" --output "type=local,dest=$temporary/output" "$temporary" <<EOF
FROM $base_image AS cache
ARG CACHE_TRANSFER_NONCE
RUN --mount=$mount test -n "\$CACHE_TRANSFER_NONCE" && mkdir /export && cp -a /root/.cache/go-build/. /export/
FROM scratch
COPY --from=cache /export/ /
EOF
  # A fresh export discards Go entries that the compiler has garbage-collected.
  # Do not alter the restored cache if exporting the new mount failed.
  rm -rf -- "$cache_directory"
  mv "$temporary/output" "$cache_directory"
fi
