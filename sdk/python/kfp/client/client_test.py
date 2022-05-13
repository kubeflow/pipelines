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

import unittest

from absl.testing import parameterized
from kfp.client import client


class TestValidatePipelineName(parameterized.TestCase):

    @parameterized.parameters([
        'pipeline',
        'my-pipeline',
        'my-pipeline-1',
        '1pipeline',
        'pipeline1',
    ])
    def test_valid(self, name: str):
        client.validate_pipeline_resource_name(name)

    @parameterized.parameters([
        'my_pipeline',
        "person's-pipeline",
        'my pipeline',
        'pipeline.yaml',
    ])
    def test_invalid(self, name: str):
        with self.assertRaisesRegex(ValueError, r'Invalid pipeline name:'):
            client.validate_pipeline_resource_name(name)


if __name__ == '__main__':
    unittest.main()
