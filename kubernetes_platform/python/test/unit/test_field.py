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


class TestUseFieldPathAsEnv:

    def test_use_one(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_field_path_as_env(
                task,
                env_name="KFP_RUN_NAME",
                field_path="metadata.annotations['pipelines.kubeflow.org/run_name']"
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'fieldPathAsEnv': [{
                                    'name':
                                        'KFP_RUN_NAME',
                                    'fieldPath':
                                        'metadata.annotations[\'pipelines.kubeflow.org/run_name\']'
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
            kubernetes.use_field_path_as_env(
                task,
                env_name="KFP_RUN_NAME",
                field_path="metadata.annotations['pipelines.kubeflow.org/run_name']"
            )
            kubernetes.use_field_path_as_env(
                task,
                env_name="POD_NAME",
                field_path="metadata.name"
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'fieldPathAsEnv': [{
                                    'name':
                                        'KFP_RUN_NAME',
                                    'fieldPath':
                                        'metadata.annotations[\'pipelines.kubeflow.org/run_name\']'
                                },
                                {
                                    'name':
                                        'POD_NAME',
                                    'fieldPath':
                                        'metadata.name'
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
