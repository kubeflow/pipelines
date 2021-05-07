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

from kfp import components
from kfp import dsl


@components.create_component_from_func
def training_op(n: int) -> int:
    # quickly allocate a lot of memory to verify memory is enough
    a = [i for i in range(n)]
    return len(a)


@dsl.pipeline(
    name='pipeline-with-resource-spec',
    description='A pipeline with resource specification.'
)
def my_pipeline(n: int = 11234567):
    # For units of these resource limits,
    # refer to https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes
    # 11234567 roughly needs 400Mi+ memory.
    training_task = training_op(n=n
                               ).set_cpu_limit('1').set_memory_limit('500Mi')
    # There are other resource spec you can set.
    # For example, to use TPU, add the following:
    # .add_node_selector_constraint('cloud.google.com/gke-accelerator', 'tpu-v3')
    # .set_gpu_limit(1)

    # Disable cache for KFP v1 mode.
    training_task.execution_options.caching_strategy.max_cache_staleness = "P0D"
