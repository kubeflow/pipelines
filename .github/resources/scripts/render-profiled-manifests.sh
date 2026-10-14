#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
# https://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
set -euo pipefail
if [[ "${PROFILE_RUNTIME_TIMINGS:-false}" != true && "${TUNE_KUBERNETES_CLIENT:-false}" != true ]]; then
  exec kubectl kustomize --load-restrictor LoadRestrictionsNone "$1"
fi
# Patch before initial apply/readiness/port-forward; replacing the live API pod
# later would invalidate the test connection. Never modify the source overlay.
profile_overlay=$(mktemp -d)
trap 'rm -rf "$profile_overlay"' EXIT
kubectl kustomize --load-restrictor LoadRestrictionsNone "$1" > "$profile_overlay/base.yaml"
cat > "$profile_overlay/kustomization.yaml" <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - base.yaml
patches:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: ml-pipeline
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: ml-pipeline
      spec:
        template:
          spec:
            containers:
              - name: ml-pipeline-api-server
                env:
EOF
if [[ "${PROFILE_RUNTIME_TIMINGS:-false}" == true ]]; then
  cat >> "$profile_overlay/kustomization.yaml" <<EOF
                  - name: KFP_RUNTIME_PROFILE_TIMINGS
                    value: "true"
EOF
fi
if [[ "${TUNE_KUBERNETES_CLIENT:-false}" == true ]]; then
  # Existing API-server settings; retain a finite limiter and every auth check.
  cat >> "$profile_overlay/kustomization.yaml" <<EOF
                  - name: CLIENTQPS
                    value: "20"
                  - name: CLIENTBURST
                    value: "40"
EOF
fi
kubectl kustomize --load-restrictor LoadRestrictionsNone "$profile_overlay"
