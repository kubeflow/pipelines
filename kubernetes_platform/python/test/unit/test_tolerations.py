# Copyright 2024 The Kubeflow Authors
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


class TestTolerations:

    def test_add_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration(
                task,
                key='key1',
                operator='Equal',
                value='value1',
                effect='NoSchedule',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [{
                                    'key': 'key1',
                                    'operator': 'Equal',
                                    'value': 'value1',
                                    'effect': 'NoSchedule',
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_one_with_toleration_seconds(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration(
                task,
                key='key1',
                operator='Equal',
                value='value1',
                effect='NoExecute',
                toleration_seconds=10,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [{
                                    'key': 'key1',
                                    'operator': 'Equal',
                                    'value': 'value1',
                                    'effect': 'NoExecute',
                                    'tolerationSeconds': '10',
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_two(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration(
                task,
                key='key1',
                operator='Equal',
                value='value1',
            )
            kubernetes.add_toleration(
                task,
                key='key2',
                operator='Equal',
                value='value2',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [
                                    {
                                        'key': 'key1',
                                        'operator': 'Equal',
                                        'value': 'value1',
                                    },
                                    {
                                        'key': 'key2',
                                        'operator': 'Equal',
                                        'value': 'value2',
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_respects_other_configuration(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_volume(
                task, secret_name='my-secret', mount_path='/mnt/my_vol',)
            kubernetes.add_toleration(
                task,
                key='key1',
                operator='Equal',
                value='value1',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [{
                                    'key': 'key1',
                                    'operator': 'Equal',
                                    'value': 'value1',
                                },],
                                'secretAsVolume': [{
                                    'secretName': 'my-secret',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'my-secret'}},
                                    'mountPath': '/mnt/my_vol',
                                    'optional': False
                                },],
                            },
                        }
                    }
                }
            }
        }


class TestTolerationsJSON:

    def test_component_pipeline_input_one(self):
        # checks that a pipeline input for
        # tasks is supported
        @dsl.pipeline
        def my_pipeline(toleration_input: str):
            task = comp()
            kubernetes.add_toleration_json(
                task,
                toleration_json=toleration_input,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [
                                    {
                                        'tolerationJson': {
                                            'componentInputParameter': 'toleration_input'
                                        }
                                    },
                                ]
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
        def my_pipeline(toleration_input_1: str, toleration_input_2: str):
            t1 = comp()
            kubernetes.add_toleration_json(
                t1,
                toleration_json=toleration_input_1,
            )
            t2 = comp()
            kubernetes.add_toleration_json(
                t2,
                toleration_json=toleration_input_1,
            )
            kubernetes.add_toleration_json(
                t2,
                toleration_json=toleration_input_2,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [
                                    {
                                        'tolerationJson': {
                                            'componentInputParameter': 'toleration_input_1'
                                        }
                                    },
                                ]
                            },
                            'exec-comp-2': {
                                'tolerations': [
                                    {
                                        'tolerationJson': {
                                            'componentInputParameter': 'toleration_input_1'
                                        }
                                    },
                                    {
                                        'tolerationJson': {
                                            'componentInputParameter': 'toleration_input_2'
                                        }
                                    }
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
            kubernetes.add_toleration_json(
                t1,
                toleration_json=t2.output,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [
                                    {
                                        'tolerationJson': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
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

    def test_component_upstream_input_two(self):
        # checks that multiple upstream task input
        # parameters are supported
        @dsl.pipeline
        def my_pipeline():
            t1 = comp()
            t2 = comp_with_output()
            t3 = comp_with_output()
            t4 = comp_with_output()
            kubernetes.add_toleration_json(
                t1,
                toleration_json=t2.output,
            )

            t5 = comp()
            kubernetes.add_toleration_json(
                t5,
                toleration_json=t3.output,
            )
            kubernetes.add_toleration_json(
                t5,
                toleration_json=t4.output,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'tolerations': [
                                    {
                                        'tolerationJson': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        }
                                    },
                                ]
                            },
                            'exec-comp-2': {
                                'tolerations': [
                                    {
                                        'tolerationJson':{
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-2'
                                            }
                                        }
                                    },
                                    {
                                        'tolerationJson': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-3'
                                            }
                                        }
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
