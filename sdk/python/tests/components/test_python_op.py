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

import subprocess
import tempfile
import unittest
from contextlib import contextmanager
from pathlib import Path
from typing import Callable

import kfp
import kfp.components as comp

def add_two_numbers(a: float, b: float) -> float:
    '''Returns sum of two arguments'''
    return a + b


@contextmanager
def components_local_output_dir_context(output_dir: str):
    old_dir = comp._components._outputs_dir
    try:
        comp._components._outputs_dir = output_dir
        yield output_dir
    finally:
        comp._components._outputs_dir = old_dir


@contextmanager
def components_override_input_output_dirs_context(inputs_dir: str, outputs_dir: str):
    old_inputs_dir = comp._components._inputs_dir
    old_outputs_dir = comp._components._outputs_dir
    try:
        if inputs_dir:
            comp._components._inputs_dir = inputs_dir
        if outputs_dir:
            comp._components._outputs_dir = outputs_dir
        yield
    finally:
        comp._components._inputs_dir = old_inputs_dir
        comp._components._outputs_dir = old_outputs_dir


module_level_variable = 10


class ModuleLevelClass:
    def class_method(self, x):
        return x * module_level_variable


def module_func(a: float) -> float:
    return a * 5


def module_func_with_deps(a: float, b: float) -> float:
    return ModuleLevelClass().class_method(a) + module_func(b)


