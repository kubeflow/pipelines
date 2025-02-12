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
import pytest


class TestImagePullSecret:

    def test_add_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_image_pull_secrets(task, ['secret-name'])
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullSecret': [{
                                    'secretName': 'secret-name',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'secret-name'}}
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
            kubernetes.set_image_pull_secrets(task,
                                              ['secret-name1', 'secret-name2'])
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullSecret': [
                                    {
                                        'secretName': 'secret-name1',
                                        'secretNameParameter': {'runtimeValue': {'constant': 'secret-name1'}},
                                    },
                                    {
                                        'secretName': 'secret-name2',
                                        'secretNameParameter': {'runtimeValue': {'constant': 'secret-name2'}}
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

            # Load the secret as a volume
            kubernetes.use_secret_as_volume(
                task, secret_name='secret-name', mount_path='/mnt/my_vol')

            # Set image pull secrets for a task using secret names
            kubernetes.set_image_pull_secrets(task, ['secret-name'])

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsVolume': [{
                                    'secretName': 'secret-name',
                                    'mountPath': '/mnt/my_vol',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'secret-name'}},
                                    'optional': False
                                }],
                                'imagePullSecret': [{
                                    'secretName': 'secret-name',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'secret-name'}}
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
        def my_pipeline(input_1: str):
            t1 = comp()
            kubernetes.set_image_pull_secrets(t1, [input_1])
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullSecret': [
                                    {
                                        'secretNameParameter': {
                                            'componentInputParameter': 'input_1'
                                        }
                                    }
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
        def my_pipeline(input_1: str, input_2: str):
            t1 = comp()
            kubernetes.set_image_pull_secrets(t1, [input_1])
            t2 = comp()
            kubernetes.set_image_pull_secrets(t2, [input_1, input_2])
            kubernetes.set_image_pull_secrets(t2, ["another-secret"])
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullSecret': [
                                    {
                                        'secretNameParameter': {
                                            'componentInputParameter': 'input_1'
                                        }
                                    }
                                ]
                            },
                            'exec-comp-2': {
                                'imagePullSecret': [
                                    {
                                        'secretNameParameter': {
                                            'componentInputParameter': 'input_1'
                                        }
                                    },
                                    {
                                        'secretNameParameter': {
                                            'componentInputParameter': 'input_2'
                                        }
                                    },
                                    {
                                        'secretName': 'another-secret',
                                        'secretNameParameter': {'runtimeValue': {'constant': 'another-secret'}}
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
        def my_pipeline(input_1: str):
            t1 = comp()
            t2 = comp_with_output()
            kubernetes.set_image_pull_secrets(t1, [t2.output])

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullSecret': [
                                    {
                                        'secretNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
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

    def test_component_upstream_input_two(self):
        # checks that multiple upstream task input
        # parameters are supported
        @dsl.pipeline
        def my_pipeline(input_1: str, input_2: str):
            t1 = comp()
            t2 = comp_with_output()
            t3 = comp_with_output()
            t4 = comp_with_output()
            kubernetes.set_image_pull_secrets(t1, [t2.output])

            t5 = comp()
            kubernetes.set_image_pull_secrets(t5, [t2.output, t3.output])
            kubernetes.set_image_pull_secrets(t5, [t4.output])
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullSecret': [
                                    {
                                        'secretNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        }
                                    }
                                ]
                            },
                            'exec-comp-2': {
                                'imagePullSecret': [
                                    {
                                        'secretNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        }
                                    },
                                    {
                                        'secretNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-2'
                                            }
                                        }
                                    },
                                    {
                                        'secretNameParameter': {
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

    def test_invalid_input(self):
        # check for invalid input

        with pytest.raises(
                ValueError,
                match="must be a list of strings or a list of pipeline input parameters"
        ):
            @dsl.pipeline
            def my_pipeline():
                t1 = comp()
                kubernetes.set_image_pull_secrets(t1, ['a_str', 4])

@dsl.component
def comp():
    pass

@dsl.component()
def comp_with_output() -> str:
    pass
