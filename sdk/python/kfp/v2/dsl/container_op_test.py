"""Tests for kfp.v2.dsl.container_op"""
import unittest

from kfp.v2.dsl import container_op


class ContainerOpTest(unittest.TestCase):
  def test_illegal_resource_setter_fail(self):
    task = container_op.ContainerOp(name='test_task', image='python:3.7')
    with self.assertRaisesRegex(ValueError, 'Expecting container_spec '
                                            'attribute'):
      task.set_cpu_limit('1')


if __name__ == '__main__':
  unittest.main()
