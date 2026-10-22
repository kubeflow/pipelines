#!/usr/bin/env python3
# Copyright 2019 The Kubeflow Authors
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
"""Opt-in runtime checks for the built inverse-proxy image.

KFP_INVERSE_PROXY_IMAGE=<tag> python -m unittest discover -s proxy \
    -p docker_image_test.py
"""

import os
import subprocess
import unittest


@unittest.skipUnless(
    os.environ.get('KFP_INVERSE_PROXY_IMAGE'),
    'Set KFP_INVERSE_PROXY_IMAGE to a locally built image')
class ProxyImageTest(unittest.TestCase):

    def setUp(self):
        self.image = os.environ['KFP_INVERSE_PROXY_IMAGE']

    def test_registration_runtime_dependencies_and_defaults(self):
        script = '''set -eu
/opt/bin/proxy-forwarding-agent --help >/dev/null 2>&1
gcloud auth activate-service-account --help >/dev/null
gcloud auth print-access-token --help >/dev/null
kubectl version --client -o json
python3 /opt/proxy/get_proxy_url_test.py
python3 -c 'import requests; import ssl'
command -v curl wget jq bash
bash -n /opt/proxy/attempt-register-vm-on-proxy.sh
test "$DEBUG" = false
test "$PROXY_TIMEOUT" = 60s
test "$HEALTH_CHECK_PATH" = /
test "$HEALTH_CHECK_INTERVAL_SECONDS" = 0
test "$HEALTH_CHECK_UNHEALTHY_THRESHOLD" = 2
'''
        result = subprocess.run([
            'docker', 'run', '--rm', '--platform', 'linux/amd64',
            '--entrypoint', '/bin/bash', self.image, '-c', script
        ],
                                capture_output=True,
                                text=True,
                                check=False)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('v1.36.1', result.stdout)


if __name__ == '__main__':
    unittest.main()
