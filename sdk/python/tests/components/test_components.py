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
from pathlib import Path

_this_file = Path(__file__).resolve()
_this_dir = _this_file.parent
_test_data_dir = _this_dir.joinpath('test_data')
_tests_root_dir = _this_dir.parent
_sdk_root_dir = _tests_root_dir.parent

sys.path.insert(0, _sdk_root_dir)

import kfp.components as comp
from kfp.components._yaml_utils import load_yaml

class LoadComponentTestCase(unittest.TestCase):
    def test_load_component_from_file(self):
        component_path_obj = _test_data_dir.joinpath('python_add.component.yaml')
        component_text = component_path_obj.read_text()
        component_dict = load_yaml(component_text)
        task_factory1 = comp.load_component_from_file(str(component_path_obj))
        assert task_factory1.__doc__ == component_dict['description']

        arg1 = 3
        arg2 = 5
        task1 = task_factory1(arg1, arg2)
        assert task1.human_name == component_dict['name']
        assert task1.image == component_dict['implementation']['dockerContainer']['image']

        assert task1.arguments[0] == str(arg1)
        assert task1.arguments[1] == str(arg2)

    @unittest.expectedFailure #The repo is non-public and will change soon. TODO: Update the URL and enable the test once we move to a public repo
    def test_load_component_from_url(self):
        url = 'https://raw.githubusercontent.com/kubeflow/pipelines/638045974d688b473cda9f4516a2cf1d7d1e02dd/sdk/python/tests/components/test_data/python_add.component.yaml'

        import requests
        resp = requests.get(url)
        component_text = resp.content
        component_dict = load_yaml(component_text)
        task_factory1 = comp.load_component_from_url(url)
        assert task_factory1.__doc__ == component_dict['description']

        arg1 = 3
        arg2 = 5
        task1 = task_factory1(arg1, arg2)
        assert task1.human_name == component_dict['name']
        assert task1.image == component_dict['implementation']['dockerContainer']['image']

        assert task1.arguments[0] == str(arg1)
        assert task1.arguments[1] == str(arg2)

    def test_loading_minimal_component(self):
        component_text = '''\
implementation:
  dockerContainer:
    image: busybox
'''
        component_dict = load_yaml(component_text)
        task_factory1 = comp.load_component(text=component_text)

        task1 = task_factory1()
        assert task1.image == component_dict['implementation']['dockerContainer']['image']

    @unittest.expectedFailure
    def test_fail_on_duplicate_input_names(self):
        component_text = '''\
inputs:
- {name: Data1}
- {name: Data1}
implementation:
  dockerContainer:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    @unittest.expectedFailure
    def test_fail_on_duplicate_output_names(self):
        component_text = '''\
outputs:
- {name: Data1}
- {name: Data1}
implementation:
  dockerContainer:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_underscored_input_names(self):
        component_text = '''\
inputs:
- {name: Data}
- {name: _Data}
implementation:
  dockerContainer:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_underscored_output_names(self):
        component_text = '''\
outputs:
- {name: Data}
- {name: _Data}
implementation:
  dockerContainer:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_input_names_with_spaces(self):
        component_text = '''\
inputs:
- {name: Training data}
implementation:
  dockerContainer:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_output_names_with_spaces(self):
        component_text = '''\
outputs:
- {name: Training data}
implementation:
  dockerContainer:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    def test_handle_file_outputs_with_spaces(self):
        component_text = '''\
outputs:
- {name: Output data}
implementation:
  dockerContainer:
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
  dockerContainer:
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
  dockerContainer:
    image: busybox
'''
        task_factory1 = comp.load_component_from_text(component_text)

    @unittest.skip #TODO: FIX:
    @unittest.expectedFailure
    def test_fail_on_unknown_value_argument(self):
        component_text = '''\
inputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    arguments:
        - [value, Wrong]
'''
        task_factory1 = comp.load_component_from_text(component_text)

    @unittest.expectedFailure
    def test_fail_on_unknown_file_output(self):
        component_text = '''\
outputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    fileOutputs:
        Wrong: '/outputs/output.txt'
'''
        task_factory1 = comp.load_component_from_text(component_text)

    @unittest.expectedFailure
    def test_load_component_fail_on_no_sources(self):
        comp.load_component()

    @unittest.expectedFailure
    def test_load_component_fail_on_multiple_sources(self):
        comp.load_component(filename='', text='')

    @unittest.expectedFailure
    def test_load_component_fail_on_none_arguments(self):
        comp.load_component(filename=None, url=None, text=None)

    @unittest.expectedFailure
    def test_load_component_from_file_fail_on_none_arg(self):
        comp.load_component_from_file(None)

    @unittest.expectedFailure
    def test_load_component_from_url_fail_on_none_arg(self):
        comp.load_component_from_url(None)

    @unittest.expectedFailure
    def test_load_component_from_text_fail_on_none_arg(self):
        comp.load_component_from_text(None)

    def test_input_value_resolving_syntax1(self):
        component_text = '''\
inputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - --data
      - [value, Data]
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1('some-data')

        self.assertEqual(task1.arguments, ['--data', 'some-data'])

    def test_input_value_resolving_syntax3(self):
        component_text = '''\
inputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - --data
      - value: Data
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1('some-data')

        self.assertEqual(task1.arguments, ['--data', 'some-data'])

    def test_output_resolving_syntax1(self):
        component_text = '''\
outputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - --output-data
      - [output, Data]
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1(data='/outputs/some-data')

        self.assertEqual(task1.arguments, ['--output-data', '/outputs/some-data'])

    def test_output_resolving(self):
        component_text = '''\
outputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - --output-data
      - output: Data
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1(data='/outputs/some-data')

        self.assertEqual(task1.arguments, ['--output-data', '/outputs/some-data'])

    def test_automatic_output_resolving_syntax1(self):
        component_text = '''\
outputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - --output-data
      - [output, Data]
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1()

        self.assertEqual(len(task1.arguments), 2)

    def test_automatic_output_resolving(self):
        component_text = '''\
outputs:
- {name: Data}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - --output-data
      - {output: Data}
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1()

        self.assertEqual(len(task1.arguments), 2)

    def test_command_concat(self):
        component_text = '''\
inputs:
- {name: In1}
- {name: In2}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - concat: [{value: In1}, {value: In2}]
'''
        task_factory1 = comp.load_component(text=component_text)
        task1 = task_factory1('some', 'data')

        self.assertEqual(task1.arguments, ['somedata'])

    def test_command_if_then_else_syntax1(self):
        component_text = '''\
inputs:
- {name: In, required: false}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - [if, [isPresent, In], [--in, [value, In]], --no-in]
'''
        task_factory1 = comp.load_component(text=component_text)

        task_then = task_factory1('data')
        self.assertEqual(task_then.arguments, ['--in', 'data']) 
        
        #TODO: Fix optional arguments
        #task_else = task_factory1() #Error: TypeError: Component() missing 1 required positional argument: 'in'
        #self.assertEqual(task_else.arguments, ['--no-in'])

    def test_command_if_then_syntax1(self):
        component_text = '''\
inputs:
- {name: In, required: false}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - [if, [isPresent, In], [--in, [value, In]]]
'''
        task_factory1 = comp.load_component(text=component_text)

        task_then = task_factory1('data')
        self.assertEqual(task_then.arguments, ['--in', 'data']) 
        
        #TODO: Fix optional arguments
        #task_else = task_factory1() #Error: TypeError: Component() missing 1 required positional argument: 'in'
        #self.assertEqual(task_else.arguments, [])

    def test_command_if_then(self):
        component_text = '''\
inputs:
- {name: In, required: false}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - if:
          cond: {isPresent: In}
          then: [--in, {value: In}]
          #else: --no-in
'''
        task_factory1 = comp.load_component(text=component_text)

        task_then = task_factory1('data')
        self.assertEqual(task_then.arguments, ['--in', 'data']) 
        
        #TODO: Fix optional arguments
        #task_else = task_factory1() #Error: TypeError: Component() missing 1 required positional argument: 'in'
        #self.assertEqual(task_else.arguments, [])

    def test_command_if_then_else(self):
        component_text = '''\
inputs:
- {name: In, required: false}
implementation:
  dockerContainer:
    image: busybox
    arguments:
      - if:
          cond: {isPresent: In}
          then: [--in, {value: In}]
          else: --no-in
'''
        task_factory1 = comp.load_component(text=component_text)

        task_then = task_factory1('data')
        self.assertEqual(task_then.arguments, ['--in', 'data']) 
        
        #TODO: Fix optional arguments
        #task_else = task_factory1() #Error: TypeError: Component() missing 1 required positional argument: 'in'
        #self.assertEqual(task_else.arguments, ['--no-in'])


if __name__ == '__main__':
    unittest.main()
