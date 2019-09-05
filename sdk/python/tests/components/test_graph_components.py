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
from pathlib import Path


import kfp.components as comp
from kfp.components._structures import ComponentReference, ComponentSpec, ContainerSpec, GraphInputArgument, GraphSpec, InputSpec, InputValuePlaceholder, GraphImplementation, OutputPathPlaceholder, OutputSpec, TaskOutputArgument, TaskSpec

from kfp.components._yaml_utils import load_yaml

class GraphComponentTestCase(unittest.TestCase):
    def test_handle_constructing_graph_component(self):
        task1 = TaskSpec(component_ref=ComponentReference(name='comp 1'), arguments={'in1 1': 11})
        task2 = TaskSpec(component_ref=ComponentReference(name='comp 2'), arguments={'in2 1': 21, 'in2 2': TaskOutputArgument.construct(task_id='task 1', output_name='out1 1')})
        task3 = TaskSpec(component_ref=ComponentReference(name='comp 3'), arguments={'in3 1': TaskOutputArgument.construct(task_id='task 2', output_name='out2 1'), 'in3 2': GraphInputArgument(input_name='graph in 1')})

        graph_component1 = ComponentSpec(
            inputs=[
                InputSpec(name='graph in 1'),
                InputSpec(name='graph in 2'),
            ],
            outputs=[
                OutputSpec(name='graph out 1'),
                OutputSpec(name='graph out 2'),
            ],
            implementation=GraphImplementation(graph=GraphSpec(
                tasks={
                    'task 1': task1,
                    'task 2': task2,
                    'task 3': task3,
                },
                output_values={
                    'graph out 1': TaskOutputArgument.construct(task_id='task 3', output_name='out3 1'),
                    'graph out 2': TaskOutputArgument.construct(task_id='task 1', output_name='out1 2'),
                }
            ))
        )

    def test_handle_parsing_graph_component(self):
        component_text = '''\
inputs:
- {name: graph in 1}
- {name: graph in 2}
outputs:
- {name: graph out 1}
- {name: graph out 2}
implementation:
  graph:
    tasks:
      task 1:
        componentRef: {name: Comp 1}
        arguments:
            in1 1: 11
      task 2:
        componentRef: {name: Comp 2}
        arguments:
            in2 1: 21
            in2 2: {taskOutput: {taskId: task 1, outputName: out1 1}}
      task 3:
        componentRef: {name: Comp 3}
        arguments:
            in3 1: {taskOutput: {taskId: task 2, outputName: out2 1}}
            in3 2: {graphInput: graph in 1}
    outputValues:
      graph out 1: {taskOutput: {taskId: task 3, outputName: out3 1}}
      graph out 2: {taskOutput: {taskId: task 1, outputName: out1 2}}
'''
        struct = load_yaml(component_text)
        ComponentSpec.from_dict(struct)

    @unittest.expectedFailure
    def test_fail_on_cyclic_references(self):
        component_text = '''\
implementation:
  graph:
    tasks:
      task 1:
        componentRef: {name: Comp 1}
        arguments:
            in1 1: {taskOutput: {taskId: task 2, outputName: out2 1}}
      task 2:
        componentRef: {name: Comp 2}
        arguments:
            in2 1: {taskOutput: {taskId: task 1, outputName: out1 1}}
'''
        struct = load_yaml(component_text)
        ComponentSpec.from_dict(struct)

    def test_handle_parsing_predicates(self):
        component_text = '''\
implementation:
  graph:
    tasks:
      task 1:
        componentRef: {name: Comp 1}
      task 2:
        componentRef: {name: Comp 2}
        arguments:
            in2 1: 21
            in2 2: {taskOutput: {taskId: task 1, outputName: out1 1}}
        isEnabled:
            not:
                and:
                    op1:
                        '>':
                            op1: {taskOutput: {taskId: task 1, outputName: out1 1}}
                            op2: 0
                    op2:
                        '==':
                            op1: {taskOutput: {taskId: task 1, outputName: out1 2}}
                            op2: 'head'
'''
        struct = load_yaml(component_text)
        ComponentSpec.from_dict(struct)

    def test_handle_parsing_task_container_spec_options(self):
        component_text = '''\
implementation:
  graph:
    tasks:
      task 1:
        componentRef: {name: Comp 1}
        k8sContainerOptions:
          resources:
            requests:
              memory: 1024Mi
              cpu: 200m

'''
        struct = load_yaml(component_text)
        component_spec = ComponentSpec.from_dict(struct)
        self.assertEqual(component_spec.implementation.graph.tasks['task 1'].k8s_container_options.resources.requests['memory'], '1024Mi')


    def test_handle_parsing_task_volumes_and_mounts(self):
        component_text = '''\
implementation:
  graph:
    tasks:
      task 1:
        componentRef: {name: Comp 1}
        k8sContainerOptions:
          volumeMounts:
          - name: workdir
            mountPath: /mnt/vol
        k8sPodOptions:
          spec:
            volumes:
            - name: workdir
              emptyDir: {}
'''
        struct = load_yaml(component_text)
        component_spec = ComponentSpec.from_dict(struct)
        self.assertEqual(component_spec.implementation.graph.tasks['task 1'].k8s_pod_options.spec.volumes[0].name, 'workdir')
        self.assertTrue(component_spec.implementation.graph.tasks['task 1'].k8s_pod_options.spec.volumes[0].empty_dir is not None)
    
    def test_load_graph_component(self):
        component_text = '''\
inputs:
- {name: graph in 1}
- {name: graph in 2}
outputs:
- {name: graph out 1}
- {name: graph out 2}
- {name: graph out 3}
- {name: graph out 4}
implementation:
  graph:
    tasks:
      task 1:
        componentRef:
          spec:
            name: Component 1
            inputs:
            - {name: in1_1}
            outputs:
            - {name: out1_1}
            - {name: out1_2}
            implementation:
              container:
                image: busybox
                command: [sh, -c, 'echo "$0" > $1; echo "$0" > $2', {inputValue: in1_1}, {outputPath: out1_1}, {outputPath: out1_2}]
        arguments:
            in1_1: 11
      task 2:
        componentRef:
          spec:
            name: Component 2
            inputs:
            - {name: in2_1}
            - {name: in2_2}
            outputs:
            - {name: out2_1}
            implementation:
              container:
                image: busybox
                command: [sh, -c, 'cat "$0" "$1" > $2', {inputValue: in2_1}, {inputValue: in2_2}, {outputPath: out2_1}]
        arguments:
            in2_1: 21
            in2_2: {taskOutput: {taskId: task 1, outputName: out1_1}}
      task 3:
        componentRef:
          spec:
            name: Component 3
            inputs:
            - {name: in3_1}
            - {name: in3_2}
            outputs:
            - {name: out3_1}
            implementation:
              container:
                image: busybox
                command: [sh, -c, 'cat "$0" "$1" > $2', {inputValue: in3_1}, {inputValue: in3_2}, {outputPath: out3_1}]
        arguments:
            in3_1: {taskOutput: {taskId: task 2, outputName: out2_1}}
            in3_2: {graphInput: graph in 1}
    outputValues:
      graph out 1: {taskOutput: {taskId: task 3, outputName: out3_1}}
      graph out 2: {taskOutput: {taskId: task 1, outputName: out1_2}}
      graph out 3: {graphInput: graph in 2}
      graph out 4: 42
'''
        op = comp.load_component_from_text(component_text)
        task = op('graph 1', 'graph 2')
        self.assertEqual(len(task.outputs), 4)
    
    def test_load_nested_graph_components(self):
        component_text = '''\
inputs:
- {name: graph in 1}
- {name: graph in 2}
outputs:
- {name: graph out 1}
- {name: graph out 2}
- {name: graph out 3}
- {name: graph out 4}
implementation:
  graph:
    tasks:
      task 1:
        componentRef:
          spec:
            name: Component 1
            inputs:
            - {name: in1_1}
            outputs:
            - {name: out1_1}
            - {name: out1_2}
            implementation:
              container:
                image: busybox
                command: [sh, -c, 'echo "$0" > $1; echo "$0" > $2', {inputValue: in1_1}, {outputPath: out1_1}, {outputPath: out1_2}]
        arguments:
            in1_1: 11
      task 2:
        componentRef:
          spec:
            name: Component 2
            inputs:
            - {name: in2_1}
            - {name: in2_2}
            outputs:
            - {name: out2_1}
            implementation:
              container:
                image: busybox
                command: [sh, -c, 'cat "$0" "$1" > $2', {inputValue: in2_1}, {inputValue: in2_2}, {outputPath: out2_1}]
        arguments:
            in2_1: 21
            in2_2: {taskOutput: {taskId: task 1, outputName: out1_1}}
      task 3:
        componentRef:
          spec:
            inputs:
            - {name: in3_1}
            - {name: in3_2}
            outputs:
            - {name: out3_1}
            implementation:
              graph:
                tasks:
                  graph subtask:
                    componentRef:
                      spec:
                        name: Component 3
                        inputs:
                        - {name: in3_1}
                        - {name: in3_2}
                        outputs:
                        - {name: out3_1}
                        implementation:
                          container:
                            image: busybox
                            command: [sh, -c, 'cat "$0" "$1" > $2', {inputValue: in3_1}, {inputValue: in3_2}, {outputPath: out3_1}]
                    arguments:
                      in3_1: {graphInput: in3_1}
                      in3_2: {graphInput: in3_1}
                outputValues:
                  out3_1: {taskOutput: {taskId: graph subtask, outputName: out3_1}}
        arguments:
          in3_1: {taskOutput: {taskId: task 2, outputName: out2_1}}
          in3_2: {graphInput: graph in 1}
    outputValues:
      graph out 1: {taskOutput: {taskId: task 3, outputName: out3_1}}
      graph out 2: {taskOutput: {taskId: task 1, outputName: out1_2}}
      graph out 3: {graphInput: graph in 2}
      graph out 4: 42
'''
        op = comp.load_component_from_text(component_text)
        task = op('graph 1', 'graph 2')
        self.assertIn('out3_1', str(task.outputs['graph out 1'])) # Checks that the outputs coming from tasks in nested subgraphs are properly resolved.
        self.assertIn('out1_2', str(task.outputs['graph out 2']))
        self.assertEqual(task.outputs['graph out 3'], 'graph 2')
        self.assertEqual(task.outputs['graph out 4'], 42)

#TODO: Test task name conversion to Argo-compatible names

if __name__ == '__main__':
    unittest.main()
