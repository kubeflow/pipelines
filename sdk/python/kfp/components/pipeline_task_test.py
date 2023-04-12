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

import textwrap
import unittest

from absl.testing import parameterized
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
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        self.assertEqual(task._task_spec, expected_task_spec)
        self.assertEqual(task.component_spec, expected_component_spec)
        self.assertEqual(task.container_spec, expected_container_spec)

    def test_create_pipeline_task_invalid_wrong_input(self):
        with self.assertRaisesRegex(
                ValueError,
                "Component 'component1' got an unexpected input: 'input0'."):
            task = pipeline_task.PipelineTask(
                component_spec=structures.ComponentSpec.from_yaml_documents(
                    V2_YAML),
                args={
                    'input1': 'value',
                    'input0': 'abc',
                },
            )

    def test_set_caching_options(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_caching_options(False)
        self.assertEqual(False, task._task_spec.enable_caching)

    @parameterized.parameters(
        {
            'cpu': '123',
            'expected_cpu_number': 123,
        },
        {
            'cpu': '123m',
            'expected_cpu_number': 0.123,
        },
        {
            'cpu': '123.0',
            'expected_cpu_number': 123,
        },
        {
            'cpu': '123.0m',
            'expected_cpu_number': 0.123,
        },
    )
    def test_set_valid_cpu_request_limit(self, cpu: str,
                                         expected_cpu_number: float):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_cpu_request(cpu)
        self.assertEqual(expected_cpu_number,
                         task.container_spec.resources.cpu_request)
        task.set_cpu_limit(cpu)
        self.assertEqual(expected_cpu_number,
                         task.container_spec.resources.cpu_limit)

    @parameterized.parameters(
        {
            'gpu_limit': '123',
            'expected_gpu_number': 123,
        },)
    def test_set_valid_gpu_limit(self, gpu_limit: str,
                                 expected_gpu_number: int):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        with self.assertWarnsRegex(
                DeprecationWarning,
                "'set_gpu_limit' is deprecated. Please use 'set_accelerator_limit' instead."
        ):
            task.set_gpu_limit(gpu_limit)
        self.assertEqual(expected_gpu_number,
                         task.container_spec.resources.accelerator_count)

    def test_add_valid_node_selector_constraint(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        with self.assertWarnsRegex(
                DeprecationWarning,
                "'add_node_selector_constraint' is deprecated. Please use 'set_accelerator_type' instead."
        ):
            task.add_node_selector_constraint('TPU_V3')
        self.assertEqual(task.container_spec.resources.accelerator_type,
                         'TPU_V3')

    @parameterized.parameters(
        {
            'limit': '123',
            'expected': 123,
        },
        {
            'limit': 123,
            'expected': 123,
        },
    )
    def test_set_accelerator_limit(self, limit, expected):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )

        task.set_accelerator_limit(limit)
        self.assertEqual(expected,
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
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_memory_request(memory)
        self.assertEqual(expected_memory_number,
                         task.container_spec.resources.memory_request)
        task.set_memory_limit(memory)
        self.assertEqual(expected_memory_number,
                         task.container_spec.resources.memory_limit)

    def test_set_accelerator_type_with_type_only(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_accelerator_type('NVIDIA_TESLA_K80')
        self.assertEqual(
            structures.ResourceSpec(
                accelerator_type='NVIDIA_TESLA_K80', accelerator_count=1),
            task.container_spec.resources)

    def test_set_accelerator_type_with_accelerator_count(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_accelerator_limit('5').set_accelerator_type('TPU_V3')
        self.assertEqual(
            structures.ResourceSpec(
                accelerator_type='TPU_V3', accelerator_count=5),
            task.container_spec.resources)

    def test_set_env_variable(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_env_variable('env_name', 'env_value')
        self.assertEqual({'env_name': 'env_value'}, task.container_spec.env)

    def test_set_display_name(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_display_name('test_name')
        self.assertEqual('test_name', task._task_spec.display_name)

    def test_set_cpu_limit_on_pipeline_should_raise(self):

        @dsl.component
        def comp():
            print('hello')

        @dsl.pipeline
        def graph():
            comp()
            comp()

        with self.assertRaisesRegex(
                ValueError,
                r'set_cpu_limit can only be used on single-step components'):

            @dsl.pipeline
            def my_pipeline():
                graph().set_cpu_limit('1')


class TestPlatformSpecificFunctionality(unittest.TestCase):

    def test_platform_config_to_platform_spec(self):

        @dsl.component
        def comp():
            pass

        @dsl.pipeline
        def my_pipeline():
            t = comp()
            t.platform_config = {'platform1': {'feature': [1, 2, 3]}}
            with self.assertRaisesRegex(
                    ValueError,
                    r"Can only access '\.platform_spec' property on a tasks created from pipelines\. Use '\.platform_config' for tasks created from primitive components\."
            ):
                t.platform_spec


if __name__ == '__main__':
    unittest.main()
