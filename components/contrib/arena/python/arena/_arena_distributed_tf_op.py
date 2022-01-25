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

# flake8: noqa TODO

import kfp.dsl as dsl
import datetime
import logging


def estimator_op(name, image, command,
                      chief_cpu_limit, chief_memory_limit, chief_port,
                      workers, worker_image, worker_cpu_limit, worker_memory_limit,
                      parameter_servers, ps_image, ps_cpu_limit, ps_memory_limit, ps_port,
                      gpus, rdma,
                      tensorboard,
                      worker_port, annotations=[],
                      evaluator=False, evaluator_cpu_limit='0', evaluator_memory_limit='0',
                      env=[], data=[], sync_source=None,
                      metrics=['Train-accuracy:PERCENTAGE'],
                      arena_image='cheyang/arena_launcher:v0.7',
                      timeout_hours=240):

    """This function submits Distributed TFJob in Estimator mode.

          Args:
            name: the name of parameter_servers_op
            image: the docker image name of training job
            data: specify the datasource to mount to the job, like <name_of_datasource>:<mount_point_on_job>
            command: the command to run
          """
    return distributed_tf_op(name=name, image=image, command=command, envs=envs, data=data, sync_source=sync_source,
                      workers=workers, worker_image=worker_image, worker_cpu_limit=worker_cpu_limit, worker_memory_limit=worker_memory,
                      parameter_servers=parameter_servers, ps_image=ps_image, ps_cpu_limit=ps_cpu_limit, ps_memory_limit=ps_memory_limit,
                      gpus=gpus, rdma=rdma,
                      chief=True,
                      chief_cpu_limit=chief_cpu_limit,
                      worker_port=worker_port,
                      ps_port=ps_port,
                      tensorboard=tensorboard,
                      metrics=metrics,
                      arena_image=arena_image,
                      timeout_hours=timeout_hours)

# def DistributeTFOp(name, image, gpus: int, ):

def parameter_servers_op(name, image, command, env, data, sync_source, annotations,
                      workers, worker_image, worker_cpu_limit, worker_memory,
                      parameter_servers, ps_image, ps_cpu_limit, ps_memory_limit,
                      gpus, rdma,
                      tensorboard,
                      worker_port, ps_port,
                      metrics=['Train-accuracy:PERCENTAGE'],
                      arena_image='cheyang/arena_launcher:v0.7',
                      timeout_hours=240):

    """This function submits Distributed TFJob in Parameter Servers mode.

          Args:
            name: the name of parameter_servers_op
            image: the docker image name of training job
            data: specify the datasource to mount to the job, like <name_of_datasource>:<mount_point_on_job>
            command: the command to run
          """
    return distributed_tf_op(name=name, image=image, command=command, envs=envs, data=data, sync_source=sync_source,
                      workers=workers, worker_image=worker_image, worker_cpu_limit=worker_cpu_limit, worker_memory_limit=worker_memory,
                      parameter_servers=parameter_servers, ps_image=ps_image, ps_cpu_limit=ps_cpu_limit, ps_memory_limit=ps_memory_limit,
                      gpus=gpus, rdma=rdma,
                      worker_port=worker_port,
                      ps_port=ps_port,
                      tensorboard=tensorboard,
                      metrics=metrics,
                      arena_image=arena_image,
                      timeout_hours=timeout_hours)



def distributed_tf_op(name, image, command, env=[], data=[], sync_source=None,
                      chief=False, chief_cpu_limit='0', chief_memory_limit='0',
                      workers=0, worker_image=None, worker_cpu_limit='0', worker_memory_limit='0',
                      parameter_servers=0, ps_image=None, ps_cpu_limit='0', ps_memory_limit='0',
                      evaluator=False, evaluator_cpu_limit='0', evaluator_memory_limit='0',
                      gpus=0, rdma=False,
                      chief_port=22222,
                      worker_port=22222,
                      ps_port=22224,
                      tensorboard=False,
                      metrics=['Train-accuracy:PERCENTAGE'],
                      arena_image='cheyang/arena_launcher:v0.7',
                      timeout_hours=240):
          """This function submits Distributed TFJob in Distributed mode.

          Args:
            name: the name of distributed_tf_op
            image: the docker image name of training job
            data: specify the datasource to mount to the job, like <name_of_datasource>:<mount_point_on_job>
            command: the command to run
          """
          return dsl.ContainerOp(
            name=name,
          image=arena_image,
          command=['python','arena_launcher.py'],
          arguments=[ "--name", name,
                      "--tensorboard", tensorboard,
                      "--rdma", rdma,
                      "--data", data,
                      "--output-data", output_data,
                      "--image", image,
                      "--gpus", gpus,
                      "--worker-cpu", worker_cpu_limit,
                      "--worker-memory", worker_memory_limit,
                      "--timeout-hours", timeout_hours,
                      "--metric-name", metric_name,
                      "--metric-unit", metric_unit,
                      "--step-name", '{{pod.name}}',
                      "--workflow-name", '{{workflow.name}}',
                      "tfjob",
                      "--workers", workers,
                      "--", command],
          file_outputs={'train': '/output.txt',
                        'workflow':'/workflow-name.txt',
                        'step':'/step-name.txt',
                        'name':'/name.txt'}
          )
