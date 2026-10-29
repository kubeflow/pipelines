# Copyright 2026 The Kubeflow Authors
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
"""Security and compatibility tests for local condition evaluation."""

import os
import tempfile
import unittest

from kfp.compiler import pipeline_spec_builder
from kfp.dsl import pipeline_channel
from kfp.local import io
from kfp.local.orchestrator.enhanced_dag_orchestrator import ConditionEvaluator


class ConditionEvaluatorTest(unittest.TestCase):

    def setUp(self):
        self.store = io.IOStore()

    def test_task_output_is_data_for_both_reference_forms(self):
        with tempfile.TemporaryDirectory() as directory:
            marker = os.path.join(directory, 'host-executed')
            payload = ("x' == 'expected' or ([c for c in ().__class__.__base__"
                       ".__subclasses__() if c.__name__ == 'catch_warnings']"
                       "[0]()._module.__builtins__['__import__']('os')"
                       f".system('touch {marker}') == 0) or 'x")
            self.store.put_task_output('producer', 'Output', payload)
            for ref in (
                    'pipelinechannel--producer-Output',
                    "inputs.parameter_values['pipelinechannel--producer-Output']",
            ):
                with self.subTest(ref=ref):
                    self.assertFalse(
                        ConditionEvaluator.evaluate_condition(
                            f"{ref} == 'expected'", self.store))
                    self.assertFalse(os.path.exists(marker))
                    self.assertTrue(
                        ConditionEvaluator.evaluate_condition(
                            f'{ref} == {payload!r}', self.store))

    def test_normal_conditions(self):
        self.store.put_task_output('producer', 'Output', 'hello')
        self.store.put_task_output('number', 'Output', '2')
        cases = [
            (
                "inputs.parameter_values['pipelinechannel--producer-Output'] == 'hello'",
                True,
            ),
            ("pipelinechannel--producer-Output != 'hello'", False),
            (
                "int(inputs.parameter_values['pipelinechannel--number-Output']) == 2",
                True,
            ),
            ('double(pipelinechannel--number-Output) >= 2.0 && true', True),
            ("!(pipelinechannel--producer-Output == 'bad') || false", True),
            ('true && null == None', True),
            ('', True),
        ]
        for expression, expected in cases:
            with self.subTest(expression=expression):
                self.assertEqual(
                    expected,
                    ConditionEvaluator.evaluate_condition(
                        expression, self.store),
                )

    def test_cel_operators_inside_quoted_literals(self):
        reference = 'pipelinechannel--producer-Output'
        for output in ('a&&b', 'a||b', 'a!(b)', 'a! (b)'):
            self.store.put_task_output('producer', 'Output', output)
            for channel in (
                    reference,
                    f"inputs.parameter_values['{reference}']",
            ):
                for quoted_output in (repr(output), f'"{output}"'):
                    with self.subTest(
                            output=output,
                            channel=channel,
                            quoted_output=quoted_output,
                    ):
                        self.assertTrue(
                            ConditionEvaluator.evaluate_condition(
                                f'{channel} == {quoted_output}', self.store))
                        self.assertFalse(
                            ConditionEvaluator.evaluate_condition(
                                f'{channel} != {quoted_output}', self.store))

        self.store.put_task_output('producer', 'Output', "a'&&b")
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                "pipelinechannel--producer-Output == 'a\\'&&b'", self.store))

    def test_compiler_string_promotion_binds_channel_value(self):
        self.store.put_task_output('flag', 'Output', True)
        promoted = ("'inputs.parameter_values['pipelinechannel--flag-Output']'")
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(f"{promoted} == 'True'",
                                                  self.store))
        self.assertFalse(
            ConditionEvaluator.evaluate_condition(f"{promoted} == '_channel_0'",
                                                  self.store))
        self.store.put_task_output('number', 'Output', '2')
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                "'int(inputs.parameter_values['pipelinechannel--number-Output'])' == '2'",
                self.store,
            ))

    def test_channel_text_inside_literal_is_not_resolved(self):
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                "'pipelinechannel--unknown-Output' == 'pipelinechannel--unknown-Output'",
                self.store,
            ))

    def test_compiler_string_literal_keeps_backslashes(self):
        self.store.put_task_output('producer', 'Output', r'C:\bin')
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(
                r"pipelinechannel--producer-Output == 'C:\bin'", self.store))

    def test_compiler_string_literal_ends_with_two_backslashes(self):
        value = r'\\'
        channel = pipeline_channel.PipelineParameterChannel(
            name='Output', channel_type='String', task_name='producer')
        condition = pipeline_spec_builder._single_binary_operation_to_cel_condition(
            pipeline_channel.ConditionOperation('==', channel, value))
        self.store.put_task_output('producer', 'Output', value)
        self.assertTrue(
            ConditionEvaluator.evaluate_condition(condition, self.store))

    def test_large_numeric_conversion_fails_closed(self):
        self.store.put_task_output('number', 'Output', 10**400)
        self.assertFalse(
            ConditionEvaluator.evaluate_condition(
                "double(int(inputs.parameter_values['pipelinechannel--number-Output'])) > 1.0",
                self.store,
            ))

    def test_unsupported_code_is_rejected(self):
        for expression in (
                "__import__('os').system('true')",
                '(1).__class__',
                '[x for x in range(1)]',
        ):
            with self.subTest(expression=expression):
                self.assertFalse(
                    ConditionEvaluator.evaluate_condition(
                        expression, self.store))
