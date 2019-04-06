#!/usr/bin/env python3
#
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


import kfp.dsl as dsl
import datetime
import logging

# def DistributeTFOp(name, image, gpus: int, ):

def distributed_tf_op(name, image, workers, ps, gpus, worker_cpu, worker_memory, ps_cpu, ps_memory,
          rdma,
          tensorboard, tensorboard_image, command, chief=0, evaluator=0,
          data='None', output_data='None',
          arena_image='cheyang/arena_launcher',
          timeout_hours='240',
          metric_name='Train-accuracy',
          metric_unit='PERCENTAGE'):
          """Submit Distributed TFJob with Parameter Server mode."""
          return dsl.ContainerOp(
            name=name,
          image=arena_image,
          command=['python','arena_launcher.py'],
          arguments=[ "--name", '%s-{{workflow.name}}' % name,
                      "--tensorboard", tensorboard,
                      "--rdma", rdma,
                      "--data", data,
                      "--output-data", output_data,
                      "--image", image,
                      "--gpus", gpus,
                      "--cpu", cpu,
                      "--memory", memory,
                      "--timeout-hours", timeout_hours,
                      "--metric-name", metric_name,
                      "--metric-unit", metric_unit,
                      "tfjob",
                      "--workers", workers,
                      "--", command],
          file_outputs={'train': '/output.txt'}
          )
