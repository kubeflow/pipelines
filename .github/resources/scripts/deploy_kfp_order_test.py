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
import unittest

SCRIPT = Path(__file__).with_name('deploy-kfp.sh')


class DeployKfpOrderTest(unittest.TestCase):

    def test_profile_controller_readiness_overlaps_kfp_rollout(self):
        script = SCRIPT.read_text(encoding='utf-8')

        profile_apply = script.index('Installing Profile Controller Resources')
        kfp_apply = script.index('echo "Deploying ${TEST_MANIFESTS}..."')
        kfp_ready = script.index('wait_for_pods || EXIT_CODE=$?')
        profile_ready = script.index(
            'wait_for_pods_ready kubeflow '
            '"app.kubernetes.io/name=profile-controller"')
        iam_ready = script.index('"${C_DIR}/wait-for-seaweedfs-iam.sh"')

        self.assertLess(profile_apply, kfp_apply)
        self.assertLess(kfp_apply, kfp_ready)
        self.assertLess(kfp_ready, profile_ready)
        self.assertLess(profile_ready, iam_ready)


if __name__ == '__main__':
    unittest.main()
