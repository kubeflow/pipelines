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


from kfp.dsl import Pipeline, PipelineParam, ContainerOp, pipeline
import unittest


class TestPipeline(unittest.TestCase):
  
  def test_basic(self):
    """Test basic usage."""
    with Pipeline('somename') as p:
      self.assertTrue(Pipeline.get_default_pipeline() is not None)
      op1 = ContainerOp(name='op1', image='image')
      op2 = ContainerOp(name='op2', image='image')
      
    self.assertTrue(Pipeline.get_default_pipeline() is None)
    self.assertEqual(p.ops['op1'].name, 'op1')
    self.assertEqual(p.ops['op2'].name, 'op2')

  def test_nested_pipelines(self):
    """Test nested pipelines"""
    with self.assertRaises(Exception):
      with Pipeline('somename1') as p1:
        with Pipeline('somename2') as p2:
          pass

  def test_decorator(self):
    """Test @pipeline decorator."""
    @pipeline(
      name='p1',
      description='description1'
    )
    def my_pipeline1():
      pass

    @pipeline(
      name='p2',
      description='description2'
    )
    def my_pipeline2():
      pass
    
    self.assertEqual(('p1', 'description1'), Pipeline.get_pipeline_functions()[my_pipeline1])
    self.assertEqual(('p2', 'description2'), Pipeline.get_pipeline_functions()[my_pipeline2])
