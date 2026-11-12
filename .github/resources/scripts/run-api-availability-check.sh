#!/usr/bin/env bash
set -euo pipefail

# Run only after the ordinary API tests on this job's disposable multi-user
# Kind cluster. Both clients run in tenant namespaces, not via port-forward.
readonly BENIGN_NAMESPACE=kubeflow-user-example-com
readonly ATTACK_NAMESPACE=kfp-audit-attacker
readonly CLIENT_CONFIGMAP=kfp-availability-client
readonly CLIENT_SOURCE=.github/resources/scripts/api-availability-client.py
readonly SUMMARY_FILE="${GITHUB_STEP_SUMMARY:-/dev/null}"

wait_for_namespace() {
  local namespace="$1"
  for _ in $(seq 1 60); do
    if kubectl get namespace "$namespace" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "Profile namespace $namespace was not reconciled" >&2
  return 1
}

api_snapshot() {
  kubectl get pods -n kubeflow -l app=ml-pipeline -o json |
    python3 -c 'import json,sys; pods=json.load(sys.stdin)["items"]; print("|".join(sorted("{}:{}".format(p["metadata"]["uid"], next((c["restartCount"] for c in p["status"].get("containerStatuses", []) if c["name"] == "ml-pipeline-api-server"), -1)) for p in pods)))'
}

apply_client_pod() {
  local name="$1" namespace="$2" service_account="$3" mode="$4"
  local pod_command
  if [[ "$mode" == hold ]]; then
    pod_command='["python3", "-c", "import time; time.sleep(180)"]'
  else
    pod_command="[\"python3\", \"/client/api-availability-client.py\", \"${mode}\"]"
  fi
  kubectl apply -f - <<EOF
apiVersion: v1
kind: Pod
metadata:
  name: ${name}
  namespace: ${namespace}
  annotations:
    sidecar.istio.io/inject: "false"
spec:
  restartPolicy: Never
  activeDeadlineSeconds: 240
  serviceAccountName: ${service_account}
  automountServiceAccountToken: false
  securityContext:
    runAsNonRoot: true
    runAsUser: 65534
    runAsGroup: 65534
    fsGroup: 65534
    seccompProfile:
      type: RuntimeDefault
  containers:
  - name: client
    image: python:3.11.18-alpine3.23
    imagePullPolicy: IfNotPresent
    command: ${pod_command}
    env:
    - name: PYTHONDONTWRITEBYTECODE
      value: "1"
    resources:
      requests:
        cpu: 50m
        memory: 64Mi
      limits:
        cpu: "1"
        memory: 512Mi
        ephemeral-storage: 64Mi
    securityContext:
      allowPrivilegeEscalation: false
      readOnlyRootFilesystem: true
      capabilities:
        drop: ["ALL"]
    volumeMounts:
    - name: client
      mountPath: /client
      readOnly: true
    - name: scratch
      mountPath: /scratch
    - name: audience-token
      mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      readOnly: true
  volumes:
  - name: client
    configMap:
      name: ${CLIENT_CONFIGMAP}
      defaultMode: 0444
  - name: scratch
    emptyDir:
      medium: Memory
      sizeLimit: 48Mi
  - name: audience-token
    projected:
      sources:
      - serviceAccountToken:
          path: token
          audience: pipelines.kubeflow.org
          expirationSeconds: 3600
EOF
}

wait_for_running_pod() {
  kubectl wait -n "$1" --for=condition=Ready "pod/$2" --timeout=90s
}

wait_for_ready_baseline() {
  local pod="$1"
  for _ in $(seq 1 45); do
    if kubectl logs -n "$BENIGN_NAMESPACE" "$pod" -c client 2>/dev/null |
      grep -q '^READY '; then
      return 0
    fi
    if [[ "$(kubectl get pod -n "$BENIGN_NAMESPACE" "$pod" -o jsonpath='{.status.phase}' 2>/dev/null || true)" == Failed ]]; then
      break
    fi
    sleep 1
  done
  kubectl logs -n "$BENIGN_NAMESPACE" "$pod" -c client 2>/dev/null || true
  echo "Authenticated benign baseline did not become ready" >&2
  return 1
}

wait_for_pod_success() {
  local namespace="$1" pod="$2"
  if ! kubectl wait -n "$namespace" --for=jsonpath='{.status.phase}'=Succeeded "pod/$pod" --timeout=100s; then
    kubectl get pod -n "$namespace" "$pod" -o wide || true
    kubectl logs -n "$namespace" "$pod" -c client 2>/dev/null || true
    return 1
  fi
}

