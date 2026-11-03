#!/usr/bin/env bash
#
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

if command -v gh >/dev/null 2>&1; then
  gh --version
  exit 0
fi

: "${RUNNER_TEMP:?RUNNER_TEMP must be set}"
: "${GITHUB_PATH:?GITHUB_PATH must be set}"
if [[ "$(uname -s)" != Linux ]]; then
  echo "GitHub CLI provisioning supports Linux runners only." >&2
  exit 2
fi

version=2.102.0
case "$(uname -m)" in
  x86_64)
    architecture=amd64
    checksum=bb766f710eef8ede859c18578c72c327597cd4c8a85b06001b1f3843c6019386
    ;;
  aarch64|arm64)
    architecture=arm64
    checksum=7862c86c72f43df3a2d93ddde6f473285b4e2af61b494849846827e513ef6484
    ;;
  *)
    echo "Unsupported runner architecture for GitHub CLI provisioning." >&2
    exit 2
    ;;
esac

# The checksums are pinned from the official cli/cli release assets. Do not
# fetch a mutable checksum alongside the archive being verified.
installation=$(mktemp -d "${RUNNER_TEMP}/kfp-gh.XXXXXX")
trap 'rm -rf "$installation"' EXIT
archive="${installation}/gh.tar.gz"
package="gh_${version}_linux_${architecture}"
curl --fail --location --proto '=https' --retry 3 --retry-all-errors \
  --connect-timeout 15 --max-time 120 \
  "https://github.com/cli/cli/releases/download/v${version}/${package}.tar.gz" \
  --output "$archive"
printf '%s  %s\n' "$checksum" "$archive" | sha256sum --check --status
tar -xzf "$archive" -C "$installation" --strip-components=1 "${package}/bin/gh"
"${installation}/bin/gh" --version
printf '%s\n' "${installation}/bin" >> "$GITHUB_PATH"
rm "$archive"
# The runner's temporary directory owns cleanup after subsequent action steps.
trap - EXIT
