#!/usr/bin/env python3
# Copyright 2018 Google LLC
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


# def arena_submit_standalone_job_op(name, image, gpus: int, ):

class MPIOp(dsl.ContainerOp):
  """Submit MPI Job."""

  # arena Image is "cheyang/arena_launcher"
  def __init__(self, name, image, workers, gpus, cpu, memory, rdma,
          tensorboard, tensorboard_image, command,
          data='None', output_data='None',
          arenaImage='cheyang/arena_launcher',
          metric_name='Train-accuracy',
          metric_unit='PERCENTAGE'):

    super(MPIOp, self).__init__(
          name=name,
          image=arenaImage,
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
                      "--metric-name", metric_name,
                      "--metric-unit", metric_unit,
                      "mpijob",
                      "--workers", workers,
                      "--", command],
          file_outputs={'train': '/output.txt'})


