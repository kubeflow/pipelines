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
"""Tests for kfp.dsl.pipeline_task."""

import textwrap
import unittest

from absl.testing import parameterized
from kfp import dsl
from kfp.dsl import pipeline_task
from kfp.dsl import placeholders
from kfp.dsl import structures

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
            'expected_cpu': '123',
        },
        {
            'cpu': '123m',
            'expected_cpu': '123m',
        },
        {
            'cpu': '123.0',
            'expected_cpu': '123.0',
        },
        {
            'cpu': '123.0m',
            'expected_cpu': '123.0m',
        },
    )
    def test_set_valid_cpu_request_limit(self, cpu: str, expected_cpu: str):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_cpu_request(cpu)
        self.assertEqual(expected_cpu,
                         task.container_spec.resources.cpu_request)
        task.set_cpu_limit(cpu)
        self.assertEqual(expected_cpu, task.container_spec.resources.cpu_limit)

    @parameterized.parameters(
        {
            'gpu_limit': '1',
            'expected_gpu_number': '1',
        },)
    def test_set_valid_gpu_limit(self, gpu_limit: str,
                                 expected_gpu_number: str):
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
            'limit': '1',
            'expected_limit': '1',
        },
        {
            'limit': 1,
            'expected_limit': '1',
        },
        {
            'limit': 16,
            'expected_limit': '16',
        },
    )
    def test_set_accelerator_limit(self, limit, expected_limit):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )

        task.set_accelerator_limit(limit)
        self.assertEqual(expected_limit,
                         task.container_spec.resources.accelerator_count)

    @parameterized.parameters(
        {
            'memory': '1E',
            'expected_memory': '1E',
        },
        {
            'memory': '15Ei',
            'expected_memory': '15Ei',
        },
        {
            'memory': '2P',
            'expected_memory': '2P',
        },
        {
            'memory': '25Pi',
            'expected_memory': '25Pi',
        },
        {
            'memory': '3T',
            'expected_memory': '3T',
        },
        {
            'memory': '35Ti',
            'expected_memory': '35Ti',
        },
        {
            'memory': '4G',
            'expected_memory': '4G',
        },
        {
            'memory': '45Gi',
            'expected_memory': '45Gi',
        },
        {
            'memory': '5M',
            'expected_memory': '5M',
        },
        {
            'memory': '55Mi',
            'expected_memory': '55Mi',
        },
        {
            'memory': '6K',
            'expected_memory': '6K',
        },
        {
            'memory': '65Ki',
            'expected_memory': '65Ki',
        },
        {
            'memory': '7000',
            'expected_memory': '7000',
        },
    )
    def test_set_memory_limit(self, memory: str, expected_memory: str):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_memory_request(memory)
        self.assertEqual(expected_memory,
                         task.container_spec.resources.memory_request)
        task.set_memory_limit(memory)
        self.assertEqual(expected_memory,
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
                accelerator_type='NVIDIA_TESLA_K80', accelerator_count='1'),
            task.container_spec.resources)

    def test_set_accelerator_type_with_accelerator_count(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.set_accelerator_limit('4').set_accelerator_type('TPU_V3')
        self.assertEqual(
            structures.ResourceSpec(
                accelerator_type='TPU_V3', accelerator_count='4'),
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


class TestTaskInFinalState(unittest.TestCase):
    """Tests PipelineTask in the state FINAL.

    Many properties and methods will be blocked.

    Also tests that the .output and .outputs behavior behaves as expected when the outputs are values, not placeholders, as will be the case when PipelineTask is in the state FINAL.
    """

    def test_output_property(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.state = pipeline_task.TaskState.FINAL
        task._outputs = {'Output': 1}
        self.assertEqual(task.output, 1)
        self.assertEqual(task.outputs['Output'], 1)

    def test_outputs_property(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.state = pipeline_task.TaskState.FINAL
        task._outputs = {
            'int_output':
                1,
            'str_output':
                'foo',
            'dataset_output':
                dsl.Dataset(
                    name='dataset_output',
                    uri='foo/bar/dataset_output',
                    metadata={'key': 'value'})
        }
        self.assertEqual(task.outputs['int_output'], 1)
        self.assertEqual(task.outputs['str_output'], 'foo')
        assert_artifacts_equal(
            self,
            task.outputs['dataset_output'],
            dsl.Dataset(
                name='dataset_output',
                uri='foo/bar/dataset_output',
                metadata={'key': 'value'}),
        )

    def test_platform_spec_property(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.state = pipeline_task.TaskState.FINAL
        with self.assertRaisesRegex(
                Exception,
                r'Platform-specific features are not supported for local execution\.'
        ):
            task.platform_spec

    def test_name_property(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.state = pipeline_task.TaskState.FINAL
        self.assertEqual(task.name, 'component1')

    def test_inputs_property(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.state = pipeline_task.TaskState.FINAL
        self.assertEqual(task.inputs, {'input1': 'value'})

    def test_dependent_tasks_property(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.state = pipeline_task.TaskState.FINAL
        with self.assertRaisesRegex(
                Exception,
                r'Task has no dependent tasks since it is executed independently\.'
        ):
            task.dependent_tasks

    def test_sampling_of_task_configuration_methods(self):
        task = pipeline_task.PipelineTask(
            component_spec=structures.ComponentSpec.from_yaml_documents(
                V2_YAML),
            args={'input1': 'value'},
        )
        task.state = pipeline_task.TaskState.FINAL
        with self.assertRaisesRegex(
                Exception,
                r"Task configuration methods are not supported for local execution\. Got call to '\.set_caching_options\(\)'\."
        ):
            task.set_caching_options(enable_caching=True)
        with self.assertRaisesRegex(
                Exception,
                r"Task configuration methods are not supported for local execution\. Got call to '\.set_env_variable\(\)'\."
        ):
            task.set_env_variable(name='foo', value='BAR')
        with self.assertRaisesRegex(
                Exception,
                r"Task configuration methods are not supported for local execution\. Got call to '\.ignore_upstream_failure\(\)'\."
        ):
            task.ignore_upstream_failure()


def assert_artifacts_equal(
    test_class: unittest.TestCase,
    a1: dsl.Artifact,
    a2: dsl.Artifact,
) -> None:
    test_class.assertEqual(a1.name, a2.name)
    test_class.assertEqual(a1.uri, a2.uri)
    test_class.assertEqual(a1.metadata, a2.metadata)
    test_class.assertEqual(a1.schema_title, a2.schema_title)
    test_class.assertEqual(a1.schema_version, a2.schema_version)
    test_class.assertIsInstance(a1, type(a2))


if __name__ == '__main__':
    unittest.main()
