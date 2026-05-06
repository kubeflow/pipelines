# Copyright 2024 The Kubeflow Authors
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
"""Tests for the enhanced DAG orchestrator's pure helpers."""

import logging
import unittest
from unittest import mock

from kfp.local import io
from kfp.local import status
from kfp.local.orchestrator import cel
from kfp.local.orchestrator import enhanced_dag_orchestrator as edo
from kfp.pipeline_spec import pipeline_spec_pb2 as ps


def _task_with_deps(dep_list):
    spec = ps.PipelineTaskSpec()
    spec.dependent_tasks.extend(dep_list)
    return spec


def _task_with_output_parameter(producer, key='Output'):
    spec = ps.PipelineTaskSpec()
    input_spec = spec.inputs.parameters['x']
    input_spec.task_output_parameter.producer_task = producer
    input_spec.task_output_parameter.output_parameter_key = key
    return spec


def _task_with_output_artifact(producer, key='Output'):
    spec = ps.PipelineTaskSpec()
    input_spec = spec.inputs.artifacts['a']
    input_spec.task_output_artifact.producer_task = producer
    input_spec.task_output_artifact.output_artifact_key = key
    return spec


class TestBuildFullDependencyMap(unittest.TestCase):

    def test_explicit_dependent_tasks(self):
        tasks = {
            'a': _task_with_deps([]),
            'b': _task_with_deps(['a']),
            'c': _task_with_deps(['a', 'b']),
        }
        deps = edo._build_full_dependency_map(tasks)
        self.assertEqual(deps['a'], [])
        self.assertEqual(deps['b'], ['a'])
        self.assertEqual(deps['c'], ['a', 'b'])

    def test_implicit_output_parameter_edges(self):
        tasks = {
            'producer': _task_with_deps([]),
            'consumer': _task_with_output_parameter('producer'),
        }
        deps = edo._build_full_dependency_map(tasks)
        self.assertEqual(deps['consumer'], ['producer'])

    def test_implicit_output_artifact_edges(self):
        tasks = {
            'producer': _task_with_deps([]),
            'consumer': _task_with_output_artifact('producer'),
        }
        deps = edo._build_full_dependency_map(tasks)
        self.assertEqual(deps['consumer'], ['producer'])

    def test_unknown_producer_is_ignored(self):
        tasks = {
            'consumer': _task_with_output_parameter('not-in-dag'),
        }
        deps = edo._build_full_dependency_map(tasks)
        self.assertEqual(deps['consumer'], [])

    def test_parallel_for_iterator_edge(self):
        producer = _task_with_deps([])
        loop = ps.PipelineTaskSpec()
        loop.parameter_iterator.items.input_parameter = (
            'pipelinechannel--producer-Output-loop-item')
        tasks = {'producer': producer, 'loop': loop}
        deps = edo._build_full_dependency_map(tasks)
        self.assertEqual(deps['loop'], ['producer'])


class TestMarkSkipped(unittest.TestCase):

    def test_skipped_task_is_reflected_in_io_store(self):
        store = io.IOStore()
        skipped = set()
        with mock.patch.object(logging.getLogger(), 'info') as log_mock:
            edo._mark_skipped(
                'x', store, skipped, reason='condition evaluated to False')
        self.assertTrue(store.is_task_skipped('x'))
        self.assertEqual(skipped, {'x'})
        # Log emitted contains task name and SKIPPED token.
        joined = ' '.join(str(c) for c in log_mock.call_args_list)
        self.assertIn("'x'", joined)
        self.assertIn('SKIPPED', joined)

    def test_get_task_output_of_skipped_returns_sentinel(self):
        store = io.IOStore()
        store.mark_task_skipped('x')
        value = store.get_task_output('x', 'Output')
        self.assertIs(value, io.SKIPPED)
        self.assertTrue(store.is_skipped(value))


class TestConditionEvaluatorIntegration(unittest.TestCase):

    def test_true_branch(self):
        self.assertTrue(
            edo.ConditionEvaluator.evaluate_condition(
                "inputs.parameter_values['pipelinechannel--x'] == 'foo'",
                {'pipelinechannel--x': 'foo'}))

    def test_false_branch(self):
        self.assertFalse(
            edo.ConditionEvaluator.evaluate_condition(
                "inputs.parameter_values['pipelinechannel--x'] == 'foo'",
                {'pipelinechannel--x': 'bar'}))

    def test_missing_parameter_propagates_cel_error(self):
        with self.assertRaises(cel.CELError):
            edo.ConditionEvaluator.evaluate_condition(
                "inputs.parameter_values['pipelinechannel--missing'] == 'x'",
                {})


if __name__ == '__main__':
    unittest.main()
