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
import sys
import unittest
from contextlib import contextmanager
from pathlib import Path


import kfp
import kfp.components as comp
from kfp.components._yaml_utils import load_yaml
from kfp.dsl.types import InconsistentTypeException


@contextmanager
def no_task_resolving_context():
    old_handler = kfp.components._components._container_task_constructor
    try:
        kfp.components._components._container_task_constructor = kfp.components._components._create_task_spec_from_component_and_arguments
        yield None
    finally:
        kfp.components._components._container_task_constructor = old_handler

class LoadComponentTestCase(unittest.TestCase):
    def setUp(self):
        self.old_container_task_constructor = kfp.components._components._container_task_constructor
        kfp.components._components._container_task_constructor = kfp.components._dsl_bridge._create_container_op_from_component_and_arguments

    def tearDown(self):
        kfp.components._components._container_task_constructor = self.old_container_task_constructor

    def _test_load_component_from_file(self, component_path: str):
        task_factory1 = comp.load_component_from_file(component_path)

        arg1 = 3
        arg2 = 5
        task1 = task_factory1(arg1, arg2)

        self.assertEqual(task1.human_name, 'Add')
        self.assertEqual(task_factory1.__doc__.strip(), 'Add\nReturns sum of two arguments')
        self.assertEqual(task1.container.image, 'python:3.5')
        self.assertEqual(task1.container.args[0], str(arg1))
        self.assertEqual(task1.container.args[1], str(arg2))

    def test_load_component_from_yaml_file(self):
        _this_file = Path(__file__).resolve()
        _this_dir = _this_file.parent
        _test_data_dir = _this_dir.joinpath('test_data')
        component_path = _test_data_dir.joinpath('python_add.component.yaml')
        self._test_load_component_from_file(str(component_path))

    def test_load_component_from_zipped_yaml_file(self):
        _this_file = Path(__file__).resolve()
        _this_dir = _this_file.parent
        _test_data_dir = _this_dir.joinpath('test_data')
        component_path = _test_data_dir.joinpath('python_add.component.zip')
        self._test_load_component_from_file(str(component_path))

    def test_load_component_from_url(self):
        url = 'https://raw.githubusercontent.com/kubeflow/pipelines/e54fe675432cfef1d115a7a2909f08ed95ea8933/sdk/python/tests/components/test_data/python_add.component.yaml'

        import requests
        resp = requests.get(url)
        component_text = resp.content
        component_dict = load_yaml(component_text)
        task_factory1 = comp.load_component_from_url(url)
        self.assertEqual(task_factory1.__doc__, component_dict['name'] + '\n' + component_dict['description'])

        arg1 = 3
        arg2 = 5
        task1 = task_factory1(arg1, arg2)
        self.assertEqual(task1.human_name, component_dict['name'])
        self.assertEqual(task1.container.image, component_dict['implementation']['container']['image'])

        self.assertEqual(task1.arguments[0], str(arg1))
        self.assertEqual(task1.arguments[1], str(arg2))

    def test_loading_minimal_component(self):
        component_text = '''\
implementation:
  container:
    image: busybox
'''
        component_dict = load_yaml(component_text)
        task_factory1 = comp.load_component(text=component_text)

        task1 = task_factory1()
        self.assertEqual(task1.container.image, component_dict['implementation']['container']['image'])

    def test_accessing_component_spec_from_task_factory(self):
        component_text = '''\
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

        actual_component_spec = task_factory1.component_spec
        actual_component_spec_dict = actual_component_spec.to_dict()
        expected_component_spec_dict = load_yaml(component_text)
        expected_component_spec = kfp.components.structures.ComponentSpec.from_dict(expected_component_spec_dict)
        self.assertEqual(expected_component_spec_dict, actual_component_spec_dict)
        self.assertEqual(expected_component_spec, task_factory1.component_spec)

    def test_fail_on_duplicate_input_names(self):
        component_text = '''\
inputs:
- {name: Data1}
- {name: Data1}
implementation:
  container:
    image: busybox
'''
        with self.assertRaises(ValueError):
            task_factory1 = comp.load_component_from_text(component_text)

    def test_fail_on_duplicate_output_names(self):
        component_text = '''\
