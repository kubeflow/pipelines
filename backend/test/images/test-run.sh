#!/bin/bash

set -o allexport
source .env
source "./patch-csv.sh"
set +o allexport

export DEPLOYMENT_NAME="ds-pipeline-$DSPA_NAME"

SKIP_DEPLOYMENT="${SKIP_DEPLOYMENT:-true}"
RANDOM_SUFFIX=$(head /dev/urandom | tr -dc a-z0-9 | head -c 8)
export NAMESPACE="$NAMESPACE-$RANDOM_SUFFIX"

TEST_LABEL="smoke"
DSPO_OWNER="opendatahub-io"
DSPO_REPO="data-science-pipelines-operator"
DSPO_BRANCH="stable"
DSP_TAG="stable"
DEPLOY_DSPO="true"
NUM_PARALLEL_NODES=10

while [ "$#" -gt 0 ]; do
  case "$1" in
    --label-filter=*)
      TEST_LABEL="${1#*=}"
      shift
      ;;
    --dsp-tag=*)
      DSP_TAG="${1#*=}"
      shift
      ;;
    --dspo-owner=*)
      DSPO_OWNER="${1#*=}"
      shift
      ;;
    --dspo-repo=*)
      DSPO_REPO="${1#*=}"
      shift
      ;;
    --dspo-branch=*)
      DSPO_BRANCH="${1#*=}"
      shift
      ;;
    --deploy-dspo=*)
      DEPLOY_DSPO="${1#*=}"
      shift
      ;;
    --num-parallel-tests=*)
      NUM_PARALLEL_NODES="${1#*=}"
      shift
      ;;
  esac
done

# 1. Create a temporary file to store dspa config
dspa_deployment=$(mktemp)
dspa_role_binding=$(mktemp)

# Check if AWS credentials are provided to determine storage type
if [[ -n "$AWS_ACCESS_KEY_ID" && -n "$AWS_SECRET_ACCESS_KEY" ]]; then
  STORAGE_TYPE="s3"
  echo "Using external S3 storage (AWS credentials detected)"
else
  STORAGE_TYPE="minio"
  echo "Using embedded Minio storage (no AWS credentials provided)"
fi

# Create DSPA CR template
cat <<EOF >> "$dspa_deployment"
apiVersion: datasciencepipelinesapplications.opendatahub.io/v1
kind: DataSciencePipelinesApplication
metadata:
  name: $DSPA_NAME
  namespace: $NAMESPACE
spec:
  dspVersion: v2
EOF

if [ "$SKIP_DEPLOYMENT" = "false" ]; then
  cat <<EOF >> "$dspa_deployment"
  apiServer:
    image: "quay.io/opendatahub/ds-pipelines-api-server:$DSP_TAG"
    argoDriverImage: "quay.io/opendatahub/ds-pipelines-driver:$DSP_TAG"
    argoLauncherImage: "quay.io/opendatahub/ds-pipelines-launcher:$DSP_TAG"
    cacheEnabled: true
    enableSamplePipeline: false
  persistenceAgent:
    image: "quay.io/opendatahub/ds-pipelines-persistenceagent:$DSP_TAG"
  scheduledWorkflow:
    image: "quay.io/opendatahub/ds-pipelines-scheduledworkflow:$DSP_TAG"
EOF
else
  cat <<EOF >> "$dspa_deployment"
  apiServer:
    cacheEnabled: true
    enableSamplePipeline: false
EOF
fi

# Add storage-specific configuration
if [ "$STORAGE_TYPE" = "s3" ]; then
  cat <<EOF >> "$dspa_deployment"
  objectStorage:
    externalStorage:
      bucket: $BUCKET
      host: $AWS_ENDPOINT
      region: $REGION
      s3CredentialsSecret:
        accessKey: AWS_ACCESS_KEY
        secretKey: AWS_SECRET_ACCESS_KEY
        secretName: $SECRET_NAME
      scheme: https
  podToPodTLS: true
EOF
else
  cat <<EOF >> "$dspa_deployment"
  objectStorage:
    enableExternalRoute: true
    minio:
      deploy: true
      image: 'quay.io/opendatahub/minio:RELEASE.2019-08-14T20-37-41Z-license-compliance'
  podToPodTLS: true
EOF
fi

cat <<EOF >> "$dspa_role_binding"
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: dashboard-permissions-$DSPA_NAME
  namespace: $NAMESPACE
subjects:
  - kind: ServiceAccount
    name: $DEPLOYMENT_NAME
    namespace: $NAMESPACE
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admin
EOF

