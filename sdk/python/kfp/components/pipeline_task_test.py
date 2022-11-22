# Copyright 2021 The Kubeflow Authors
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
"""Tests for kfp.components.pipeline_task."""

import os
import tempfile
import textwrap
import unittest

from absl.testing import parameterized
from kfp import compiler
from kfp import dsl
from kfp.components import pipeline_task
from kfp.components import placeholders
from kfp.components import structures

V2_YAML = textwrap.dedent("""\
components:
  comp-component1:
    executorLabel: exec-component1
    inputDefinitions:
      parameters:
        input1:
          parameterType: STRING
    outputDefinitions:
      artifacts:
        output1:
          artifactType:
            schemaTitle: system.Artifact
            schemaVersion: 0.0.1
deploymentSpec:
  executors:
    exec-component1:
      container:
        args:
        - '{{$.inputs.parameters[''input1'']}}'
        - '{{$.outputs.artifacts[''output1''].path}}'
        command:
        - sh
        - -c
        - echo "$0" >> "$1"
        image: alpine
pipelineInfo:
  name: component1
root:
  dag:
    tasks:
      component1:
        cachingOptions:
          enableCache: true
        componentRef:
          name: comp-component1
        inputs:
          parameters:
            input1:
              componentInputParameter: input1
        taskInfo:
          name: component1
  inputDefinitions:
    parameters:
      input1:
        parameterType: STRING
schemaVersion: 2.1.0
sdkVersion: kfp-2.0.0-alpha.2
""")


