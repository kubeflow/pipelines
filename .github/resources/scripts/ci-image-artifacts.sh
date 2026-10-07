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

CONTROL_PLANE_IMAGE_ARTIFACTS=(
  "apiserver"
  "scheduledworkflow"
  "persistenceagent"
  "frontend"
  "viewer-crd-controller"
  "profile-controller"
)
RUNTIME_IMAGE_ARTIFACTS=("driver" "launcher")
ALL_CI_IMAGE_ARTIFACTS=(
  "${CONTROL_PLANE_IMAGE_ARTIFACTS[@]}"
  "${RUNTIME_IMAGE_ARTIFACTS[@]}"
  "runtime-base-images"
)

# The native tooling producer and its consumers use these same output names.
TOOL_GENERATOR_IMAGE="kfp-api-generator"
TOOL_RELEASE_IMAGE="kfp-release"
TOOL_IMAGE_ARTIFACTS=("$TOOL_GENERATOR_IMAGE" "$TOOL_RELEASE_IMAGE")
TOOL_ARCHITECTURES=("amd64" "arm64")
TOOL_API_OUTPUT_SUFFIX=".sha256"
TOOL_MANIFESTS_OUTPUT="manifests.sha256"
TOOL_CHANGELOG_OUTPUT="CHANGELOG.md"

tool_output_files() {
  local image
  for image in "${TOOL_IMAGE_ARTIFACTS[@]}"; do
    printf '%s%s\n' "$image" "$TOOL_API_OUTPUT_SUFFIX"
  done
  printf '%s\n' "$TOOL_MANIFESTS_OUTPUT" "$TOOL_CHANGELOG_OUTPUT"
}

ci_artifact_files() {
  local kind="${1:?Expected artifact inventory command}" architecture image file attempt
  case "$kind" in
    deploy-image-files)
      case "${2:-false}" in
        true|false) ;;
        *) echo "Expected true or false for Modelcar fixture" >&2; return 1 ;;
      esac
      for image in "${ALL_CI_IMAGE_ARTIFACTS[@]}"; do
        printf '%s/%s.tar\n' "$image" "$image"
      done
      if [[ "${2:-false}" == true ]]; then
        printf '%s\n' 'runtime-base-images/modelcar.tar'
      fi
      ;;
    arm64-image-files)
      for image in "${CONTROL_PLANE_IMAGE_ARTIFACTS[@]}" "${RUNTIME_IMAGE_ARTIFACTS[@]}"; do
        printf '%s-arm64.tar\n%s-arm64.json\n' "$image" "$image"
      done
      ;;
    platform-digest-files)
      PYTHONPATH="$(dirname -- "${BASH_SOURCE[0]}")" python3 -c \
        'import sys; from publish_image_index import parse_platforms; print("\n".join(platform.split("/")[1] + ".json" for platform in sorted(parse_platforms(sys.argv[1]))))' \
        "${2:?Expected Linux platforms}"
      ;;
    tool-output-files)
      tool_output_files
      ;;
    tool-output-artifacts)
      attempt="${2:?Expected workflow run attempt}"
      if [[ ! "$attempt" =~ ^[1-9][0-9]*$ ]]; then
        echo "Expected a positive workflow run attempt" >&2
        return 1
      fi
      for architecture in "${TOOL_ARCHITECTURES[@]}"; do
        while IFS= read -r file; do
          printf 'tool-output-%s-%s/%s\n' "$architecture" "$attempt" "$file"
        done < <(tool_output_files)
      done
      ;;
    tool-digest-files)
      for image in "${TOOL_IMAGE_ARTIFACTS[@]}"; do
        for architecture in "${TOOL_ARCHITECTURES[@]}"; do
          printf '%s/%s.json\n' "$image" "$architecture"
        done
      done
      ;;
    published-image-files)
      PYTHONPATH="$(dirname -- "${BASH_SOURCE[0]}")" python3 -c \
        'from arm64_smoke import IMAGES; print("\n".join(f"{image}.json" for image in sorted(IMAGES)))'
      ;;
    *)
      echo "Unknown artifact inventory: $kind" >&2
      return 1
      ;;
  esac
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  set -euo pipefail
  if [[ "${1:-}" == --github-output ]]; then
    shift
    manifest=$(ci_artifact_files "$@")
    printf 'required-files<<EOF\n%s\nEOF\n' "$manifest" >> "${GITHUB_OUTPUT:?Expected GITHUB_OUTPUT}"
  else
    ci_artifact_files "$@"
  fi
fi
