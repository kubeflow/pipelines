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
                                    'secretName': 'secret-name'
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
                                        'secretName': 'secret-name1'
                                    },
                                    {
                                        'secretName': 'secret-name2'
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
                                    'optional': False
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
