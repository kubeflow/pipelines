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

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON_EXEC="${PYTHON_EXEC:-python3}"
JUNIT_SCRIPT="${JUNIT_SCRIPT:-${SCRIPT_DIR}/junit_to_summary.py}"

# Build arguments array
args=()

# Add custom data if provided
if [[ -n "${CUSTOM_DATA_INPUT:-}" ]]; then
  args+=(--custom-data "$CUSTOM_DATA_INPUT")
fi

# Add custom data file if provided
if [[ -n "${CUSTOM_DATA_FILE_INPUT:-}" ]]; then
  args+=(--custom-data-file "$CUSTOM_DATA_FILE_INPUT")
fi

# Add fail_on_test_failures flag
if [[ "${FAIL_ON_TEST_FAILURES:-true}" != "true" ]]; then
  args+=(--no-fail-on-test-failures)
fi

# Add output file if specified via environment
if [[ -n "${OUTPUT_PATH:-}" ]]; then
  args+=(--output "$OUTPUT_PATH")
fi

# XML files / targets specified as positional args or via XML_FILES environment variable
xml_targets=()
if [[ $# -gt 0 ]]; then
  xml_targets=("$@")
elif [[ -n "${XML_FILES:-}" ]]; then
  read -r -a xml_targets <<< "$XML_FILES"
fi

if [[ ${#xml_targets[@]} -eq 0 ]]; then
  echo "Error: No XML files or targets specified" >&2
  exit 1
fi

"$PYTHON_EXEC" "$JUNIT_SCRIPT" "${xml_targets[@]}" "${args[@]}"
