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
