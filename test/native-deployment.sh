#!/usr/bin/env bash
# Copyright 2026 The Kubeflow Authors
# SPDX-License-Identifier: Apache-2.0
set -euo pipefail
cd "$(dirname "$0")/.."
cluster=native-kfp
namespace=kfp-argo
kubeconfig=$(mktemp)
export KUBECONFIG="$kubeconfig"
kind create cluster --name "$cluster" --kubeconfig "$kubeconfig" --wait 120s
forward_pid=''
cleanup() {
  if [[ -n "$forward_pid" ]]; then kill "$forward_pid" 2>/dev/null || true; fi
  kind delete cluster --name "$cluster"
  rm -f "$kubeconfig"
}
trap cleanup EXIT
context="kind-$cluster"
port=${KFP_TEST_PORT:-$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')}
uv build --package kfp --wheel
npm --prefix frontend run build
docker build -f samples/argo-native/Dockerfile -t kfp-argo-demo:dev .
docker build -f samples/argo-native/Dockerfile.ui -t kfp-argo-ui:test .
uv run python samples/argo-native/compile_demo.py --output /tmp/native-version.yaml
image_digest=$(docker image inspect kfp-argo-demo:dev --format '{{.Id}}')
image="kfp-argo-demo:kfp-${image_digest#sha256:}"
kind load docker-image --name "$cluster" "$image" kfp-argo-ui:test
uv run python samples/argo-native/deploy.py --namespace "$namespace" --ui-image kfp-argo-ui:test > /tmp/native-install.yaml
kubectl --context "$context" -n "$namespace" apply --server-side -f /tmp/native-install.yaml
for deployment in postgres minio workflow-controller argo-server kfp-argo-ui; do
  kubectl --context "$context" -n "$namespace" rollout status "deployment/$deployment" --timeout=180s
done
kubectl --context "$context" -n "$namespace" apply -f /tmp/native-version.yaml
kubectl --context "$context" -n "$namespace" port-forward svc/kfp-argo-ui "$port:80" > /tmp/native-port-forward.log 2>&1 &
forward_pid=$!
for _ in $(seq 1 60); do
  kill -0 "$forward_pid"
  if grep -q 'Forwarding from 127.0.0.1:' /tmp/native-port-forward.log && curl -fsS "http://127.0.0.1:$port/argo.html" -o /dev/null; then break; fi
  sleep 1
done
export ARGO_EDITOR_TOKEN ARGO_READER_TOKEN
ARGO_EDITOR_TOKEN=$(kubectl --context "$context" -n "$namespace" create token argo-ui-editor --duration=2h)
ARGO_READER_TOKEN=$(kubectl --context "$context" -n "$namespace" create token argo-ui-reader --duration=2h)
if [[ ${GITHUB_ACTIONS:-} == true ]]; then
  printf '::add-mask::%s\n' "$ARGO_EDITOR_TOKEN" "$ARGO_READER_TOKEN"
fi
export ARGO_SERVER="127.0.0.1:$port" ARGO_UI_URL="http://127.0.0.1:$port" ARGO_BASE_HREF=/argo ARGO_HTTP1=true ARGO_SECURE=false
export ARGO_TOKEN="Bearer $ARGO_EDITOR_TOKEN"
uv run python samples/argo-native/verify_compatibility.py --context "$context" --namespace "$namespace" --image-digest "$image_digest"
export ARGO_CONTEXT="$context" ARGO_NAMESPACE="$namespace"
export ARGO_VERSION_NAME
ARGO_VERSION_NAME=$(uv run python -c "import yaml; print(yaml.safe_load(open('/tmp/native-version.yaml'))['metadata']['name'])")
export PLAYWRIGHT_CHANNEL=${PLAYWRIGHT_CHANNEL:-chromium}
node --test frontend/scripts/argo-deployment.smoke.mjs
