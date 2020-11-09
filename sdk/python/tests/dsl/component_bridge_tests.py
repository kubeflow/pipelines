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

import tempfile
import textwrap
import unittest
import warnings
import kfp
from pathlib import Path
from kfp.components import load_component_from_text, create_component_from_func
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

    def test_volatile_components(self):
        component_text = textwrap.dedent('''\
            metadata:
                annotations:
                    volatile_component: "true"
            implementation:
                container:
                    image: busybox
            '''
        )
        task_factory1 = load_component_from_text(text=component_text)

        task1 = task_factory1()
        self.assertEqual(task1.execution_options.caching_strategy.max_cache_staleness, 'P0D')

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
            @kfp.dsl.pipeline(
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
            kfp.compiler.Compiler().compile(calc_pipeline, pipeline_filename)

    def test_handling_list_arguments_containing_pipelineparam(self):
        '''Checks that lists containing PipelineParam can be properly serialized'''
        def consume_list(list_param: list) -> int:
            pass

        import kfp
        task_factory = create_component_from_func(consume_list)
        task = task_factory([1, 2, 3, kfp.dsl.PipelineParam('aaa'), 4, 5, 6])

        full_command_line = task.command + task.arguments
        for arg in full_command_line:
            self.assertNotIn('PipelineParam', arg)

    def test_converted_outputs(self):
        component_text = textwrap.dedent('''\
            outputs:
            - name: Output 1
            implementation:
                container:
                    image: busybox
                    command:
                    - producer
                    - {outputPath: Output 1}  # Outputs must be used in the implementation
            '''
        )
        task_factory1 = load_component_from_text(component_text)
        task1 = task_factory1()

        self.assertSetEqual(set(task1.outputs.keys()), {'Output 1', 'output_1'})
        self.assertIsNotNone(task1.output)

    def test_reusable_component_warnings(self):
        op1 = load_component_from_text('''\
            implementation:
                container:
                    image: busybox
            '''
        )
        with warnings.catch_warnings(record=True) as warning_messages:
            op1()
            deprecation_messages = list(str(message) for message in warning_messages if message.category == DeprecationWarning)
            self.assertListEqual(deprecation_messages, [])

        with self.assertWarnsRegex(FutureWarning, expected_regex='reusable'):
            kfp.dsl.ContainerOp(name='name', image='image')

    def test_prevent_passing_container_op_as_argument(self):
        component_text = textwrap.dedent('''\
            inputs:
            - {name: input 1}
            - {name: input 2}
            implementation:
                container:
                    image: busybox
                    command:
                    - prog
                    - {inputValue: input 1}
                    - {inputPath: input 2}
            '''
        )
        component = load_component_from_text(component_text)
        # Passing normal values to component
        task1 = component(input_1="value 1", input_2="value 2")
        # Passing unserializable values to component
        with self.assertRaises(TypeError):
            component(input_1=task1, input_2="value 2")
        with self.assertRaises(TypeError):
            component(input_1="value 1", input_2=task1)

    def test_pythonic_container_output_handled_by_graph(self):
        component_a = textwrap.dedent('''\
          inputs: []
          outputs:
            - {name: out1, type: str}
          implementation:
            graph:
              tasks:
                some-container:
                  arguments: {}
                  componentRef:
                    spec:
                      outputs:
                      - {name: out1, type: str}
                      implementation:
                        container:
                          image: busybox
                          command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out1}]
              outputValues:
                out1:
                  taskOutput:
                    taskId: some-container
                    outputName: out1
        ''')
        component_b = textwrap.dedent('''\
            inputs:
            - {name: in1, type: str}
            implementation:
              container:
                image: busybox
                command: [echo, {inputValue: in1}]
        '''
                                      )
        task_factory_a = load_component_from_text(component_a)
        task_factory_b = load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])

    def test_nonpythonic_container_output_handled_by_graph(self):
        component_a = textwrap.dedent('''\
          inputs: []
          outputs:
            - {name: out1, type: str}
          implementation:
            graph:
              tasks:
                some-container:
                  arguments: {}
                  componentRef:
                    spec:
                      outputs:
                      - {name: out-1, type: str}
                      implementation:
                        container:
                          image: busybox
                          command: [bash, -c, 'mkdir -p "$(dirname "$0")"; date > "$0"', {outputPath: out-1}]
              outputValues:
                out1:
                  taskOutput:
                    taskId: some-container
                    outputName: out-1
        ''')
        component_b = textwrap.dedent('''\
            inputs:
            - {name: in1, type: str}
            implementation:
              container:
                image: busybox
                command: [echo, {inputValue: in1}]
        ''')
        task_factory_a = load_component_from_text(component_a)
        task_factory_b = load_component_from_text(component_b)
        a_task = task_factory_a()
        b_task = task_factory_b(in1=a_task.outputs['out1'])
