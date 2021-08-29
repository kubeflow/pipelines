# Copyright 2021 The Kubeflow Authors
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
"""Tests for kfp.v2.components.experimental.pipeline_param."""

import unittest

from absl.testing import parameterized
from kfp.v2.components.experimental import component_spec, pipeline_param


class PipelineParamTest(parameterized.TestCase):

    def test_invalid_name(self):
        with self.assertRaisesRegex(
                ValueError,
                'Only letters, numbers, spaces, "_", and "-" are allowed in the '
                'name. Must begin with a letter. Got name: 123_abc'):
            p = pipeline_param.PipelineParam(name='123_abc')

    def test_op_name_and_value_both_set(self):
        with self.assertRaisesRegex(ValueError,
                                    'op_name and value cannot be both set.'):
            p = pipeline_param.PipelineParam(
                name='abc', op_name='op1', value=123)

    @parameterized.parameters(
        {
            'pipeline_param':
                pipeline_param.PipelineParam(
                    name='param1', op_name='op1', param_type='String'),
            'str_repr':
                '{{pipelineparam:op=op1;name=param1;type=String;}}',
        },
        {
            'pipeline_param':
                pipeline_param.PipelineParam(
                    name='param2', param_type='Integer'),
            'str_repr':
                '{{pipelineparam:op=;name=param2;type=Integer;}}',
        },
        {
            'pipeline_param':
                pipeline_param.PipelineParam(
                    name='param3', param_type={'type_a': {
                        'property_b': 'c'
                    }}),
            'str_repr':
                '{{pipelineparam:op=;name=param3;type={"type_a": {"property_b": "c"}};}}',
        },
        {
            'pipeline_param':
                pipeline_param.PipelineParam(
                    name='param4', param_type='Float', value=1.23),
            'str_repr':
                '{{pipelineparam:op=;name=param4;type=Float;}}',
        },
        {
            'pipeline_param': pipeline_param.PipelineParam(name='param5'),
            'str_repr': '{{pipelineparam:op=;name=param5;type=;}}',
        },
    )
    def test_str_repr(self, pipeline_param, str_repr):
        self.assertEqual(str_repr, str(pipeline_param))

    def test_extract_pipelineparam(self):
        p1 = pipeline_param.PipelineParam(name='param1', op_name='op1')
        p2 = pipeline_param.PipelineParam(
            name='param2', param_type='customized_type_b')
        p3 = pipeline_param.PipelineParam(
            name='param3',
            value='value3',
            param_type={'customized_type_c': {
                'property_c': 'value_c'
            }})
        stuff_chars = ' between '
        payload = str(p1) + stuff_chars + str(p2) + stuff_chars + str(p3)
        params = pipeline_param.extract_pipelineparams(payload)
        self.assertListEqual([p1, p2, p3], params)

        # Expecting the extract_pipelineparam_from_any to dedup PipelineParams
        # among all the payloads.
        payload = [
            str(p1) + stuff_chars + str(p2),
            str(p2) + stuff_chars + str(p3)
        ]
        params = pipeline_param.extract_pipelineparams_from_any(payload)
        self.assertListEqual([p1, p2, p3], params)


if __name__ == '__main__':
    unittest.main()
