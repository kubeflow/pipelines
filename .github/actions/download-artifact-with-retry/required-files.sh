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

mode="${1:?Expected prepare or verify}"
case "$mode" in
  prepare|verify) ;;
  *) echo "::error::Unknown required-file operation: $mode"; exit 1 ;;
esac

required_files="${REQUIRED_FILES:-}"
if [[ -z "${required_files//[[:space:]]/}" ]]; then
  echo "::error::Set required-files to the complete list of expected artifact files"
  exit 1
fi
# Preparation and download must resolve the destination identically. Unlike
# the download action, shell paths do not expand a tilde stored in a variable.
case "$DOWNLOAD_PATH" in
  "~"*)
    echo "::error::required-files needs an absolute or workspace-relative destination without tilde expansion"
    exit 1
    ;;
esac
mkdir -p "$DOWNLOAD_PATH"
cd "$DOWNLOAD_PATH"

files=()
while IFS= read -r file || [[ -n "$file" ]]; do
  [[ "$file" =~ [^[:space:]] ]] || continue
  case "/$file/" in
    //*|*/../*|*/./*|*//*)
      echo "::error::Required file must be a relative file path: $file"
      exit 1
      ;;
  esac
  # Never follow a symlink while clearing previous download output.
  component="$file"
  while [[ "$component" != "." ]]; do
    if [[ -L "$component" ]]; then
      echo "::error::Required file path contains a symlink: $file"
      exit 1
    fi
    component="$(dirname -- "$component")"
  done
  if [[ -e "$file" && ! -f "$file" ]]; then
    echo "::error::Required file path is not a regular file: $file"
    exit 1
  fi
  files+=("$file")
done <<< "$required_files"

missing=0
for file in "${files[@]}"; do
  if [[ "$mode" == "prepare" ]]; then
    # Require every attempt to supply its own complete set. A failed download
    # may leave unverified bytes that must not satisfy the next attempt.
    rm -f -- "$file"
  elif [[ ! -s "$file" ]]; then
    echo "Missing or empty required artifact file: $file"
    missing=1
  fi
done
exit "$missing"
