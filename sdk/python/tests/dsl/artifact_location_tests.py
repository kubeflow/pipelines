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

from kfp.compiler._k8s_helper import convert_k8s_obj_to_json
from kfp.dsl import ArtifactLocation
from kubernetes.client.models import V1SecretKeySelector

import unittest


class TestArtifactLocation(unittest.TestCase):

  def test_artifact_location_constructor(self):
    artifact_location = ArtifactLocation.s3(
        bucket="foo",
        endpoint="s3.amazonaws.com",
        insecure=False,
        region="ap-southeast-1",
        access_key_secret={"name": "s3-secret", "key": "accesskey"},
        secret_key_secret=V1SecretKeySelector(name="s3-secret", key="secretkey")
    )

    expected = {
        "bucket": "foo",
        "endpoint": "s3.amazonaws.com",
        "insecure": False,
        "region": "ap-southeast-1",
        "access_key_secret": {"name": "s3-secret", "key": "accesskey"},
        "secret_key_secret": {"name": "s3-secret", "key": "secretkey"}
    }

    self.assertEqual(artifact_location.s3.bucket, "foo")
    self.assertEqual(artifact_location.s3.endpoint, "s3.amazonaws.com")
    self.assertEqual(artifact_location.s3.insecure, False)
    self.assertEqual(artifact_location.s3.region, "ap-southeast-1")
    self.assertEqual(artifact_location.s3.access_key_secret.name, "s3-secret")
    self.assertEqual(artifact_location.s3.access_key_secret.key, "accesskey")
    self.assertEqual(artifact_location.s3.secret_key_secret.name, "s3-secret")
    self.assertEqual(artifact_location.s3.secret_key_secret.key, "secretkey")

  def test_create_artifact_for_s3_with_default(self):
    # should trigger pending deprecation warning about not having a default
    # artifact_location if artifact_location is not provided.
    artifact = ArtifactLocation.create_artifact_for_s3(
        None,
        name="foo",
        path="path/to",
        key="key")

    self.assertEqual(artifact.name, "foo")
    self.assertEqual(artifact.path, "path/to")

  def test_create_artifact_for_s3(self):
    artifact_location = ArtifactLocation.s3(
        bucket="foo",
        endpoint="s3.amazonaws.com",
        insecure=False,
        region="ap-southeast-1",
        access_key_secret={"name": "s3-secret", "key": "accesskey"},
        secret_key_secret=V1SecretKeySelector(name="s3-secret", key="secretkey")
    )
    artifact = ArtifactLocation.create_artifact_for_s3(
        artifact_location,
        name="foo",
        path="path/to",
        key="key")

    self.assertEqual(artifact.name, "foo")
    self.assertEqual(artifact.path, "path/to")
    self.assertEqual(artifact.s3.endpoint, "s3.amazonaws.com")
    self.assertEqual(artifact.s3.bucket, "foo")
    self.assertEqual(artifact.s3.key, "key")
    self.assertEqual(artifact.s3.access_key_secret.name, "s3-secret")
    self.assertEqual(artifact.s3.access_key_secret.key, "accesskey")
    self.assertEqual(artifact.s3.secret_key_secret.name, "s3-secret")
    self.assertEqual(artifact.s3.secret_key_secret.key, "secretkey")

  def test_create_artifact_for_s3_with_dict(self):
    # use the convert_k8s_obj_to_json to mimick the compiler
    artifact_location_dict = convert_k8s_obj_to_json(ArtifactLocation.s3(
        bucket="foo",
        endpoint="s3.amazonaws.com",
        insecure=False,
        region="ap-southeast-1",
        access_key_secret={"name": "s3-secret", "key": "accesskey"},
        secret_key_secret=V1SecretKeySelector(name="s3-secret", key="secretkey")
    ))
    artifact = ArtifactLocation.create_artifact_for_s3(
        artifact_location_dict,
        name="foo",
        path="path/to",
        key="key")

    self.assertEqual(artifact.name, "foo")
    self.assertEqual(artifact.path, "path/to")
    self.assertEqual(artifact.s3.endpoint, "s3.amazonaws.com")
    self.assertEqual(artifact.s3.bucket, "foo")
    self.assertEqual(artifact.s3.key, "key")
    self.assertEqual(artifact.s3.access_key_secret.name, "s3-secret")
    self.assertEqual(artifact.s3.access_key_secret.key, "accesskey")
    self.assertEqual(artifact.s3.secret_key_secret.name, "s3-secret")
    self.assertEqual(artifact.s3.secret_key_secret.key, "secretkey")