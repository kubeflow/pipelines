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
import pytest


class TestMountPVC:

    def test_mount_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'constant': 'pvc-name',
                                    'pvcNameParameter': {'runtimeValue': {'constant': 'pvc-name'}},
                                    'mountPath': 'path'
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_mount_two(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path1',
            )
            kubernetes.mount_pvc(
                task,
                pvc_name='other-pvc-name',
                mount_path='path2',
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
                                        'pvcNameParameter': {'runtimeValue': {'constant': 'pvc-name'}},
                                        'mountPath': 'path1'
                                    },
                                    {
                                        'constant': 'other-pvc-name',
                                        'pvcNameParameter': {'runtimeValue': {'constant': 'other-pvc-name'}},
                                        'mountPath': 'path2'
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_mount_with_sub_path(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='/data',
                sub_path='models',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'constant': 'pvc-name',
                                    'pvcNameParameter': {'runtimeValue': {'constant': 'pvc-name'}},
                                    'mountPath': '/data',
                                    'subPath': 'models'
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_mount_with_empty_sub_path(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='/data',
                sub_path='',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'constant': 'pvc-name',
                                    'pvcNameParameter': {'runtimeValue': {'constant': 'pvc-name'}},
                                    'mountPath': '/data'
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_mount_preserves_secret_as_env(self):
        # checks that mount_pvc respects previously set secrets
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_env(
                task,
                secret_name='secret-name',
                secret_key_to_env={'password': 'SECRET_VAR'},
            )
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'constant': 'pvc-name',
                                    'pvcNameParameter': {'runtimeValue': {'constant': 'pvc-name'}},
                                    'mountPath': 'path'
                                }],
                                'secretAsEnv': [{
                                    'secretName': 'secret-name',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'secret-name'}},
                                    'keyToEnv': [{
                                        'secretKey': 'password',
                                        'envVar': 'SECRET_VAR'
                                    }],
                                    'optional': False
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_mount_preserves_secret_as_vol(self):
        # checks that mount_pvc respects previously set secrets
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name',
                mount_path='secretpath',
            )
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'constant': 'pvc-name',
                                    'pvcNameParameter': {'runtimeValue': {'constant': 'pvc-name'}},
                                    'mountPath': 'path'
                                }],
                                'secretAsVolume': [{
                                    'secretName': 'secret-name',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'secret-name'}},
                                    'mountPath': 'secretpath',
                                    'optional': False
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_illegal_pvc_name(self):

        @dsl.component
        def identity(string: str) -> str:
            return string

        with pytest.raises(
                ValueError,
                match=r"Argument for 'input_param' must be an instance of str, dict, or PipelineChannel. "
                      r"Got unknown input type: <class 'int'>.",
        ):

            @dsl.pipeline
            def my_pipeline(string: str = 'string'):
                op1 = kubernetes.mount_pvc(
                    identity(string=string),
                    pvc_name=1,
                    mount_path='/path',
                )

    def test_component_pipeline_input_one(self):
        # checks that a pipeline input for
        # tasks is supported
        @dsl.pipeline
        def my_pipeline(pvc_name_input: str):
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name=pvc_name_input,
                mount_path='path',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'componentInputParameter': 'pvc_name_input',
                                    'pvcNameParameter': {
                                        'componentInputParameter': 'pvc_name_input'
                                    },
                                    'mountPath': 'path'
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
        def my_pipeline(pvc_name_input_1: str, pvc_name_input_2: str):
            t1 = comp()
            kubernetes.mount_pvc(
                t1,
                pvc_name=pvc_name_input_1,
                mount_path='path',
            )
            t2 = comp()
            kubernetes.mount_pvc(
                t2,
                pvc_name=pvc_name_input_1,
                mount_path='path',
            )
            kubernetes.mount_pvc(
                t2,
                pvc_name=pvc_name_input_2,
                mount_path='path',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [
                                    {
                                        'componentInputParameter': 'pvc_name_input_1',
                                        'pvcNameParameter': {
                                            'componentInputParameter': 'pvc_name_input_1'
                                        },
                                        'mountPath': 'path'
                                    },
                                ]
                            },
                            'exec-comp-2': {
                                'pvcMount': [
                                    {
                                        'componentInputParameter': 'pvc_name_input_1',
                                        'pvcNameParameter': {
                                            'componentInputParameter': 'pvc_name_input_1'
                                        },
                                        'mountPath': 'path'
                                    },
                                    {
                                        'componentInputParameter': 'pvc_name_input_2',
                                        'pvcNameParameter': {
                                            'componentInputParameter': 'pvc_name_input_2'
                                        },
                                        'mountPath': 'path'
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
            kubernetes.mount_pvc(
                t1,
                pvc_name=t2.output,
                mount_path='path',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [{
                                    'taskOutputParameter': {
                                        'outputParameterKey': 'Output',
                                        'producerTask': 'comp-with-output'
                                    },
                                    'pvcNameParameter': {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output'
                                        }
                                    },
                                    'mountPath': 'path'
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
            t4 = comp_with_output()
            kubernetes.mount_pvc(
                t1,
                pvc_name=t2.output,
                mount_path='path1',
            )
            t5 = comp()
            kubernetes.mount_pvc(
                t5,
                pvc_name=t3.output,
                mount_path='path2',
            )
            kubernetes.mount_pvc(
                t5,
                pvc_name=t4.output,
                mount_path='path3',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'pvcMount': [
                                    {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output'
                                        },
                                        'pvcNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output'
                                            }
                                        },
                                        'mountPath': 'path1'
                                    },
                                ]
                            },
                            'exec-comp-2': {
                                'pvcMount': [
                                    {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output-2'
                                        },
                                        'pvcNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-2'
                                            }
                                        },
                                        'mountPath': 'path2'
                                    },
                                    {
                                        'taskOutputParameter': {
                                            'outputParameterKey': 'Output',
                                            'producerTask': 'comp-with-output-3'
                                        },
                                        'pvcNameParameter': {
                                            'taskOutputParameter': {
                                                'outputParameterKey': 'Output',
                                                'producerTask': 'comp-with-output-3'
                                            }
                                        },
                                        'mountPath': 'path3'
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

class TestGenericEphemeralVolume:

    def test_mount_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_ephemeral_volume(
                task,
                volume_name='pvc-name',
                mount_path='path',
                access_modes=['ReadWriteOnce'],
                size='5Gi',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'genericEphemeralVolume': [{
                                    'volumeName': 'pvc-name',
                                    'mountPath': 'path',
                                    'accessModes': ['ReadWriteOnce'],
                                    'defaultStorageClass': True,
                                    'size': '5Gi',
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_mount_two(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_ephemeral_volume(
                task,
                volume_name='pvc-name',
                mount_path='path1',
                access_modes=['ReadWriteOnce'],
                size='5Gi',
            )
            kubernetes.add_ephemeral_volume(
                task,
                volume_name='other-pvc-name',
                mount_path='path2',
                access_modes=['ReadWriteMany'],
                size='10Ti',
                storage_class_name='gp2',
                labels={
                    'label1': 'l1',
                },
                annotations={
                    'annotation1': 'a1',
                }
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'genericEphemeralVolume': [
                                    {
                                        'volumeName': 'pvc-name',
                                        'mountPath': 'path1',
                                        'accessModes': ['ReadWriteOnce'],
                                        'defaultStorageClass': True,
                                        'size': '5Gi',
                                    },
                                    {
                                        'volumeName': 'other-pvc-name',
                                        'mountPath': 'path2',
                                        'accessModes': ['ReadWriteMany'],
                                        'size': '10Ti',
                                        'storageClassName': 'gp2',
                                        'metadata': {
                                            'labels': {'label1': 'l1'},
                                            'annotations': {'annotation1': 'a1'},
                                        },
                                    },
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
