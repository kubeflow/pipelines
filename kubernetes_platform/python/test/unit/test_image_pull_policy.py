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


class TestImagePullPolicy:

    def test_always(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_image_pull_policy(task, 'Always')

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullPolicy': 'Always'
                            }
                        }
                    }
                }
            }
        }

    def test_if_not_present(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_image_pull_policy(task, 'IfNotPresent')

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullPolicy': 'IfNotPresent'
                            }
                        }
                    }
                }
            }
        }

    def test_never(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.set_image_pull_policy(task, 'Never')

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'imagePullPolicy': 'Never'
                            }
                        }
                    }
                }
            }
        }


@dsl.component
def comp():
    pass
