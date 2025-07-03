#!/bin/bash
set -euo pipefail

NAMESPACES=("istio-system" "auth" "cert-manager" "oauth2-proxy" "kubeflow" "knative-serving")
for NAMESPACE in "${NAMESPACES[@]}"; do
    if kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        PATCH_OUTPUT=$(kubectl label namespace $NAMESPACE pod-security.kubernetes.io/enforce=restricted --overwrite 2>&1)
        if echo "$PATCH_OUTPUT" | grep -q "violate the new PodSecurity"; then
            echo "WARNING: PSS violation detected for namespace $NAMESPACE"
            echo "$PATCH_OUTPUT" | grep -A 5 "violate the new PodSecurity"
        else
            echo "✅ Namespace '$NAMESPACE' labeled successfully."
        fi
    fi
done