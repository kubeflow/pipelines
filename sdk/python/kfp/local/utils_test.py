# Copyright 2024 The Kubeflow Authors
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
"""Test for utils.py."""
import unittest

from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp.local import utils
from kfp.pipeline_spec import pipeline_spec_pb2


class TestDictToExecutorSpec(unittest.TestCase):

    def test_simple(self):
        input_struct = struct_pb2.Struct()
        input_dict = {
            'container': {
                'image': 'alpine',
                'command': ['echo'],
                'args': ['foo'],
            }
        }
        json_format.ParseDict(input_dict, input_struct)
        expected = pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
            container=pipeline_spec_pb2.PipelineDeploymentConfig
            .PipelineContainerSpec(
                image='alpine', command=['echo'], args=['foo']))

        actual = utils.struct_to_executor_spec(input_struct)
        self.assertEqual(actual, expected)


if __name__ == '__main__':
    unittest.main()
