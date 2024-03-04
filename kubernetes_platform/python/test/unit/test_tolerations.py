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
from kfp import compiler
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

        compiler.Compiler().compile(
            pipeline_func=my_pipeline, package_path='my_pipeline.yaml')

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

        compiler.Compiler().compile(
            pipeline_func=my_pipeline, package_path='my_pipeline.yaml')

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
                task, secret_name='my-secret', mount_path='/mnt/my_vol')
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
                                    'mountPath': '/mnt/my_vol',
                                },],
                            },
                        }
                    }
                }
            }
        }


@dsl.component
def comp():
    pass
