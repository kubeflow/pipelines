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

__all__ = [
    'run_pipeline_func_on_cluster',
]


from typing import Mapping, Callable

from . import Client
from . import dsl


def run_pipeline_func_on_cluster(pipeline_func: Callable, arguments: Mapping[str, str], run_name : str = None, experiment_name : str = None, kfp_client : Client = None, pipeline_conf: dsl.PipelineConf = None):
    '''Runs pipeline on KFP-enabled Kubernetes cluster.
    This command compiles the pipeline function, creates or gets an experiment and submits the pipeline for execution.

    Args:
      pipeline_func: A function that describes a pipeline by calling components and composing them into execution graph.
      arguments: Arguments to the pipeline function provided as a dict.
      run_name: Optional. Name of the run to be shown in the UI.
      experiment_name: Optional. Name of the experiment to add the run to.
      kfp_client: Optional. An instance of kfp.Client configured for the desired KFP cluster.
      pipeline_conf: Optional. kfp.dsl.PipelineConf instance. Can specify op transforms, image pull secrets and other pipeline-level configuration options.ta
    '''
    kfp_client = kfp_client or Client()
    return kfp_client.create_run_from_pipeline_func(pipeline_func, arguments, run_name, experiment_name, pipeline_conf)
