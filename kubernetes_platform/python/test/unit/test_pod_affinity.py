# Copyright 2026 The Kubeflow Authors
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

@dsl.component
def comp():
    pass

class TestPodAffinity:

    def test_add_pod_affinity(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_pod_affinity(
                task=task,
                match_pod_expressions=[{
                    'key': 'security',
                    'operator': 'In',
                    'values': ['S1']
                }],
                topology_key='topology.kubernetes.io/zone',
                weight=10,
                anti=True,
            )
        
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podAffinity': [{
                                    'matchPodExpressions': [{
                                        'key': 'security',
                                        'operator': 'In',
                                        'values': ['S1']
                                    }],
                                    'topologyKey': 'topology.kubernetes.io/zone',
                                    'weight': 10,
                                    'anti': True
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_pod_affinity_json(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            pod_affinity_json = {
                'labelSelector': {
                    'matchExpressions': [{
                        'key': 'security',
                        'operator': 'In',
                        'values': ['S1']
                    }]
                },
                'topologyKey': 'topology.kubernetes.io/zone'
            }
            
            kubernetes.add_pod_affinity_json(
                task=task,
                pod_affinity_json=pod_affinity_json,
                anti=False
            )
        
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'podAffinity': [{
                                    'podAffinityJson': {
                                        'runtimeValue': {
                                            'constant': {
                                                'topologyKey': 'topology.kubernetes.io/zone',
                                                'labelSelector': {
                                                    'matchExpressions': [{
                                                        'operator': 'In',
                                                        'values': ['S1'],
                                                        'key': 'security'
                                                    }]
                                                }
                                            }
                                        }
                                    },
                                    'anti': False
                                }]
                            }
                        }
                    }
                }
            }
        }
