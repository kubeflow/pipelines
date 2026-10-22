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
"""Exercise a built cache-deployer image without a Kubernetes cluster.

Set KFP_CACHE_DEPLOYER_IMAGE to the locally built image to run the contract.
"""

import json
import os
import subprocess
import unittest


@unittest.skipUnless(
    os.environ.get("KFP_CACHE_DEPLOYER_IMAGE"),
    "Set KFP_CACHE_DEPLOYER_IMAGE to test a built image",
)
class CacheDeployerImageTest(unittest.TestCase):
    def test_no_fixable_vulnerabilities_including_packaged_kubectl(self):
        result = subprocess.run(
            [
                "trivy",
                "image",
                "--scanners",
                "vuln",
                "--format",
                "json",
                "--detection-priority",
                "comprehensive",
                "--ignorefile",
                os.devnull,
                "--config",
                os.devnull,
                "--severity",
                "UNKNOWN,LOW,MEDIUM,HIGH,CRITICAL",
                os.environ["KFP_CACHE_DEPLOYER_IMAGE"],
            ],
            check=True,
            text=True,
            capture_output=True,
            timeout=180,
        )
        results = json.loads(result.stdout)["Results"]
        self.assertTrue(
            any(
                target.get("Type") == "gobinary"
                and target["Target"].endswith("/kubectl")
                for target in results
            ),
            "Trivy must inspect kubectl's embedded Go dependencies",
        )
        fixable = [
            (vulnerability["VulnerabilityID"], vulnerability["PkgName"])
            for target in results
            for vulnerability in target.get("Vulnerabilities", [])
            if vulnerability.get("FixedVersion")
        ]
        self.assertEqual(fixable, [])

    def test_runtime_tools_and_scripts_without_unused_cloud_sdk(self):
        result = subprocess.run(
            [
                "docker",
                "run",
                "--rm",
                "--entrypoint",
                "/bin/sh",
                os.environ["KFP_CACHE_DEPLOYER_IMAGE"],
                "-ec",
                """
for tool in bash kubectl curl jq openssl; do
    command -v "$tool"
done
if command -v gcloud; then
    echo 'Unused Google Cloud SDK must not ship in the cache deployer' >&2
    exit 1
fi
kubectl version --client --output=json | jq -e '.clientVersion.gitVersion'
curl --version
openssl version
openssl genrsa -out /tmp/cache-test-key.pem 2048
openssl req -new -key /tmp/cache-test-key.pem -subj /CN=cache-test -out /tmp/cache-test.csr
openssl req -in /tmp/cache-test.csr -noout -verify
for script in deploy-cache-service.sh webhook-create-signed-cert.sh webhook-patch-ca-bundle.sh; do
    test -x "$script"
    bash -n "$script"
done
printf 'Zm9v' > /tmp/cache-test-ca
printf '%s\\n' '${CA_BUNDLE} ${NAMESPACE}' | NAMESPACE=cache-test ./webhook-patch-ca-bundle.sh --cert_input_path /tmp/cache-test-ca > /tmp/cache-test-rendered
grep -qx 'Zm9v cache-test' /tmp/cache-test-rendered
test -s /etc/ssl/certs/ca-certificates.crt
test -s /third_party/license.txt
""",
            ],
            check=False,
            text=True,
            capture_output=True,
            timeout=60,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
