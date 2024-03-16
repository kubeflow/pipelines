# Copyright 2020-2021 The Kubeflow Authors
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

from kfp import dsl


@dsl.component
def training_op(n: int) -> int:
    # quickly allocate a lot of memory to verify memory is enough
    a = [i for i in range(n)]
    return len(a)


@dsl.pipeline(
    name='pipeline-with-resource-spec',
    description='A pipeline with resource specification.')
def my_pipeline(n: int = 11234567):
    # For units of these resource limits,
    # refer to https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes
    # 11234567 roughly needs 400Mi+ memory.
    #
    # Note, with v2 python components, there's a larger memory overhead caused
    # by installing KFP SDK in the component, so we had to increase memory limit to 650M.
    training_task = training_op(n=n).set_cpu_limit('1').set_memory_limit('650M')
    
    # TODO(gkcalat): enable requests once SDK implements the feature
    # training_task = training_task.set_cpu_request('1').set_memory_request('650M')

    # TODO(Bobgy): other resource specs like cpu requests, memory requests and
    # GPU limits are not available yet: https://github.com/kubeflow/pipelines/issues/6354.
    # There are other resource spec you can set.
    # For example, to use TPU, add the following:
    # .add_node_selector_constraint('cloud.google.com/gke-accelerator', 'tpu-v3')
    # .set_gpu_limit(1)
