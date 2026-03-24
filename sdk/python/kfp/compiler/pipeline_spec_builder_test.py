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
"""Tests for kfp.compiler.pipeline_spec_builder."""

import os
import tempfile
import unittest

from absl.testing import parameterized
from google.protobuf import json_format
from google.protobuf import struct_pb2
import kfp
from kfp import dsl
from kfp import kubernetes
from kfp.compiler import compiler
from kfp.compiler import pipeline_spec_builder
from kfp.dsl import TaskConfigField
from kfp.pipeline_spec import pipeline_spec_pb2
import yaml


class PipelineSpecBuilderTest(parameterized.TestCase):

    def setUp(self):
        self.maxDiff = None

    @parameterized.parameters(
        {
            'parameter_type': pipeline_spec_pb2.ParameterType.NUMBER_INTEGER,
            'default_value': None,
            'expected': struct_pb2.Value(),
        },
        {
            'parameter_type': pipeline_spec_pb2.ParameterType.NUMBER_INTEGER,
            'default_value': 1,
            'expected': struct_pb2.Value(number_value=1),
        },
        {
            'parameter_type': pipeline_spec_pb2.ParameterType.NUMBER_DOUBLE,
            'default_value': 1.2,
            'expected': struct_pb2.Value(number_value=1.2),
        },
        {
            'parameter_type': pipeline_spec_pb2.ParameterType.STRING,
            'default_value': 'text',
            'expected': struct_pb2.Value(string_value='text'),
        },
        {
            'parameter_type': pipeline_spec_pb2.ParameterType.BOOLEAN,
            'default_value': True,
            'expected': struct_pb2.Value(bool_value=True),
        },
        {
            'parameter_type': pipeline_spec_pb2.ParameterType.BOOLEAN,
            'default_value': False,
            'expected': struct_pb2.Value(bool_value=False),
        },
        {
            'parameter_type':
                pipeline_spec_pb2.ParameterType.STRUCT,
            'default_value': {
                'a': 1,
                'b': 2,
            },
            'expected':
                struct_pb2.Value(
                    struct_value=struct_pb2.Struct(
                        fields={
                            'a': struct_pb2.Value(number_value=1),
                            'b': struct_pb2.Value(number_value=2),
                        })),
        },
        {
            'parameter_type':
                pipeline_spec_pb2.ParameterType.LIST,
            'default_value': ['a', 'b'],
            'expected':
                struct_pb2.Value(
                    list_value=struct_pb2.ListValue(values=[
                        struct_pb2.Value(string_value='a'),
                        struct_pb2.Value(string_value='b'),
                    ])),
        },
        {
            'parameter_type':
                pipeline_spec_pb2.ParameterType.LIST,
            'default_value': [{
                'a': 1,
                'b': 2
            }, {
                'a': 10,
                'b': 20
            }],
            'expected':
                struct_pb2.Value(
                    list_value=struct_pb2.ListValue(values=[
                        struct_pb2.Value(
                            struct_value=struct_pb2.Struct(
                                fields={
                                    'a': struct_pb2.Value(number_value=1),
                                    'b': struct_pb2.Value(number_value=2),
                                })),
                        struct_pb2.Value(
                            struct_value=struct_pb2.Struct(
                                fields={
                                    'a': struct_pb2.Value(number_value=10),
                                    'b': struct_pb2.Value(number_value=20),
                                })),
                    ])),
        },
    )
    def test_fill_in_component_input_default_value(self, parameter_type,
                                                   default_value, expected):
        component_spec = pipeline_spec_pb2.ComponentSpec(
            input_definitions=pipeline_spec_pb2.ComponentInputsSpec(
                parameters={
                    'input1':
                        pipeline_spec_pb2.ComponentInputsSpec.ParameterSpec(
                            parameter_type=parameter_type)
                }))
        pipeline_spec_builder._fill_in_component_input_default_value(
            component_spec=component_spec,
            input_name='input1',
            default_value=default_value)

        self.assertEqual(
            expected,
            component_spec.input_definitions.parameters['input1'].default_value,
        )

    def test_merge_deployment_spec_and_component_spec(self):
        main_deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
        main_deployment_config.executors['exec-1'].CopyFrom(
            pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
                container=pipeline_spec_pb2.PipelineDeploymentConfig
                .PipelineContainerSpec(image='img-1', command=['cmd'])))
        main_pipeline_spec = pipeline_spec_pb2.PipelineSpec()
        main_pipeline_spec.components['comp-1'].CopyFrom(
            pipeline_spec_pb2.ComponentSpec(executor_label='exec-1'))

        sub_deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig()
        sub_deployment_config.executors['exec-1'].CopyFrom(
            pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
                container=pipeline_spec_pb2.PipelineDeploymentConfig
                .PipelineContainerSpec(image='img-a', command=['cmd'])))
        sub_deployment_config.executors['exec-2'].CopyFrom(
            pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
                container=pipeline_spec_pb2.PipelineDeploymentConfig
                .PipelineContainerSpec(image='img-b', command=['cmd'])))

        sub_pipeline_spec = pipeline_spec_pb2.PipelineSpec()
        sub_pipeline_spec.deployment_spec.update(
            json_format.MessageToDict(sub_deployment_config))
        sub_pipeline_spec.components['comp-1'].CopyFrom(
            pipeline_spec_pb2.ComponentSpec(executor_label='exec-1'))
        sub_pipeline_spec.components['comp-2'].CopyFrom(
            pipeline_spec_pb2.ComponentSpec(executor_label='exec-2'))
        sub_pipeline_spec.root.CopyFrom(
            pipeline_spec_pb2.ComponentSpec(dag=pipeline_spec_pb2.DagSpec()))
        sub_pipeline_spec.root.dag.tasks['task-1'].CopyFrom(
            pipeline_spec_pb2.PipelineTaskSpec(
                component_ref=pipeline_spec_pb2.ComponentRef(name='comp-1')))
        sub_pipeline_spec.root.dag.tasks['task-2'].CopyFrom(
            pipeline_spec_pb2.PipelineTaskSpec(
                component_ref=pipeline_spec_pb2.ComponentRef(name='comp-2')))

        expected_sub_pipeline_spec = pipeline_spec_pb2.PipelineSpec.FromString(
            sub_pipeline_spec.SerializeToString())

        expected_main_deployment_config = pipeline_spec_pb2.PipelineDeploymentConfig(
        )
        expected_main_deployment_config.executors['exec-1'].CopyFrom(
            pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
                container=pipeline_spec_pb2.PipelineDeploymentConfig
                .PipelineContainerSpec(image='img-1', command=['cmd'])))
        expected_main_deployment_config.executors['exec-1-2'].CopyFrom(
            pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
                container=pipeline_spec_pb2.PipelineDeploymentConfig
                .PipelineContainerSpec(image='img-a', command=['cmd'])))
        expected_main_deployment_config.executors['exec-2'].CopyFrom(
            pipeline_spec_pb2.PipelineDeploymentConfig.ExecutorSpec(
                container=pipeline_spec_pb2.PipelineDeploymentConfig
                .PipelineContainerSpec(image='img-b', command=['cmd'])))

        expected_main_pipeline_spec = pipeline_spec_pb2.PipelineSpec()
        expected_main_pipeline_spec.components['comp-1'].CopyFrom(
            pipeline_spec_pb2.ComponentSpec(executor_label='exec-1'))
        expected_main_pipeline_spec.components['comp-1-2'].CopyFrom(
            pipeline_spec_pb2.ComponentSpec(executor_label='exec-1-2'))
        expected_main_pipeline_spec.components['comp-2'].CopyFrom(
            pipeline_spec_pb2.ComponentSpec(executor_label='exec-2'))

        expected_sub_pipeline_spec_root = pipeline_spec_pb2.ComponentSpec(
            dag=pipeline_spec_pb2.DagSpec())
        expected_sub_pipeline_spec_root.dag.tasks['task-1'].CopyFrom(
            pipeline_spec_pb2.PipelineTaskSpec(
                component_ref=pipeline_spec_pb2.ComponentRef(name='comp-1-2')))
        expected_sub_pipeline_spec_root.dag.tasks['task-2'].CopyFrom(
            pipeline_spec_pb2.PipelineTaskSpec(
                component_ref=pipeline_spec_pb2.ComponentRef(name='comp-2')))

        sub_pipeline_spec_copy, sub_platform_spec = pipeline_spec_builder.merge_deployment_spec_and_component_spec(
            main_pipeline_spec=main_pipeline_spec,
            main_deployment_config=main_deployment_config,
            sub_pipeline_spec=sub_pipeline_spec,
            main_platform_spec=pipeline_spec_pb2.PlatformSpec(),
            sub_platform_spec=pipeline_spec_pb2.PlatformSpec(),
        )

        self.assertEqual(sub_pipeline_spec, expected_sub_pipeline_spec)
        self.assertEqual(main_pipeline_spec, expected_main_pipeline_spec)
        self.assertEqual(main_deployment_config,
                         expected_main_deployment_config)
        self.assertEqual(sub_pipeline_spec_copy.root,
                         expected_sub_pipeline_spec_root)


