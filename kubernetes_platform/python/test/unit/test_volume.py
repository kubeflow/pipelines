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
                                        'mountPath': 'path1'
                                    },
                                    {
                                        'constant': 'other-pvc-name',
                                        'mountPath': 'path2'
                                    },
                                ]
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
                                    'mountPath': 'path'
                                }],
                                'secretAsEnv': [{
                                    'secretName':
                                        'secret-name',
                                    'keyToEnv': [{
                                        'secretKey': 'password',
                                        'envVar': 'SECRET_VAR'
                                    }]
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
                                    'mountPath': 'path'
                                }],
                                'secretAsVolume': [{
                                    'secretName': 'secret-name',
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
                match=r'Argument for \'pvc_name\' must be an instance of str or PipelineChannel\. Got unknown input type: <class \'int\'>\.',
        ):

            @dsl.pipeline
            def my_pipeline(string: str = 'string'):
                op1 = kubernetes.mount_pvc(
                    identity(string=string),
                    pvc_name=1,
                    mount_path='/path',
                )


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
