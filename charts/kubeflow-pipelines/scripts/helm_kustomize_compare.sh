#!/usr/bin/env bash
# Compare Helm vs Kustomize manifests for Kubeflow Pipelines (base only)

set -euo pipefail

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
ROOT_DIR="$( cd "$SCRIPT_DIR/../../.." && pwd )"

KUSTOMIZE_PATH="${1:-}"
CHART_DIR="${2:-}"

if [[ -z "$KUSTOMIZE_PATH" || -z "$CHART_DIR" ]]; then
    echo "ERROR: Missing required arguments"
    echo "Usage: $0 <kustomize-base-path> <helm-chart-path>"
    echo ""
    echo "Example:"
    echo "  $0 ../manifests/kustomize/base charts/kubeflow-pipelines"
    exit 1
fi

echo "Comparing kubeflow-pipelines manifests (base)"

# ----------------------------
# Validate Paths
# ----------------------------

if [ ! -d "$KUSTOMIZE_PATH" ]; then
    echo "ERROR: Kustomize path does not exist: $KUSTOMIZE_PATH"
    exit 1
fi

if [ ! -d "$CHART_DIR" ]; then
    echo "ERROR: Helm chart directory does not exist: $CHART_DIR"
    exit 1
fi

if ! command -v kustomize >/dev/null 2>&1; then
    echo "ERROR: kustomize binary not found"
    exit 1
fi

if ! command -v helm >/dev/null 2>&1; then
    echo "ERROR: helm binary not found"
    exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
    echo "ERROR: python3 binary not found"
    exit 1
fi

# ----------------------------
# Temp Files
# ----------------------------

KUSTOMIZE_OUTPUT="$(mktemp)"
HELM_OUTPUT="$(mktemp)"

cleanup() {
    rm -f "$KUSTOMIZE_OUTPUT" "$HELM_OUTPUT"
}
trap cleanup EXIT

# ----------------------------
# Build Kustomize Output
# ----------------------------

cd "$ROOT_DIR"

echo "Building Kustomize output..."
kustomize build "$KUSTOMIZE_PATH" > "$KUSTOMIZE_OUTPUT"

# ----------------------------
# Render Helm Output
# ----------------------------

echo "Rendering Helm chart..."

helm template kubeflow-pipelines "$CHART_DIR" \
    --namespace kubeflow \
    --include-crds \
    > "$HELM_OUTPUT"

# ----------------------------
# Run Strict Parity Comparison
# ----------------------------

echo "Running strict parity comparison..."

python3 "$SCRIPT_DIR/helm_kustomize_compare.py" \
    "$KUSTOMIZE_OUTPUT" \
    "$HELM_OUTPUT"

COMPARISON_RESULT=$?

exit $COMPARISON_RESULT
