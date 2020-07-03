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

import kfp
from kfp.dsl import Pipeline, PipelineParam, ContainerOp, pipeline
from kfp.dsl._metadata import _extract_pipeline_metadata
from kfp.dsl.types import GCSPath, Integer
from kfp.components.structures import ComponentSpec, InputSpec
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
    
    self.assertEqual(my_pipeline1._component_human_name, 'p1')
    self.assertEqual(my_pipeline2._component_human_name, 'p2')
    self.assertEqual(my_pipeline1._component_description, 'description1')
    self.assertEqual(my_pipeline2._component_description, 'description2')

  def test_decorator_metadata(self):
    """Test @pipeline decorator with metadata."""
    @pipeline(
        name='p1',
        description='description1'
    )
    def my_pipeline1(a: {'Schema': {'file_type': 'csv'}}='good', b: Integer()=12):
      pass

    golden_meta = ComponentSpec(name='p1', description='description1', inputs=[])
    golden_meta.inputs.append(InputSpec(name='a', type={'Schema': {'file_type': 'csv'}}, default='good', optional=True))
    golden_meta.inputs.append(InputSpec(name='b', type={'Integer': {'openapi_schema_validator': {"type": "integer"}}}, default="12", optional=True))

    pipeline_meta = _extract_pipeline_metadata(my_pipeline1)
    self.assertEqual(pipeline_meta, golden_meta)
