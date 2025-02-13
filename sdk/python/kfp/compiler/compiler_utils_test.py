# Copyright 2023 The Kubeflow Authors
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
from kfp.compiler import compiler_utils
from kfp.dsl import pipeline_channel


class TestAdditionalInputNameForPipelineChannel(parameterized.TestCase):

    @parameterized.parameters(
        {
            'channel':
                pipeline_channel.create_pipeline_channel(
                    name='output1', task_name='task1', channel_type='String'),
            'expected':
                'pipelinechannel--task1-output1',
        },
        {
            'channel':
                pipeline_channel.create_pipeline_channel(
                    name='output1',
                    task_name='task1',
                    channel_type='system.Artifact@0.0.1',
                ),
            'expected':
                'pipelinechannel--task1-output1',
        },
        {
            'channel':
                pipeline_channel.create_pipeline_channel(
                    name='param1', channel_type='String'),
            'expected':
                'pipelinechannel--param1',
        },
    )
    def test_additional_input_name_for_pipeline_channel(self, channel,
                                                        expected):
        self.assertEqual(
            expected,
            compiler_utils.additional_input_name_for_pipeline_channel(channel))


if __name__ == '__main__':
    unittest.main()
