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


class TestAffinities:

    def test_add_one_node_affinity(self):

        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_node_affinity(
                task,
                match_expressions=[kubernetes.SelectorRequirement(key="key1", operator="In", values=["value1"])],
            )

        compiler.Compiler().compile(
            pipeline_func=my_pipeline, package_path='my_pipeline.yaml')
        print(json_format.MessageToDict(my_pipeline.platform_spec))
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [
                                    {
                                        'matchExpressions': [
                                            {
                                            'key': 'key1',
                                            'operator': 'In',
                                            'values': ['value1']
                                            }
                                        ]
                                    } 
                                ]
                            }
                        }
                    }
                }
            }
        }


@dsl.component
def comp():
    pass