class TestPlatformConfigToPlatformSpec(unittest.TestCase):

    def test(self):
        platform_config = {
            'platform_key': {
                'feat1': [1, 2, 3],
                'feat2': 'hello',
                'feat3': {
                    'k': 'v'
                }
            }
        }
        actual = pipeline_spec_builder.platform_config_to_platform_spec(
            platform_config=platform_config, executor_label='exec-comp')
        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_key': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'feat1': [1, 2, 3],
                                    'feat2': 'hello',
                                    'feat3': {
                                        'k': 'v'
                                    }
                                }
                            }
                        }
                    }
                }
            }, expected)
        self.assertEqual(actual, expected)


class TestMergePlatformSpecs(unittest.TestCase):

    def test_merge_different_executors(self):
        main_spec = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_key': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'feat1': [1, 2, 3],
                                    'feat2': 'hello',
                                    'feat3': {
                                        'k': 'v'
                                    }
                                }
                            }
                        }
                    }
                }
            }, main_spec)

        sub_spec = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_key': {
                        'deployment_spec': {
                            'executors': {
                                'exec-other': {
                                    'feat1': [4, 5, 6],
                                    'feat2': 'goodbye',
                                    'feat3': {
                                        'k2': 'v2'
                                    }
                                }
                            }
                        }
                    }
                }
            }, sub_spec)

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform_key': {
                        'deployment_spec': {
                            'executors': {
                                'exec-other': {
                                    'feat1': [4, 5, 6],
                                    'feat2': 'goodbye',
                                    'feat3': {
                                        'k2': 'v2'
                                    }
                                },
                                'exec-comp': {
                                    'feat1': [1, 2, 3],
                                    'feat2': 'hello',
                                    'feat3': {
                                        'k': 'v'
                                    }
                                }
                            }
                        }
                    }
                }
            }, expected)

        pipeline_spec_builder.merge_platform_specs(main_spec, sub_spec)
        self.assertEqual(main_spec, expected)

    def test_merge_different_platforms(self):
        base_spec = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform1': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'feat1': [1, 2, 3],
                                    'feat2': 'hello',
                                    'feat3': {
                                        'k': 'v'
                                    }
                                }
                            }
                        }
                    }
                }
            }, base_spec)

        sub_spec = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform2': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'feat1': [4, 5, 6],
                                    'feat2': 'goodbye',
                                    'feat3': {
                                        'k2': 'v2'
                                    }
                                }
                            }
                        }
                    }
                }
            }, sub_spec)

        expected = pipeline_spec_pb2.PlatformSpec()
        json_format.ParseDict(
            {
                'platforms': {
                    'platform1': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'feat1': [1, 2, 3],
                                    'feat2': 'hello',
                                    'feat3': {
                                        'k': 'v'
                                    }
                                }
                            }
                        }
                    },
                    'platform2': {
                        'deployment_spec': {
                            'executors': {
                                'exec-comp': {
                                    'feat1': [4, 5, 6],
                                    'feat2': 'goodbye',
                                    'feat3': {
                                        'k2': 'v2'
                                    }
                                }
                            }
                        }
                    }
                }
            }, expected)

        pipeline_spec_builder.merge_platform_specs(base_spec, sub_spec)
        self.assertEqual(base_spec, expected)


