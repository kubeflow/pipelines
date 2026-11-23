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

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('deploy-kfp.sh')


class DeployKfpOrderTest(unittest.TestCase):
    def test_istio_upgrade_waits_for_replacement_pods(self):
        with tempfile.TemporaryDirectory() as directory:
            kubectl = Path(directory) / 'kubectl'
            kubectl.write_text(
                """#!/bin/sh
case "$*" in
  *"wait --for=condition=Ready pods"*)
    echo 'Error from server (NotFound): pod deleted during rollout' >&2
    exit 1 ;;
  *"rollout status deployment/istiod deployment/istio-ingressgateway"*)
    echo 'Istio replacement deployments available' ;;
  *"apply -f manifests/kustomize/third-party/metacontroller/base/crd.yaml"*)
    exit 42 ;;  # Stop after the readiness gate.
esac
""",
                encoding='utf-8',
            )
            kubectl.chmod(0o755)
            result = subprocess.run(
                ['bash', str(SCRIPT.resolve()), '--multi-user'],
                cwd=directory,
                env={
                    **os.environ,
                    'PATH': directory + os.pathsep + os.environ['PATH'],
                },
                capture_output=True,
                text=True,
            )
            self.assertEqual(
                result.returncode, 42, result.stdout + result.stderr
            )
            self.assertIn(
                'Istio replacement deployments available', result.stdout
            )

    def test_profile_controller_readiness_overlaps_kfp_rollout(self):
        script = SCRIPT.read_text(encoding='utf-8')

        profile_apply = script.index('Installing Profile Controller Resources')
        kfp_apply = script.index('echo "Deploying ${TEST_MANIFESTS}..."')
        kfp_ready = script.index('wait_for_pods || EXIT_CODE=$?')
        profile_ready = script.index(
            'wait_for_readiness kubeflow '
            '"app.kubernetes.io/name=profile-controller"'
        )
        iam_ready = script.index('"${C_DIR}/wait-for-seaweedfs-iam.sh"')

        self.assertLess(profile_apply, kfp_apply)
        self.assertLess(kfp_apply, kfp_ready)
        self.assertLess(kfp_ready, profile_ready)
        self.assertLess(profile_ready, iam_ready)


if __name__ == '__main__':
    unittest.main()
