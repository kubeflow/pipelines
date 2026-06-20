#!/bin/bash
set -euxo pipefail

echo "SeaweedFS Security Test - Namespace Isolation + KServe HeadBucket Check"

# Requires SeaweedFS >= 4.32 (PR #9792): older builds ignore the s3:prefix
# condition on ListBucket and leak cross-namespace listings.
#
# s3_helper.py exit codes: 0 = ALLOWED, 1 = DENIED/failed.
# This script runs ALL checks and reports each, so a single run shows the full
# picture instead of bailing on the first failure.

HELPER="test/seaweedfs/s3_helper.py"
ENDPOINT="http://localhost:8333"
BUCKET="mlpipeline"
SECRET="mlpipeline-minio-artifact"

for cmd in kubectl python3; do
    command -v "$cmd" >/dev/null || { echo "Error: $cmd is required"; exit 1; }
done
python3 -c "import boto3" 2>/dev/null || pip3 install boto3

port_forward_pid=""
cleanup() {
    [ -n "$port_forward_pid" ] && kill "$port_forward_pid" 2>/dev/null || true
    kubectl delete profile test-profile-1 test-profile-2 --ignore-not-found
}
trap cleanup EXIT

create_profiles() {
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
    for _ in {1..6}; do
        kubectl get namespace test-profile-1 test-profile-2 >/dev/null 2>&1 && return 0
        sleep 10
    done
    echo "Error: namespaces not created"; exit 1
}

wait_for_credentials() {
    for _ in {1..6}; do
        kubectl get secret -n "$1" "$SECRET" >/dev/null 2>&1 && return 0
        sleep 10
    done
    echo "Error: no credentials in $1"; return 1
}

# Run the helper as the given namespace.
# Usage: run_as <namespace> <operation> [helper args...]
run_as() {
    local namespace=$1; shift
    local access_key secret_key
    access_key=$(kubectl get secret -n "$namespace" "$SECRET" -o jsonpath='{.data.accesskey}' | base64 -d)
    secret_key=$(kubectl get secret -n "$namespace" "$SECRET" -o jsonpath='{.data.secretkey}' | base64 -d)
    python3 "$HELPER" "$1" \
        --access-key "$access_key" --secret-key "$secret_key" \
        --endpoint-url "$ENDPOINT" --bucket "$BUCKET" "${@:2}"
}

start_port_forward() {
    local pod
    pod=$(kubectl get pod -n kubeflow -l app=seaweedfs -o jsonpath='{.items[0].metadata.name}')
    kubectl port-forward -n kubeflow "pod/$pod" 8333:8333 >/dev/null 2>&1 &
    port_forward_pid=$!
    sleep 3
}

failures=0

# check <description> <expected: allow|deny> <run_as args...>
check() {
    local desc=$1 expect=$2; shift 2
    local result
    if run_as "$@"; then result=allow; else result=deny; fi
    if [ "$result" = "$expect" ]; then
        echo "PASS [$result]: $desc"
    else
        echo "FAIL [got $result, wanted $expect]: $desc"
        failures=$((failures + 1))
    fi
}

main() {
    create_profiles
    echo "Waiting for profile controller to create credentials..."
    sleep 30
    wait_for_credentials test-profile-1
    wait_for_credentials test-profile-2
    start_port_forward

    # Seed one file in each namespace's own prefix (best-effort).
    run_as test-profile-1 upload --key "private-artifacts/test-profile-1/test-file.txt" --content "file for test-profile-1" || true
    run_as test-profile-2 upload --key "private-artifacts/test-profile-2/test-file.txt" --content "file for test-profile-2" || true

    # KServe support: bucket existence probe MUST be allowed.
    check "HeadBucket (KServe storage-initializer probe)" allow \
        test-profile-1 headbucket

    # Positive: a namespace MUST access its own data.
    check "own-namespace download" allow \
        test-profile-1 download --key "private-artifacts/test-profile-1/test-file.txt"
    check "own-namespace prefix list" allow \
        test-profile-1 list --prefix "private-artifacts/test-profile-1/"

    # Top-level delimited browse MUST be allowed (folder names only).
    check "top-level delimited browse" allow \
        test-profile-2 list --prefix "" --delimiter "/"

    # Negative: cross-namespace object read MUST be denied.
    check "cross-namespace download" deny \
        test-profile-2 download --key "private-artifacts/test-profile-1/test-file.txt"

    # Negative: copying FROM another namespace's private object MUST be denied
    # (CopyObject requires GetObject on the source, which is not granted cross-tenant).
    check "cross-namespace copy source" deny \
        test-profile-2 copy \
        --key "private-artifacts/test-profile-2/stolen.txt" \
        --source-key "private-artifacts/test-profile-1/test-file.txt"

    # Positive: a namespace MUST be able to delete its own objects.
    check "own-namespace delete" allow \
        test-profile-1 delete --key "private-artifacts/test-profile-1/test-file.txt"

    # Negative: deleting another namespace's private object MUST be denied.
    check "cross-namespace delete" deny \
        test-profile-2 delete --key "private-artifacts/test-profile-1/test-file.txt"

    # Negative: deep cross-namespace prefix listing MUST be denied.
    check "cross-namespace deep prefix list" deny \
        test-profile-2 list --prefix "private-artifacts/test-profile-1/"

    # Negative: flat (no-delimiter) bucket-wide listing MUST be denied,
    # since it would enumerate every namespace's object keys.
    check "flat empty-prefix list (no delimiter)" deny \
        test-profile-2 list --prefix ""
    check "flat no-prefix list (no delimiter)" deny \
        test-profile-2 list --no-prefix

    if [ "$failures" -eq 0 ]; then
        echo "ALL PASS: HeadBucket works, own access works, top-level browse works, deep/flat cross-namespace listing is blocked"
    else
        echo "$failures CHECK(S) FAILED"
        exit 1
    fi
}

main