class TestTaskConfigPassthroughValidation(unittest.TestCase):

    def test_resources_set_without_passthrough_raises(self):

        @dsl.component(task_config_passthroughs=[TaskConfigField.ENV])
        def comp():
            pass

        with self.assertRaisesRegex(
                ValueError,
                r"Task 'comp' cannot handle resource type 'RESOURCES'"):

            @dsl.pipeline
            def pipe():
                t = comp()
                t.set_cpu_limit('1')

    def test_env_set_without_passthrough_raises(self):

        @dsl.component(task_config_passthroughs=[TaskConfigField.RESOURCES])
        def comp():
            pass

        with self.assertRaisesRegex(
                ValueError, r"Task 'comp' cannot handle resource type 'ENV'"):

            @dsl.pipeline
            def pipe():
                t = comp()
                t.set_env_variable('A', 'B')

    def test_tolerations_without_passthrough_raises(self):

        @dsl.component(task_config_passthroughs=[TaskConfigField.RESOURCES])
        def comp():
            pass

        with self.assertRaisesRegex(
                ValueError,
                r"Task 'comp' cannot handle resource type 'KUBERNETES_TOLERATIONS'"
        ):

            @dsl.pipeline
            def pipe():
                t = comp()
                kubernetes.add_toleration(t, key='k', operator='Exists')

    def test_node_selector_without_passthrough_raises(self):

        @dsl.component(task_config_passthroughs=[TaskConfigField.RESOURCES])
        def comp():
            pass

        with self.assertRaisesRegex(
                ValueError,
                r"Task 'comp' cannot handle resource type 'KUBERNETES_NODE_SELECTOR'"
        ):

            @dsl.pipeline
            def pipe():
                t = comp()
                kubernetes.add_node_selector(t, label_key='k', label_value='v')

    def test_affinity_without_passthrough_raises(self):

        @dsl.component(task_config_passthroughs=[TaskConfigField.RESOURCES])
        def comp():
            pass

        with self.assertRaisesRegex(
                ValueError,
                r"Task 'comp' cannot handle resource type 'KUBERNETES_AFFINITY'"
        ):

            @dsl.pipeline
            def pipe():
                t = comp()
                kubernetes.add_node_affinity(
                    t,
                    match_expressions=[{
                        'key': 'disktype',
                        'operator': 'In',
                        'values': ['ssd']
                    }],
                )

    def test_volumes_without_passthrough_raises(self):

        @dsl.component(task_config_passthroughs=[TaskConfigField.RESOURCES])
        def comp():
            pass

        with self.assertRaisesRegex(
                ValueError,
                r"Task 'comp' cannot handle resource type 'KUBERNETES_VOLUMES'"
        ):

            @dsl.pipeline
            def pipe():
                t = comp()
                kubernetes.mount_pvc(t, pvc_name='my-pvc', mount_path='/mnt')


