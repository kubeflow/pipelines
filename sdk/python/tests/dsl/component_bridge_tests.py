# Copyright 2020 Google LLC
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


import textwrap
import unittest
import kfp
from kfp.components import load_component_from_text
from kfp.dsl.types import InconsistentTypeException


class TestComponentBridge(unittest.TestCase):
    # Alternatively, we could use kfp.dsl.Pipleine().__enter__ and __exit__
    def setUp(self):
        self.old_container_task_constructor = kfp.components._components._container_task_constructor
        kfp.components._components._container_task_constructor = kfp.dsl._component_bridge._create_container_op_from_component_and_arguments

    def tearDown(self):
        kfp.components._components._container_task_constructor = self.old_container_task_constructor

    def test_conversion_to_container_op(self):
        component_text = textwrap.dedent('''\
            name: Custom component
            implementation:
                container:
                    image: busybox
            '''
        )
        task_factory1 = load_component_from_text(component_text)
        task1 = task_factory1()

        self.assertEqual(task1.human_name, 'Custom component')

    def test_passing_env_to_container_op(self):
        component_text = textwrap.dedent('''\
            implementation:
                container:
                    image: busybox
                    env:
                        key1: value 1
                        key2: value 2
            '''
        )
        task_factory1 = load_component_from_text(component_text)
        
        task1 = task_factory1()
        actual_env = {env_var.name: env_var.value for env_var in task1.container.env}
        expected_env = {'key1': 'value 1', 'key2': 'value 2'}
        self.assertDictEqual(expected_env, actual_env)

    def test_input_path_placeholder_with_constant_argument(self):
        component_text = textwrap.dedent('''\
            inputs:
            - {name: input 1}
            implementation:
                container:
                    image: busybox
                    command:
                    - --input-data
                    - {inputPath: input 1}
            '''
        )
        task_factory1 = load_component_from_text(component_text)
        task1 = task_factory1('Text')

        self.assertEqual(task1.command, ['--input-data', task1.input_artifact_paths['input 1']])
        self.assertEqual(task1.artifact_arguments, {'input 1': 'Text'})

    def test_passing_component_metadata_to_container_op(self):
        component_text = textwrap.dedent('''\
            metadata:
                annotations:
                    key1: value1
                labels:
                    key1: value1
            implementation:
                container:
                    image: busybox
            '''
        )
        task_factory1 = load_component_from_text(text=component_text)

        task1 = task_factory1()
        self.assertEqual(task1.pod_annotations['key1'], 'value1')
        self.assertEqual(task1.pod_labels['key1'], 'value1')

    def test_type_compatibility_check_not_failing_when_disabled(self):
        component_a = textwrap.dedent('''\
            outputs:
            - {name: out1, type: type_A}
            implementation:
                container:
                    image: busybox
                    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
            '''
        )
        component_b = textwrap.dedent('''\
            inputs:
            - {name: in1, type: type_Z}
            implementation:
                container:
                    image: busybox
                    command: [echo, {inputValue: in1}]
            '''
        )
        kfp.TYPE_CHECK = False
        task_factory_a = load_component_from_text(component_a)
        task_factory_b = load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])
        kfp.TYPE_CHECK = True

    def test_type_compatibility_check_not_failing_when_type_is_ignored(self):
        component_a = textwrap.dedent('''\
            outputs:
            - {name: out1, type: type_A}
            implementation:
                container:
                    image: busybox
                    command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
            '''
        )
        component_b = textwrap.dedent('''\
            inputs:
            - {name: in1, type: type_Z}
            implementation:
                container:
                    image: busybox
                    command: [echo, {inputValue: in1}]
            '''
        )
        task_factory_a = load_component_from_text(component_a)
        task_factory_b = load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'].ignore_type())