outputs:
- {name: Data1}
- {name: Data1}
implementation:
  container:
    image: busybox
'''
        with self.assertRaises(ValueError):
            task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_underscored_input_names(self):
        component_text = '''\
inputs:
- {name: Data}
- {name: _Data}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_underscored_output_names(self):
        component_text = '''\
outputs:
- {name: Data}
- {name: _Data}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_input_names_with_spaces(self):
        component_text = '''\
inputs:
- {name: Training data}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_output_names_with_spaces(self):
        component_text = '''\
outputs:
- {name: Training data}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_file_outputs_with_spaces(self):
        component_text = '''\
outputs:
- {name: Output data}
implementation:
  container:
    image: busybox
    fileOutputs:
      Output data: /outputs/output-data
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_similar_input_names(self):
        component_text = '''\
inputs:
- {name: Input 1}
- {name: Input_1}
- {name: Input-1}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_duplicate_input_output_names(self):
        component_text = '''\
inputs:
- {name: Data}
outputs:
- {name: Data}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_fail_on_unknown_value_argument(self):
        component_text = '''\
inputs:
- {name: Data}
implementation:
  container:
    image: busybox
    args:
      - {inputValue: Wrong}
'''
        with self.assertRaises(TypeError):
            task_factory1 = comp.load_component_from_text(component_text)

    def test_fail_on_unknown_file_output(self):
        component_text = '''\
outputs:
- {name: Data}
implementation:
  container:
    image: busybox
    fileOutputs:
        Wrong: '/outputs/output.txt'
'''
        with self.assertRaises(TypeError):
            task_factory1 = comp.load_component_from_text(component_text)

    def test_load_component_fail_on_no_sources(self):
        with self.assertRaises(ValueError):
            comp.load_component()

    def test_load_component_fail_on_multiple_sources(self):
        with self.assertRaises(ValueError):
            comp.load_component(filename='', text='')

    def test_load_component_fail_on_none_arguments(self):
        with self.assertRaises(ValueError):
            comp.load_component(filename=None, url=None, text=None)

    def test_load_component_from_file_fail_on_none_arg(self):
        with self.assertRaises(TypeError):
            comp.load_component_from_file(None)

    def test_load_component_from_url_fail_on_none_arg(self):
        with self.assertRaises(TypeError):
            comp.load_component_from_url(None)

    def test_load_component_from_text_fail_on_none_arg(self):
        with self.assertRaises(TypeError):
            comp.load_component_from_text(None)

    def test_input_value_resolving(self):
        component_text = '''\
inputs:
- {name: Data}
implementation:
  container:
    image: busybox
    args:
      - --data
      - inputValue: Data
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1('some-data')

        self.assertEqual(task1.arguments, ['--data', 'some-data'])

    def test_automatic_output_resolving(self):
        component_text = '''\
outputs:
- {name: Data}
implementation:
  container:
    image: busybox
    args:
      - --output-data
      - {outputPath: Data}
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1()

        self.assertEqual(len(task1.arguments), 2)
        self.assertEqual(task1.arguments[0], '--output-data')
        self.assertTrue(task1.arguments[1].startswith('/'))

    def test_input_path_placeholder_with_constant_argument(self):
        component_text = '''\
inputs:
- {name: input 1}
implementation:
  container:
    image: busybox
    command:
      - --input-data
      - {inputPath: input 1}
'''
        task_factory1 = comp.load_component_from_text(component_text)
        task1 = task_factory1('Text')

        self.assertEqual(task1.command, ['--input-data', task1.input_artifact_paths['input 1']])
        self.assertEqual(task1.artifact_arguments, {'input 1': 'Text'})

    def test_optional_inputs_reordering(self):
        '''Tests optional input reordering.
        In python signature, optional arguments must come after the required arguments.
        '''
        component_text = '''\
inputs:
- {name: in1}
- {name: in2, optional: true}
- {name: in3}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)
        import inspect
        signature = inspect.signature(task_factory1)
        actual_signature = list(signature.parameters.keys())
        self.assertSequenceEqual(actual_signature, ['in1', 'in3', 'in2'], str)

    def test_inputs_reordering_when_inputs_have_defaults(self):
        '''Tests reordering of inputs with default values.
        In python signature, optional arguments must come after the required arguments.
        '''
        component_text = '''\
