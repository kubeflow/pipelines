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
from kfp.local.orchestrator import cel
from kfp.local.orchestrator import enhanced_dag_orchestrator as edo


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
