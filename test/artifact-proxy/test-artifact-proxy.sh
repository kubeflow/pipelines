#!/usr/bin/env bash
set -euxo pipefail

# Artifact Proxy E2E Test
# Validates the user-facing artifact path through the authenticated shared UI.
# Direct tenant access to receiver pods is intentionally denied by NetworkPolicy.

NAMESPACE="${1:-${USER_NAMESPACE:-kubeflow-user-example-com}}"
UI_URL="http://ml-pipeline-ui.kubeflow.svc.cluster.local"
WORKFLOW_NAME=""

cleanup() {
  local result=$?
  trap - EXIT
  kubectl --request-timeout=30s -n "$NAMESPACE" delete \
    pod/kfp-proxy-curl role/kfp-artifact-smoke-reader \
    rolebinding/kfp-artifact-smoke-reader --ignore-not-found=true --wait=false || result=1
  if [[ -n "$WORKFLOW_NAME" ]]; then
    kubectl --request-timeout=30s -n "$NAMESPACE" delete "workflow/$WORKFLOW_NAME" \
      --ignore-not-found=true --wait=false || result=1
  fi
  exit "$result"
}
trap cleanup EXIT

# This CI installation does not install the kubeflow-admin role referenced by
# Profile owners. Grant only the artifact namespace check used by the shared UI;
# the fixture user still cannot read another namespace or bypass object storage.
kubectl -n "$NAMESPACE" apply -f - <<'EOF'
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: kfp-artifact-smoke-reader
rules:
- apiGroups: [kubeflow.org]
  resources: [viewers]
  verbs: [get]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: kfp-artifact-smoke-reader
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kfp-artifact-smoke-reader
subjects:
- apiGroup: rbac.authorization.k8s.io
  kind: User
  name: user@example.com
EOF

# Create curl pod for testing
kubectl -n "$NAMESPACE" run kfp-proxy-curl --image=curlimages/curl:8.7.1 --restart=Never \
  --annotations="sidecar.istio.io/inject=false" --command -- sleep 3600
kubectl -n "$NAMESPACE" wait --for=condition=Ready pod/kfp-proxy-curl --timeout=300s

# Test 1: Verify shared UI and API server health
HEALTH_RESPONSE=""
for attempt in $(seq 1 30); do
  if HEALTH_RESPONSE=$(kubectl -n "$NAMESPACE" exec kfp-proxy-curl -- \
    curl --connect-timeout 5 --max-time 30 -fsS -H 'kubeflow-userid: user@example.com' \
    "${UI_URL}/apis/v2beta1/healthz") && \
    jq -e '.apiServerReady == true' >/dev/null <<<"$HEALTH_RESPONSE"; then
    break
  fi

  if [[ "$attempt" -eq 30 ]]; then
    echo "ERROR: Artifact proxy API server was not ready after 30 attempts"
    echo "Response: $HEALTH_RESPONSE"
    kubectl -n "$NAMESPACE" logs deploy/ml-pipeline-ui-artifact -c ml-pipeline-ui-artifact --tail=50 || true
    exit 1
  fi

  echo "Waiting for artifact proxy API server to become ready... ($attempt/30)"
  sleep 5
done

# Wait for the profile controller to observe the protected receiver rollout.
for attempt in $(seq 1 60); do
  state=$(kubectl -n kubeflow get secret "kfp-artifact-storage-${NAMESPACE}" \
    -o jsonpath='{.data.authorization\.json}' 2>/dev/null | base64 -d | jq -r .state || true)
  if [[ "$state" == "ready" ]]; then
    break
  fi
  if [[ "$attempt" -eq 60 ]]; then
    echo "ERROR: Artifact storage authorization did not become ready (state=${state:-absent})"
    kubectl -n "$NAMESPACE" get deployments,pods,networkpolicies
    exit 1
  fi
  sleep 5
done

# Test 2: Submit workflow and fetch artifact
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

KEY=$(
  kubectl -n "$NAMESPACE" get workflow "$WORKFLOW_NAME" -o json |
    jq -r '[.status.nodes[].outputs.artifacts[]? | select(.name=="out") | .s3.key] | first // empty'
)
if [[ -z "$KEY" ]]; then
  echo "ERROR: Could not locate S3 key for artifact 'out' in workflow status"
  exit 1
fi
ARTIFACT_URL="${UI_URL}/artifacts/get?source=minio&bucket=mlpipeline&namespace=${NAMESPACE}&key=${KEY}"
if ! kubectl -n "$NAMESPACE" exec kfp-proxy-curl -- sh -c \
  "curl --connect-timeout 5 --max-time 30 -fsS -H 'kubeflow-userid: user@example.com' '${ARTIFACT_URL}' 2>/dev/null | grep -qx 'artifact-proxy-e2e-test-content'"; then
  echo "ERROR: Artifact content validation failed"
  kubectl -n "$NAMESPACE" logs deploy/ml-pipeline-ui-artifact -c ml-pipeline-ui-artifact --tail=50 || true
  exit 1
fi
