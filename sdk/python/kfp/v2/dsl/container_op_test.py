"""Tests for kfp.v2.dsl.container_op"""
import unittest

from kfp.v2.dsl import container_op
from kfp.v2.proto import pipeline_spec_pb2

from google.protobuf import text_format
from google.protobuf import json_format

_EXPECTED_CONTAINER_WITH_RESOURCE = """
resources {
  cpu_limit: 1.0
  memory_limit: 1.0
  accelerator {
    type: 'NVIDIA_TESLA_K80'
    count: 1
  }
}
"""

# Shorthand for PipelineContainerSpec
_PipelineContainerSpec = pipeline_spec_pb2.PipelineDeploymentConfig.PipelineContainerSpec


class ContainerOpTest(unittest.TestCase):
  def test_illegal_resource_setter_fail(self):
    task = container_op.ContainerOp(name='test_task', image='python:3.7')
    with self.assertRaisesRegex(TypeError, 'ContainerOp.container_spec '
                                           'is expected to be'):
      task.set_cpu_limit('1')

  def test_illegal_label_fail(self):
    task = container_op.ContainerOp(name='test_task', image='python:3.7')
    task.container_spec = _PipelineContainerSpec()
    with self.assertRaisesRegex(
        ValueError, 'Currently add_node_selector_constraint only supports'):
      task.add_node_selector_constraint('cloud.google.com/test-label',
                                        'test-value')

  def test_chained_call_resource_setter(self):
    task = container_op.ContainerOp(name='test_task', image='python:3.7')
    task.container_spec = _PipelineContainerSpec()
    task.set_cpu_limit(
        '1').set_memory_limit(
        '1G').add_node_selector_constraint(
        'cloud.google.com/gke-accelerator',
        'nvidia-tesla-k80').set_gpu_limit(1)

    expected_container_spec = text_format.Parse(
        _EXPECTED_CONTAINER_WITH_RESOURCE, _PipelineContainerSpec())

    self.assertDictEqual(json_format.MessageToDict(task.container_spec),
                         json_format.MessageToDict(expected_container_spec))


if __name__ == '__main__':
  unittest.main()
