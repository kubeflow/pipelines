#!/usr/bin/env python3
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

from pathlib import Path
import subprocess
import textwrap
import unittest


SCRIPT = (
    Path(__file__).parents[3]
    / 'test'
    / 'seaweedfs'
    / 'namespace_isolation_test.sh'
)


class NamespaceIsolationCredentialWaitTest(unittest.TestCase):

    def _run(self, shell_body: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [
                'bash',
                '-c',
                textwrap.dedent(f'''\
                    source "{SCRIPT}"
                    {shell_body}
                '''),
            ],
            capture_output=True,
            text=True,
            check=False,
        )

    def test_waits_for_both_profile_credentials(self):
        result = self._run('''
            kubectl() {
              if [[ "$*" == "get secret -n test-profile-1 mlpipeline-minio-artifact" ||
                    "$*" == "get secret -n test-profile-2 mlpipeline-minio-artifact" ]]; then
                return 0
              fi
              return 1
            }
            wait_for_credentials test-profile-1 test-profile-2
        ''')

        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn(
            'Credentials found in all test Profile namespaces.', result.stdout
        )

    def test_timeout_reports_each_missing_profile_and_controller(self):
        result = self._run('''
            CREDENTIAL_TIMEOUT_SECONDS=0
            kubectl() {
              echo "kubectl $*"
              return 1
            }
            wait_for_credentials test-profile-1 test-profile-2
        ''')

        self.assertEqual(result.returncode, 1)
        self.assertIn(
            'credentials were not created within 0s', result.stdout
        )
        self.assertIn(
            'credential reconciliation diagnostics for test-profile-1',
            result.stdout,
        )
        self.assertIn(
            'credential reconciliation diagnostics for test-profile-2',
            result.stdout,
        )
        self.assertIn(
            'deployment/kubeflow-pipelines-profile-controller', result.stdout
        )


if __name__ == '__main__':
    unittest.main()
