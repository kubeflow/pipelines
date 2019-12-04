# Copyright 2019 Google LLC
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

from kfp.dsl import ContainerOp
from kfp.dsl.extensions.kubernetes import use_secret
import unittest
import inspect


class TestAddSecrets(unittest.TestCase):

    def test_use_default_use_secret(self):
        op1 = ContainerOp(name="op1", image="image")
        secret_name = "my-secret"
        secret_path = "/here/are/my/secret"
        op1 = op1.apply(use_secret(secret_name=secret_name, 
            secret_volume_mount_path=secret_path))
        self.assertEqual(type(op1.container.env), type(None))
        container_dict = op1.container.to_dict()
        volume_mounts = container_dict["volume_mounts"][0]
        self.assertEqual(volume_mounts["name"], secret_name)
        self.assertEqual(type(volume_mounts), dict)
        self.assertEqual(volume_mounts["mount_path"], secret_path)

    def test_use_set_volume_use_secret(self):
        op1 = ContainerOp(name="op1", image="image")
        secret_name = "my-secret"
        secret_path = "/here/are/my/secret"
        op1 = op1.apply(use_secret(secret_name=secret_name, 
            secret_volume_mount_path=secret_path))
        self.assertEqual(type(op1.container.env), type(None))
        container_dict = op1.container.to_dict()
        volume_mounts = container_dict["volume_mounts"][0]
        self.assertEqual(type(volume_mounts), dict)
        self.assertEqual(volume_mounts["mount_path"], secret_path)

    def test_use_set_env_use_secret(self):
        op1 = ContainerOp(name="op1", image="image")
        secret_name = "my-secret"
        secret_path = "/here/are/my/secret/"
        env_variable = "MY_SECRET"
        secret_file_path_in_volume = "secret.json"
        op1 = op1.apply(use_secret(secret_name=secret_name, 
            secret_volume_mount_path=secret_path,
            env_variable=env_variable,
            secret_file_path_in_volume=secret_file_path_in_volume))
        self.assertEqual(len(op1.container.env), 1)
        container_dict = op1.container.to_dict()
        volume_mounts = container_dict["volume_mounts"][0]
        self.assertEqual(type(volume_mounts), dict)
        self.assertEqual(volume_mounts["mount_path"], secret_path)
        env_dict = op1.container.env[0].to_dict()
        self.assertEqual(env_dict["name"], env_variable)
        self.assertEqual(env_dict["value"], secret_path + secret_file_path_in_volume)