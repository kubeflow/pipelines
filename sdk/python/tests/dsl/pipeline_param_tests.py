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


from kfp.dsl import PipelineParam
import unittest


class TestPipelineParam(unittest.TestCase):
  
  def test_invalid(self):
    """Invalid pipeline param name and op_name."""
    with self.assertRaises(ValueError):
      p = PipelineParam(name='123_abc')

    with self.assertRaises(ValueError):
      p = PipelineParam(name='param1', op_name='a b')

  def test_str_repr(self):
    """Test string representation."""

    p = PipelineParam(name='param1', op_name='op1')
    self.assertEqual('{{pipelineparam:op=op1;name=param1;value=}}', str(p))

    p = PipelineParam(name='param2')
    self.assertEqual('{{pipelineparam:op=;name=param2;value=}}', str(p))

    p = PipelineParam(name='param3', value='value3')
    self.assertEqual('{{pipelineparam:op=;name=param3;value=value3}}', str(p))
