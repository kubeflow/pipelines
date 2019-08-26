# Copyright 2018 Google LLC
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

from kfp.dsl._metadata import ComponentMeta, ParameterMeta
import unittest


class TestComponentMeta(unittest.TestCase):

  def test_to_dict(self):
    component_meta = ComponentMeta(name='foobar',
                                   description='foobar example',
                                   inputs=[ParameterMeta(name='input1',
                                                         description='input1 desc',
                                                         param_type={'GCSPath': {
                                                             'bucket_type': 'directory',
                                                             'file_type': 'csv'
                                                         }},
                                                         default='default1'
                                                         ),
                                           ParameterMeta(name='input2',
                                                         description='input2 desc',
                                                         param_type={'TFModel': {
                                                            'input_data': 'tensor',
                                                            'version': '1.8.0'
                                                         }},
                                                         default='default2'
                                                         ),
                                           ParameterMeta(name='input3',
                                                         description='input3 desc',
                                                         param_type='Integer',
                                                         default='default3'
                                                         ),
                                           ],
                                   outputs=[ParameterMeta(name='output1',
                                                          description='output1 desc',
                                                          param_type={'Schema': {
                                                              'file_type': 'tsv'
                                                          }},
                                                          default='default_output1'
                                                          )
                                            ]
                                   )
    golden_meta = {
        'name': 'foobar',
        'description': 'foobar example',
        'inputs': [
            {
                'name': 'input1',
                'description': 'input1 desc',
                'type': {
                    'GCSPath': {
                        'bucket_type': 'directory',
                        'file_type': 'csv'
                    }
                },
                'default': 'default1'
            },
            {
                'name': 'input2',
                'description': 'input2 desc',
                'type': {
                    'TFModel': {
                        'input_data': 'tensor',
                        'version': '1.8.0'
                    }
                },
                'default': 'default2'
            },
            {
                'name': 'input3',
                'description': 'input3 desc',
                'type': 'Integer',
                'default': 'default3'
            }
        ],
        'outputs': [
            {
                'name': 'output1',
                'description': 'output1 desc',
                'type': {
                    'Schema': {
                        'file_type': 'tsv'
                    }
                },
                'default': 'default_output1'
            }
        ]
    }
    self.assertEqual(component_meta.to_dict(), golden_meta)
