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

from kubernetes.client.models import V1ConfigMap, V1Container, V1EnvVar
from kfp.dsl import PipelineParam
from kfp.dsl._pipeline_param import _extract_pipelineparams, extract_pipelineparams_from_any
import unittest


class TestPipelineParam(unittest.TestCase):

  def test_invalid(self):
    """Invalid pipeline param name and op_name."""
    with self.assertRaises(ValueError):
      p = PipelineParam(name='123_abc')

  def test_str_repr(self):
    """Test string representation."""

    p = PipelineParam(name='param1', op_name='op1')
    self.assertEqual('{{pipelineparam:op=op1;name=param1}}', str(p))

    p = PipelineParam(name='param2')
    self.assertEqual('{{pipelineparam:op=;name=param2}}', str(p))

    p = PipelineParam(name='param3', value='value3')
    self.assertEqual('{{pipelineparam:op=;name=param3}}', str(p))

  def test_extract_pipelineparams(self):
    """Test _extract_pipeleineparams."""

    p1 = PipelineParam(name='param1', op_name='op1')
    p2 = PipelineParam(name='param2')
    p3 = PipelineParam(name='param3', value='value3')
    stuff_chars = ' between '
    payload = str(p1) + stuff_chars + str(p2) + stuff_chars + str(p3)
    params = _extract_pipelineparams(payload)
    self.assertListEqual([p1, p2, p3], params)
    payload = [str(p1) + stuff_chars + str(p2), str(p2) + stuff_chars + str(p3)]
    params = _extract_pipelineparams(payload)
    self.assertListEqual([p1, p2, p3], params)

  def test_extract_pipelineparams_from_any(self):
    """Test extract_pipeleineparams."""
    p1 = PipelineParam(name='param1', op_name='op1')
    p2 = PipelineParam(name='param2')
    p3 = PipelineParam(name='param3', value='value3')
    stuff_chars = ' between '
    payload = str(p1) + stuff_chars + str(p2) + stuff_chars + str(p3)

    container = V1Container(name=p1,
                            image=p2,
                            env=[V1EnvVar(name="foo", value=payload)])

    params = extract_pipelineparams_from_any(container)
    self.assertListEqual(sorted([p1, p2, p3]), sorted(params))

  def test_extract_pipelineparams_from_dict(self):
    """Test extract_pipeleineparams."""
    p1 = PipelineParam(name='param1', op_name='op1')
    p2 = PipelineParam(name='param2')

    configmap = V1ConfigMap(data={str(p1): str(p2)})

    params = extract_pipelineparams_from_any(configmap)
    self.assertListEqual(sorted([p1, p2]), sorted(params))

  def test_extract_pipelineparam_with_types(self):
    """Test _extract_pipelineparams. """
    p1 = PipelineParam(name='param1', op_name='op1', param_type={'customized_type_a': {'property_a': 'value_a'}})
    p2 = PipelineParam(name='param2', param_type='customized_type_b')
    p3 = PipelineParam(name='param3', value='value3', param_type={'customized_type_c': {'property_c': 'value_c'}})
    stuff_chars = ' between '
    payload = str(p1) + stuff_chars + str(p2) + stuff_chars + str(p3)
    params = _extract_pipelineparams(payload)
    self.assertListEqual([p1, p2, p3], params)
    # Expecting the _extract_pipelineparam to dedup the pipelineparams among all the payloads.
    payload = [str(p1) + stuff_chars + str(p2), str(p2) + stuff_chars + str(p3)]
    params = _extract_pipelineparams(payload)
    self.assertListEqual([p1, p2, p3], params)
