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


class TestNodeSelector:

    def test_add_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_node_selector(
                task,
                label_key='cloud.google.com/gke-accelerator',
                label_value='nvidia-tesla-p4',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeSelector': {
                                    'labels': {
                                        'cloud.google.com/gke-accelerator':
                                            'nvidia-tesla-p4'
                                    }
                                }
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
            kubernetes.add_node_selector(
                task,
                label_key='cloud.google.com/gke-accelerator',
                label_value='nvidia-tesla-p4',
            )
            kubernetes.add_node_selector(
                task,
                label_key='other_label_key',
                label_value='other_label_value',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeSelector': {
                                    'labels': {
                                        'cloud.google.com/gke-accelerator':
                                            'nvidia-tesla-p4',
                                        'other_label_key':
                                            'other_label_value'
                                    },
                                }
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
                task, secret_name='my-secret', mount_path='/mnt/my_vol')
            kubernetes.add_node_selector(
                task,
                label_key='cloud.google.com/gke-accelerator',
                label_value='nvidia-tesla-p4',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeSelector': {
                                    'labels': {
                                        'cloud.google.com/gke-accelerator':
                                            'nvidia-tesla-p4'
                                    }
                                },
                                'secretAsVolume': [{
                                    'secretName': 'my-secret',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'my-secret'}},
                                    'mountPath': '/mnt/my_vol',
                                    'optional': False
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
        def my_pipeline(selector_input: str):
            task = comp()
            kubernetes.add_node_selector(
                task,
                node_selector_json=selector_input,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeSelector': {
                                    'nodeSelectorJson': {
                                        'componentInputParameter': 'selector_input'
                                    },
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_component_pipeline_input_two(self):
        # checks that multiple pipeline inputs for
        # different tasks are supported. And confirm
        # that applying node selector json multiple
        # times overwrites the previous.
        @dsl.pipeline
        def my_pipeline(selector_input_1: str, selector_input_2: str):
            t1 = comp()
            kubernetes.add_node_selector(
                t1,
                node_selector_json=selector_input_1,
            )

            t2 = comp()
            kubernetes.add_node_selector(
                t2,
                node_selector_json=selector_input_1,
            )
            # This should overwrite the previous selector json
            kubernetes.add_node_selector(
                t2,
                node_selector_json=selector_input_2,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeSelector': {
                                    'nodeSelectorJson': {
                                        'componentInputParameter': 'selector_input_1'
                                    },
                                }
                            },
                            'exec-comp-2': {
                                'nodeSelector': {
                                    'nodeSelectorJson': {
                                        'componentInputParameter': 'selector_input_2'
                                    },
                                }
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
            kubernetes.add_node_selector(
                t1,
                node_selector_json=t2.output,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeSelector': {
                                    'nodeSelectorJson': {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output'
                                        }
                                    },
                                }
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
        def my_pipeline(selector_input: str):
            t1 = comp()
            t2 = comp_with_output()
            kubernetes.add_node_selector(
                t1,
                node_selector_json=t2.output,
            )
            t3 = comp()
            t4 = comp_with_output()
            kubernetes.add_node_selector(
                t3,
                node_selector_json=t4.output,
            )
            # overwrites the previous
            t5 = comp_with_output()
            kubernetes.add_node_selector(
                t3,
                node_selector_json=t5.output,
            )
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeSelector': {
                                    'nodeSelectorJson': {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output'
                                        }
                                    },
                                }
                            },
                            'exec-comp-2': {
                                'nodeSelector': {
                                    'nodeSelectorJson': {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output-3'
                                        }
                                    },
                                }
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
