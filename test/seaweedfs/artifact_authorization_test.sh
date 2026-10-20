#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
set -euo pipefail

# Runs entirely against disposable local storage. Requires Docker, Node, installed
# frontend/server dependencies, and a Python interpreter with boto3 installed.
# PYTHON=/path/to/venv/bin/python bash test/seaweedfs/artifact_authorization_test.sh
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
python="${PYTHON:-python3}"
"$python" -c 'import boto3' || {
  echo 'Install boto3 in a temporary Python environment and set PYTHON to that interpreter.' >&2
  exit 1
}
fixture_dir=$(mktemp -d)
container="kfp-artifact-auth-${RANDOM}-$$"
cleanup() {
  result=$?
  if (( result != 0 )); then
    docker logs --tail 100 "$container" >&2 || true
  fi
  docker rm --force --volumes "$container" >/dev/null 2>&1 || true
  rm -rf "$fixture_dir"
  exit "$result"
}
trap cleanup EXIT

# Exercise this branch's shipped storage version; overrides are for compatibility runs.
manifest="$repo_root/manifests/kustomize/third-party/seaweedfs/base/seaweedfs/seaweedfs-deployment.yaml"
image="${SEAWEEDFS_IMAGE:-$("$python" -c 'import shlex, sys; print(next(shlex.split(line)[1] for line in open(sys.argv[1]) if line.strip().startswith("image:")))' "$manifest")}"
if [[ -z "$image" ]]; then
  echo 'Could not determine SeaweedFS image from the branch manifest.' >&2
  exit 1
fi
printf 'Testing artifact authorization with %s\n' "$image"
docker run --detach --name "$container" \
  --publish 127.0.0.1::8333 --publish 127.0.0.1::8111 \
  "$image" \
  server -dir=/tmp -s3 -iam -filer -master.volumePreallocate=false >/dev/null
s3_port=$(docker inspect --format '{{(index (index .NetworkSettings.Ports "8333/tcp") 0).HostPort}}' "$container")
iam_port=$(docker inspect --format '{{(index (index .NetworkSettings.Ports "8111/tcp") 0).HostPort}}' "$container")
export S3_ENDPOINT_URL="http://127.0.0.1:${s3_port}"
export AWS_ENDPOINT_URL="http://127.0.0.1:${iam_port}"
export AWS_ACCESS_KEY_ID=fixture-admin
export AWS_SECRET_ACCESS_KEY=fixture-admin-secret
export AWS_DEFAULT_REGION=us-east-1

ready=false
for _ in $(seq 1 60); do
  if curl --fail --silent "${S3_ENDPOINT_URL}/status" >/dev/null; then
    ready=true
    break
  fi
  sleep 1
done
if [[ "$ready" != true ]]; then
  echo 'Disposable SeaweedFS did not become ready within 60 seconds.' >&2
  exit 1
fi
docker exec "$container" sh -c \
  'echo "s3.configure -user kubeflow-admin -access_key fixture-admin -secret_key fixture-admin-secret -actions Admin -apply" | weed shell' >/dev/null
"$python" "$repo_root/test/seaweedfs/artifact_authorization_fixture.py" "$fixture_dir/identities.json"
cd "$repo_root/frontend/server"
KFP_LIVE_ARTIFACT_AUTH_FIXTURE="$fixture_dir/identities.json" \
  npm test -- integration-tests/artifact-storage-authorization-live.test.ts
