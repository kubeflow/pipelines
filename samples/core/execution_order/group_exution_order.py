#!/usr/bin/env python3
# Copyright 2019 Google LLC
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


import kfp
from kfp import dsl


@dsl.graph_component
def echo1_graph_component(text1):
  """A two step graph component with an explicitly defined execution order."""
  step1_task = dsl.ContainerOp(
      name='echo1-task1',
      image='library/bash:4.4.23',
      command=['sh', '-c'],
      arguments=['echo "$0"', text1])
  step2_task = dsl.ContainerOp(
      name='echo1-task2',
      image='library/bash:4.4.23',
      command=['sh', '-c'],
      arguments=['echo "$0"', text1])
  step2_task.after(step1_task)


@dsl.graph_component
def echo2_graph_component(text2):
  """A two step graph component with an explicitly defined execution order."""
  step1_task = dsl.ContainerOp(
      name='echo2-task1',
      image='library/bash:4.4.23',
      command=['sh', '-c'],
      arguments=['echo "$0"', text2])
  step2_task = dsl.ContainerOp(
      name='echo2-task2',
      image='library/bash:4.4.23',
      command=['sh', '-c'],
      arguments=['echo "$0"', text2])
  step2_task.after(step1_task)


@dsl.pipeline(
    name='Execution order pipeline',
    description='A pipeline to demonstrate execution order management.'
)
def execution_order_pipeline(text1='message 1', text2='message 2'):
  """A two step pipeline with an explicitly defined execution order."""
  step1_graph_component = echo1_graph_component(text1)
  step2_graph_component = echo2_graph_component(text2)
  step2_graph_component.after(step1_graph_component)

if __name__ == '__main__':
  kfp.compiler.Compiler().compile(execution_order_pipeline, __file__ + '.yaml')
