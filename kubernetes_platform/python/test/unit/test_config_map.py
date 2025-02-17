# Copyright 2023 The Kubeflow Authors
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

from google.protobuf import json_format
from kfp import dsl
from kfp import kubernetes
from kfp.dsl import OutputPath

class TestUseConfigMapAsVolume:

    def test_use_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name',
                mount_path='cmpath',
            )
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name',
                                    'mountPath': 'cmpath',
                                    'optional': False,
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name'
                                        }
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_use_one_optional_true(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name',
                mount_path='cmpath',
                optional=True)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name',
                                    'mountPath': 'cmpath',
                                    'optional': True,
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name'
                                        }
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_use_one_optional_false(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name',
                mount_path='cmpath',
                optional=False)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name',
                                    'mountPath': 'cmpath',
                                    'optional': False,
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name'
                                        }
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_use_two(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name1',
                mount_path='cmpath1',
            )
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name2',
                mount_path='cmpath2',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [
                                    {
                                        'configMapName': 'cm-name1',
                                        'mountPath': 'cmpath1',
                                        'optional': False,
                                        'configNameParameter': {
                                            'runtimeValue': {
                                                'constant': 'cm-name1'
                                            }
                                        }
                                    },
                                    {
                                        'configMapName': 'cm-name2',
                                        'mountPath': 'cmpath2',
                                        'optional': False,
                                        'configNameParameter': {
                                            'runtimeValue': {
                                                'constant': 'cm-name2'
                                            }
                                        }
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_preserves_config_map_as_env(self):
        # checks that use_config map_as_volume respects previously set config maps as env

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_env(
                task,
                config_map_name='cm-name1',
                config_map_key_to_env={'foo': 'CM_VAR'},
            )
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name2',
                mount_path='cmpath2',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [{
                                    'configMapName':
                                        'cm-name1',
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name1'
                                        }
                                    },
                                    'keyToEnv': [{
                                        'configMapKey': 'foo',
                                        'envVar': 'CM_VAR'
                                    }]
                                }],
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name2',
                                    'mountPath': 'cmpath2',
                                    'optional': False,
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name2'
                                        }
                                    }
                                },]
                            }
                        }
                    }
                }
            }
        }

    def test_alongside_pvc_mount(self):
        # checks that use_config_map_as_volume respects previously set pvc
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path',
            )
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name',
                mount_path='cmpath',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'constant': 'pvc-name',
                                    'pvcNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'pvc-name'
                                        }
                                    },
                                    'mountPath': 'path'
                                }],
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name',
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name'
                                        }
                                    },
                                    'mountPath': 'cmpath',
                                    'optional': False,
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_component_pipeline_input_one(self):
        # checks that a pipeline input for
        # tasks is supported
        @dsl.pipeline
        def my_pipeline(cm_name_input_1: str):
            task = comp()
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name=cm_name_input_1,
                mount_path="cmpath"
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [{
                                    'configNameParameter': {
                                        'componentInputParameter': 'cm_name_input_1'
                                    },
                                    'mountPath': 'cmpath',
                                    'optional': False,
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_component_pipeline_input_two(self):
        # checks that multiple pipeline inputs for
        # different tasks are supported
        @dsl.pipeline
        def my_pipeline(cm_name_input_1: str, cm_name_input_2: str):
            t1 = comp()
            kubernetes.use_config_map_as_volume(
                t1,
                config_map_name=cm_name_input_1,
                mount_path="cmpath"
            )
            kubernetes.use_config_map_as_volume(
                t1,
                config_map_name=cm_name_input_2,
                mount_path="cmpath"
            )

            t2 = comp()
            kubernetes.use_config_map_as_volume(
                t2,
                config_map_name=cm_name_input_2,
                mount_path="cmpath"
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [
                                    {
                                        'configNameParameter': {
                                            'componentInputParameter': 'cm_name_input_1'
                                        },
                                        'mountPath': 'cmpath',
                                        'optional': False,
                                    },
                                    {
                                        'configNameParameter': {
                                            'componentInputParameter': 'cm_name_input_2'
                                        },
                                        'mountPath': 'cmpath',
                                        'optional': False,
                                    }

                                ]
                            },
                            'exec-comp-2': {
                                'configMapAsVolume': [
                                    {
                                        'configNameParameter': {
                                            'componentInputParameter': 'cm_name_input_2'
                                        },
                                        'mountPath': 'cmpath',
                                        'optional': False,
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_component_upstream_input_one(self):
        # checks that upstream task input parameters
        # are supported
        @dsl.pipeline
        def my_pipeline():
            t1 = comp()
            t2 = comp_with_output()
            kubernetes.use_config_map_as_volume(
                t1,
                config_map_name=t2.output,
                mount_path="cmpath"
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [{
                                    'configNameParameter': {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output'
                                        }
                                    },
                                    'mountPath': 'cmpath',
                                    'optional': False,
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_component_upstream_input_two(self):
        # checks that multiple upstream task input
        # parameters are supported
        @dsl.pipeline
        def my_pipeline():
            t1 = comp()
            t2 = comp_with_output()
            t3 = comp_with_output()

            kubernetes.use_config_map_as_volume(
                t1,
                config_map_name=t2.output,
                mount_path="cmpath"
            )
            kubernetes.use_config_map_as_volume(
                t1,
                config_map_name=t3.output,
                mount_path="cmpath"
            )
            t4 = comp()
            kubernetes.use_config_map_as_volume(
                t4,
                config_map_name=t2.output,
                mount_path="cmpath"
            )
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsVolume': [
                                    {
                                        'configNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        },
                                        'mountPath': 'cmpath',
                                        'optional': False,
                                    },
                                    {
                                        'configNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-2'
                                            }
                                        },
                                        'mountPath': 'cmpath',
                                        'optional': False,
                                    }
                                ]
                            },
                            'exec-comp-2': {
                                'configMapAsVolume': [
                                    {
                                        'configNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        },
                                        'mountPath': 'cmpath',
                                        'optional': False,
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }

class TestUseConfigMapAsEnv:

    def test_use_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_env(
                task,
                config_map_name='cm-name',
                config_map_key_to_env={
                    'foo': 'FOO',
                    'bar': 'BAR',
                },
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [{
                                    'configMapName':
                                        'cm-name',
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name'
                                        }
                                    },
                                    'keyToEnv': [
                                        {
                                            'configMapKey': 'foo',
                                            'envVar': 'FOO'
                                        },
                                        {
                                            'configMapKey': 'bar',
                                            'envVar': 'BAR'
                                        },
                                    ]
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_use_two(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_env(
                task,
                config_map_name='cm-name1',
                config_map_key_to_env={'foo1': 'CM_VAR1'},
            )
            kubernetes.use_config_map_as_env(
                task,
                config_map_name='cm-name2',
                config_map_key_to_env={'foo2': 'CM_VAR2'},
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [
                                    {
                                        'configMapName':
                                            'cm-name1',
                                        'configNameParameter': {
                                            'runtimeValue': {
                                                'constant': 'cm-name1'
                                            }
                                        },
                                        'keyToEnv': [{
                                            'configMapKey': 'foo1',
                                            'envVar': 'CM_VAR1'
                                        }]
                                    },
                                    {
                                        'configMapName': 'cm-name2',
                                        'configNameParameter': {
                                            'runtimeValue': {
                                                'constant': 'cm-name2'
                                            }
                                        },
                                        'keyToEnv': [{
                                            'configMapKey': 'foo2',
                                            'envVar': 'CM_VAR2'
                                        }]
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_preserves_config_map_as_volume(self):
        # checks that use_config_map_as_env respects previously set ConfigMaps as vol

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_config_map_as_volume(
                task,
                config_map_name='cm-name2',
                mount_path='cmpath2',
            )
            kubernetes.use_config_map_as_env(
                task,
                config_map_name='cm-name1',
                config_map_key_to_env={'foo': 'CM_VAR'},
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [{
                                    'configMapName':
                                        'cm-name1',
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name1'
                                        }
                                    },
                                    'keyToEnv': [{
                                        'configMapKey': 'foo',
                                        'envVar': 'CM_VAR'
                                    }]
                                }],
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name2',
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name2'
                                        }
                                    },
                                    'mountPath': 'cmpath2',
                                    'optional': False
                                },]
                            }
                        }
                    }
                }
            }
        }

    def test_preserves_pvc_mount(self):
        # checks that use_config_map_as_env respects previously set pvc
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path',
            )
            kubernetes.use_config_map_as_env(
                task,
                config_map_name='cm-name',
                config_map_key_to_env={'foo': 'CM_VAR'},
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [
                                    {
                                        'constant': 'pvc-name',
                                        'pvcNameParameter': {
                                            'runtimeValue': {
                                                'constant': 'pvc-name'
                                            }
                                        },
                                        'mountPath': 'path'
                                    },
                                ],
                                'configMapAsEnv': [{
                                    'configMapName': 'cm-name',
                                    'configNameParameter': {
                                        'runtimeValue': {
                                            'constant': 'cm-name'
                                        }
                                    },
                                    'keyToEnv': [
                                        {
                                            'configMapKey': 'foo',
                                            'envVar': 'CM_VAR'
                                        },
                                    ]
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_component_pipeline_input_one(self):
        # checks that a pipeline input for
        # tasks is supported
        @dsl.pipeline
        def my_pipeline(cm_name_input_1: str):
            task = comp()
            kubernetes.use_config_map_as_env(
                task,
                config_map_name=cm_name_input_1,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [{
                                    'configNameParameter': {
                                        'componentInputParameter': 'cm_name_input_1'
                                    },
                                    'keyToEnv': [
                                        {
                                            'configMapKey': 'foo',
                                            'envVar': 'CM_VAR'
                                        },
                                    ]
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_component_pipeline_input_two(self):
        # checks that multiple pipeline inputs for
        # different tasks are supported
        @dsl.pipeline
        def my_pipeline(cm_name_input_1: str, cm_name_input_2: str):
            t1 = comp()
            kubernetes.use_config_map_as_env(
                t1,
                config_map_name=cm_name_input_1,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )
            kubernetes.use_config_map_as_env(
                t1,
                config_map_name=cm_name_input_2,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )
            t2 = comp()
            kubernetes.use_config_map_as_env(
                t2,
                config_map_name=cm_name_input_2,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [
                                    {
                                        'configNameParameter': {
                                            'componentInputParameter': 'cm_name_input_1'
                                        },
                                        'keyToEnv': [
                                            {
                                                'configMapKey': 'foo',
                                                'envVar': 'CM_VAR'
                                            },
                                        ]
                                    },
                                    {
                                        'configNameParameter': {
                                            'componentInputParameter': 'cm_name_input_2'
                                        },
                                        'keyToEnv': [
                                            {
                                                'configMapKey': 'foo',
                                                'envVar': 'CM_VAR'
                                            },
                                        ]
                                    },
                                ]
                            },
                            'exec-comp-2': {
                                'configMapAsEnv': [
                                    {
                                        'configNameParameter': {
                                            'componentInputParameter': 'cm_name_input_2'
                                        },
                                        'keyToEnv': [
                                            {
                                                'configMapKey': 'foo',
                                                'envVar': 'CM_VAR'
                                            },
                                        ]
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_component_upstream_input_one(self):
        # checks that upstream task input parameters
        # are supported
        @dsl.pipeline
        def my_pipeline():
            t1 = comp()
            t2 = comp_with_output()
            kubernetes.use_config_map_as_env(
                t1,
                config_map_name=t2.output,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [{
                                    'configNameParameter': {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output'
                                        }
                                    },
                                    'keyToEnv': [
                                        {
                                            'configMapKey': 'foo',
                                            'envVar': 'CM_VAR'
                                        },
                                    ]
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_component_upstream_input_two(self):
        # checks that multiple upstream task input
        # parameters are supported
        @dsl.pipeline
        def my_pipeline():
            t1 = comp()
            t2 = comp_with_output()
            t3 = comp_with_output()
            kubernetes.use_config_map_as_env(
                t1,
                config_map_name=t2.output,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )
            kubernetes.use_config_map_as_env(
                t1,
                config_map_name=t3.output,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )

            t4 = comp()
            kubernetes.use_config_map_as_env(
                t4,
                config_map_name=t2.output,
                config_map_key_to_env={
                    'foo': 'CM_VAR',
                },
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'configMapAsEnv': [
                                    {
                                        'configNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        },
                                        'keyToEnv': [
                                            {
                                                'configMapKey': 'foo',
                                                'envVar': 'CM_VAR'
                                            },
                                        ]
                                    },
                                    {
                                        'configNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-2'
                                            }
                                        },
                                        'keyToEnv': [
                                            {
                                                'configMapKey': 'foo',
                                                'envVar': 'CM_VAR'
                                            },
                                        ]
                                    }
                                ]
                            },
                            'exec-comp-2': {
                                'configMapAsEnv': [
                                    {
                                        'configNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        },
                                        'keyToEnv': [
                                            {
                                                'configMapKey': 'foo',
                                                'envVar': 'CM_VAR'
                                            },
                                        ]
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }

@dsl.component
def comp():
    pass

@dsl.component()
def comp_with_output() -> str:
    pass
