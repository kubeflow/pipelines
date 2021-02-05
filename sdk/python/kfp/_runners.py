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
    "run_pipeline_func_on_cluster",
    "run_pipeline_func_locally",
]


from typing import Callable, List, Mapping

from . import Client, LocalClient, dsl


def run_pipeline_func_on_cluster(
    pipeline_func: Callable,
    arguments: Mapping[str, str],
    run_name: str = None,
    experiment_name: str = None,
    kfp_client: Client = None,
    pipeline_conf: dsl.PipelineConf = None,
):
    """Runs pipeline on KFP-enabled Kubernetes cluster.

    This command compiles the pipeline function, creates or gets an experiment
    and submits the pipeline for execution.

    Feature stage:
    [Alpha](https://github.com/kubeflow/pipelines/blob/07328e5094ac2981d3059314cc848fbb71437a76/docs/release/feature-stages.md#alpha)

    Args:
      pipeline_func: A function that describes a pipeline by calling components
      and composing them into execution graph.
      arguments: Arguments to the pipeline function provided as a dict.
      run_name: Optional. Name of the run to be shown in the UI.
      experiment_name: Optional. Name of the experiment to add the run to.
      kfp_client: Optional. An instance of kfp.Client configured for the desired
        KFP cluster.
      pipeline_conf: Optional. kfp.dsl.PipelineConf instance. Can specify op
        transforms, image pull secrets and other pipeline-level configuration
        options.
    """
    kfp_client = kfp_client or Client()
    return kfp_client.create_run_from_pipeline_func(
        pipeline_func, arguments, run_name, experiment_name, pipeline_conf
    )


def run_pipeline_func_locally(
    pipeline_func: Callable,
    arguments: Mapping[str, str],
    local_client: LocalClient = None,
    local_env_images: List[str] = None,
):
    """Run kubeflow pipeline in docker or locally

    Feature stage:
    [Alpha](https://github.com/kubeflow/pipelines/blob/master/docs/release/feature-stages.md#alpha)

    Args:
      pipeline_func: A function that describes a pipeline by calling components
        and composing them into execution graph.
      arguments: Arguments to the pipeline function provided as a dict.
        reference to `kfp.client.create_run_from_pipeline_func`
      local_client: Optional. An instance of kfp.LocalClient
      local_env_images: list of images
        If the image of component equals to one of `local_env_images`, local runner will run
        this component locally in forked process, otherwise, local runner will run this
        component on docker.
    """
    local_client = local_client or LocalClient()
    return local_client.create_run_from_pipeline_func(
        pipeline_func, arguments, local_env_images
    )