class PipelineTaskTest(parameterized.TestCase):

    def test_create_pipeline_task_valid(self):
        expected_component_spec = structures.ComponentSpec(
            name='component1',
            implementation=structures.Implementation(
                container=structures.ContainerSpecImplementation(
                    image='alpine',
                    command=['sh', '-c', 'echo "$0" >> "$1"'],
                    args=[
                        placeholders.InputValuePlaceholder(
                            input_name='input1')._to_string(),
                        placeholders.OutputPathPlaceholder(
                            output_name='output1')._to_string(),
                    ],
                )),
            inputs={
                'input1': structures.InputSpec(type='String'),
            },
            outputs={
                'output1': structures.OutputSpec(type='system.Artifact@0.0.1'),
            },
        )
        expected_task_spec = structures.TaskSpec(
            name='component1',
            inputs={'input1': 'value'},
            dependent_tasks=[],
            component_ref='component1',
        )
        expected_container_spec = structures.ContainerSpecImplementation(
            image='alpine',
            command=['sh', '-c', 'echo "$0" >> "$1"'],
            args=[
                "{{$.inputs.parameters['input1']}}",
                "{{$.outputs.artifacts['output1'].path}}",
            ],
        )

        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        self.assertEqual(task._task_spec, expected_task_spec)
        self.assertEqual(task.component_spec, expected_component_spec)
        self.assertEqual(task.container_spec, expected_container_spec)

    def test_create_pipeline_task_invalid_wrong_input(self):
        with self.assertRaisesRegex(
                ValueError,
                'Component "component1" got an unexpected input: input0.'):
            task = pipeline_task.PipelineTask(
                component_spec=structures.ComponentSpec
                .load_from_component_yaml(V2_YAML),
                args={
                    'input1': 'value',
                    'input0': 'abc',
                },
            )

    def test_set_caching_options(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_caching_options(False)
        self.assertEqual(False, task._task_spec.enable_caching)

    @parameterized.parameters(
        {
            'cpu_limit': '123',
            'expected_cpu_number': 123,
        },
        {
            'cpu_limit': '123m',
            'expected_cpu_number': 0.123,
        },
        {
            'cpu_limit': '123.0',
            'expected_cpu_number': 123,
        },
        {
            'cpu_limit': '123.0m',
            'expected_cpu_number': 0.123,
        },
    )
    def test_set_valid_cpu_limit(self, cpu_limit: str,
                                 expected_cpu_number: float):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_cpu_limit(cpu_limit)
        self.assertEqual(expected_cpu_number,
                         task.container_spec.resources.cpu_limit)

    @parameterized.parameters(
        {
            'gpu_limit': '666',
            'expected_gpu_number': 666,
        },)
    def test_set_valid_gpu_limit(self, gpu_limit: str,
                                 expected_gpu_number: int):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_gpu_limit(gpu_limit)
        self.assertEqual(expected_gpu_number,
                         task.container_spec.resources.accelerator_count)

    @parameterized.parameters(
        {
            'memory': '1E',
            'expected_memory_number': 1000000000,
        },
        {
            'memory': '15Ei',
            'expected_memory_number': 17293822569.102703,
        },
        {
            'memory': '2P',
            'expected_memory_number': 2000000,
        },
        {
            'memory': '25Pi',
            'expected_memory_number': 28147497.6710656,
        },
        {
            'memory': '3T',
            'expected_memory_number': 3000,
        },
        {
            'memory': '35Ti',
            'expected_memory_number': 38482.90697216,
        },
        {
            'memory': '4G',
            'expected_memory_number': 4,
        },
        {
            'memory': '45Gi',
            'expected_memory_number': 48.31838208,
        },
        {
            'memory': '5M',
            'expected_memory_number': 0.005,
        },
        {
            'memory': '55Mi',
            'expected_memory_number': 0.05767168,
        },
        {
            'memory': '6K',
            'expected_memory_number': 0.000006,
        },
        {
            'memory': '65Ki',
            'expected_memory_number': 0.00006656,
        },
        {
            'memory': '7000',
            'expected_memory_number': 0.000007,
        },
    )
    def test_set_memory_limit(self, memory: str, expected_memory_number: int):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_memory_limit(memory)
        self.assertEqual(expected_memory_number,
                         task.container_spec.resources.memory_limit)

    def test_add_node_selector_constraint_type_only(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.add_node_selector_constraint('NVIDIA_TESLA_K80')
        self.assertEqual(
            structures.ResourceSpec(
                accelerator_type='NVIDIA_TESLA_K80', accelerator_count=1),
            task.container_spec.resources)

    def test_add_node_selector_constraint_accelerator_count(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_gpu_limit('5').add_node_selector_constraint('TPU_V3')
        self.assertEqual(
            structures.ResourceSpec(
                accelerator_type='TPU_V3', accelerator_count=5),
            task.container_spec.resources)

    def test_set_env_variable(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_env_variable('env_name', 'env_value')
        self.assertEqual({'env_name': 'env_value'}, task.container_spec.env)

    def test_set_display_name(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.load_from_component_yaml(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_display_name('test_name')
        self.assertEqual('test_name', task._task_spec.display_name)


class TestCannotUseAfterCrossDAG(unittest.TestCase):

    def test_inner_task_prevented(self):
        with self.assertRaisesRegex(ValueError,
                                    r'Cannot use \.after\(\) across'):

            @dsl.component
            def print_op(message: str):
                print(message)

            @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
            def my_pipeline():
                first_exit_task = print_op(message='First exit task.')

                with dsl.ExitHandler(first_exit_task):
                    first_print_op = print_op(
                        message='Inside first exit handler.')

                second_exit_task = print_op(message='Second exit task.')
                with dsl.ExitHandler(second_exit_task):
                    print_op(message='Inside second exit handler.').after(
                        first_print_op)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_exit_handler_task_prevented(self):
        with self.assertRaisesRegex(ValueError,
                                    r'Cannot use \.after\(\) across'):

            @dsl.component
            def print_op(message: str):
                print(message)

            @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
            def my_pipeline():
                first_exit_task = print_op(message='First exit task.')

                with dsl.ExitHandler(first_exit_task):
                    first_print_op = print_op(
                        message='Inside first exit handler.')

                second_exit_task = print_op(message='Second exit task.')
                with dsl.ExitHandler(second_exit_task):
                    x = print_op(message='Inside second exit handler.')
                    x.after(first_exit_task)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_within_same_exit_handler_permitted(self):

        @dsl.component
        def print_op(message: str):
            print(message)

        @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
        def my_pipeline():
            first_exit_task = print_op(message='First exit task.')

            with dsl.ExitHandler(first_exit_task):
                first_print_op = print_op(
                    message='First task inside first exit handler.')
                second_print_op = print_op(
                    message='Second task inside first exit handler.').after(
                        first_print_op)

            second_exit_task = print_op(message='Second exit task.')
            with dsl.ExitHandler(second_exit_task):
                print_op(message='Inside second exit handler.')

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)

    def test_outside_of_condition_blocked(self):
        with self.assertRaisesRegex(ValueError,
                                    r'Cannot use \.after\(\) across'):

            @dsl.component
            def print_op(message: str):
                print(message)

            @dsl.component
            def return_1() -> int:
                return 1

            @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
            def my_pipeline():
                return_1_task = return_1()

                with dsl.Condition(return_1_task.output == 1):
                    one = print_op(message='1')
                    two = print_op(message='2')
                three = print_op(message='3').after(one)

            with tempfile.TemporaryDirectory() as tempdir:
                package_path = os.path.join(tempdir, 'pipeline.yaml')
                compiler.Compiler().compile(
                    pipeline_func=my_pipeline, package_path=package_path)

    def test_inside_of_condition_permitted(self):

        @dsl.component
        def print_op(message: str):
            print(message)

        @dsl.component
        def return_1() -> int:
            return 1

        @dsl.pipeline(name='pipeline-with-multiple-exit-handlers')
        def my_pipeline():
            return_1_task = return_1()

            with dsl.Condition(return_1_task.output == '1'):
                one = print_op(message='1')
                two = print_op(message='2').after(one)
            three = print_op(message='3')

        with tempfile.TemporaryDirectory() as tempdir:
            package_path = os.path.join(tempdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=my_pipeline, package_path=package_path)


if __name__ == '__main__':
    unittest.main()
