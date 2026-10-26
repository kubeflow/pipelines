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
from contextlib import nullcontext
import io
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock
from unittest.mock import patch

import third_party_upgrade_compatibility as compatibility


class CompatibilityTest(unittest.TestCase):

    def test_secret_credentials_string_and_encoded_data(self):
        self.assertEqual(
            compatibility.secret_credentials({
                "data": {
                    "accesskey": "bWluaW8=",
                    "secretkey": "bWluaW8xMjM="
                }
            }), ("minio", "minio123"))
        self.assertEqual(
            compatibility.secret_credentials({
                "data": {
                    "accesskey": "b2xk",
                    "secretkey": "bWluaW8xMjM="
                },
                "stringData": {
                    "accesskey": "override"
                }
            }), ("override", "minio123"))

    def test_invalid_secret_credentials_fail_closed(self):
        for secret in ({}, {
                "data": {
                    "accesskey": "not base64"
                }
        }, {
                "stringData": {
                    "accesskey": "",
                    "secretkey": "value"
                }
        }, {
                "stringData": {
                    "accesskey": 123,
                    "secretkey": "value"
                }
        }, {
                "data": {
                    "accesskey": "/w==",
                    "secretkey": "bWluaW8="
                }
        }):
            with self.subTest(secret=secret), self.assertRaisesRegex(
                    ValueError, "invalid or missing"):
                compatibility.secret_credentials(secret)

    def test_exercise_uses_manifest_credentials_without_overwriting_secret(
            self):
        storage = [{
            "kind": "Deployment",
            "spec": {
                "template": {
                    "spec": {
                        "containers": [{
                            "image": "seaweedfs:new"
                        }]
                    }
                }
            }
        }, {
            "kind": "Secret",
            "metadata": {
                "name": "mlpipeline-minio-artifact"
            },
            "stringData": {
                "accesskey": "manifest-user",
                "secretkey": "manifest-key"
            }
        }]
        user_secret = {"data": {"accesskey": "dXNlcg==", "secretkey": "a2V5"}}
        with tempfile.TemporaryDirectory() as directory, \
             patch.object(compatibility, "rendered", side_effect=[storage, []]), \
             patch.object(compatibility, "apply") as apply, \
             patch.object(compatibility, "wait_ready"), \
             patch.object(compatibility, "kubectl", return_value='{"metadata":{"uid":"same-pvc"}}'), \
             patch.object(compatibility, "forward", side_effect=lambda *args: nullcontext("http://localhost")), \
             patch.object(compatibility, "client") as client, \
             patch.object(compatibility, "verify_state"), \
             patch.object(compatibility, "verify_payload"), \
             patch.object(compatibility, "sync_profile", return_value=user_secret):
            compatibility.exercise(Path(directory))
        self.assertFalse(
            any(obj["kind"] == "Secret"
                for obj in apply.call_args_list[0].args[0]))
        self.assertEqual(apply.call_args_list[1].args[0], storage)
        self.assertEqual([call.args[1:] for call in client.call_args_list],
                         [("manifest-user", "manifest-key"),
                          ("manifest-user", "manifest-key"), ("user", "key"),
                          ("manifest-user", "manifest-key"), ("user", "key")])

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
