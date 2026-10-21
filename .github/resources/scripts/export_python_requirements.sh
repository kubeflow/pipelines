#!/bin/bash
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      https://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Run with uv 0.13.4, matching the Python dependency check.
set -euo pipefail

if [[ $# -ne 1 || -z "$1" ]]; then
  echo "Usage: $0 OUTPUT_DIRECTORY (temporary or release artifact directory)" >&2
  exit 2
fi
mkdir -p -- "$1"
OUTPUT_DIRECTORY="$(cd -- "$1" && pwd)"
cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../../.."
# Exclude local workspace projects so these dependency-only artifacts are portable.
uv export --frozen --no-dev --no-hashes --no-emit-workspace --format requirements-txt -o "$OUTPUT_DIRECTORY/workspace-requirements.txt"
uv export --frozen --no-dev --no-hashes --no-emit-workspace --package kfp --format requirements-txt -o "$OUTPUT_DIRECTORY/kfp-requirements.txt"
