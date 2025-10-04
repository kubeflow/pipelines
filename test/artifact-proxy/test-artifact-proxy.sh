set -euo pipefail

# Artifact Proxy E2E Test
# Validates artifact proxy functionality in multi-user mode

NS="${1:-${USER_NAMESPACE:-kubeflow-user-example-com}}"

# Ensure the artifact proxy is deployed and ready
echo "Checking artifact proxy deployment in namespace $NS..."
if ! kubectl get deployment -n "$NS" ml-pipeline-ui-artifact &>/dev/null; then
  echo "ERROR: Artifact proxy deployment not found in namespace $NS"
  exit 1
fi

echo "Waiting for artifact proxy to become ready..."
kubectl -n "$NS" rollout status deploy/ml-pipeline-ui-artifact --timeout=180s || {
  echo "ERROR: Artifact proxy failed to become ready"
  exit 1
}

# Create curl pod for testing
if ! kubectl -n "$NS" get pod kfp-proxy-curl >/dev/null 2>&1; then
  kubectl -n "$NS" run kfp-proxy-curl --image=curlimages/curl:8.7.1 --restart=Never --command -- sleep 3600
fi
kubectl -n "$NS" wait --for=condition=Ready pod/kfp-proxy-curl --timeout=120s

# Test 1: Verify artifact proxy health endpoint
echo "Test 1: Checking artifact proxy health..."
HEALTH_RESPONSE=$(kubectl -n "$NS" exec kfp-proxy-curl -- \
  curl -fsS -H 'kubeflow-userid: user@example.com' \
  "http://ml-pipeline-ui-artifact.${NS}.svc.cluster.local/apis/v1beta1/healthz")

if ! echo "$HEALTH_RESPONSE" | grep -q '"apiServerReady":true'; then
  echo "ERROR: apiServerReady=false"
  echo "Response: $HEALTH_RESPONSE"
  kubectl -n "$NS" logs deploy/ml-pipeline-ui-artifact -c ml-pipeline-ui-artifact --tail=50 || true
  exit 1
fi
echo "Test 1 PASSED"

# Test 2: Verify proxy can list pipelines
echo "Test 2: Listing pipelines via proxy..."
PIPELINES_RESPONSE=$(kubectl -n "$NS" exec kfp-proxy-curl -- \
  curl -fsS -H 'kubeflow-userid: user@example.com' \
  "http://ml-pipeline-ui-artifact.${NS}.svc.cluster.local/apis/v2beta1/pipelines?page_size=1")

if ! echo "$PIPELINES_RESPONSE" | grep -q '"pipelines"'; then
  echo "ERROR: Failed to list pipelines"
  echo "Response: $PIPELINES_RESPONSE"
  exit 1
fi
echo "Test 2 PASSED"

# Test 3: Submit workflow and fetch artifact
echo "Test 3: Submitting workflow..."
WF_MANIFEST="$(cat <<'EOF'
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: artifact-proxy-e2e-test-
spec:
  serviceAccountName: default-editor
  entrypoint: main
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

WF_NAME=$(kubectl -n "$NS" create -f - -o name <<<$"$WF_MANIFEST" | awk -F/ '{print $2}')
echo "Waiting for workflow $WF_NAME to complete..."
kubectl -n "$NS" wait --for=condition=Completed "wf/${WF_NAME}" --timeout=300s

# Check workflow succeeded
WF_PHASE=$(kubectl -n "$NS" get wf "$WF_NAME" -o jsonpath='{.status.phase}')
if [[ "$WF_PHASE" != "Succeeded" ]]; then
  echo "ERROR: Workflow phase: $WF_PHASE"
  kubectl -n "$NS" get wf "$WF_NAME" -o jsonpath='{.status.message}'
  exit 1
fi

# Get artifact key from workflow
KEY=$(kubectl -n "$NS" get wf "$WF_NAME" -o jsonpath='{.status.nodes.*.outputs.artifacts[?(@.name=="out")].s3.key}')

echo "Fetching artifact via proxy..."
ART_URL="http://ml-pipeline-ui-artifact.${NS}.svc.cluster.local/artifacts/get?source=minio&bucket=mlpipeline&key=${KEY}"

if ! kubectl -n "$NS" exec kfp-proxy-curl -- sh -c \
  "curl -fsS -H 'kubeflow-userid: user@example.com' '${ART_URL}' 2>/dev/null | grep -qx 'artifact-proxy-e2e-test-content'"; then
  echo "ERROR: Artifact content validation failed"
  kubectl -n "$NS" logs deploy/ml-pipeline-ui-artifact -c ml-pipeline-ui-artifact --tail=50 || true
  exit 1
fi
echo "Test 3 PASSED"

# Cleanup
kubectl -n "$NS" delete wf "$WF_NAME" --ignore-not-found=true
kubectl -n "$NS" delete pod kfp-proxy-curl --ignore-not-found=true

echo ""
echo "All artifact proxy tests passed"
