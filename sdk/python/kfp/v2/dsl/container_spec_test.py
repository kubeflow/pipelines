# Copyright 2021 Google LLC
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
"""Tests for kfp.v2.dsl.container_spec."""

import unittest

from kfp.v2.dsl import container_spec as dsl_container_spec
from kfp.pipeline_spec import pipeline_spec_pb2

from google.protobuf import json_format


class ContainerSpecTest(unittest.TestCase):

  def test_build_container_spec(self):
    expected_dict = {
        "image": "gcr.io/image",
        "command": ["cmd"],
        "args": ["arg1", "arg2"]
    }
    expected_spec = (
        pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec())
    json_format.ParseDict(expected_dict, expected_spec)

    container_spec = (
        dsl_container_spec.build_container_spec("gcr.io/image", ["cmd"],
                                                ["arg1", "arg2"]))

    self.assertEqual(expected_spec, container_spec)


if __name__ == "__main__":
  unittest.main()
