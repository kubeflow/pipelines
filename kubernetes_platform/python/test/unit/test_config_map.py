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
                                    'mountPath': 'cmpath'
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
                                        'mountPath': 'cmpath1'
                                    },
                                    {
                                        'configMapName': 'cm-name2',
                                        'mountPath': 'cmpath2'
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
                                    'keyToEnv': [{
                                        'configMapKey': 'foo',
                                        'envVar': 'CM_VAR'
                                    }]
                                }],
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name2',
                                    'mountPath': 'cmpath2'
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
                                    'mountPath': 'path'
                                }],
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name',
                                    'mountPath': 'cmpath'
                                }]
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
                                        'keyToEnv': [{
                                            'configMapKey': 'foo1',
                                            'envVar': 'CM_VAR1'
                                        }]
                                    },
                                    {
                                        'configMapName':
                                            'cm-name2',
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
                                    'keyToEnv': [{
                                        'configMapKey': 'foo',
                                        'envVar': 'CM_VAR'
                                    }]
                                }],
                                'configMapAsVolume': [{
                                    'configMapName': 'cm-name2',
                                    'mountPath': 'cmpath2'
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
                                'pvcMount': [{
                                    'constant': 'pvc-name',
                                    'mountPath': 'path'
                                }],
                                'configMapAsEnv': [{
                                    'configMapName':
                                        'cm-name',
                                    'keyToEnv': [{
                                        'configMapKey': 'foo',
                                        'envVar': 'CM_VAR'
                                    }]
                                }]
                            }
                        }
                    }
                }
            }
        }


@dsl.component
def comp():
    pass
