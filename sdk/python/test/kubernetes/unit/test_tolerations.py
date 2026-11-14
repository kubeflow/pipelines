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

import copy

import pytest
from google.protobuf import json_format
from kfp import dsl, kubernetes


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
                                'tolerations': [
                                    {
                                        'key': 'key1',
                                        'operator': 'Equal',
                                        'value': 'value1',
                                        'effect': 'NoSchedule',
                                    }
                                ]
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
                                'tolerations': [
                                    {
                                        'key': 'key1',
                                        'operator': 'Equal',
                                        'value': 'value1',
                                        'effect': 'NoExecute',
                                        'tolerationSeconds': '10',
                                    }
                                ]
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
                task,
                secret_name='my-secret',
                mount_path='/mnt/my_vol',
            )
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
                                'tolerations': [
                                    {
                                        'key': 'key1',
                                        'operator': 'Equal',
                                        'value': 'value1',
                                    },
                                ],
                                'secretAsVolume': [
                                    {
                                        'secretName': 'my-secret',
                                        'secretNameParameter': {
                                            'runtimeValue': {
                                                'constant': 'my-secret'
                                            }
                                        },
                                        'mountPath': '/mnt/my_vol',
                                        'optional': False,
                                    },
                                ],
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
        def my_pipeline(toleration_input: dict):
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
        def my_pipeline(toleration_input_1: dict, toleration_input_2: list):
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
                                    },
                                ]
                            },
                        }
                    }
                }
            }
        }

    def test_component_pipeline_input_three(self):
        # check list or dict types for add
        # toleration json
        @dsl.pipeline
        def my_pipeline(toleration_input: dict):
            t1 = comp()
            kubernetes.add_toleration_json(
                t1,
                toleration_json=toleration_input,
            )
            kubernetes.add_toleration_json(
                t1,
                toleration_json={
                    'key': 'key3',
                    'operator': 'Equal',
                    'value': 'value3',
                    'effect': 'NoSchedule',
                },
            )
            kubernetes.add_toleration_json(
                t1,
                toleration_json=[
                    {
                        'key': 'key1',
                        'operator': 'Equal',
                        'value': 'value1',
                        'effect': 'NoSchedule',
                    },
                    {
                        'key': 'key2',
                        'operator': 'Exists',
                        'effect': 'NoExecute',
                    },
                ],
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
                                    {
                                        'key': 'key3',
                                        'operator': 'Equal',
                                        'value': 'value3',
                                        'effect': 'NoSchedule',
                                    },
                                    {
                                        'key': 'key1',
                                        'operator': 'Equal',
                                        'value': 'value1',
                                        'effect': 'NoSchedule',
                                    },
                                    {
                                        'key': 'key2',
                                        'operator': 'Exists',
                                        'effect': 'NoExecute',
                                    },
                                ]
                            },
                        }
                    }
                }
            }
        }

    def test_static_dict_camel_case_toleration_seconds(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration_json(
                task,
                toleration_json={
                    'key': 'key1',
                    'operator': 'Equal',
                    'value': 'value1',
                    'effect': 'NoExecute',
                    'tolerationSeconds': 300,
                },
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
                                        'effect': 'NoExecute',
                                        'tolerationSeconds': '300',
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_static_list_camel_case_toleration_seconds(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration_json(
                task,
                toleration_json=[
                    {
                        'key': 'key1',
                        'operator': 'Equal',
                        'value': 'value1',
                        'effect': 'NoExecute',
                        'tolerationSeconds': 300,
                    },
                    {
                        'key': 'key2',
                        'operator': 'Equal',
                        'value': 'value2',
                        'effect': 'NoExecute',
                        'tolerationSeconds': 60,
                    },
                ],
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
                                        'effect': 'NoExecute',
                                        'tolerationSeconds': '300',
                                    },
                                    {
                                        'key': 'key2',
                                        'operator': 'Equal',
                                        'value': 'value2',
                                        'effect': 'NoExecute',
                                        'tolerationSeconds': '60',
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_static_dict_camel_case_zero_toleration_seconds(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration_json(
                task,
                toleration_json={
                    'key': 'key1',
                    'operator': 'Equal',
                    'value': 'value1',
                    'effect': 'NoExecute',
                    'tolerationSeconds': 0,
                },
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
                                        'effect': 'NoExecute',
                                        'tolerationSeconds': '0',
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_static_snake_case_toleration_seconds_still_supported(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration_json(
                task,
                toleration_json={
                    'key': 'key1',
                    'operator': 'Equal',
                    'value': 'value1',
                    'effect': 'NoExecute',
                    'toleration_seconds': 50,
                },
            )
            kubernetes.add_toleration_json(
                task,
                toleration_json=[
                    {
                        'key': 'key2',
                        'operator': 'Equal',
                        'value': 'value2',
                        'effect': 'NoExecute',
                        'toleration_seconds': 0,
                    }
                ],
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
                                        'effect': 'NoExecute',
                                        'tolerationSeconds': '50',
                                    },
                                    {
                                        'key': 'key2',
                                        'operator': 'Equal',
                                        'value': 'value2',
                                        'effect': 'NoExecute',
                                        'tolerationSeconds': '0',
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_static_omitted_toleration_seconds(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration_json(
                task,
                toleration_json={
                    'key': 'key1',
                    'operator': 'Equal',
                    'value': 'value1',
                    'effect': 'NoSchedule',
                },
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
                                        'effect': 'NoSchedule',
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_static_camel_case_input_is_not_mutated(self):
        input_dict = {
            'key': 'key1',
            'operator': 'Equal',
            'value': 'value1',
            'effect': 'NoExecute',
            'tolerationSeconds': 300,
        }
        input_list = [
            {
                'key': 'key2',
                'operator': 'Equal',
                'value': 'value2',
                'effect': 'NoExecute',
                'tolerationSeconds': 60,
            },
            {
                'key': 'key3',
                'operator': 'Equal',
                'value': 'value3',
                'effect': 'NoExecute',
                'tolerationSeconds': 0,
            },
        ]
        expected_dict = copy.deepcopy(input_dict)
        expected_list = copy.deepcopy(input_list)

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_toleration_json(task, toleration_json=input_dict)
            kubernetes.add_toleration_json(task, toleration_json=input_list)

        assert input_dict == expected_dict
        assert input_list == expected_list

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
                                                'producerTask': 'comp-with-output',
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
                                                'producerTask': 'comp-with-output',
                                            }
                                        }
                                    },
                                ]
                            },
                            'exec-comp-2': {
                                'tolerations': [
                                    {
                                        'tolerationJson': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-2',
                                            }
                                        }
                                    },
                                    {
                                        'tolerationJson': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-3',
                                            }
                                        }
                                    },
                                ]
                            },
                        }
                    }
                }
            }
        }

    @pytest.mark.parametrize('operator', ['Equal', 'Exists', 'Gt', 'Lt'])
    def test_valid_operators(self, operator):

        @dsl.pipeline
        def my_pipeline():
            kubernetes.add_toleration(comp(), key='k', operator=operator)

        tolerations = json_format.MessageToDict(my_pipeline.platform_spec)[
            'platforms'
        ]['kubernetes']['deploymentSpec']['executors']['exec-comp'][
            'tolerations'
        ]
        assert tolerations[0]['operator'] == operator

    @pytest.mark.parametrize(
        'effect', ['NoExecute', 'NoSchedule', 'PreferNoSchedule']
    )
    def test_valid_effects(self, effect):

        @dsl.pipeline
        def my_pipeline():
            kubernetes.add_toleration(comp(), key='k', effect=effect)

        tolerations = json_format.MessageToDict(my_pipeline.platform_spec)[
            'platforms'
        ]['kubernetes']['deploymentSpec']['executors']['exec-comp'][
            'tolerations'
        ]
        assert tolerations[0]['effect'] == effect

    @pytest.mark.parametrize('operator,effect', [(None, None), ('', '')])
    def test_omitted_or_empty_operator_and_effect(self, operator, effect):

        @dsl.pipeline
        def my_pipeline():
            kubernetes.add_toleration(
                comp(), key='k', operator=operator, effect=effect
            )

        tolerations = json_format.MessageToDict(my_pipeline.platform_spec)[
            'platforms'
        ]['kubernetes']['deploymentSpec']['executors']['exec-comp'][
            'tolerations'
        ]
        assert tolerations == [{'key': 'k'}]

    def test_invalid_operator_raises(self):
        with pytest.raises(
            ValueError, match="Invalid toleration operator 'Bad'"
        ):

            @dsl.pipeline
            def my_pipeline():
                kubernetes.add_toleration(comp(), key='k', operator='Bad')

    def test_invalid_effect_raises(self):
        with pytest.raises(ValueError, match="Invalid toleration effect 'Bad'"):

            @dsl.pipeline
            def my_pipeline():
                kubernetes.add_toleration(comp(), key='k', effect='Bad')

    @pytest.mark.parametrize(
        'toleration_json',
        [
            {'key': 'k', 'operator': 'Bad'},
            {'key': 'k', 'effect': 'Bad'},
            [{'key': 'k', 'operator': 'Equal'}, {'key': 'k', 'effect': 'Bad'}],
        ],
    )
    def test_add_toleration_json_static_invalid_raises(self, toleration_json):
        with pytest.raises(ValueError, match='Invalid toleration'):

            @dsl.pipeline
            def my_pipeline():
                kubernetes.add_toleration_json(comp(), toleration_json)

    def test_add_toleration_json_static_valid_gt(self):

        @dsl.pipeline
        def my_pipeline():
            kubernetes.add_toleration_json(
                comp(),
                {
                    'key': 'k',
                    'operator': 'Gt',
                    'value': '5',
                    'effect': 'NoSchedule',
                },
            )

        tolerations = json_format.MessageToDict(my_pipeline.platform_spec)[
            'platforms'
        ]['kubernetes']['deploymentSpec']['executors']['exec-comp'][
            'tolerations'
        ]
        assert tolerations[0]['operator'] == 'Gt'


@dsl.component
def comp():
    pass


@dsl.component()
def comp_with_output() -> str:
    pass
