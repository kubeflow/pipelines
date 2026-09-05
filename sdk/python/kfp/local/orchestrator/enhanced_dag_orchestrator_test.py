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
"""Tests for enhanced_dag_orchestrator.py."""
import unittest

from kfp.local import io
from kfp.local.orchestrator import enhanced_dag_orchestrator

ConditionEvaluator = enhanced_dag_orchestrator.ConditionEvaluator


class TestConditionEvaluator(unittest.TestCase):

    def _store_with_output(self, value):
        io_store = io.IOStore()
        io_store.put_task_output('t', 'Output', value)
        return io_store

    def test_empty_condition_is_true(self):
        self.assertTrue(ConditionEvaluator.evaluate_condition('', io.IOStore()))

    def test_channel_equality_match(self):
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                "pipelinechannel--t-Output == 'expected'",
                self._store_with_output('expected')))

    def test_channel_equality_no_match(self):
        self.assertFalse(
            ConditionEvaluator.evaluate_condition(
                "pipelinechannel--t-Output == 'expected'",
                self._store_with_output('other')))

    def test_int_value_preserved(self):
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                'pipelinechannel--t-Output == 5', self._store_with_output(5)))

    def test_bool_value_preserved(self):
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                'pipelinechannel--t-Output', self._store_with_output(True)))

    def test_cel_parameter_reference(self):
        io_store = io.IOStore()
        io_store.put_parent_input('p', 'a')
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                "inputs.parameter_values['p'] == 'a'", io_store))

    def test_quote_breakout_is_not_injected(self):
        # A task output that carries data from outside the pipeline must be
        # compared as a literal, never spliced into the evaluated expression.
        # Before binding values, this payload closed the string literal and
        # OR-ed in `True`, forcing the branch to run regardless of the value.
        payload = "x' or True or 'x"
        self.assertFalse(
            ConditionEvaluator.evaluate_condition(
                "pipelinechannel--t-Output == 'expected'",
                self._store_with_output(payload)))

    def test_sandbox_escape_payload_is_treated_as_data(self):
        # `__builtins__` being empty does not stop object-graph traversal such
        # as ().__class__.__base__.__subclasses__(); if the value reached the
        # expression text this would evaluate. Bound as data, it stays inert
        # and the equality is simply False.
        payload = ("x' == '' or [c for c in "
                   "().__class__.__base__.__subclasses__()] or 'x")
        self.assertFalse(
            ConditionEvaluator.evaluate_condition(
                "pipelinechannel--t-Output == 'expected'",
                self._store_with_output(payload)))


if __name__ == '__main__':
    unittest.main()