class TestTaskConfigPassthroughValidationPositive(unittest.TestCase):

    def test_all_passthroughs_allow_all_settings_compiles(self):

        @dsl.component(task_config_passthroughs=[
            TaskConfigField.RESOURCES,
            TaskConfigField.ENV,
            TaskConfigField.KUBERNETES_TOLERATIONS,
            TaskConfigField.KUBERNETES_NODE_SELECTOR,
            TaskConfigField.KUBERNETES_AFFINITY,
            TaskConfigField.KUBERNETES_VOLUMES,
        ])
        def comp():
            pass

        @dsl.pipeline
        def pipe():
            t = comp()
            # Set container resources and env
            t.set_cpu_limit('1')
            t.set_env_variable('A', 'B')
            # Set Kubernetes platform configs
            kubernetes.add_toleration(t, key='k', operator='Exists')
            kubernetes.add_node_selector(t, label_key='k', label_value='v')
            kubernetes.add_node_affinity(
                t,
                match_expressions=[{
                    'key': 'disktype',
                    'operator': 'In',
                    'values': ['ssd']
                }])
            kubernetes.mount_pvc(t, pvc_name='my-pvc', mount_path='/mnt')

        with tempfile.TemporaryDirectory() as tmpdir:
            package_path = os.path.join(tmpdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipe, package_path=package_path)


