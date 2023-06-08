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
                                    'mountPath': 'secretpath'
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


@dsl.component
def comp():
    pass
