#!/bin/bash
# Copyright 2026 The Kubeflow Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#     http://www.apache.org/licenses/LICENSE-2.0
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Run inside the cache-deployer image as its deployment identity (1000:0).
set -euo pipefail

fail() { echo "FAIL: $*" >&2; exit 1; }
[ "$(id -u)" = 1000 ] || fail 'expected deployment UID 1000'
[ "$(id -g)" = 0 ] || fail 'expected deployment GID 0'
for tool in bash curl jq openssl kubectl grep tr sed base64 seq mktemp; do
    command -v "$tool" >/dev/null || fail "missing $tool"
done
kubectl version --client=true >/dev/null
[ -w "$HOME" ] || fail 'HOME is not writable'
[ -w /kfp/cache/deployer ] || fail 'deployment workdir is not writable'

TEST_ROOT=$(mktemp -d)
trap 'rm -rf "$TEST_ROOT"' EXIT
export TEST_ROOT
mkdir -p "$TEST_ROOT/bin"
# The API returns a certificate to the real certificate-generation script.
openssl req -x509 -newkey rsa:2048 -nodes -keyout "$TEST_ROOT/ca.key" \
    -out "$TEST_ROOT/ca.pem" -days 1 -subj /CN=cache-test >/dev/null 2>&1
base64 < "$TEST_ROOT/ca.pem" | tr -d '\n' > "$TEST_ROOT/certificate"

cat > "$TEST_ROOT/bin/curl" <<'MOCK'
#!/bin/sh
# Exercise the existing offline fallback without fetching another executable.
exit 22
MOCK
cat > "$TEST_ROOT/bin/sleep" <<'MOCK'
#!/bin/sh
[ "$1" = infinity ] || exit 1
# A distinctive failure exits the real set -e script at its successful idle state.
echo idle >> "$TEST_ROOT/operations"
exit 73
MOCK
cat > "$TEST_ROOT/bin/kubectl" <<'MOCK'
#!/bin/sh
set -eu
echo "$*" >> "$TEST_ROOT/operations"
case "$*" in
    'version --output json')
        echo '{"serverVersion":{"major":"1","minor":"36"}}' ;;
    'api-versions') echo admissionregistration.k8s.io/v1 ;;
    'get mutatingwebhookconfigurations '* )
        if [ "$TEST_CASE" = existing ] || [ -f "$TEST_ROOT/webhook" ]; then
            echo cache-webhook-cache-test
        fi ;;
    'get secrets '* )
        if [ "$TEST_CASE" = existing ]; then echo webhook-server-tls; fi ;;
    'delete csr '* ) : ;;
    'create -f -') cat > "$TEST_ROOT/csr.yaml" ;;
    'get csr '* )
        case "$*" in
            *jsonpath*) cat "$TEST_ROOT/certificate" ;;
            *) echo cache-server.cache-test ;;
        esac ;;
    'certificate approve '* ) : ;;
    'create secret generic '* )
        # Validate the actual generated key and decoded certificate files.
        for arg in "$@"; do
            case "$arg" in
                --from-file=key.pem=*) openssl rsa -check -noout -in "${arg#--from-file=key.pem=}" >/dev/null ;;
                --from-file=cert.pem=*) openssl x509 -noout -in "${arg#--from-file=cert.pem=}" ;;
            esac
        done
        printf 'apiVersion: v1\nkind: Secret\nmetadata:\n  name: webhook-server-tls\n' ;;
    '-n cache-test apply -f -') cat > "$TEST_ROOT/secret.yaml" ;;
    'apply -f ./cache-configmap-ca-bundle.yaml --namespace cache-test')
        cp ./cache-configmap-ca-bundle.yaml "$TEST_ROOT/webhook" ;;
    *) echo "Unexpected kubectl call: $*" >&2; exit 1 ;;
esac
MOCK
chmod +x "$TEST_ROOT/bin/"*
export PATH="$TEST_ROOT/bin:$PATH"
export NAMESPACE_TO_WATCH=cache-test
cd /kfp/cache/deployer

run_case() {
    export TEST_CASE=$1
    : > "$TEST_ROOT/operations"
    local status=0
    /bin/sh ./deploy-cache-service.sh > "$TEST_ROOT/$1.log" 2>&1 || status=$?
    if [ "$status" != 73 ]; then
        cat "$TEST_ROOT/$1.log" >&2
        fail "$1 exited $status instead of reaching idle state"
    fi
    grep -qx idle "$TEST_ROOT/operations" || fail "$1 did not idle"
}

run_case fresh
grep -q 'kind: CertificateSigningRequest' "$TEST_ROOT/csr.yaml"
grep -q 'name: cache-server.cache-test' "$TEST_ROOT/csr.yaml"
grep -q 'kind: Secret' "$TEST_ROOT/secret.yaml"
grep -q 'name: cache-webhook-cache-test' "$TEST_ROOT/webhook"
grep -q 'namespace: cache-test' "$TEST_ROOT/webhook"
grep -Fq "$(cat "$TEST_ROOT/certificate")" "$TEST_ROOT/webhook"
! grep -q '\${' "$TEST_ROOT/webhook" || fail 'unexpanded webhook template'

run_case existing
if grep -Eq '^(create |apply |certificate |delete )' "$TEST_ROOT/operations"; then
    fail 'already-installed deployment mutated resources'
fi
echo 'cache-deployer image tools, fresh installation and idempotency passed'
