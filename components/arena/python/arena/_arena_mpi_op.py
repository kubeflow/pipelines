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

def mpi_job_op(name, image, command, workers=0, gpus=0, cpu=0, memory=0, envs=[],
          mounts=[],
          rdma=False,
          tensorboard=False, 
          metrics=['Train-accuracy:PERCENTAGE'],
          arenaImage='cheyang/arena_launcher',
          timeout_hours=240):
    """This function submits MPI Job, it can run Allreduce-style Distributed Training.

    Args:
      name: the name of mpi_job_op
      image: the docker image name of training job
      data: specify the datasource to mount to the job, like <name_of_datasource>:<mount_point_on_job>
      command: the command to run
    """
    return dsl.ContainerOp(
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
                      "--timeout-hours", timeout_hours,
                      "--metric-name", metric_name,
                      "--metric-unit", metric_unit,
                      "mpijob",
                      "--workers", workers,
                      "--", command],
          file_outputs={'train': '/output.txt'}
    )