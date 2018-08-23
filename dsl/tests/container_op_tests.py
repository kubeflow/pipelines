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


import mlp
import unittest


class TestContainerOp(unittest.TestCase):

  def test_basic(self):
    """Test basic usage."""
    with mlp.Pipeline('somename') as p:
      param1 = mlp.PipelineParam('param1')
      param2 = mlp.PipelineParam('param2')
      op1 = mlp.ContainerOp(name='op1', image='image',
          arguments=['%s hello %s' % (param1, param2)],
          file_outputs={'out1': '/tmp/b'})
      
    self.assertCountEqual([x.name for x in op1.inputs], ['param1', 'param2'])
    self.assertCountEqual(list(op1.outputs.keys()), ['out1'])
    self.assertCountEqual([x.op_name for x in op1.outputs.values()], ['op1'])
    self.assertEqual(op1.output.name, 'out1')

  def test_after_op(self):
    """Test duplicate ops."""
    with mlp.Pipeline('somename') as p:
      op1 = mlp.ContainerOp(name='op1', image='image')
      op2 = mlp.ContainerOp(name='op2', image='image')
      op2.after(op1)
    self.assertCountEqual(op2.dependent_op_names, ['op1'])

  def test_invalid_name(self):
    """Invalid pipeline param name and op_name."""
    with self.assertRaises(ValueError):
      with mlp.Pipeline('somename') as p:
        p = mlp.ContainerOp(name='a b', image='image')
      