inputs:
- {name: in1}
- {name: in2, default: val}
- {name: in3}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)
        import inspect
        signature = inspect.signature(task_factory1)
        actual_signature = list(signature.parameters.keys())
        self.assertSequenceEqual(actual_signature, ['in1', 'in3', 'in2'], str)

    def test_inputs_reordering_stability(self):
        '''Tests input reordering stability. Required inputs and optional/default inputs should keep the ordering.
        In python signature, optional arguments must come after the required arguments.
        '''
        component_text = '''\
inputs:
- {name: a1}
- {name: b1, default: val}
- {name: a2}
- {name: b2, optional: True}
- {name: a3}
- {name: b3, default: val}
- {name: a4}
- {name: b4, optional: True}
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)
        import inspect
        signature = inspect.signature(task_factory1)
        actual_signature = list(signature.parameters.keys())
        self.assertSequenceEqual(actual_signature, ['a1', 'a2', 'a3', 'a4', 'b1', 'b2', 'b3', 'b4'], str)

    def test_missing_optional_input_value_argument(self):
        '''Missing optional inputs should resolve to nothing'''
        component_text = '''\
inputs:
- {name: input 1, optional: true}
implementation:
  container:
    image: busybox
    command:
      - a
      - {inputValue: input 1}
      - z
'''
        task_factory1 = comp.load_component_from_text(component_text)
        task1 = task_factory1()

        self.assertEqual(task1.command, ['a', 'z'])

    def test_missing_optional_input_file_argument(self):
        '''Missing optional inputs should resolve to nothing'''
        component_text = '''\
inputs:
- {name: input 1, optional: true}
implementation:
  container:
    image: busybox
    command:
      - a
      - {inputPath: input 1}
      - z
'''
        task_factory1 = comp.load_component_from_text(component_text)
        task1 = task_factory1()

        self.assertEqual(task1.command, ['a', 'z'])

    def test_command_concat(self):
        component_text = '''\