#################### PATCH CSV #######################
if [ "$DEPLOY_DSPO" == "true" ] && [ "$SKIP_DEPLOYMENT" = "false" ]; then
  find_csv_and_update "$DSPO_OWNER" "$DSPO_REPO" "$DSPO_BRANCH"
fi

#################### DEPLOY DSPA #######################

# Create Namespace
echo "Creating namespace: $NAMESPACE"
oc create namespace "$NAMESPACE"

# Define cleanup function
cleanup() {
  echo "Cleaning up resources..."
  if [ -n "$NAMESPACE" ] && oc get namespace "$NAMESPACE" &> /dev/null; then
    echo "Deleting namespace: $NAMESPACE"
    oc delete namespace "$NAMESPACE"
  fi
  # Clean up temporary files
  if [ -n "$dspa_deployment" ] && [ -f "$dspa_deployment" ]; then
    rm -f "$dspa_deployment"
  fi
  if [ -n "$dspa_role_binding" ] && [ -f "$dspa_role_binding" ]; then
    rm -f "$dspa_role_binding"
  fi
}

# Set trap to run cleanup on script exit (success or failure)
trap cleanup EXIT

# Create AWS Secret (only for S3 storage)
if [ "$STORAGE_TYPE" = "s3" ]; then
  echo "Creating AWS credentials secret for S3 storage"
  oc -n "$NAMESPACE" create secret generic "$SECRET_NAME" \
    --from-literal=AWS_ACCESS_KEY="$AWS_ACCESS_KEY_ID" \
    --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY"
else
  echo "Skipping AWS secret creation (using Minio storage)"
fi

# Create DSPA deployment
echo "Creating DSPA deployment"
cat "$dspa_deployment"
oc apply -n "$NAMESPACE" -f "$dspa_deployment"
timeout 1m bash -c \
  "until oc -n $NAMESPACE get deployment $DEPLOYMENT_NAME &> /dev/null; do echo 'Waiting for the deployment $DEPLOYMENT_NAME...'; sleep 10; done"
if oc wait --for=condition=available deployment/ds-pipeline-"$DSPA_NAME" --timeout=10m -n "$NAMESPACE"; then
  echo "Operator pod is ready"
else
  echo "Warning: Operator pod did not become ready within timeout, continuing anyway..."
  exit 1
fi

# Create Role Binding
echo "Create role binding to allow service account access to DSPA API"
oc apply -f "$dspa_role_binding" -n "$NAMESPACE"

# Get API URL
echo "Fetching route to $DEPLOYMENT_NAME"
timeout 1m bash -c \
  "until oc -n $NAMESPACE get route "$DEPLOYMENT_NAME" &> /dev/null; do echo 'Waiting for the route to $DEPLOYMENT_NAME...'; sleep 10; done"
export API_URL="https://$(oc -n $NAMESPACE get route "$DEPLOYMENT_NAME" -o jsonpath={.spec.host})"

# Get API Token
echo "Generate Token for $DEPLOYMENT_NAME"
export API_TOKEN=$(oc create token "$DEPLOYMENT_NAME" --namespace "$NAMESPACE" --duration=60m)


#################### TESTS #######################
# Run Tests
# Traverse up directories until we find a directory containing $TEST_DIRECTORY
current_dir="$(pwd)"
found_test_dir=""
while [ "$current_dir" != "/" ]; do
  if [ -d "$current_dir/$TEST_DIRECTORY" ]; then
    found_test_dir="$current_dir/$TEST_DIRECTORY"
    break
  fi
  current_dir="$(dirname "$current_dir")"
done

if [ -n "$found_test_dir" ]; then
  echo "Found test directory at: $found_test_dir"
  cd "$found_test_dir" || exit
else
  echo "Error: Could not find $TEST_DIRECTORY in any parent directory"
  exit 1
fi
echo "Running Tests now..."
go run github.com/onsi/ginkgo/v2/ginkgo -r -v -p \
  --nodes=$NUM_PARALLEL_NODES \
  --keep-going \
  --label-filter="$TEST_LABEL" \
  -- -namespace="$NAMESPACE" \
  -apiUrl="$API_URL" \
  -authToken="$API_TOKEN" \
  -disableTlsCheck=true \
  -serviceAccountName=pipeline-runner-"$DSPA_NAME" \
  -baseImage="registry.redhat.io/ubi9/python-312@sha256:e80ff3673c95b91f0dafdbe97afb261eab8244d7fd8b47e20ffcbcfee27fb168"
