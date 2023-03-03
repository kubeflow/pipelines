# Copyright 2023 The Kubeflow Authors
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
"""Sample pipeline that uses accelerator config in KFP v2."""
import kfp
import kfp.dsl as dsl
from kfp.v2.dsl import component

@component
def add(a: float, b: float) -> float:
  '''Calculates sum of two arguments'''
  return a + b

@dsl.pipeline(
  name='addition-pipeline',
  description='Only CPU v2',
)
def pipeline_cpu(a: float = 1, b: float = 7):
  add_task = add(a=a, b=b).set_cpu_limit('1').set_memory_limit('650M')

@dsl.pipeline(
  name='addition-pipeline',
  description='Nvidia Tesla K80 v2',
)
def pipeline_k80(a: float = 1, b: float = 7):
  add_task = add(a=a, b=b).set_cpu_limit('1').set_memory_limit('650M').add_node_selector_constraint('nvidia-tesla-k80').set_gpu_limit(1)

@dsl.pipeline(
  name='addition-pipeline',
  description='Nvidia Tesla K80 x2 v2',
)
def pipeline_k80_x2(a: float = 1, b: float = 7):
  add_task = add(a=a, b=b).set_cpu_limit('1').set_memory_limit('650M').add_node_selector_constraint('nvidia-tesla-k80').set_gpu_limit(2)

@dsl.pipeline(
  name='addition-pipeline',
  description='Nvidia Tesla v100 v2',
)
def pipeline_v100(a: float = 1, b: float = 7):
  add_task = add(a=a, b=b).set_cpu_limit('1').set_memory_limit('650M').add_node_selector_constraint('nvidia-tesla-v100').set_gpu_limit(1)

@dsl.pipeline(
  name='addition-pipeline',
  description='Nvidia Tesla p100 v2',
)
def pipeline_p100(a: float = 1, b: float = 7):
  add_task = add(a=a, b=b).set_cpu_limit('1').set_memory_limit('650M').add_node_selector_constraint('nvidia-tesla-p100').set_gpu_limit(1)
