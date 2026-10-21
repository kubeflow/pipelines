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

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/../../.."
export GOPROXY="${GOPROXY:-https://proxy.golang.org|direct}"

# Linux CI supplies GNU timeout. One deadline covers downloads and retry sleeps,
# including go/git children; a stuck process gets ten seconds to terminate.
if timeout --kill-after=10s 300s bash -c '
  set -euo pipefail
  source "$1"
  retry 3 10 go mod download
' download-go-modules "$SCRIPT_DIR/helper-functions.sh"; then
  exit 0
else
  status=$?
  echo "ERROR: Go module downloads failed; check the download errors above before running tests." >&2
  exit "$status"
fi
