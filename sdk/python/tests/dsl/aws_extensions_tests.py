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

from kfp.dsl import ContainerOp
from kfp.aws import use_aws_secret
import unittest
import inspect


class TestAwsExtension(unittest.TestCase):
  def test_default_aws_secret_name(self):
    spec = inspect.getfullargspec(use_aws_secret)
    assert len(spec.defaults) == 3
    assert spec.defaults[0] == 'aws-secret'
    assert spec.defaults[1] == 'AWS_ACCESS_KEY_ID'
    assert spec.defaults[2] == 'AWS_SECRET_ACCESS_KEY'

  def test_use_aws_secret(self):
      op1 = ContainerOp(name='op1', image='image')
      op1 = op1.apply(use_aws_secret('myaws-secret', 'key_id', 'access_key'))
      assert len(op1.container.env) == 2

      index = 0
      for expected_name, expected_key in [('AWS_ACCESS_KEY_ID', 'key_id'), ('AWS_SECRET_ACCESS_KEY', 'access_key')]:
          assert op1.container.env[index].name == expected_name
          assert op1.container.env[index].value_from.secret_key_ref.name == 'myaws-secret'
          assert op1.container.env[index].value_from.secret_key_ref.key == expected_key
          index += 1
