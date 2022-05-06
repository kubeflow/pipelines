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

from absl.testing import parameterized
from kfp.compiler import helpers


class TestValidatePipelineName(parameterized.TestCase):

    @parameterized.parameters(
        {
            'pipeline_name': 'my-pipeline',
            'is_valid': True,
        },
        {
            'pipeline_name': 'p' * 128,
            'is_valid': True,
        },
        {
            'pipeline_name': 'p' * 129,
            'is_valid': False,
        },
        {
            'pipeline_name': 'my_pipeline',
            'is_valid': False,
        },
        {
            'pipeline_name': '-my-pipeline',
            'is_valid': False,
        },
        {
            'pipeline_name': 'My pipeline',
            'is_valid': False,
        },
    )
    def test(self, pipeline_name, is_valid):

        if is_valid:
            helpers.validate_pipeline_name(pipeline_name)
        else:
            with self.assertRaisesRegex(ValueError, 'Invalid pipeline name: '):
                helpers.validate_pipeline_name('my_pipeline')
