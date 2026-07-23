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
"""Tests for the CEL subset evaluator."""

from kfp.local.orchestrator import cel
import pytest

PV_KEY = 'pipelinechannel--flip-coin-Output'


@pytest.mark.parametrize(
    'expression, parameter_values, expected',
    [
        # equality on strings
        (
            f"inputs.parameter_values['{PV_KEY}'] == 'heads'",
            {
                PV_KEY: 'heads'
            },
            True,
        ),
        (
            f"inputs.parameter_values['{PV_KEY}'] == 'heads'",
            {
                PV_KEY: 'tails'
            },
            False,
        ),
        # int() cast with comparison
        (
            "int(inputs.parameter_values['pipelinechannel--n']) > 5",
            {
                'pipelinechannel--n': '7'
            },
            True,
        ),
        (
            "int(inputs.parameter_values['pipelinechannel--n']) > 5",
            {
                'pipelinechannel--n': '3'
            },
            False,
        ),
        # int() on both sides of >=
        (
            ("int(inputs.parameter_values['pipelinechannel--x']) >= "
             "int(inputs.parameter_values['pipelinechannel--y'])"),
            {
                'pipelinechannel--x': 5,
                'pipelinechannel--y': 5
            },
            True,
        ),
        # unary !
        (
            f"!(inputs.parameter_values['{PV_KEY}'] == 'heads')",
            {
                PV_KEY: 'tails'
            },
            True,
        ),
        # mixed && with negation (if_elif_else_complex pattern)
        (
            ("!(int(inputs.parameter_values['pipelinechannel--x']) < 5000)"
             " && int(inputs.parameter_values['pipelinechannel--x']) >= 7500"),
            {
                'pipelinechannel--x': '8000'
            },
            True,
        ),
        (
            ("!(int(inputs.parameter_values['pipelinechannel--x']) < 5000)"
             " && int(inputs.parameter_values['pipelinechannel--x']) >= 7500"),
            {
                'pipelinechannel--x': '6000'
            },
            False,
        ),
        # || short-circuit
        (
            ("inputs.parameter_values['pipelinechannel--x'] == 'a'"
             " || inputs.parameter_values['pipelinechannel--y'] == 'b'"),
            {
                'pipelinechannel--x': 'a',
                'pipelinechannel--y': 'z'
            },
            True,
        ),
        # bool literal
        ('true', {}, True),
        ('false || true', {}, True),
        # empty string is treated as True (no condition => run)
        ('', {}, True),
    ],
)
def test_evaluate_known_forms(expression, parameter_values, expected):
    assert cel.evaluate(expression, parameter_values) is expected


def test_evaluate_missing_parameter_raises():
    with pytest.raises(cel.CELError):
        cel.evaluate(
            "inputs.parameter_values['pipelinechannel--missing'] == 'x'",
            parameter_values={})


def test_evaluate_rejects_arbitrary_expressions():
    # There is no `__import__` / builtins surface — attempts to reach them
    # should fail at parse/eval time rather than execute.
    with pytest.raises(cel.CELError):
        cel.evaluate("__import__('os').system('true')", parameter_values={})


def test_evaluate_unknown_function():
    with pytest.raises(cel.CELError):
        cel.evaluate('min(1, 2) == 1', parameter_values={})


def test_parser_errors_on_garbage():
    with pytest.raises(cel.CELError):
        cel.evaluate('inputs.parameter_values[ == 1', parameter_values={})
