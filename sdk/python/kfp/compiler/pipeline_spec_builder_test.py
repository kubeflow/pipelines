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


def pipeline_spec_from_file(filepath: str) -> str:
    with open(filepath, 'r') as f:
        dictionary = yaml.safe_load(f)
    return json_format.ParseDict(dictionary, pipeline_spec_pb2.PipelineSpec())


class ToProtobufValueTest(unittest.TestCase):

    def test_none_value(self):
        result = pipeline_spec_builder.to_protobuf_value(None)
        self.assertIsInstance(result, struct_pb2.Value)
        self.assertEqual(result.null_value, struct_pb2.NULL_VALUE)

    def test_bool_true(self):
        result = pipeline_spec_builder.to_protobuf_value(True)
        self.assertEqual(result.bool_value, True)

    def test_bool_false(self):
        result = pipeline_spec_builder.to_protobuf_value(False)
        self.assertEqual(result.bool_value, False)

    def test_string_value(self):
        result = pipeline_spec_builder.to_protobuf_value("test")
        self.assertEqual(result.string_value, "test")

    def test_int_value(self):
        result = pipeline_spec_builder.to_protobuf_value(42)
        self.assertEqual(result.number_value, 42)

    def test_float_value(self):
        result = pipeline_spec_builder.to_protobuf_value(3.14)
        self.assertAlmostEqual(result.number_value, 3.14)

    def test_dict_value(self):
        test_dict = {"key1": "value1", "key2": 123}
        result = pipeline_spec_builder.to_protobuf_value(test_dict)
        self.assertIsInstance(result.struct_value, struct_pb2.Struct)
        self.assertEqual(result.struct_value.fields["key1"].string_value, "value1")
        self.assertEqual(result.struct_value.fields["key2"].number_value, 123)

    def test_dict_with_none_value(self):
        test_dict = {"key1": "value1", "key2": None}
        result = pipeline_spec_builder.to_protobuf_value(test_dict)
        self.assertIsInstance(result.struct_value, struct_pb2.Struct)
        self.assertEqual(result.struct_value.fields["key1"].string_value, "value1")
        self.assertEqual(result.struct_value.fields["key2"].null_value, struct_pb2.NULL_VALUE)

    def test_list_value(self):
        test_list = ["string", 123, True]
        result = pipeline_spec_builder.to_protobuf_value(test_list)
        self.assertIsInstance(result.list_value, struct_pb2.ListValue)
        self.assertEqual(len(result.list_value.values), 3)
        self.assertEqual(result.list_value.values[0].string_value, "string")
        self.assertEqual(result.list_value.values[1].number_value, 123)
        self.assertEqual(result.list_value.values[2].bool_value, True)

    def test_list_with_none_value(self):
        test_list = ["string", None, 123]
        result = pipeline_spec_builder.to_protobuf_value(test_list)
        self.assertIsInstance(result.list_value, struct_pb2.ListValue)
        self.assertEqual(len(result.list_value.values), 3)
        self.assertEqual(result.list_value.values[0].string_value, "string")
        self.assertEqual(result.list_value.values[1].null_value, struct_pb2.NULL_VALUE)
        self.assertEqual(result.list_value.values[2].number_value, 123)

    def test_nested_dict_with_none(self):
        test_dict = {
            "level1": {
                "level2": {
                    "value": None,
                    "key": "test"
                },
                "null_key": None
            }
        }
        result = pipeline_spec_builder.to_protobuf_value(test_dict)
        level1 = result.struct_value.fields["level1"].struct_value
        level2 = level1.fields["level2"].struct_value
        self.assertEqual(level2.fields["value"].null_value, struct_pb2.NULL_VALUE)
        self.assertEqual(level2.fields["key"].string_value, "test")
        self.assertEqual(level1.fields["null_key"].null_value, struct_pb2.NULL_VALUE)

    def test_mixed_nested_structure(self):
        test_data = {
            "string": "value",
            "number": 42,
            "list": [1, None, "three", {"nested": None}],
            "dict": {"key": None},
            "null": None
        }
        result = pipeline_spec_builder.to_protobuf_value(test_data)
        
        struct = result.struct_value
        self.assertEqual(struct.fields["string"].string_value, "value")
        self.assertEqual(struct.fields["number"].number_value, 42)
        self.assertEqual(struct.fields["null"].null_value, struct_pb2.NULL_VALUE)
        
        list_values = struct.fields["list"].list_value.values
        self.assertEqual(list_values[0].number_value, 1)
        self.assertEqual(list_values[1].null_value, struct_pb2.NULL_VALUE)
        self.assertEqual(list_values[2].string_value, "three")
        self.assertEqual(list_values[3].struct_value.fields["nested"].null_value, struct_pb2.NULL_VALUE)
        
        self.assertEqual(struct.fields["dict"].struct_value.fields["key"].null_value, struct_pb2.NULL_VALUE)

    def test_invalid_type_raises_error(self):
        with self.assertRaises(ValueError) as context:
            pipeline_spec_builder.to_protobuf_value(object())
        
        self.assertIn("Value must be one of the following types", str(context.exception))


if __name__ == '__main__':
    unittest.main()
