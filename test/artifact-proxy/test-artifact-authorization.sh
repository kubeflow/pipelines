#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
set -euo pipefail

# This mutates a disposable multi-user test installation. Never run on production.
# Credentials are handled in Python memory; do not enable shell tracing.
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
python="${PYTHON:-python3}"
command -v kubectl >/dev/null
"$python" -c 'import boto3' || {
  echo 'Install boto3 in a temporary Python environment and set PYTHON to that interpreter.' >&2
  exit 1
}
exec "$python" "$repo_root/test/artifact-proxy/artifact_authorization_acceptance.py" "$@"
