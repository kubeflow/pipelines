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
import io
import json
import unittest
from unittest.mock import Mock
from unittest.mock import patch

import third_party_upgrade_compatibility as compatibility


class CompatibilityTest(unittest.TestCase):

    def test_corrupted_persistent_object_fails(self):
        s3 = Mock()
        s3.get_object.return_value = {"Body": io.BytesIO(b"changed")}
        with self.assertRaisesRegex(AssertionError, "changed during upgrade"):
            compatibility.verify_payload(s3, "object")

    def test_missing_lifecycle_rule_fails(self):
        s3 = Mock()
        s3.get_object.return_value = {"Body": io.BytesIO(compatibility.PAYLOAD)}
        s3.get_bucket_lifecycle_configuration.return_value = {"Rules": []}
        with self.assertRaisesRegex(AssertionError, "Lifecycle configuration"):
            compatibility.verify_state(s3)

    def test_persistent_data_and_lifecycle_pass(self):
        s3 = Mock()
        s3.get_object.return_value = {"Body": io.BytesIO(compatibility.PAYLOAD)}
        s3.get_bucket_lifecycle_configuration.return_value = {
            "Rules": [{
                "Status": "Enabled",
                "Expiration": {
                    "Days": 30
                }
            }]
        }
        compatibility.verify_state(s3)

    def test_reconciliation_sends_existing_credentials(self):
        secret = {"kind": "Secret", "data": {"accesskey": "test"}}
        response = io.BytesIO(json.dumps({"attachments": [secret]}).encode())
        with patch.object(
                compatibility.urllib.request, "urlopen",
                return_value=response) as request:
            self.assertEqual(
                compatibility.sync_profile("http://localhost", "user", secret),
                secret)
        observed = json.loads(request.call_args.args[0].data)
        self.assertEqual(
            observed["attachments"]["Secret.v1"],
            {"user/mlpipeline-minio-artifact": secret},
        )


if __name__ == "__main__":
    unittest.main()
