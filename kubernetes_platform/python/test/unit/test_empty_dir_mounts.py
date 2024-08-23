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


class TestEmptyDirMounts:

    def test_add_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.empty_dir_mount(
                task, 
                volume_name='emptydir-vol-1', 
                mount_path='/mnt/my_vol_1',
                medium='Memory',
                size_limit='1Gi'
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'emptyDirMounts': [{
                                    'medium': 'Memory',
                                    'mountPath': '/mnt/my_vol_1',
                                    'sizeLimit': '1Gi',
                                    'volumeName': 'emptydir-vol-1'
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
            kubernetes.empty_dir_mount(
                task, 
                volume_name='emptydir-vol-1',
                mount_path='/mnt/my_vol_1',
                medium='Memory',
                size_limit='1Gi'
            )
            kubernetes.empty_dir_mount(
                task, 
                volume_name='emptydir-vol-2',
                mount_path='/mnt/my_vol_2'
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'emptyDirMounts': [{
                                    'medium': 'Memory',
                                    'mountPath': '/mnt/my_vol_1',
                                    'sizeLimit': '1Gi',
                                    'volumeName': 'emptydir-vol-1'
                                },
                                {
                                    'mountPath': '/mnt/my_vol_2',
                                    'volumeName': 'emptydir-vol-2'
                                }]
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

            kubernetes.empty_dir_mount(
                task, 
                volume_name='emptydir-vol-1',
                mount_path='/mnt/my_vol_1',
                medium='Memory',
                size_limit='1Gi'
            )

            # this should exist too
            kubernetes.set_image_pull_secrets(task, ['secret-name'])

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'emptyDirMounts': [{
                                    'medium': 'Memory',
                                    'mountPath': '/mnt/my_vol_1',
                                    'sizeLimit': '1Gi',
                                    'volumeName': 'emptydir-vol-1'
                                }],
                                'imagePullSecret': [{
                                    'secretName': 'secret-name'
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
