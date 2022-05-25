# Copyright 2022 The Kubeflow Authors
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
import tempfile
import unittest

from kfp.client import set_volume_credentials
from kfp.client import token_credentials_base
from kubernetes.client import configuration


class TestServiceAccountTokenVolumeCredentials(unittest.TestCase):

    def test_get_token(self):
        with tempfile.TemporaryDirectory() as tempdir:
            token_path = os.path.join(tempdir, 'token.txt')
            with open(token_path, 'w') as f:
                f.write('my_token')
            service_account_tvc = set_volume_credentials.ServiceAccountTokenVolumeCredentials(
                token_path)

            self.assertEqual(service_account_tvc._get_token(), 'my_token')

    def test_refresh_api_key_hook(self):
        with tempfile.TemporaryDirectory() as tempdir:
            token_path = os.path.join(tempdir, 'token.txt')
            with open(token_path, 'w') as f:
                f.write('my_token')
            service_account_tvc = set_volume_credentials.ServiceAccountTokenVolumeCredentials(
                token_path)
            config = configuration.Configuration()

            with self.assertRaisesRegex(KeyError, r'authorization'):
                config.api_key['authorization']

            service_account_tvc.refresh_api_key_hook(config)
            self.assertEqual(config.api_key['authorization'], 'my_token')
