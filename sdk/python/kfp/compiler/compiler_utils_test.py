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

    @parameterized.parameters(
        {
            'data': [{
                'container_spec': {
                    'image_uri':
                        'gcr.io/ml-pipeline/google-cloud-pipeline-components:2.5.0',
                    'command': ['echo'],
                    'args': ['foo']
                },
                'machine_spec': {
                    'machine_type':
                        pipeline_channel.PipelineParameterChannel(
                            name='Output',
                            channel_type='String',
                            task_name='machine-type'),
                    'accelerator_type':
                        pipeline_channel.PipelineParameterChannel(
                            name='Output',
                            channel_type='String',
                            task_name='accelerator-type'),
                    'accelerator_count':
                        1
                },
                'replica_count': 1
            }],
            'old_value':
                '{{channel:task=machine-type;name=Output;type=String;}}',
            'new_value':
                '{{$.inputs.parameters['
                'pipelinechannel--machine-type-Output'
                ']}}',
            'expected': [{
                'container_spec': {
                    'image_uri':
                        'gcr.io/ml-pipeline/google-cloud-pipeline-components:2.5.0',
                    'command': ['echo'],
                    'args': ['foo']
                },
                'machine_spec': {
                    'machine_type':
                        '{{$.inputs.parameters['
                        'pipelinechannel--machine-type-Output'
                        ']}}',
                    'accelerator_type':
                        pipeline_channel.PipelineParameterChannel(
                            name='Output',
                            channel_type='String',
                            task_name='accelerator-type'),
                    'accelerator_count':
                        1
                },
                'replica_count': 1
            }],
        },
        {
            'data': [{
                'first_name':
                    f'my_first_name: {str(pipeline_channel.PipelineParameterChannel(name="Output", channel_type="String", task_name="first_name"))}',
                'last_name':
                    f'my_last_name: {str(pipeline_channel.PipelineParameterChannel(name="Output", channel_type="String", task_name="last_name"))}',
            }],
            'old_value':
                '{{channel:task=first_name;name=Output;type=String;}}',
            'new_value':
                '{{$.inputs.parameters['
                'pipelinechannel--first_name-Output'
                ']}}',
            'expected': [{
                'first_name':
                    'my_first_name: {{$.inputs.parameters[pipelinechannel--first_name-Output]}}',
                'last_name':
                    f'my_last_name: {str(pipeline_channel.PipelineParameterChannel(name="Output", channel_type="String", task_name="last_name"))}',
            }],
        },
        {
            'data': [{
                'project': 'project',
                'location': 'US',
                'job_configuration_query': {
                    'query': 'SELECT * FROM `project.dataset.input_table`',
                    'destinationTable': {
                        'projectId':
                            'project',
                        'datasetId':
                            'dataset',
                        'tableId':
                            f'output_table_{str(pipeline_channel.PipelineParameterChannel(name="Output", channel_type="String", task_name="table_suffix"))}',
                    },
                    'writeDisposition': 'WRITE_TRUNCATE',
                },
            }],
            'old_value':
                '{{channel:task=table_suffix;name=Output;type=String;}}',
            'new_value':
                '{{$.inputs.parameters['
                'pipelinechannel--table_suffix-Output'
                ']}}',
            'expected': [{
                'project': 'project',
                'location': 'US',
                'job_configuration_query': {
                    'query': 'SELECT * FROM `project.dataset.input_table`',
                    'destinationTable': {
                        'projectId':
                            'project',
                        'datasetId':
                            'dataset',
                        'tableId':
                            'output_table_{{$.inputs.parameters[pipelinechannel--table_suffix-Output]}}',
                    },
                    'writeDisposition': 'WRITE_TRUNCATE',
                },
            }],
        },
    )
    def test_recursive_replace_placeholders(self, data, old_value, new_value,
                                            expected):
        self.assertEqual(
            expected,
            compiler_utils.recursive_replace_placeholders(
                data, old_value, new_value))


if __name__ == '__main__':
    unittest.main()