class TestPlatformConfigDAGBoundaryHandling(unittest.TestCase):
    """Tests that platform config input references are correctly rewritten
    when tasks are inside sub-DAGs (e.g. ParallelFor)."""

    def _compile_and_parse(self, pipeline_func):
        """Compile a pipeline and return (pipeline_spec, platform_spec)."""
        with tempfile.TemporaryDirectory() as tmpdir:
            package_path = os.path.join(tmpdir, 'pipeline.yaml')
            compiler.Compiler().compile(
                pipeline_func=pipeline_func, package_path=package_path)
            with open(package_path, 'r') as f:
                docs = list(yaml.safe_load_all(f))
            pipeline_spec = json_format.ParseDict(
                docs[0], pipeline_spec_pb2.PipelineSpec())
            platform_spec = json_format.ParseDict(
                docs[1], pipeline_spec_pb2.PlatformSpec()
            ) if len(docs) > 1 else pipeline_spec_pb2.PlatformSpec()
            return pipeline_spec, platform_spec

    def test_simple_secret_no_subdag(self):
        """Baseline: secret_name from pipeline param at root level.
        No rewriting needed - componentInputParameter stays unprefixed."""

        @dsl.component
        def my_comp():
            print('hello')

        @dsl.pipeline
        def pipe(secret_name: str):
            task = my_comp()
            kubernetes.use_secret_as_env(
                task,
                secret_name=secret_name,
                secret_key_to_env={'key': 'VAL'},
            )

        pipeline_spec, platform_spec = self._compile_and_parse(pipe)

        # At root level the param name should NOT be prefixed
        secret_param = (
            platform_spec.platforms['kubernetes'].deployment_spec.executors[
                'exec-my-comp'].fields['secretAsEnv'].list_value.values[0]
            .struct_value.fields['secretNameParameter'].struct_value.fields[
                'componentInputParameter'].string_value)
        self.assertEqual(secret_param, 'secret_name')

    def test_parallelfor_pipeline_input_secret(self):
        """Bug scenario: secret_name from pipeline param inside ParallelFor.
        The sub-DAG must surface the param, and the platform config must
        reference the prefixed name."""

        @dsl.component
        def my_comp(item: str):
            print(item)

        @dsl.pipeline
        def pipe(secret_name: str):
            with dsl.ParallelFor(items=['a', 'b'], parallelism=1) as item:
                t = my_comp(item=item)
                kubernetes.use_secret_as_env(
                    t,
                    secret_name=secret_name,
                    secret_key_to_env={'key': 'VAL'},
                )

        pipeline_spec, platform_spec = self._compile_and_parse(pipe)

        # Sub-DAG component must have the surfaced input
        loop_component = pipeline_spec.components['comp-for-loop-2']
        self.assertIn('pipelinechannel--secret_name',
                      loop_component.input_definitions.parameters)

        # Root DAG task must wire the input
        root_task_params = pipeline_spec.root.dag.tasks[
            'for-loop-2'].inputs.parameters
        self.assertEqual(
            root_task_params[
                'pipelinechannel--secret_name'].component_input_parameter,
            'secret_name',
        )

        # Platform config must reference the prefixed name
        secret_param = (
            platform_spec.platforms['kubernetes'].deployment_spec.executors[
                'exec-my-comp'].fields['secretAsEnv'].list_value.values[0]
            .struct_value.fields['secretNameParameter'].struct_value.fields[
                'componentInputParameter'].string_value)
        self.assertEqual(secret_param, 'pipelinechannel--secret_name')

    def test_parallelfor_outer_task_output_secret(self):
        """Cross-DAG: secret_name from outer task output inside ParallelFor.
        The taskOutputParameter must be rewritten to componentInputParameter
        pointing to the surfaced input."""

        @dsl.component
        def emit_secret_name() -> str:
            return 'secret'

        @dsl.component
        def my_comp():
            print('hello')

        @dsl.pipeline
        def pipe():
            secret_task = emit_secret_name()
            with dsl.ParallelFor(items=[1, 2], parallelism=1):
                t = my_comp()
                kubernetes.use_secret_as_env(
                    t,
                    secret_name=secret_task.output,
                    secret_key_to_env={'key': 'VAL'},
                )

        pipeline_spec, platform_spec = self._compile_and_parse(pipe)

        # Sub-DAG component must have the surfaced task output
        loop_component = pipeline_spec.components['comp-for-loop-2']
        self.assertIn('pipelinechannel--emit-secret-name-Output',
                      loop_component.input_definitions.parameters)

        # Root DAG task must wire the task output
        root_task_params = pipeline_spec.root.dag.tasks[
            'for-loop-2'].inputs.parameters
        self.assertEqual(
            root_task_params[
                'pipelinechannel--emit-secret-name-Output']
            .task_output_parameter.producer_task,
            'emit-secret-name',
        )

        # Platform config must use componentInputParameter (NOT taskOutputParameter)
        secret_name_fields = (
            platform_spec.platforms['kubernetes'].deployment_spec.executors[
                'exec-my-comp'].fields['secretAsEnv'].list_value.values[0]
            .struct_value.fields['secretNameParameter'].struct_value.fields)
        self.assertEqual(
            secret_name_fields['componentInputParameter'].string_value,
            'pipelinechannel--emit-secret-name-Output',
        )
        self.assertNotIn('taskOutputParameter', secret_name_fields)

    def test_parallelfor_literal_secret_unchanged(self):
        """Literal secret names should not be affected by the rewriting."""

        @dsl.component
        def my_comp(item: str):
            print(item)

        @dsl.pipeline
        def pipe():
            with dsl.ParallelFor(items=['a', 'b'], parallelism=1) as item:
                t = my_comp(item=item)
                kubernetes.use_secret_as_env(
                    t,
                    secret_name='my-literal-secret',
                    secret_key_to_env={'key': 'VAL'},
                )

        pipeline_spec, platform_spec = self._compile_and_parse(pipe)

        # Platform config should have the constant value, not a parameter ref
        secret_name_fields = (
            platform_spec.platforms['kubernetes'].deployment_spec.executors[
                'exec-my-comp'].fields['secretAsEnv'].list_value.values[0]
            .struct_value.fields['secretNameParameter'].struct_value.fields)
        self.assertNotIn('componentInputParameter', secret_name_fields)
        self.assertNotIn('taskOutputParameter', secret_name_fields)


