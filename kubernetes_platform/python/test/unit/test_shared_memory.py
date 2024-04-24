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


class TestEnableSharedMemory:

    def test_default_options(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_shared_memory(task)

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            "platforms": {
                "kubernetes": {
                    "deploymentSpec": {
                        "executors": {
                            "exec-comp": {"enabledSharedMemory": {"volumeName": "shm"}}
                        }
                    }
                }
            }
        }

    def test_custom_volume_name(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_shared_memory(task, volume_name="Random-Name")

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            "platforms": {
                "kubernetes": {
                    "deploymentSpec": {
                        "executors": {
                            "exec-comp": {
                                "enabledSharedMemory": {"volumeName": "Random-Name"}
                            }
                        }
                    }
                }
            }
        }

    def test_custom_volume_size(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_shared_memory(task, size="100Mi")

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            "platforms": {
                "kubernetes": {
                    "deploymentSpec": {
                        "executors": {
                            "exec-comp": {
                                "enabledSharedMemory": {
                                    "volumeName": "shm",
                                    "size": "100Mi",
                                }
                            }
                        }
                    }
                }
            }
        }

    def test_custom_volume_name_and_size(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.enable_shared_memory(
                task, volume_name="Random-Name", size="100Mi"
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            "platforms": {
                "kubernetes": {
                    "deploymentSpec": {
                        "executors": {
                            "exec-comp": {
                                "enabledSharedMemory": {
                                    "volumeName": "Random-Name",
                                    "size": "100Mi",
                                }
                            }
                        }
                    }
                }
            }
        }


@dsl.component
def comp():
    pass
