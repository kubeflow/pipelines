#!/bin/bash
set -euxo pipefail

echo "SeaweedFS Security Test - Unauthorized Access Check"
echo "Testing if one namespace can access files from another namespace"

# Check dependencies
for cmd in kubectl python3; do
    if ! command -v $cmd &> /dev/null; then
        echo "Error: $cmd is required but not installed"
        exit 1
    fi
done

# Install boto3 if not available
if ! python3 -c "import boto3" 2>/dev/null; then
    echo "Installing boto3..."
    pip3 install boto3
fi

PORT_FORWARD_PID=""
# Cleanup function
cleanup() {
    echo "Cleaning up..."
    if [ -n "$PORT_FORWARD_PID" ]; then
        kill $PORT_FORWARD_PID 2>/dev/null || true
    fi
    rm -f test-file.txt accessed-file.txt
    kubectl delete profile test-profile-1 test-profile-2 --ignore-not-found
}
trap cleanup EXIT

# Create test profiles
create_profiles() {
    echo "Creating test profiles..."

    # Create both profiles
    kubectl apply -f - <<EOF
apiVersion: kubeflow.org/v1
kind: Profile
metadata:
  name: test-profile-1
spec:
  owner:
    kind: User
    name: test-user-1@example.com
---
apiVersion: kubeflow.org/v1
kind: Profile
metadata:
  name: test-profile-2
spec:
  owner:
    kind: User
    name: test-user-2@example.com
EOF

    # Wait for namespaces
    echo "Waiting for namespaces..."
    for i in {1..6}; do
        if kubectl get namespace test-profile-1 test-profile-2 >/dev/null 2>&1; then
            echo "Namespaces created"
            return 0
        fi
        sleep 10
    done

    echo "Error: Namespaces not created"
    exit 1
}

# Wait for S3 credentials
wait_for_credentials() {
    local namespace=$1
    echo "Waiting for S3 credentials in $namespace..."

    for i in {1..6}; do
        if kubectl get secret -n $namespace mlpipeline-minio-artifact >/dev/null 2>&1; then
            echo "Credentials found"
            return 0
        fi
        sleep 10
    done

    echo "Error: No credentials found"
    return 1
}

# Get credentials for namespace
get_credentials() {
    local namespace=$1
    local access_key=$(kubectl get secret -n $namespace mlpipeline-minio-artifact -o jsonpath='{.data.accesskey}' | base64 -d)
    local secret_key=$(kubectl get secret -n $namespace mlpipeline-minio-artifact -o jsonpath='{.data.secretkey}' | base64 -d)
    echo "$access_key:$secret_key"
}

# Setup port forward to SeaweedFS
setup_port_forward() {
    if [ -n "$PORT_FORWARD_PID" ]; then
        return 0  # Already running
    fi

    echo "Setting up port-forward..."
    local pod=$(kubectl get pod -n kubeflow -l app=seaweedfs -o jsonpath='{.items[0].metadata.name}')
    kubectl port-forward -n kubeflow pod/$pod 8333:8333 >/dev/null 2>&1 &
    PORT_FORWARD_PID=$!
    sleep 3
}

# Upload test file
upload_file() {
    local namespace=$1
    echo "Uploading test file to $namespace..."

    local credentials=$(get_credentials $namespace)
    local access_key=$(echo $credentials | cut -d: -f1)
    local secret_key=$(echo $credentials | cut -d: -f2)

    setup_port_forward

    python3 tests/s3_helper_test.py upload \
        --access-key "$access_key" \
        --secret-key "$secret_key" \
        --endpoint-url "http://localhost:8333" \
        --bucket "mlpipeline" \
        --key "private-artifacts/$namespace/test-file.txt" \
        --content "Test file for $namespace"
}

# Test unauthorized access
test_unauthorized_access() {
    local from_namespace=$1
    local target_namespace=$2

    echo "Testing unauthorized access from $from_namespace to $target_namespace..."

    local credentials=$(get_credentials $from_namespace)
    local access_key=$(echo $credentials | cut -d: -f1)
    local secret_key=$(echo $credentials | cut -d: -f2)

    setup_port_forward

    # Try to access the other namespace's file
    # Note: Python script returns 0 when access is denied (good), 1 when access succeeds (bad)
    if python3 tests/s3_helper_test.py download \
        --access-key "$access_key" \
        --secret-key "$secret_key" \
        --endpoint-url "http://localhost:8333" \
        --bucket "mlpipeline" \
        --key "private-artifacts/$target_namespace/test-file.txt"; then

        echo "Security OK: Access denied as expected"
        return 0
    else
        echo "SECURITY ISSUE: Unauthorized access successful!"
        return 1
    fi
}

# Main test function
main() {
    echo "Starting security test..."

    # Create test profiles
    create_profiles

    # Wait for credentials to be created
    echo "Waiting for profile controller to create credentials..."
    sleep 30

    wait_for_credentials "test-profile-1" || {
        echo "Failed to get credentials for test-profile-1"
        exit 1
    }

    wait_for_credentials "test-profile-2" || {
        echo "Failed to get credentials for test-profile-2"
        exit 1
    }

    # Upload file to first namespace
    upload_file "test-profile-1" || {
        echo "Failed to upload file"
        exit 1
    }

    # Test unauthorized access
    if test_unauthorized_access "test-profile-2" "test-profile-1"; then
        echo "SECURITY TEST PASSED: No unauthorized access detected"
    else
        echo "SECURITY TEST FAILED: Unauthorized access detected"
        echo "This indicates a security vulnerability in the SeaweedFS setup"
        exit 1
    fi
}

main