inputs:
- {name: In1}
- {name: In2}
implementation:
  container:
    image: busybox
    args:
      - concat: [{inputValue: In1}, {inputValue: In2}]
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1('some', 'data')

        self.assertEqual(task1.arguments, ['somedata'])

    def test_command_if_boolean_true_then_else(self):
        component_text = '''\
implementation:
  container:
    image: busybox
    args:
      - if:
          cond: true
          then: --true-arg
          else: --false-arg
'''
        task_factory1 = comp.load_component(text=component_text)
        task = task_factory1()
        self.assertEqual(task.arguments, ['--true-arg']) 

    def test_command_if_boolean_false_then_else(self):
        component_text = '''\
implementation:
  container:
    image: busybox
    args:
      - if:
          cond: false
          then: --true-arg
          else: --false-arg
'''
        task_factory1 = comp.load_component(text=component_text)
        task = task_factory1()
        self.assertEqual(task.arguments, ['--false-arg']) 

    def test_command_if_true_string_then_else(self):
        component_text = '''\
implementation:
  container:
    image: busybox
    args:
      - if:
          cond: 'true'
          then: --true-arg
          else: --false-arg
'''
        task_factory1 = comp.load_component(text=component_text)
        task = task_factory1()
        self.assertEqual(task.arguments, ['--true-arg']) 

    def test_command_if_false_string_then_else(self):
        component_text = '''\
implementation:
  container:
    image: busybox
    args:
      - if:
          cond: 'false'
          then: --true-arg
          else: --false-arg
'''
        task_factory1 = comp.load_component(text=component_text)

        task = task_factory1()
        self.assertEqual(task.arguments, ['--false-arg']) 

    def test_command_if_is_present_then(self):
        component_text = '''\
inputs:
- {name: In, optional: true}
implementation:
  container:
    image: busybox
    args:
      - if:
          cond: {isPresent: In}
          then: [--in, {inputValue: In}]
          #else: --no-in
'''
        task_factory1 = comp.load_component(text=component_text)

        task_then = task_factory1('data')
        self.assertEqual(task_then.arguments, ['--in', 'data']) 
        
        task_else = task_factory1()
        self.assertEqual(task_else.arguments, [])

    def test_command_if_is_present_then_else(self):
        component_text = '''\
inputs:
- {name: In, optional: true}
implementation:
  container:
    image: busybox
    args:
      - if:
          cond: {isPresent: In}
          then: [--in, {inputValue: In}]
          else: --no-in
'''
        task_factory1 = comp.load_component(text=component_text)

        task_then = task_factory1('data')
        self.assertEqual(task_then.arguments, ['--in', 'data']) 
        
        task_else = task_factory1()
        self.assertEqual(task_else.arguments, ['--no-in'])


    def test_command_if_input_value_then(self):
        component_text = '''\
inputs:
- {name: Do test, type: Boolean, optional: true}
- {name: Test data, type: Integer, optional: true}
- {name: Test parameter 1, optional: true}
implementation:
  container:
    image: busybox
    args:
      - if:
          cond: {inputValue: Do test}
          then: [--test-data, {inputValue: Test data}, --test-param1, {inputValue: Test parameter 1}]
'''
        task_factory1 = comp.load_component(text=component_text)

        task_then = task_factory1(True, 'test_data.txt', '42')
        self.assertEqual(task_then.arguments, ['--test-data', 'test_data.txt', '--test-param1', '42'])
        
        task_else = task_factory1()
        self.assertEqual(task_else.arguments, [])

    def test_handling_env(self):
        component_text = '''\
implementation:
  container:
    image: busybox
    env:
      key1: value 1
      key2: value 2
'''
        task_factory1 = comp.load_component_from_text(component_text)
        
        task1 = task_factory1()
        actual_env = {env_var.name: env_var.value for env_var in task1.container.env}
        expected_env = {'key1': 'value 1', 'key2': 'value 2'}
        self.assertDictEqual(expected_env, actual_env)

    def test_handle_default_values_in_task_factory(self):
        component_text = '''\
inputs:
- {name: Data, default: '123'}
implementation:
  container:
    image: busybox
    args:
      - {inputValue: Data}
'''
        task_factory1 = comp.load_component_from_text(text=component_text)

        task1 = task_factory1()
        self.assertEqual(task1.arguments, ['123'])

        task2 = task_factory1('456')
        self.assertEqual(task2.arguments, ['456'])

    def test_passing_component_metadata_to_container_op(self):
        component_text = '''\
metadata:
  annotations:
    key1: value1
  labels:
    key1: value1
implementation:
  container:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(text=component_text)

        task1 = task_factory1()
        self.assertEqual(task1.pod_annotations['key1'], 'value1')
        self.assertEqual(task1.pod_labels['key1'], 'value1')

    def test_check_task_spec_outputs_dictionary(self):
        component_text = '''\
outputs:
- {name: out 1}
- {name: out 2}
implementation:
  container:
    image: busybox
    command: [touch, {outputPath: out 1}, {outputPath: out 2}]
'''
        op = comp.load_component_from_text(component_text)
        with no_task_resolving_context():
          task = op()

        self.assertEqual(list(task.outputs.keys()), ['out 1', 'out 2'])

    def test_check_type_validation_of_task_spec_outputs(self):
        producer_component_text = '''\
outputs:
- {name: out1, type: Integer}
- {name: out2, type: String}
implementation:
  container:
    image: busybox
    command: [touch, {outputPath: out1}, {outputPath: out2}]
'''
        consumer_component_text = '''\
inputs:
- {name: data, type: Integer}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: data}]
'''
        producer_op = comp.load_component_from_text(producer_component_text)
        consumer_op = comp.load_component_from_text(consumer_component_text)
        with no_task_resolving_context():
          producer_task = producer_op()

          consumer_op(producer_task.outputs['out1'])
          consumer_op(producer_task.outputs['out2'].without_type())
          consumer_op(producer_task.outputs['out2'].with_type('Integer'))
          with self.assertRaises(TypeError):
            consumer_op(producer_task.outputs['out2'])

    def test_type_compatibility_check_for_simple_types(self):
        component_a = '''\
