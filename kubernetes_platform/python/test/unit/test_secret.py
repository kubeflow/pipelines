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


class TestUseSecretAsVolume:

    def test_use_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name',
                mount_path='secretpath',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
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

    def test_use_one_optional_true(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name',
                mount_path='secretpath',
                optional=True)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsVolume': [{
                                    'secretName': 'secret-name',
                                    'mountPath': 'secretpath',
                                    'optional': True
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
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name',
                mount_path='secretpath',
                optional=False)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
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

    def test_use_two(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name1',
                mount_path='secretpath1',
            )
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name2',
                mount_path='secretpath2',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsVolume': [
                                    {
                                        'secretName': 'secret-name1',
                                        'mountPath': 'secretpath1',
                                        'optional': False
                                    },
                                    {
                                        'secretName': 'secret-name2',
                                        'mountPath': 'secretpath2',
                                        'optional': False
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_preserves_secret_as_env(self):
        # checks that use_secret_as_volume respects previously set secrets as env

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_env(
                task,
                secret_name='secret-name1',
                secret_key_to_env={'password': 'SECRET_VAR'},
            )
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name2',
                mount_path='secretpath2',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsEnv': [{
                                    'secretName':
                                        'secret-name1',
                                    'keyToEnv': [{
                                        'secretKey': 'password',
                                        'envVar': 'SECRET_VAR'
                                    }]
                                }],
                                'secretAsVolume': [{
                                    'secretName': 'secret-name2',
                                    'mountPath': 'secretpath2',
                                    'optional': False
                                },]
                            }
                        }
                    }
                }
            }
        }

    def test_alongside_pvc_mount(self):
        # checks that use_secret_as_volume respects previously set pvc
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path',
            )
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name',
                mount_path='secretpath',
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


class TestUseSecretAsEnv:

    def test_use_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_env(
                task,
                secret_name='secret-name',
                secret_key_to_env={
                    'username': 'USERNAME',
                    'password': 'PASSWORD',
                },
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsEnv': [{
                                    'secretName':
                                        'secret-name',
                                    'keyToEnv': [
                                        {
                                            'secretKey': 'username',
                                            'envVar': 'USERNAME'
                                        },
                                        {
                                            'secretKey': 'password',
                                            'envVar': 'PASSWORD'
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
            kubernetes.use_secret_as_env(
                task,
                secret_name='secret-name1',
                secret_key_to_env={'password1': 'SECRET_VAR1'},
            )
            kubernetes.use_secret_as_env(
                task,
                secret_name='secret-name2',
                secret_key_to_env={'password2': 'SECRET_VAR2'},
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsEnv': [
                                    {
                                        'secretName':
                                            'secret-name1',
                                        'keyToEnv': [{
                                            'secretKey': 'password1',
                                            'envVar': 'SECRET_VAR1'
                                        }]
                                    },
                                    {
                                        'secretName':
                                            'secret-name2',
                                        'keyToEnv': [{
                                            'secretKey': 'password2',
                                            'envVar': 'SECRET_VAR2'
                                        }]
                                    },
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_preserves_secret_as_volume(self):
        # checks that use_secret_as_env respects previously set secrets as vol

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_volume(
                task,
                secret_name='secret-name2',
                mount_path='secretpath2',
            )
            kubernetes.use_secret_as_env(
                task,
                secret_name='secret-name1',
                secret_key_to_env={'password': 'SECRET_VAR'},
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'secretAsEnv': [{
                                    'secretName':
                                        'secret-name1',
                                    'keyToEnv': [{
                                        'secretKey': 'password',
                                        'envVar': 'SECRET_VAR'
                                    }]
                                }],
                                'secretAsVolume': [{
                                    'secretName': 'secret-name2',
                                    'mountPath': 'secretpath2',
                                    'optional': False
                                },]
                            }
                        }
                    }
                }
            }
        }

    def test_preserves_pvc_mount(self):
        # checks that use_secret_as_env respects previously set pvc
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.mount_pvc(
                task,
                pvc_name='pvc-name',
                mount_path='path',
            )
            kubernetes.use_secret_as_env(
                task,
                secret_name='secret-name',
                secret_key_to_env={'password': 'SECRET_VAR'},
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


@dsl.component
def comp():
    pass
