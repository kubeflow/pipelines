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


from kfp.dsl import Pipeline, PipelineParam, ContainerOp
from kfp.dsl import ComponentMeta, ParameterMeta, TypeMeta
import unittest

class TestComponentMeta(unittest.TestCase):

  def test_to_dict(self):
    component_meta = ComponentMeta(name='foobar',
                                   description='foobar example',
                                   inputs=[ParameterMeta(name='input1',
                                                         description='input1 desc',
                                                         param_type=TypeMeta(name='GCSPath',
                                                                             properties={'bucket_type': 'directory',
                                                                                         'file_type': 'csv'
                                                                                         }
                                                                             )
                                                         ),
                                           ParameterMeta(name='input2',
                                                         description='input2 desc',
                                                         param_type=TypeMeta(name='TFModel',
                                                                             properties={'input_data': 'tensor',
                                                                                         'version': '1.8.0'
                                                                                         }
                                                                             )
                                                         ),
                                           ],
                                   outputs=[ParameterMeta(name='output1',
                                                          description='output1 desc',
                                                          param_type=TypeMeta(name='Schema',
                                                                              properties={'file_type': 'tsv'
                                                                                          }
                                                                              )
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
                }
            },
            {
                'name': 'input2',
                'description': 'input2 desc',
                'type': {
                    'TFModel': {
                        'input_data': 'tensor',
                        'version': '1.8.0'
                    }
                }
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
                }
            }
        ]
    }
    self.assertEqual(component_meta.to_dict(), golden_meta)

  def test_type_meta_from_dict(self):
    component_dict = {
        'GCSPath': {
            'bucket_type': 'directory',
            'file_type': 'csv'
        }
    }
    golden_type_meta = TypeMeta(name='GCSPath', properties={'bucket_type': 'directory',
                                                            'file_type': 'csv'})
    self.assertEqual(TypeMeta.from_dict(component_dict), golden_type_meta)

class TestContainerOp(unittest.TestCase):

  def test_basic(self):
    """Test basic usage."""
    with Pipeline('somename') as p:
      param1 = PipelineParam('param1')
      param2 = PipelineParam('param2')
      op1 = ContainerOp(name='op1', image='image',
          arguments=['%s hello %s %s' % (param1, param2, param1)],
          file_outputs={'out1': '/tmp/b'})
      
    self.assertCountEqual([x.name for x in op1.inputs], ['param1', 'param2'])
    self.assertCountEqual(list(op1.outputs.keys()), ['out1'])
    self.assertCountEqual([x.op_name for x in op1.outputs.values()], ['op1'])
    self.assertEqual(op1.output.name, 'out1')

  def test_after_op(self):
    """Test duplicate ops."""
    with Pipeline('somename') as p:
      op1 = ContainerOp(name='op1', image='image')
      op2 = ContainerOp(name='op2', image='image')
      op2.after(op1)
    self.assertCountEqual(op2.dependent_op_names, [op1.name])
