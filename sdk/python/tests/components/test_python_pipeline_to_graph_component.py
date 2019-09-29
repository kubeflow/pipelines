# Copyright 2018 Google LLC
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

import os
import sys
import unittest
from collections import OrderedDict
from pathlib import Path

sys.path.insert(0, __file__ + '/../../../')

import kfp.components as comp
from kfp.components._python_to_graph_component import create_graph_component_spec_from_pipeline_func


component_with_2_inputs_and_0_outputs = '''\
name: Component with 2 inputs and 0 outputs
inputs:
- {name: Input parameter}
- {name: Input artifact}
implementation:
  container:
    image: busybox
    command: [sh, -c, '
        echo "Input parameter = $0"
        echo "Input artifact = $(< $1)"
        '
    ]
    args: 
    - {inputValue: Input parameter}
    - {inputPath: Input artifact}
'''


component_with_0_inputs_and_2_outputs = '''\
name: Component with 0 inputs and 2 outputs
outputs:
- {name: Output 1}
- {name: Output 2}
implementation:
  container:
    image: busybox
    command: [sh, -c, '
        echo "Data 1" > $0
        echo "Data 2" > $1
        '
    ]
    args:
    - {outputPath: Output 1}
    - {outputPath: Output 2}
'''


component_with_2_inputs_and_2_outputs = '''\
name: Component with 2 inputs and 2 outputs
inputs:
- {name: Input parameter}
- {name: Input artifact}
outputs:
- {name: Output 1}
- {name: Output 2}
implementation:
  container:
    image: busybox
    command: [sh, -c, '
        mkdir -p $(dirname "$2")
        mkdir -p $(dirname "$3")
        echo "$0" > "$2"
        cp "$1" "$3"
        '
    ]
    args: 
    - {inputValue: Input parameter}
    - {inputPath: Input artifact}
    - {outputPath: Output 1}
    - {outputPath: Output 2}
'''


class PythonPipelineToGraphComponentTestCase(unittest.TestCase):
    def test_handle_creating_graph_component_from_pipeline_that_uses_container_components(self):
        producer_op = comp.load_component_from_text(component_with_0_inputs_and_2_outputs)
        processor_op = comp.load_component_from_text(component_with_2_inputs_and_2_outputs)
        #consumer_op = comp.load_component_from_text(component_with_2_inputs_and_0_outputs)

        #def pipeline(pipeline_param_1: int):
        def pipeline1(pipeline_param_1: int):
            producer_task = producer_op()
            processor_task = processor_op(pipeline_param_1, producer_task.outputs['Output 2'])
            #processor_task = processor_op(pipeline_param_1, producer_task.outputs.output_2)
            #consumer_task = consumer_op(processor_task.outputs['Output 1'], processor_task.outputs['Output 2'])

            #return (producer_task.outputs['Output 1'], processor_task.outputs['Output 2'])
            #return [producer_task.outputs['Output 1'], processor_task.outputs['Output 2']]
            #return {'Pipeline output 1': producer_task.outputs['Output 1'], 'Pipeline output 2': processor_task.outputs['Output 2']}
            return OrderedDict([
                ('Pipeline output 1', producer_task.outputs['Output 1']),
                ('Pipeline output 2', processor_task.outputs['Output 2']),
            ])
            #return namedtuple('Pipeline1Outputs', ['Pipeline output 1', 'Pipeline output 2'])(producer_task.outputs['Output 1'], processor_task.outputs['Output 2'])

        
        graph_component = create_graph_component_spec_from_pipeline_func(pipeline1)
        self.assertEqual(len(graph_component.inputs), 1)
        self.assertListEqual([input.name for input in graph_component.inputs], ['pipeline_param_1']) #Relies on human name conversion function stability
        self.assertListEqual([output.name for output in graph_component.outputs], ['Pipeline output 1', 'Pipeline output 2'])
        self.assertEqual(len(graph_component.implementation.graph.tasks), 2)


if __name__ == '__main__':
    unittest.main()