kubectl apply -f - <<EOF
apiVersion: kubeflow.org/v1beta1
kind: Profile
metadata:
  name: ${ATTACK_NAMESPACE}
spec:
  owner:
    kind: User
    name: security-ci@example.invalid
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: allow-kfp-audit-attacker
  namespace: kubeflow
spec:
  podSelector:
    matchLabels:
      app: ml-pipeline
  policyTypes: [Ingress]
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          kubernetes.io/metadata.name: ${ATTACK_NAMESPACE}
    ports:
    - protocol: TCP
      port: 8888
EOF
wait_for_namespace "$ATTACK_NAMESPACE"
kubectl get serviceaccount default-editor -n "$BENIGN_NAMESPACE" >/dev/null

for namespace in "$BENIGN_NAMESPACE" "$ATTACK_NAMESPACE"; do
  kubectl create configmap "$CLIENT_CONFIGMAP" -n "$namespace" \
    --from-file=api-availability-client.py="$CLIENT_SOURCE" \
    --dry-run=client -o yaml | kubectl apply -f -
done

printf '### Bounded API availability checks\n\n' >> "$SUMMARY_FILE"
for case in report multipart zip; do
  benign_pod="audit-benign-${case}"
  attack_pod="audit-attack-${case}"
  apply_client_pod "$attack_pod" "$ATTACK_NAMESPACE" default hold
  wait_for_running_pod "$ATTACK_NAMESPACE" "$attack_pod"
  if [[ "$case" == zip ]]; then
    kubectl exec -n "$ATTACK_NAMESPACE" "$attack_pod" -c client -- \
      python3 /client/api-availability-client.py prepare-zip
  fi
  apply_client_pod "$benign_pod" "$BENIGN_NAMESPACE" default-editor benign
  wait_for_ready_baseline "$benign_pod"
  before="$(api_snapshot)"
  if [[ -z "$before" ]]; then
    echo "No KFP API pod found before $case" >&2
    exit 1
  fi
  if [[ "$(kubectl get pod -n "$BENIGN_NAMESPACE" "$benign_pod" -o jsonpath='{.status.phase}')" != Running ]]; then
    echo "Benign client finished before the $case request began" >&2
    exit 1
  fi
  if ! attack_result="$(kubectl exec -n "$ATTACK_NAMESPACE" "$attack_pod" -c client -- \
    python3 /client/api-availability-client.py "$case")"; then
    echo "Attack request failed or did not reach a valid HTTP boundary: $attack_result" >&2
    exit 1
  fi
  if [[ "$(kubectl get pod -n "$BENIGN_NAMESPACE" "$benign_pod" -o jsonpath='{.status.phase}')" != Running ]]; then
    echo "Benign client ended before the $case request finished; overlap is inconclusive" >&2
    exit 1
  fi
  wait_for_pod_success "$BENIGN_NAMESPACE" "$benign_pod"
  after="$(api_snapshot)"
  benign_log="$(kubectl logs -n "$BENIGN_NAMESPACE" "$benign_pod" -c client)"
  if ! ATTACK_LOG="$attack_result" BENIGN_LOG="$benign_log" python3 - <<'PY'
import os
import re
import sys

attack = os.environ["ATTACK_LOG"]
benign = os.environ["BENIGN_LOG"]
start = re.search(r"^HTTP_SEND_START ms=(\d+)$", attack, re.MULTILINE)
end = re.search(r"^HTTP_SEND_END ms=(\d+)$", attack, re.MULTILINE)
if not start or not end:
    print("Missing attack HTTP interval", file=sys.stderr)
    sys.exit(1)
start_ms, end_ms = int(start.group(1)), int(end.group(1))
samples = re.findall(r"^SAMPLE start_ms=(\d+) end_ms=(\d+)$", benign, re.MULTILINE)
if not any(int(a) <= end_ms and int(b) >= start_ms for a, b in samples):
    print("No benign ListPipelines request overlapped the attack HTTP interval", file=sys.stderr)
    sys.exit(1)
PY
  then
    echo "$attack_result" >&2
    echo "$benign_log" | tail -5 >&2
    exit 1
  fi
  benign_result="$(printf '%s\n' "$benign_log" | tail -1)"
  printf '%s: %s; %s\n' "$case" "$attack_result" "$benign_result"
  printf -- '- `%s`: %s; %s\n' "$case" "$attack_result" "$benign_result" >> "$SUMMARY_FILE"
  if [[ "$before" != "$after" ]]; then
    echo "KFP API pod identity or restart count changed during $case: $before -> $after" >&2
    exit 1
  fi
  kubectl delete pod -n "$ATTACK_NAMESPACE" "$attack_pod" --wait=false >/dev/null
done
