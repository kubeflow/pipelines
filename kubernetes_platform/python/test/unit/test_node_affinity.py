# Copyright 2025 The Kubeflow Authors
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


class TestNodeAffinity:

    def test_add_match_expressions(self):
        """Test adding node affinity with matchExpressions (required scheduling)."""
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_node_affinity(
                task,
                match_expressions=[
                    {
                        'key': 'disktype',
                        'operator': 'In',
                        'values': ['ssd', 'nvme']
                    }
                ]
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'matchExpressions': [{
                                        'key': 'disktype',
                                        'operator': 'In',
                                        'values': ['ssd', 'nvme']
                                    }]
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_match_fields(self):
        """Test adding node affinity with matchFields (required scheduling)."""
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_node_affinity(
                task,
                match_fields=[
                    {
                        'key': 'metadata.name',
                        'operator': 'In',
                        'values': ['node-1', 'node-2']
                    }
                ]
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'matchFields': [{
                                        'key': 'metadata.name',
                                        'operator': 'In',
                                        'values': ['node-1', 'node-2']
                                    }]
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_preferred_scheduling_with_weight(self):
        """Test adding node affinity with weight (preferred scheduling)."""
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_node_affinity(
                task,
                match_expressions=[
                    {
                        'key': 'zone',
                        'operator': 'In',
                        'values': ['us-west-1']
                    }
                ],
                weight=100
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'matchExpressions': [{
                                        'key': 'zone',
                                        'operator': 'In',
                                        'values': ['us-west-1']
                                    }],
                                    'weight': 100
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_combined_match_expressions_and_fields(self):
        """Test adding node affinity with both matchExpressions and matchFields."""
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_node_affinity(
                task,
                match_expressions=[
                    {
                        'key': 'disktype',
                        'operator': 'In',
                        'values': ['ssd']
                    },
                    {
                        'key': 'gpu',
                        'operator': 'Exists'
                    }
                ],
                match_fields=[
                    {
                        'key': 'metadata.name',
                        'operator': 'NotIn',
                        'values': ['node-5', 'node-6']
                    }
                ]
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'matchExpressions': [
                                        {
                                            'key': 'disktype',
                                            'operator': 'In',
                                            'values': ['ssd']
                                        },
                                        {
                                            'key': 'gpu',
                                            'operator': 'Exists'
                                        }
                                    ],
                                    'matchFields': [{
                                        'key': 'metadata.name',
                                        'operator': 'NotIn',
                                        'values': ['node-5', 'node-6']
                                    }]
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_add_multiple_node_affinity_terms(self):
        """Test adding multiple node affinity terms."""
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.add_node_affinity(
                task,
                match_expressions=[
                    {
                        'key': 'disktype',
                        'operator': 'In',
                        'values': ['ssd']
                    }
                ]
            )
            kubernetes.add_node_affinity(
                task,
                match_fields=[
                    {
                        'key': 'metadata.name',
                        'operator': 'In',
                        'values': ['node-1']
                    }
                ]
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [
                                    {
                                        'matchExpressions': [{
                                            'key': 'disktype',
                                            'operator': 'In',
                                            'values': ['ssd']
                                        }]
                                    },
                                    {
                                        'matchFields': [{
                                            'key': 'metadata.name',
                                            'operator': 'In',
                                            'values': ['node-1']
                                        }]
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_respects_other_configuration(self):
        """Test that node affinity respects other Kubernetes configurations."""
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            kubernetes.use_secret_as_volume(
                task, secret_name='my-secret', mount_path='/mnt/my_vol')
            kubernetes.add_node_affinity(
                task,
                match_expressions=[
                    {
                        'key': 'disktype',
                        'operator': 'In',
                        'values': ['ssd']
                    }
                ]
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'matchExpressions': [{
                                        'key': 'disktype',
                                        'operator': 'In',
                                        'values': ['ssd']
                                    }]
                                }],
                                'secretAsVolume': [{
                                    'secretName': 'my-secret',
                                    'secretNameParameter': {'runtimeValue': {'constant': 'my-secret'}},
                                    'mountPath': '/mnt/my_vol',
                                    'optional': False
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_invalid_match_expression_missing_key(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            try:
                kubernetes.add_node_affinity(
                    task,
                    match_expressions=[{'operator': 'In', 'values': ['ssd']}]
                )
            except ValueError as e:
                assert "non-empty 'key'" in str(e)
            else:
                assert False, 'Expected ValueError for missing key'

    def test_invalid_match_expression_missing_operator(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            try:
                kubernetes.add_node_affinity(
                    task,
                    match_expressions=[{'key': 'disktype', 'values': ['ssd']}]
                )
            except ValueError as e:
                assert "non-empty 'operator'" in str(e)
            else:
                assert False, 'Expected ValueError for missing operator'

    def test_invalid_match_expression_invalid_operator(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            try:
                kubernetes.add_node_affinity(
                    task,
                    match_expressions=[{'key': 'disktype', 'operator': 'INVALID', 'values': ['ssd']}]
                )
            except ValueError as e:
                assert "Invalid operator" in str(e)
            else:
                assert False, 'Expected ValueError for invalid operator'

    def test_invalid_weight_too_low(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            try:
                kubernetes.add_node_affinity(
                    task,
                    match_expressions=[{'key': 'disktype', 'operator': 'In', 'values': ['ssd']}],
                    weight=0
                )
            except ValueError as e:
                assert "weight must be between 1 and 100" in str(e)
            else:
                assert False, 'Expected ValueError for weight < 1'

    def test_invalid_weight_too_high(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            try:
                kubernetes.add_node_affinity(
                    task,
                    match_expressions=[{'key': 'disktype', 'operator': 'In', 'values': ['ssd']}],
                    weight=101
                )
            except ValueError as e:
                assert "weight must be between 1 and 100" in str(e)
            else:
                assert False, 'Expected ValueError for weight > 100'


class TestNodeAffinityJSON:

    def test_component_pipeline_input_required_scheduling(self):
        """Test JSON-based node affinity with pipeline input for required scheduling."""
        @dsl.pipeline
        def my_pipeline(affinity_input: dict):
            task = comp()
            kubernetes.add_node_affinity_json(
                task,
                node_affinity_json=affinity_input,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'nodeAffinityJson': {
                                        'componentInputParameter': 'affinity_input'
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }
    
    def test_component_pipeline_input_multiple_tasks(self):
        """Test JSON-based node affinity with multiple tasks and pipeline inputs."""
        @dsl.pipeline
        def my_pipeline(affinity_input_1: dict, affinity_input_2: dict):
            t1 = comp()
            kubernetes.add_node_affinity_json(
                t1,
                node_affinity_json=affinity_input_1,
            )

            t2 = comp()
            kubernetes.add_node_affinity_json(
                t2,
                node_affinity_json=affinity_input_2,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'nodeAffinityJson': {
                                        'componentInputParameter': 'affinity_input_1'
                                    }
                                }]
                            },
                            'exec-comp-2': {
                                'nodeAffinity': [{
                                    'nodeAffinityJson': {
                                        'componentInputParameter': 'affinity_input_2'
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_component_upstream_input(self):
        """Test JSON-based node affinity with upstream task input parameters."""
        @dsl.pipeline
        def my_pipeline():
            upstream_task = comp_with_output()
            task = comp()
            kubernetes.add_node_affinity_json(
                task,
                node_affinity_json=upstream_task.output,
            )
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'nodeAffinityJson': {
                                        'taskOutputParameter': {
                                            'producerTask': 'comp-with-output',
                                            'outputParameterKey': 'Output'
                                        }
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_overwrite_previous_json(self):
        """Test that applying node affinity JSON multiple times overwrites the previous."""
        @dsl.pipeline
        def my_pipeline(affinity_input_1: dict, affinity_input_2: dict):
            task = comp()
            kubernetes.add_node_affinity_json(
                task,
                node_affinity_json=affinity_input_1,
            )
            # This should overwrite the previous JSON
            kubernetes.add_node_affinity_json(
                task,
                node_affinity_json=affinity_input_2,
            )        
        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [{
                                    'nodeAffinityJson': {
                                        'componentInputParameter': 'affinity_input_2'
                                    }
                                }]
                            }
                        }
                    }
                }
            }
        }

    def test_mixed_explicit_and_json(self):
        """Test mixing explicit node affinity with JSON-based node affinity."""
        @dsl.pipeline
        def my_pipeline(affinity_input: dict):
            task = comp()
            kubernetes.add_node_affinity(
                task,
                match_expressions=[
                    {
                        'key': 'disktype',
                        'operator': 'In',
                        'values': ['ssd']
                    }
                ]
            )
            kubernetes.add_node_affinity_json(
                task,
                node_affinity_json=affinity_input,
            )

        assert json_format.MessageToDict(my_pipeline.platform_spec) == {
            'platforms': {
                'kubernetes': {
                    'deploymentSpec': {
                        'executors': {
                            'exec-comp': {
                                'nodeAffinity': [
                                    {
                                        'matchExpressions': [{
                                            'key': 'disktype',
                                            'operator': 'In',
                                            'values': ['ssd']
                                        }]
                                    },
                                    {
                                        'nodeAffinityJson': {
                                            'componentInputParameter': 'affinity_input'
                                        }
                                    }
                                ]
                            }
                        }
                    }
                }
            }
        }

    def test_invalid_node_affinity_json(self):
        @dsl.pipeline
        def my_pipeline():
            task = comp()
            # Missing required fields for V1NodeAffinity
            invalid_json = {"foo": "bar"}
            try:
                kubernetes.add_node_affinity_json(task, node_affinity_json=invalid_json)
            except ValueError as e:
                assert "Invalid V1NodeAffinity JSON" in str(e)
            else:
                assert False, 'Expected ValueError for invalid node_affinity_json'


@dsl.component
def comp():
    pass


@dsl.component()
def comp_with_output() -> str:
    return "test_output" 
