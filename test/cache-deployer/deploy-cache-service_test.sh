#!/bin/bash
#
# Copyright 2026 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

REPO_ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
TEST_ROOT=$(mktemp -d)
trap 'rm -rf "${TEST_ROOT}"' EXIT

fail() {
    echo "FAIL: $*" >&2
    exit 1
}

write_mock_commands() {
    local mock_bin=$1
    mkdir -p "${mock_bin}"

    cat >"${mock_bin}/uname" <<'EOF'
#!/bin/sh
if [ "$1" = "-m" ]; then
    echo "${TEST_UNAME_MACHINE}"
    exit 0
fi
exec /usr/bin/uname "$@"
EOF

    cat >"${mock_bin}/jq" <<'EOF'
#!/bin/sh
cat >/dev/null
echo "1.36"
EOF

    cat >"${mock_bin}/kubectl" <<'EOF'
#!/bin/sh
if [ "$1" = "version" ] && [ "$2" = "--output" ]; then
    echo '{"serverVersion":{"major":"1","minor":"36"}}'
    exit 0
fi
echo "$@" >>"${TEST_FALLBACK_KUBECTL_LOG}"
EOF

    cat >"${mock_bin}/curl" <<'EOF'
#!/bin/sh
if [ "$1" = "-fsSL" ] && [ "$#" -eq 2 ]; then
    echo "v1.36.2"
    exit 0
fi

if [ "$1" = "-fsSL" ] && [ "$2" = "-o" ]; then
    output_path=$3
    kubectl_url=$4
    echo "${kubectl_url}" >>"${TEST_CURL_URL_LOG}"
    cat >"${output_path}" <<'DOWNLOADED_KUBECTL'
#!/bin/sh
echo "$@" >>"${TEST_DOWNLOADED_KUBECTL_LOG}"
exit "${TEST_DOWNLOADED_KUBECTL_EXIT_CODE}"
DOWNLOADED_KUBECTL
    exit 0
fi

echo "unexpected curl args: $*" >&2
exit 1
EOF

    chmod +x "${mock_bin}/uname" "${mock_bin}/jq" "${mock_bin}/kubectl" "${mock_bin}/curl"
}

run_install_test() {
    local uname_machine=$1
    local downloaded_exit_code=$2
    local expected_arch=$3
    local expect_installed=$4
    local test_dir="${TEST_ROOT}/${uname_machine}-${downloaded_exit_code}"
    local home_dir="${test_dir}/home"
    local mock_bin="${test_dir}/mock-bin"

    mkdir -p "${home_dir}"
    write_mock_commands "${mock_bin}"

    export DEPLOY_CACHE_SERVICE_LIBRARY_MODE=true
    export HOME="${home_dir}"
    export PATH="${mock_bin}:/usr/bin:/bin"
    export TEST_UNAME_MACHINE="${uname_machine}"
    export TEST_DOWNLOADED_KUBECTL_EXIT_CODE="${downloaded_exit_code}"
    export TEST_CURL_URL_LOG="${test_dir}/curl-url.log"
    export TEST_DOWNLOADED_KUBECTL_LOG="${test_dir}/downloaded-kubectl.log"
    export TEST_FALLBACK_KUBECTL_LOG="${test_dir}/fallback-kubectl.log"

    # shellcheck source=/dev/null
    source "${REPO_ROOT}/backend/src/cache/deployer/deploy-cache-service.sh"
    install_version_matched_kubectl

    grep -q "/linux/${expected_arch}/kubectl" "${TEST_CURL_URL_LOG}" ||
        fail "expected kubectl URL to use linux/${expected_arch}"
    grep -q "version --client=true" "${TEST_DOWNLOADED_KUBECTL_LOG}" ||
        fail "expected downloaded kubectl to be validated before install"

    if [ "${expect_installed}" = "true" ]; then
        [ -x "${home_dir}/bin/kubectl" ] || fail "expected kubectl to be installed"
    else
        [ ! -e "${home_dir}/bin/kubectl" ] ||
            fail "expected invalid downloaded kubectl to be removed instead of installed"
    fi
}

run_install_test "aarch64" "0" "arm64" "true"
run_install_test "aarch64" "126" "arm64" "false"
run_install_test "x86_64" "0" "amd64" "true"

echo "deploy-cache-service kubectl install tests passed"
