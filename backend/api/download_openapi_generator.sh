#!/bin/bash
#
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
codegen_file=$1
codegen_uri="https://repo1.maven.org/maven2/org/openapitools/openapi-generator-cli/4.3.1/openapi-generator-cli-4.3.1.jar"
if [[ -s "$codegen_file" ]]; then
    exit 0
fi

# Publish only completed downloads. Keep the temporary file beside the cache so
# rename is atomic; a failed transfer must not become the next invocation's JAR.
temporary=$(mktemp "${codegen_file}.XXXXXX")
trap 'rm -f "$temporary"' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# curl's default retry policy excludes TLS resets (exit 35). Bound every attempt
# explicitly rather than depending on newer curl retry-all-errors support.
for attempt in 1 2 3 4; do
    : > "$temporary"
    if curl --fail --location --connect-timeout 20 --max-time 120 \
        "$codegen_uri" --output "$temporary" && [[ -s "$temporary" ]]; then
        mv -f "$temporary" "$codegen_file"
        exit 0
    fi
    if [[ "$attempt" -lt 4 ]]; then
        sleep 2
    fi
done

echo "Failed to download the OpenAPI generator after four attempts; retry generation." >&2
exit 1
