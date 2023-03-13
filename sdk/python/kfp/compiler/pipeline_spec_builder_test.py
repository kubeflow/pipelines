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

import unittest

from absl.testing import parameterized
from google.protobuf import json_format
from google.protobuf import struct_pb2
from kfp.compiler import pipeline_spec_builder
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


def pipeline_spec_from_file(filepath: str) -> str:
    with open(filepath, 'r') as f:
        dictionary = yaml.safe_load(f)
    return json_format.ParseDict(dictionary, pipeline_spec_pb2.PipelineSpec())


if __name__ == '__main__':
    unittest.main()