outputs:
  - {name: out1, type: custom_type}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: custom_type}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_type_compatibility_check_for_types_with_parameters(self):
        component_a = '''\
outputs:
  - {name: out1, type: {parametrized_type: {property_a: value_a, property_b: value_b}}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {parametrized_type: {property_a: value_a, property_b: value_b}}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_type_compatibility_check_when_using_positional_arguments(self):
        """Tests that `op2(task1.output)` works as good as `op2(in1=task1.output)`"""
        component_a = '''\
outputs:
  - {name: out1, type: {parametrized_type: {property_a: value_a, property_b: value_b}}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {parametrized_type: {property_a: value_a, property_b: value_b}}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(a_task.outputs['out1'])

    def test_type_compatibility_check_when_input_type_is_missing(self):
        component_a = '''\
outputs:
  - {name: out1, type: custom_type}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_type_compatibility_check_when_argument_type_is_missing(self):
        component_a = '''\
outputs:
  - {name: out1}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: custom_type}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_fail_type_compatibility_check_when_simple_type_name_is_different(self):
        component_a = '''\
outputs:
  - {name: out1, type: type_A}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: type_Z}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        with self.assertRaises(InconsistentTypeException):
            b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_fail_type_compatibility_check_when_parametrized_type_name_is_different(self):
        component_a = '''\
outputs:
  - {name: out1, type: {parametrized_type_A: {property_a: value_a}}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {parametrized_type_Z: {property_a: value_a}}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        with self.assertRaises(InconsistentTypeException):
            b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_fail_type_compatibility_check_when_type_property_value_is_different(self):
        component_a = '''\
outputs:
  - {name: out1, type: {parametrized_type: {property_a: value_a}}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {parametrized_type: {property_a: DIFFERENT VALUE}}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        with self.assertRaises(InconsistentTypeException):
            b_task = task_factory_b(in1=a_task.outputs['out1'])

    @unittest.skip('Type compatibility check currently works the opposite way')
    def test_type_compatibility_check_when_argument_type_has_extra_type_parameters(self):
        component_a = '''\
outputs:
  - {name: out1, type: {parametrized_type: {property_a: value_a, extra_property: extra_value}}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {parametrized_type: {property_a: value_a}}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])

    @unittest.skip('Type compatibility check currently works the opposite way')
    def test_fail_type_compatibility_check_when_argument_type_has_missing_type_parameters(self):
        component_a = '''\
outputs:
  - {name: out1, type: {parametrized_type: {property_a: value_a}}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {parametrized_type: {property_a: value_a, property_b: value_b}}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        with self.assertRaises(InconsistentTypeException):
            b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_type_compatibility_check_not_failing_when_disabled(self):
        component_a = '''\
outputs:
  - {name: out1, type: type_A}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: type_Z}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = False
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])
        kfp.TYPE_CHECK = True

    def test_type_compatibility_check_not_failing_when_type_is_ignored(self):
        component_a = '''\
outputs:
  - {name: out1, type: type_A}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: type_Z}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'].ignore_type())

    def test_type_compatibility_check_for_types_with_schema(self):
        component_a = '''\
outputs:
  - {name: out1, type: {GCSPath: {openapi_schema_validator: {type: string, pattern: "^gs://.*$" } }}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {GCSPath: {openapi_schema_validator: {type: string, pattern: "^gs://.*$" } }}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_fail_type_compatibility_check_for_types_with_different_schemas(self):
        component_a = '''\
outputs:
  - {name: out1, type: {GCSPath: {openapi_schema_validator: {type: string, pattern: AAA } }}}
implementation:
  container:
    image: busybox
    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
'''
        component_b = '''\
inputs:
  - {name: in1, type: {GCSPath: {openapi_schema_validator: {type: string, pattern: ZZZ } }}}
implementation:
  container:
    image: busybox
    command: [echo, {inputValue: in1}]
'''
        kfp.TYPE_CHECK = True
        task_factory_a = comp.load_component_from_text(component_a)
        task_factory_b = comp.load_component_from_text(component_b)
        a_task = task_factory_a()

        with self.assertRaises(InconsistentTypeException):
            b_task = task_factory_b(in1=a_task.outputs['out1'])


if __name__ == '__main__':
    unittest.main()