class PythonOpTestCase(unittest.TestCase):
    def helper_test_2_in_1_out_component_using_local_call(self, func, op, arguments=[3., 5.]):
        expected = func(arguments[0], arguments[1])
        if isinstance(expected, tuple):
            expected = expected[0]
        expected_str = str(expected)

        with tempfile.TemporaryDirectory() as temp_dir_name:
            with components_local_output_dir_context(temp_dir_name):
                task = op(arguments[0], arguments[1])

            full_command = task.command + task.arguments
            subprocess.run(full_command, check=True)

            output_path = list(task.file_outputs.values())[0]
            actual_str = Path(output_path).read_text()

        self.assertEqual(float(actual_str), float(expected_str))

    def helper_test_2_in_2_out_component_using_local_call(self, func, op, output_names):
        arg1 = float(3)
        arg2 = float(5)

        expected_tuple = func(arg1, arg2)
        expected1_str = str(expected_tuple[0])
        expected2_str = str(expected_tuple[1])

        with tempfile.TemporaryDirectory() as temp_dir_name:
            with components_local_output_dir_context(temp_dir_name):
                task = op(arg1, arg2)

            full_command = task.command + task.arguments

            subprocess.run(full_command, check=True)

            (output_path1, output_path2) = (task.file_outputs[output_names[0]], task.file_outputs[output_names[1]])
            actual1_str = Path(output_path1).read_text()
            actual2_str = Path(output_path2).read_text()

        self.assertEqual(float(actual1_str), float(expected1_str))
        self.assertEqual(float(actual2_str), float(expected2_str))

    def helper_test_component_created_from_func_using_local_call(self, func: Callable, arguments: dict):
        task_factory = comp.func_to_container_op(func)
        self.helper_test_component_against_func_using_local_call(func, task_factory, arguments)

    def helper_test_component_against_func_using_local_call(self, func: Callable, op: Callable, arguments: dict):
        # ! This function cannot be used when component has output types that use custom serialization since it will compare non-serialized function outputs with serialized component outputs.
        # Evaluating the function to get the expected output values
        expected_output_values_list = func(**arguments)
        expected_output_values_list = [str(value) for value in expected_output_values_list]

        output_names = [output.name for output in op.outputs]
        from kfp.components._naming import generate_unique_name_conversion_table, _sanitize_python_function_name
        output_name_to_pythonic = generate_unique_name_conversion_table(output_names, _sanitize_python_function_name)
        pythonic_output_names = [output_name_to_pythonic[name] for name in output_names]
        from collections import OrderedDict
        expected_output_values_dict = OrderedDict(zip(pythonic_output_names, expected_output_values_list))

        self.helper_test_component_using_local_call(op, arguments, expected_output_values_dict)

    def helper_test_component_using_local_call(self, component_task_factory: Callable, arguments: dict, expected_output_values: dict):
        with tempfile.TemporaryDirectory() as temp_dir_name:
            # Creating task from the component.
            # We do it in a special context that allows us to control the output file locations.
            inputs_path = Path(temp_dir_name) / 'inputs'
            outputs_path = Path(temp_dir_name) / 'outputs'
            with components_override_input_output_dirs_context(str(inputs_path), str(outputs_path)):
                task = component_task_factory(**arguments)

            # Preparing input files
            for input_name, input_file_path in (task.input_artifact_paths or {}).items():
                Path(input_file_path).parent.mkdir(parents=True, exist_ok=True)
                Path(input_file_path).write_text(str(arguments[input_name]))

            # Constructing the full command-line from resolved command+args
            full_command = task.command + task.arguments

            # Executing the command-line locally
            subprocess.run(full_command, check=True)

            actual_output_values_dict = {output_name: Path(output_path).read_text() for output_name, output_path in task.file_outputs.items()}

        self.assertEqual(actual_output_values_dict, expected_output_values)

    def test_func_to_container_op_local_call(self):
        func = add_two_numbers
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_func_to_container_op_output_component_file(self):
        func = add_two_numbers
        with tempfile.TemporaryDirectory() as temp_dir_name:
            component_path = str(Path(temp_dir_name) / 'component.yaml')
            comp.func_to_container_op(func, output_component_file=component_path)
            op = comp.load_component_from_file(component_path)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_func_to_component_file(self):
        func = add_two_numbers
        with tempfile.TemporaryDirectory() as temp_dir_name:
            component_path = str(Path(temp_dir_name) / 'component.yaml')
            comp._python_op.func_to_component_file(func, output_component_file=component_path)
            op = comp.load_component_from_file(component_path)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_indented_func_to_container_op_local_call(self):
        def add_two_numbers_indented(a: float, b: float) -> float:
            '''Returns sum of two arguments'''
            return a + b

        func = add_two_numbers_indented
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_func_to_container_op_call_other_func(self):
        extra_variable = 10

        class ExtraClass:
            def class_method(self, x):
                return x * extra_variable

        def extra_func(a: float) -> float:
            return a * 5

        def main_func(a: float, b: float) -> float:
            return ExtraClass().class_method(a) + extra_func(b)

        func = main_func
        op = comp.func_to_container_op(func, use_code_pickling=True)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_func_to_container_op_check_nothing_extra_captured(self):
        def f1():
            pass

        def f2():
            pass

        def main_func(a: float, b: float) -> float:
            f1()
            try:
                eval('f2()')
            except:
                return a + b
            raise AssertionError("f2 should not be captured, because it's not a dependency.")

        expected_func = lambda a, b: a + b
        op = comp.func_to_container_op(main_func, use_code_pickling=True)

        self.helper_test_2_in_1_out_component_using_local_call(expected_func, op)

    def test_func_to_container_op_call_other_func_global(self):
        func = module_func_with_deps
        op = comp.func_to_container_op(func, use_code_pickling=True)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_func_to_container_op_with_imported_func(self):
        from .test_data.module1 import module_func_with_deps as module1_func_with_deps
        func = module1_func_with_deps
        op = comp.func_to_container_op(func, use_code_pickling=True)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_func_to_container_op_with_imported_func2(self):
        from .test_data.module2_which_depends_on_module1 import module2_func_with_deps as module2_func_with_deps
        func = module2_func_with_deps
        op = comp.func_to_container_op(func, use_code_pickling=True, modules_to_capture=[
            'tests.components.test_data.module1',
            'tests.components.test_data.module2_which_depends_on_module1'
        ])

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_func_to_container_op_multiple_named_typed_outputs(self):
        from typing import NamedTuple
        def add_multiply_two_numbers(a: float, b: float) -> NamedTuple('DummyName', [('sum', float), ('product', float)]):
            '''Returns sum and product of two arguments'''
            return (a + b, a * b)

        func = add_multiply_two_numbers
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_2_out_component_using_local_call(func, op, output_names=['sum', 'product'])

    def test_extract_component_interface(self):
        from typing import NamedTuple
        def my_func( # noqa: F722
            required_param,
            int_param: int = 42,
            float_param : float = 3.14,
            str_param : str = 'string',
            bool_param : bool = True,
            none_param = None,
            custom_type_param: 'Custom type' = None,
            ) -> NamedTuple('DummyName', [
                #('required_param',), # All typing.NamedTuple fields must have types
                ('int_param', int),
                ('float_param', float),
                ('str_param', str),
                ('bool_param', bool),
                #('custom_type_param', 'Custom type'), #SyntaxError: Forward reference must be an expression -- got 'Custom type'
                ('custom_type_param', 'CustomType'),
            ]
        ):
            '''Function docstring'''
            pass

        component_spec = comp._python_op._extract_component_interface(my_func)

        from kfp.components._structures import InputSpec, OutputSpec
        self.assertEqual(
            component_spec.inputs,
            [
                InputSpec(name='required_param'),
                InputSpec(name='int_param', type='Integer', default='42', optional=True),
                InputSpec(name='float_param', type='Float', default='3.14', optional=True),
                InputSpec(name='str_param', type='String', default='string', optional=True),
                InputSpec(name='bool_param', type='Boolean', default='True', optional=True),
                InputSpec(name='none_param', optional=True), # No default='None'
                InputSpec(name='custom_type_param', type='Custom type', optional=True),
            ]
        )
        self.assertEqual(
            component_spec.outputs,
            [
                OutputSpec(name='int_param', type='Integer'),
                OutputSpec(name='float_param', type='Float'),
                OutputSpec(name='str_param', type='String'),
                OutputSpec(name='bool_param', type='Boolean'),
                #OutputSpec(name='custom_type_param', type='Custom type', default='None'),
                OutputSpec(name='custom_type_param', type='CustomType'),
            ]
        )

        self.maxDiff = None
        self.assertDictEqual(
            component_spec.to_dict(),
            {
                'name': 'My func',
                'description': 'Function docstring\n',
                'inputs': [
                    {'name': 'required_param'},
                    {'name': 'int_param', 'type': 'Integer', 'default': '42', 'optional': True},
                    {'name': 'float_param', 'type': 'Float', 'default': '3.14', 'optional': True},
                    {'name': 'str_param', 'type': 'String', 'default': 'string', 'optional': True},
                    {'name': 'bool_param', 'type': 'Boolean', 'default': 'True', 'optional': True},
                    {'name': 'none_param', 'optional': True}, # No default='None'
                    {'name': 'custom_type_param', 'type': 'Custom type', 'optional': True},
                ],
                'outputs': [
                    {'name': 'int_param', 'type': 'Integer'},
                    {'name': 'float_param', 'type': 'Float'},
                    {'name': 'str_param', 'type': 'String'},
                    {'name': 'bool_param', 'type': 'Boolean'},
                    {'name': 'custom_type_param', 'type': 'CustomType'},
                ]
            }
        )

    @unittest.skip #TODO: #Simplified multi-output syntax is not implemented yet
    def test_func_to_container_op_multiple_named_typed_outputs_using_list_syntax(self):
        def add_multiply_two_numbers(a: float, b: float) -> [('sum', float), ('product', float)]:
            '''Returns sum and product of two arguments'''
            return (a + b, a * b)

        func = add_multiply_two_numbers
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_2_out_component_using_local_call(func, op, output_names=['sum', 'product'])

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

        self.helper_test_2_in_2_out_component_using_local_call(func, op, output_names=['a', 'b'])

    def test_handling_same_input_default_output_names(self):
        def add_two_numbers_indented(a: float, Output: float) -> float:
            '''Returns sum of two arguments'''
            return a + Output

        func = add_two_numbers_indented
        op = comp.func_to_container_op(func)

        self.helper_test_2_in_1_out_component_using_local_call(func, op)

    def test_python_component_decorator(self):
        from kfp.dsl import python_component
        import kfp.components._python_op as _python_op

        expected_name = 'Sum component name'
        expected_description = 'Sum component description'
        expected_image = 'org/image'

        @python_component(name=expected_name, description=expected_description, base_image=expected_image)
        def add_two_numbers_decorated(a: float, b: float) -> float:
            '''Returns sum of two arguments'''
            return a + b

        component_spec = _python_op._func_to_component_spec(add_two_numbers_decorated)

        self.assertEqual(component_spec.name, expected_name)
        self.assertEqual(component_spec.description.strip(), expected_description.strip())
        self.assertEqual(component_spec.implementation.container.image, expected_image)

    def test_saving_default_values(self):
        from typing import NamedTuple
        def add_multiply_two_numbers(a: float = 3, b: float = 5) -> NamedTuple('DummyName', [('sum', float), ('product', float)]):
            '''Returns sum and product of two arguments'''
            return (a + b, a * b)

        func = add_multiply_two_numbers
        component_spec = comp._python_op._func_to_component_spec(func)

        self.assertEqual(component_spec.inputs[0].default, '3')
        self.assertEqual(component_spec.inputs[1].default, '5')

    def test_handling_default_value_of_none(self):
        def assert_is_none(a, b, arg=None) -> int:
            assert arg is None
            return 1

        func = assert_is_none
        op = comp.func_to_container_op(func)
        self.helper_test_2_in_1_out_component_using_local_call(func, op)


    def test_handling_complex_default_values_of_none(self):
        def assert_values_are_default(
            a, b,
            singleton_param=None,
            function_param=ascii,
            dict_param={'b': [2, 3, 4]},
            func_call_param='_'.join(['a', 'b', 'c']),
        ) -> int:
            assert singleton_param is None
            assert function_param is ascii
            assert dict_param == {'b': [2, 3, 4]}
            assert func_call_param == '_'.join(['a', 'b', 'c'])

            return 1

        func = assert_values_are_default
        op = comp.func_to_container_op(func)
        self.helper_test_2_in_1_out_component_using_local_call(func, op)


    def test_handling_boolean_arguments(self):
        def assert_values_are_true_false(
            bool1 : bool,
            bool2 : bool,
        ) -> int:
            assert bool1 is True
            assert bool2 is False
            return 1

        func = assert_values_are_true_false
        op = comp.func_to_container_op(func)
        self.helper_test_2_in_1_out_component_using_local_call(func, op, arguments=[True, False])


    def test_handling_list_dict_arguments(self):
        def assert_values_are_same(
            list_param: list,
            dict_param: dict,
        ) -> int:
            import unittest
            unittest.TestCase().assertEqual(list_param, ["string", 1, 2.2, True, False, None, [3, 4], {'s': 5}])
            unittest.TestCase().assertEqual(dict_param, {'str': "string", 'int': 1, 'float':  2.2, 'false': False, 'true': True, 'none': None, 'list': [3, 4], 'dict': {'s': 4}})
            return 1
        
        # ! JSON map keys are always strings. Python converts all keys to strings without warnings
        func = assert_values_are_same
        op = comp.func_to_container_op(func)
        self.helper_test_2_in_1_out_component_using_local_call(func, op, arguments=[
            ["string", 1, 2.2, True, False, None, [3, 4], {'s': 5}],
            {'str': "string", 'int': 1, 'float':  2.2, 'false': False, 'true': True, 'none': None, 'list': [3, 4], 'dict': {'s': 4}},
        ])


    def test_handling_list_arguments_containing_pipelineparam(self):
        '''Checks that lists containing PipelineParam can be properly serialized'''
        def consume_list(list_param: list) -> int:
            pass

        import kfp
        task_factory = comp.func_to_container_op(consume_list)
        task = task_factory([1, 2, 3, kfp.dsl.PipelineParam("aaa"), 4, 5, 6])
        full_command_line = task.command + task.arguments
        for arg in full_command_line:
            self.assertNotIn('PipelineParam', arg)


    def test_handling_base64_pickle_arguments(self):
        def assert_values_are_same(
            obj1: 'Base64Pickle', # noqa: F821
            obj2: 'Base64Pickle', # noqa: F821
        ) -> int:
            import unittest
            unittest.TestCase().assertEqual(obj1['self'], obj1)
            unittest.TestCase().assertEqual(obj2, open)
            return 1
        
        func = assert_values_are_same
        op = comp.func_to_container_op(func)

        recursive_obj = {}
        recursive_obj['self'] = recursive_obj
        self.helper_test_2_in_1_out_component_using_local_call(func, op, arguments=[
            recursive_obj,
            open,
        ])


    def test_handling_list_dict_output_values(self):
        def produce_list() -> list:
            return (["string", 1, 2.2, True, False, None, [3, 4], {'s': 5}], )
        
        # ! JSON map keys are always strings. Python converts all keys to strings without warnings
        task_factory = comp.func_to_container_op(produce_list)

        import json
        expected_output = json.dumps(["string", 1, 2.2, True, False, None, [3, 4], {'s': 5}])

        self.helper_test_component_using_local_call(task_factory, arguments={}, expected_output_values={'output': expected_output})


    def test_input_path(self):
        from kfp.components import InputPath
        def consume_file_path(number_file_path: InputPath(int)) -> int:
            with open(number_file_path) as f:
                string_data = f.read()
            return int(string_data)

        task_factory = comp.func_to_container_op(consume_file_path)

        self.assertEqual(task_factory.component_spec.inputs[0].type, 'Integer')

        # TODO: Fix the input names: "number_file_path" parameter should be exposed as "number" input
        self.helper_test_component_using_local_call(task_factory, arguments={'number_file_path': "42"}, expected_output_values={'output': '42'})


    def test_input_text_file(self):
        from kfp.components import InputTextFile
        def consume_file_path(number_file: InputTextFile(int)) -> int:
            string_data = number_file.read()
            assert isinstance(string_data, str)
            return int(string_data)

        task_factory = comp.func_to_container_op(consume_file_path)

        self.assertEqual(task_factory.component_spec.inputs[0].type, 'Integer')

        # TODO: Fix the input names: "number_file" parameter should be exposed as "number" input
        self.helper_test_component_using_local_call(task_factory, arguments={'number_file': "42"}, expected_output_values={'output': '42'})


    def test_input_binary_file(self):
        from kfp.components import InputBinaryFile
        def consume_file_path(number_file: InputBinaryFile(int)) -> int:
            bytes_data = number_file.read()
            assert isinstance(bytes_data, bytes)
            return int(bytes_data)

        task_factory = comp.func_to_container_op(consume_file_path)

        self.assertEqual(task_factory.component_spec.inputs[0].type, 'Integer')

        # TODO: Fix the input names: "number_file" parameter should be exposed as "number" input
        self.helper_test_component_using_local_call(task_factory, arguments={'number_file': "42"}, expected_output_values={'output': '42'})


    def test_output_path(self):
        from kfp.components import OutputPath
        def write_to_file_path(number_file_path: OutputPath(int)):
            with open(number_file_path, 'w') as f:
                f.write(str(42))

        task_factory = comp.func_to_container_op(write_to_file_path)

        self.assertFalse(task_factory.component_spec.inputs)
        self.assertEqual(len(task_factory.component_spec.outputs), 1)
        self.assertEqual(task_factory.component_spec.outputs[0].type, 'Integer')

        # TODO: Fix the output names: "number_file_path" should be exposed as "number" output
        self.helper_test_component_using_local_call(task_factory, arguments={}, expected_output_values={'number_file_path': '42'})


    def test_output_text_file(self):
        from kfp.components import OutputTextFile
        def write_to_file_path(number_file: OutputTextFile(int)):
            number_file.write(str(42))

        task_factory = comp.func_to_container_op(write_to_file_path)

        self.assertFalse(task_factory.component_spec.inputs)
        self.assertEqual(len(task_factory.component_spec.outputs), 1)
        self.assertEqual(task_factory.component_spec.outputs[0].type, 'Integer')

        # TODO: Fix the output names: "number_file" should be exposed as "number" output
        self.helper_test_component_using_local_call(task_factory, arguments={}, expected_output_values={'number_file': '42'})


    def test_output_binary_file(self):
        from kfp.components import OutputBinaryFile
        def write_to_file_path(number_file: OutputBinaryFile(int)):
            number_file.write(b'42')

        task_factory = comp.func_to_container_op(write_to_file_path)

        self.assertFalse(task_factory.component_spec.inputs)
        self.assertEqual(len(task_factory.component_spec.outputs), 1)
        self.assertEqual(task_factory.component_spec.outputs[0].type, 'Integer')

        # TODO: Fix the output names: "number_file" should be exposed as "number" output
        self.helper_test_component_using_local_call(task_factory, arguments={}, expected_output_values={'number_file': '42'})


    def test_end_to_end_python_component_pipeline_compilation(self):
        import kfp.components as comp

        #Defining the Python function
        def add(a: float, b: float) -> float:
            '''Returns sum of two arguments'''
            return a + b

        with tempfile.TemporaryDirectory() as temp_dir_name:
            add_component_file = str(Path(temp_dir_name).joinpath('add.component.yaml'))

            #Converting the function to a component. Instantiate it to create a pipeline task (ContaineOp instance)
            add_op = comp.func_to_container_op(add, base_image='python:3.5', output_component_file=add_component_file)

            #Checking that the component artifact is usable:
            add_op2 = comp.load_component_from_file(add_component_file)

            #Building the pipeline
            import kfp.dsl as dsl
            @dsl.pipeline(
                name='Calculation pipeline',
                description='A pipeline that performs arithmetic calculations.'
            )
            def calc_pipeline(
                a1,
                a2='7',
                a3='17',
            ):
                task_1 = add_op(a1, a2)
                task_2 = add_op2(a1, a2)
                task_3 = add_op(task_1.output, task_2.output)
                task_4 = add_op2(task_3.output, a3)

            #Compiling the pipleine:
            pipeline_filename = str(Path(temp_dir_name).joinpath(calc_pipeline.__name__ + '.pipeline.tar.gz'))
            import kfp.compiler as compiler
            compiler.Compiler().compile(calc_pipeline, pipeline_filename)


if __name__ == '__main__':
    unittest.main()
