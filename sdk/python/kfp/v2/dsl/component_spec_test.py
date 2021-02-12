# Copyright 2021 Google LLC
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
"""Tests for kfp.v2.dsl.component_spec."""

import unittest

from kfp.components import _structures as structures
from kfp.v2 import dsl
from kfp.v2.dsl import component_spec as dsl_component_spec
from kfp.pipeline_spec import pipeline_spec_pb2

from google.protobuf import json_format


class ComponentSpecTest(unittest.TestCase):

  def test_build_component_spec_from_structure(self):
    structure_component_spec = structures.ComponentSpec(
        name='component1',
        description='component1 desc',
        inputs=[
            structures.InputSpec(
                name='input1', description='input1 desc', type='Dataset'),
            structures.InputSpec(
                name='input2', description='input2 desc', type='String'),
            structures.InputSpec(
                name='input3', description='input3 desc', type='Integer'),
        ],
        outputs=[
            structures.OutputSpec(
                name='output1', description='output1 desc', type='Model')
        ])
    expected_dict = {
        'inputDefinitions': {
            'artifacts': {
                'input1': {
                    'artifactType': {
                        'instanceSchema':
                            'properties:\ntitle: kfp.Dataset\ntype: object\n'
                    }
                }
            },
            'parameters': {
                'input2': {
                    'type': 'STRING'
                },
                'input3': {
                    'type': 'INT'
                }
            }
        },
        'outputDefinitions': {
            'artifacts': {
                'output1': {
                    'artifactType': {
                        'instanceSchema':
                            'properties:\ntitle: kfp.Model\ntype: object\n'
                    }
                }
            }
        },
        'executorLabel': 'exec-component1'
    }
    expected_spec = pipeline_spec_pb2.ComponentSpec()
    json_format.ParseDict(expected_dict, expected_spec)

    component_spec = (
        dsl_component_spec.build_component_spec_from_structure(
            structure_component_spec))

    self.assertEqual(expected_spec, component_spec)

  def test_build_component_inputs_spec(self):
    pipeline_params = [
        dsl.PipelineParam(name='input1', param_type='Dataset'),
        dsl.PipelineParam(name='input2', param_type='Integer'),
        dsl.PipelineParam(name='input3', param_type='String'),
        dsl.PipelineParam(name='input4', param_type='Float'),
    ]
    expected_dict = {
        'inputDefinitions': {
            'artifacts': {
                'input1': {
                    'artifactType': {
                        'instanceSchema':
                            'properties:\ntitle: kfp.Dataset\ntype: object\n'
                    }
                }
            },
            'parameters': {
                'input2': {
                    'type': 'INT'
                },
                'input3': {
                    'type': 'STRING'
                },
                'input4': {
                    'type': 'DOUBLE'
                }
            }
        }
    }
    expected_spec = pipeline_spec_pb2.ComponentSpec()
    json_format.ParseDict(expected_dict, expected_spec)

    component_spec = pipeline_spec_pb2.ComponentSpec()
    dsl_component_spec.build_component_inputs_spec(component_spec,
                                                   pipeline_params)

    self.assertEqual(expected_spec, component_spec)

  def test_build_component_outputs_spec(self):
    pipeline_params = [
        dsl.PipelineParam(name='output1', param_type='Dataset'),
        dsl.PipelineParam(name='output2', param_type='Integer'),
        dsl.PipelineParam(name='output3', param_type='String'),
        dsl.PipelineParam(name='output4', param_type='Float'),
    ]
    expected_dict = {
        'outputDefinitions': {
            'artifacts': {
                'output1': {
                    'artifactType': {
                        'instanceSchema':
                            'properties:\ntitle: kfp.Dataset\ntype: object\n'
                    }
                }
            },
            'parameters': {
                'output2': {
                    'type': 'INT'
                },
                'output3': {
                    'type': 'STRING'
                },
                'output4': {
                    'type': 'DOUBLE'
                }
            }
        }
    }
    expected_spec = pipeline_spec_pb2.ComponentSpec()
    json_format.ParseDict(expected_dict, expected_spec)

    component_spec = pipeline_spec_pb2.ComponentSpec()
    dsl_component_spec.build_component_outputs_spec(component_spec,
                                                    pipeline_params)

    self.assertEqual(expected_spec, component_spec)

  def test_build_task_inputs_spec(self):
    pipeline_params = [
        dsl.PipelineParam(name='output1', param_type='Dataset', op_name='op-1'),
        dsl.PipelineParam(name='output2', param_type='Integer', op_name='op-2'),
        dsl.PipelineParam(name='output3', param_type='Model', op_name='op-3'),
        dsl.PipelineParam(name='output4', param_type='Double', op_name='op-4'),
    ]
    tasks_in_current_dag = ['op-1', 'op-2']
    expected_dict = {
        'inputs': {
            'artifacts': {
                'op-1-output1': {
                    'taskOutputArtifact': {
                        'producerTask': 'task-op-1',
                        'outputArtifactKey': 'output1'
                    }
                },
                'op-3-output3': {
                    'componentInputArtifact': 'op-3-output3'
                }
            },
            'parameters': {
                'op-2-output2': {
                    'taskOutputParameter': {
                        'producerTask': 'task-op-2',
                        'outputParameterKey': 'output2'
                    }
                },
                'op-4-output4': {
                    'componentInputParameter': 'op-4-output4'
                }
            }
        }
    }
    expected_spec = pipeline_spec_pb2.PipelineTaskSpec()
    json_format.ParseDict(expected_dict, expected_spec)

    task_spec = pipeline_spec_pb2.PipelineTaskSpec()
    dsl_component_spec.build_task_inputs_spec(task_spec, pipeline_params,
                                              tasks_in_current_dag)

    self.assertEqual(expected_spec, task_spec)


if __name__ == '__main__':
  unittest.main()
