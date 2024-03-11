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


class TestPodMetadata:

    def test_add_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_pod_label(
                task,
                label_key='kubeflow.com/kfp',
                label_value='pipeline-node',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podMetadata': {
                                    'labels': {
                                        'kubeflow.com/kfp': 'pipeline-node'
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_add_same_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_pod_label(
                task,
                label_key='kubeflow.com/kfp',
                label_value='pipeline-node',
            )
            kubernetes.add_pod_label(
                task,
                label_key='kubeflow.com/kfp',
                label_value='pipeline-node2',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podMetadata': {
                                    'labels': {
                                        'kubeflow.com/kfp': 'pipeline-node2'
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_add_two_and_mix(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_pod_label(
                task,
                label_key='kubeflow.com/kfp',
                label_value='pipeline-node',
            )
            kubernetes.add_pod_label(
                task,
                label_key='kubeflow.com/common',
                label_value='test',
            )
            kubernetes.add_pod_annotation(
                task,
                annotation_key='run_id',
                annotation_value='123456',
            )
            kubernetes.add_pod_annotation(
                task,
                annotation_key='experiment_id',
                annotation_value='234567',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podMetadata': {
                                    'annotations': {
                                        'run_id': '123456',
                                        'experiment_id': '234567'
                                    },
                                    'labels': {
                                        'kubeflow.com/kfp': 'pipeline-node',
                                        'kubeflow.com/common': 'test'
                                    }
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
            kubernetes.add_pod_annotation(
                task,
                annotation_key='run_id',
                annotation_value='123456',
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podMetadata': {
                                    'annotations': {
                                        'run_id': '123456'
                                    }
                                },
                                'secretAsVolume': [{
                                    'secretName': 'my-secret',
                                    'mountPath': '/mnt/my_vol',
                                    'optional': False
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