class TestRewritePlatformConfigInputReferences(unittest.TestCase):
    """Unit tests for the _rewrite_platform_config_input_references helper."""

    def test_rewrites_unprefixed_component_input_param(self):
        platform_config = {
            'kubernetes': {
                'secretAsEnv': [{
                    'secretNameParameter': {
                        'componentInputParameter': 'secret_name'
                    },
                    'keyToEnv': [{'secretKey': 'pw', 'envVar': 'PASSWORD'}],
                }]
            }
        }
        parent_inputs = pipeline_spec_pb2.ComponentInputsSpec()
        parent_inputs.parameters[
            'pipelinechannel--secret_name'].parameter_type = (
                pipeline_spec_pb2.ParameterType.STRING)

        result = pipeline_spec_builder._rewrite_platform_config_input_references(
            platform_config, parent_inputs, [])

        self.assertEqual(
            result['kubernetes']['secretAsEnv'][0]['secretNameParameter']
            ['componentInputParameter'],
            'pipelinechannel--secret_name',
        )

    def test_no_rewrite_when_param_exists_in_parent(self):
        platform_config = {
            'kubernetes': {
                'secretAsEnv': [{
                    'secretNameParameter': {
                        'componentInputParameter': 'secret_name'
                    },
                }]
            }
        }
        parent_inputs = pipeline_spec_pb2.ComponentInputsSpec()
        parent_inputs.parameters['secret_name'].parameter_type = (
            pipeline_spec_pb2.ParameterType.STRING)

        result = pipeline_spec_builder._rewrite_platform_config_input_references(
            platform_config, parent_inputs, [])

        self.assertEqual(
            result['kubernetes']['secretAsEnv'][0]['secretNameParameter']
            ['componentInputParameter'],
            'secret_name',
        )

    def test_rewrites_task_output_from_outer_task(self):
        platform_config = {
            'kubernetes': {
                'secretAsEnv': [{
                    'secretNameParameter': {
                        'taskOutputParameter': {
                            'producerTask': 'emit-secret',
                            'outputParameterKey': 'Output',
                        }
                    },
                }]
            }
        }
        parent_inputs = pipeline_spec_pb2.ComponentInputsSpec()
        parent_inputs.parameters[
            'pipelinechannel--emit-secret-Output'].parameter_type = (
                pipeline_spec_pb2.ParameterType.STRING)

        result = pipeline_spec_builder._rewrite_platform_config_input_references(
            platform_config, parent_inputs,
            tasks_in_current_dag=['worker-task'])

        # taskOutputParameter should be replaced with componentInputParameter
        secret_ref = result['kubernetes']['secretAsEnv'][0][
            'secretNameParameter']
        self.assertNotIn('taskOutputParameter', secret_ref)
        self.assertEqual(
            secret_ref['componentInputParameter'],
            'pipelinechannel--emit-secret-Output',
        )

    def test_no_rewrite_task_output_from_local_task(self):
        platform_config = {
            'kubernetes': {
                'secretAsEnv': [{
                    'secretNameParameter': {
                        'taskOutputParameter': {
                            'producerTask': 'local-task',
                            'outputParameterKey': 'Output',
                        }
                    },
                }]
            }
        }
        parent_inputs = pipeline_spec_pb2.ComponentInputsSpec()

        result = pipeline_spec_builder._rewrite_platform_config_input_references(
            platform_config, parent_inputs,
            tasks_in_current_dag=['local-task'])

        # taskOutputParameter should remain since the producer is in the current DAG
        secret_ref = result['kubernetes']['secretAsEnv'][0][
            'secretNameParameter']
        self.assertIn('taskOutputParameter', secret_ref)
        self.assertNotIn('componentInputParameter', secret_ref)

    def test_returns_copy_when_no_parent_inputs(self):
        platform_config = {'kubernetes': {'secretAsEnv': [{'key': 'val'}]}}
        result = pipeline_spec_builder._rewrite_platform_config_input_references(
            platform_config, None, None)
        self.assertEqual(result, platform_config)
        # Must be a copy, not the same object
        self.assertIsNot(result, platform_config)

    def test_rewrites_multiple_params(self):
        platform_config = {
            'kubernetes': {
                'secretAsEnv': [{
                    'secretNameParameter': {
                        'componentInputParameter': 'secret1'
                    },
                }],
                'pvcMount': [{
                    'pvcNameParameter': {
                        'componentInputParameter': 'pvc_name'
                    },
                }],
            }
        }
        parent_inputs = pipeline_spec_pb2.ComponentInputsSpec()
        parent_inputs.parameters[
            'pipelinechannel--secret1'].parameter_type = (
                pipeline_spec_pb2.ParameterType.STRING)
        parent_inputs.parameters[
            'pipelinechannel--pvc_name'].parameter_type = (
                pipeline_spec_pb2.ParameterType.STRING)

        result = pipeline_spec_builder._rewrite_platform_config_input_references(
            platform_config, parent_inputs, [])

        self.assertEqual(
            result['kubernetes']['secretAsEnv'][0]['secretNameParameter']
            ['componentInputParameter'],
            'pipelinechannel--secret1',
        )
        self.assertEqual(
            result['kubernetes']['pvcMount'][0]['pvcNameParameter']
            ['componentInputParameter'],
            'pipelinechannel--pvc_name',
        )


def pipeline_spec_from_file(filepath: str) -> str:
    with open(filepath, 'r') as f:
        dictionary = yaml.safe_load(f)
    return json_format.ParseDict(dictionary, pipeline_spec_pb2.PipelineSpec())


if __name__ == '__main__':
    unittest.main()
