#!/usr/bin/env bash
set -euxo pipefail

# Artifact Proxy E2E Test
# Validates artifact proxy functionality in multi-user mode

NAMESPACE="${1:-${USER_NAMESPACE:-kubeflow-user-example-com}}"

# Create curl pod for testing
kubectl -n "$NAMESPACE" run kfp-proxy-curl --image=curlimages/curl:8.7.1 --restart=Never \
  --annotations="sidecar.istio.io/inject=false" --command -- sleep 3600
kubectl -n "$NAMESPACE" wait --for=condition=Ready pod/kfp-proxy-curl --timeout=300s

# Test 1: Verify artifact proxy health endpoint
HEALTH_RESPONSE=$(kubectl -n "$NAMESPACE" exec kfp-proxy-curl -- \
  curl -fsS -H 'kubeflow-userid: user@example.com' \
  "http://ml-pipeline-ui-artifact.${NAMESPACE}.svc.cluster.local/apis/v1beta1/healthz")

if ! echo "$HEALTH_RESPONSE" | grep -q '"apiServerReady":true'; then
  echo "ERROR: apiServerReady=false"
  echo "Response: $HEALTH_RESPONSE"
  kubectl -n "$NAMESPACE" logs deploy/ml-pipeline-ui-artifact -c ml-pipeline-ui-artifact --tail=50 || true
  exit 1
fi

# Test 2: Verify proxy can list pipelines
PIPELINES_RESPONSE=$(kubectl -n "$NAMESPACE" exec kfp-proxy-curl -- \
  curl -fsS -H 'kubeflow-userid: user@example.com' \
  "http://ml-pipeline-ui-artifact.${NAMESPACE}.svc.cluster.local/apis/v2beta1/pipelines?page_size=1")

if ! echo "$PIPELINES_RESPONSE" | grep -q '"pipelines"'; then
  echo "ERROR: Failed to list pipelines"
  echo "Response: $PIPELINES_RESPONSE"
  exit 1
fi

# Test 3: Submit workflow and fetch artifact
WORKFLOW_MANIFEST="$(cat <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-proxy-e2e-test-
spec:
  serviceAccountName: default-editor
  entrypoint: main
  podMetadata:
    annotations:
      sidecar.istio.io/inject: "false"
  templates:
  - name: main
    script:
      image: alpine:3.19
      command: [sh]
      source: |
        echo "artifact-proxy-e2e-test-content" > /tmp/out.txt
    outputs:
      artifacts:
      - name: out
        path: /tmp/out.txt
EOF
)"

WORKFLOW_NAME=$(kubectl -n "$NAMESPACE" create -f - -o name <<<$"$WORKFLOW_MANIFEST" | awk -F/ '{print $2}')
echo "Waiting for workflow $WORKFLOW_NAME to complete..."
kubectl -n "$NAMESPACE" wait --for=condition=Completed "workflow/${WORKFLOW_NAME}" --timeout=300s

WORKFLOW_PHASE=$(kubectl -n "$NAMESPACE" get workflow "$WORKFLOW_NAME" -o jsonpath='{.status.phase}')
if [[ "$WORKFLOW_PHASE" != "Succeeded" ]]; then
  echo "ERROR: Workflow phase: $WORKFLOW_PHASE"
  kubectl -n "$NAMESPACE" get workflow "$WORKFLOW_NAME" -o jsonpath='{.status.message}'
  exit 1
fi

KEY=$(kubectl -n "$NAMESPACE" get workflow "$WORKFLOW_NAME" -o jsonpath='{.status.nodes.*.outputs.artifacts[?(@.name=="out")].s3.key}')
ART_URL="http://ml-pipeline-ui-artifact.${NAMESPACE}.svc.cluster.local/artifacts/get?source=minio&bucket=mlpipeline&key=${KEY}"
if ! kubectl -n "$NAMESPACE" exec kfp-proxy-curl -- sh -c \
  "curl -fsS -H 'kubeflow-userid: user@example.com' '${ART_URL}' 2>/dev/null | grep -qx 'artifact-proxy-e2e-test-content'"; then
  echo "ERROR: Artifact content validation failed"
  kubectl -n "$NAMESPACE" logs deploy/ml-pipeline-ui-artifact -c ml-pipeline-ui-artifact --tail=50 || true
  exit 1
fi

# Cleanup
kubectl -n "$NAMESPACE" delete workflow "$WORKFLOW_NAME" --ignore-not-found=true
kubectl -n "$NAMESPACE" delete pod kfp-proxy-curl --ignore-not-found=true
