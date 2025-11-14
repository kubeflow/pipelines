#!/bin/bash

set -o allexport
source .env
set +o allexport

export DEPLOYMENT_NAME="ds-pipeline-$DSPA_NAME"

# 1. Create a temporary file to store dspa config
dspa_deployment=$(mktemp)
dspa_role_binding=$(mktemp)

# Create DSPA CR template
cat <<EOF >> "$dspa_deployment"
apiVersion: datasciencepipelinesapplications.opendatahub.io/v1
kind: DataSciencePipelinesApplication
metadata:
  name: $DSPA_NAME
  namespace: $NAMESPACE
spec:
  dspVersion: v2
  apiServer:
    cacheEnabled: true
    enableSamplePipeline: false
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

# Create Namespace
oc create namespace $NAMESPACE

# Create AWS Secret
oc -n "$NAMESPACE" create secret generic "$SECRET_NAME" \
  --from-literal=AWS_ACCESS_KEY="$AWS_ACCESS_KEY_ID" \
  --from-literal=AWS_SECRET_ACCESS_KEY="$AWS_SECRET_ACCESS_KEY"

# Create DSPA deployment
oc apply -n $NAMESPACE -f "$dspa_deployment"
timeout 1m bash -c \
  "until oc -n $NAMESPACE get deployment $DEPLOYMENT_NAME &> /dev/null; do echo 'Waiting for the deployment $DEPLOYMENT_NAME...'; sleep 10; done"
oc wait --for=condition=available deployment/ds-pipeline-$DSPA_NAME --timeout=10m -n $NAMESPACE

# Create Role Binding
echo "Create role binding to allow service account access to DSPA API"
oc apply -f $dspa_role_binding -n $NAMESPACE

# Get API URL
echo "Fetching route to $DEPLOYMENT_NAME"
timeout 1m bash -c \
  "until oc -n $NAMESPACE get route "$DEPLOYMENT_NAME" &> /dev/null; do echo 'Waiting for the route to $DEPLOYMENT_NAME...'; sleep 10; done"
export API_URL="https://$(oc -n $NAMESPACE get route "$DEPLOYMENT_NAME" -o jsonpath={.spec.host})"

# Get API Token
echo "Generate Token for $DEPLOYMENT_NAME"
export API_TOKEN=$(oc create token "$DEPLOYMENT_NAME" --namespace "$NAMESPACE" --duration=60m)

# Run Tests
cd $TEST_DIRECTORY
echo "Running Tests now..."
go run github.com/onsi/ginkgo/v2/ginkgo -r -v -p \
  --nodes=10 \
  --keep-going "$@" \
  -- -namespace=$NAMESPACE \
  -apiUrl=$API_URL \
  -authToken="$API_TOKEN" \
  -disableTlsCheck=true \
  -serviceAccountName=pipeline-runner-$DSPA_NAME \
  -repoName="opendatahub-io/data-science-pipelines" \
  -baseImage="registry.redhat.io/ubi9/python-312@sha256:e80ff3673c95b91f0dafdbe97afb261eab8244d7fd8b47e20ffcbcfee27fb168"

# Cleanup
echo "Cleaning up the namespace"
oc delete namespace "$NAMESPACE"