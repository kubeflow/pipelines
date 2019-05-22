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


def standalone_job_op(name, image, command, gpus=0, cpu_limit='0', memory_limit='0', env=[],
          tensorboard=False, tensorboard_image=None,
          data=[], sync_source=None, annotations=[],
          metrics=['Train-accuracy:PERCENTAGE'],
          arena_image='cheyang/arena_launcher:v0.7',
          timeout_hours=240):

    """This function submits a standalone training Job 

        Args:
          name: the name of standalone_job_op
          image: the docker image name of training job
          mount: specify the datasource to mount to the job, like <name_of_datasource>:<mount_point_on_job>
          command: the command to run
    """
    if not name:
      raise ValueError("name must be specified")
    if not image:
      raise ValueError("image must be specified")
    if not command:
      raise ValueError("command must be specified")

    options = []
    if sync_source:
       options.append('--sync-source')
       options.append(str(sync_source))

    for e in env:
      options.append('--env')
      options.append(str(e))

    for d in data:
      options.append('--data')
      options.append(str(d))

    for m in metrics:
      options.append('--metric')
      options.append(str(m))

    if tensorboard_image:
      options.append('--tensorboard-image')
      options.append(str(tensorboard_image))

    op = dsl.ContainerOp(
          name=name,
          image=arena_image,
          command=['python','arena_launcher.py'],
          arguments=[ "--name", name,
                      "--tensorboard", str(tensorboard),
                      "--image", str(image),
                      "--gpus", str(gpus),
                      "--cpu", str(cpu_limit),
                      "--step-name", '{{pod.name}}',
                      "--workflow-name", '{{workflow.name}}',
                      "--memory", str(memory_limit),
                      "--timeout-hours", str(timeout_hours),
                      ] + options +
                      [
                      "job",
                      "--", str(command)],
          file_outputs={'train': '/output.txt',
                        'workflow':'/workflow-name.txt',
                        'step':'/step-name.txt',
                        'name':'/name.txt'}
      )
    op.set_image_pull_policy('Always')
    return op
