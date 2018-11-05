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

import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

import kfp.components as comp
from kfp.components._yaml_utils import load_yaml

_this_file = Path(__file__).resolve()
_this_dir = _this_file.parent
_test_data_dir = _this_dir.joinpath('test_data')
_tests_root_dir = _this_dir.parent.parent
_sdk_root_dir = _tests_root_dir.parent

sys.path.insert(0, _sdk_root_dir)

def add_two_numbers(a: float, b: float) -> float:
    '''Returns sum of two arguments'''
    return a + b

@comp.python_op
def add_two_numbers_decorated(a: float, b: float) -> float:
    '''Returns sum of two arguments'''
    return a + b

@comp.python_op(base_image='ark7/python-fire')
def add_two_numbers_decorated_with_parameters(a: float, b: float) -> float:
    '''Returns sum of two arguments'''
    return a + b


class PythonOpTestCase(unittest.TestCase):
    def helper_test_2_in_1_out_component_using_local_call(self, func, op):
        arg1 = float(3)
        arg2 = float(5)

        expected = func(arg1, arg2)
        if isinstance(expected, tuple):
            expected = expected[0]
        expected_str = str(expected)

        output_file_obj = tempfile.NamedTemporaryFile()
        output_path = output_file_obj.name

        task = op(arg1, arg2, output_path)

        full_command = task.command + task.arguments

        process = subprocess.run(full_command)

        actual_str = Path(output_path).read_text()

        self.assertEqual(float(actual_str), float(expected_str))

    def helper_test_2_in_2_out_component_using_local_call(self, func, op):
        arg1 = float(3)
        arg2 = float(5)

        expected_tuple = func(arg1, arg2)
        expected1_str = str(expected_tuple[0])
        expected2_str = str(expected_tuple[1])

        output_path1 = tempfile.NamedTemporaryFile().name
        output_path2 = tempfile.NamedTemporaryFile().name

        task = op(arg1, arg2, output_path1, output_path2)

        full_command = task.command + task.arguments

        process = subprocess.run(full_command)

        actual1_str = Path(output_path1).read_text()
        actual2_str = Path(output_path2).read_text()

        self.assertEqual(float(actual1_str), float(expected1_str))
        self.assertEqual(float(actual2_str), float(expected2_str))

    def test_func_to_container_op_local_call(self):
        func = add_two_numbers
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_decorated_func_to_container_op_local_call(self):
        self.helper_test_2_in_1_out_component_using_local_call(add_two_numbers, add_two_numbers_decorated)

    def test_decorated_with_parameters_func_to_container_op_local_call(self):
        self.helper_test_2_in_1_out_component_using_local_call(add_two_numbers, add_two_numbers_decorated_with_parameters)

    def test_indented_func_to_container_op_local_call(self):
        def add_two_numbers_indented(a: float, b: float) -> float:
            '''Returns sum of two arguments'''
            return a + b

        func = add_two_numbers_indented
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_indented_decorated_func_to_container_op_local_call(self):
        @comp.python_op
        def add_two_numbers_indented_decorated(a: float, b: float) -> float:
            '''Returns sum of two arguments'''
            return a + b

        self.helper_test_2_in_1_out_component_using_local_call(add_two_numbers, add_two_numbers_indented_decorated)

    def test_indented_decorated_with_parameters_func_to_container_op_local_call(self):
        @comp.python_op(base_image='ark7/python-fire')
        def add_two_numbers_indented_decorated_with_parameters(a: float, b: float) -> float:
            '''Returns sum of two arguments'''
            return a + b

        self.helper_test_2_in_1_out_component_using_local_call(add_two_numbers, add_two_numbers_indented_decorated_with_parameters)

    def test_func_to_container_op_multiple_named_typed_outputs(self):
        from typing import NamedTuple
        def add_multiply_two_numbers(a: float, b: float) -> NamedTuple('DummyName', [('sum', float), ('product', float)]):
            '''Returns sum and product of two arguments'''
            return (a + b, a * b)

        func = add_multiply_two_numbers
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_2_out_component_using_local_call(func, op)

    @unittest.skip #TODO: #Simplified multi-output syntax is not implemented yet
    def test_func_to_container_op_multiple_named_typed_outputs_using_list_syntax(self):
        def add_multiply_two_numbers(a: float, b: float) -> [('sum', float), ('product', float)]:
            '''Returns sum and product of two arguments'''
            return (a + b, a * b)

        func = add_multiply_two_numbers
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_2_out_component_using_local_call(func, op)

    def test_func_to_container_op_named_typed_outputs_with_underscores(self):
        from typing import NamedTuple
        def add_two_numbers_name2(a: float, b: float) -> NamedTuple('DummyName', [('output_data', float)]):
            '''Returns sum of two arguments'''
            return (a + b,)

        func = add_two_numbers_name2
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    @unittest.skip #Python does not allow NamedTuple with spaces in names: ValueError: Type names and field names must be valid identifiers: 'Output data'
    def test_func_to_container_op_named_typed_outputs_with_spaces(self):
        from typing import NamedTuple
        def add_two_numbers_name3(a: float, b: float) -> NamedTuple('DummyName', [('Output data', float)]):
            '''Returns sum of two arguments'''
            return (a + b,)

        func = add_two_numbers_name3
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_handling_same_input_output_names(self):
        from typing import NamedTuple
        def add_multiply_two_numbers(a: float, b: float) -> NamedTuple('DummyName', [('a', float), ('b', float)]):
            '''Returns sum and product of two arguments'''
            return (a + b, a * b)

        func = add_multiply_two_numbers
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_2_out_component_using_local_call(func, op)

    def test_handling_same_input_default_output_names(self):
        def add_two_numbers_indented(a: float, Output: float) -> float:
            '''Returns sum of two arguments'''
            return a + Output

        func = add_two_numbers_indented
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)


    def test_end_to_end_python_component_pipeline_compilation(self):
        import kfp.components as comp
        from kfp.components import python_op

        #Defining the Python function
        def add(a: float, b: float) -> float:
            '''Returns sum of two arguments'''
            return a + b

        add_component_file = tempfile.NamedTemporaryFile(prefix='add', suffix='.component.yaml').name
        subtract_component_file = tempfile.NamedTemporaryFile(prefix='subtract', suffix='.component.yaml').name

        #Converting the function to a component. Instantiate it to create a pipeline task (ContaineOp instance)
        add_op = comp.func_to_container_op(add, base_image='ark7/python-fire', output_component_file=add_component_file)

        #Checking that the component artifact is usable:
        add_op2 = comp.load_component_from_file(add_component_file)

        #Using decorator to perform the same thing (convert function to a component):
        @python_op(base_image='ark7/python-fire', output_component_file=subtract_component_file)
        def subtract_op(a: float, b: float) -> float:
            '''Returns difference between two arguments'''
            return a - b

        #Checking that the component artifact is usable:
        subtract_op2 = comp.load_component_from_file(subtract_component_file)

        #Building the pipeline
        import kfp.dsl as dsl
        @dsl.pipeline(
            name='Calculation pipeline',
            description='A pipeline that performs arithmetic calculations.'
        )
        def calc_pipeline(
            a1=dsl.PipelineParam('a1'),
            a2=dsl.PipelineParam('a2', value='7'),
            a3=dsl.PipelineParam('a3', value='17'),
        ):
            task_11 = add_op(a1, a2)
            task_12 = subtract_op(task_11.output, a3)
            task_21 = add_op2(a1, a2)
            task_22 = subtract_op2(task_21.output, a3)
            task_3 = subtract_op(task_12.output, task_22.output)

        #Compiling the pipleine:
        pipeline_filename=calc_pipeline.__name__ + '.pipeline.tar.gz'
        import kfp.compiler as compiler
        compiler.Compiler().compile(calc_pipeline, calc_pipeline.__name__ + '.tar.gz')


if __name__ == '__main__':
    unittest.main()
