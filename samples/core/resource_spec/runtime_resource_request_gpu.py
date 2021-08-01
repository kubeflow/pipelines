# Copyright 2021 The Kubeflow Authors
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

import kfp
from kfp import dsl, components
from typing import NamedTuple


def training_job():
    import torch
    use_cuda = torch.cuda.is_available()
    print(f'The gpus status is: {use_cuda}')
    if not use_cuda:
        raise ValueError('GPU not available')


training_comp = components.create_component_from_func(
        training_job,
        base_image='pytorch/pytorch:1.7.1-cuda11.0-cudnn8-runtime',
        packages_to_install=[]
    )

@components.create_component_from_func
def generate_resource_constraints_request() -> NamedTuple('output', [('gpu_vendor', str), ('nbr_gpus', str), ('constrain_type', str), ('constrain_value', str)]):
    """Returns the gpu resource and constraints settings"""
    from collections import namedtuple
    output = namedtuple('output', ['gpu_vendor', 'nbr_gpu', 'constrain_type', 'constrain_value'])

    return output( 'nvidia.com/gpu', '1', 'cloud.google.com/gke-accelerator', 'nvidia-tesla-p4')

@dsl.pipeline(
    name='Runtime resource request pipeline',
    description='An example on how to make resource requests at runtime.'
)
def resource_constraint_request_pipeline():
    resource_constraints_task = generate_resource_constraints_request()

    traning_task = training_comp().set_gpu_limit(resource_constraints_task.outputs['nbr_gpus'], resource_constraints_task.outputs['gpu_vendor'])\
        .add_node_selector_constraint(resource_constraints_task.outputs['constrain_type'], resource_constraints_task.outputs['constrain_value'])
    # Disable cache for KFP v1 mode.
    traning_task.execution_options.caching_strategy.max_cache_staleness = 'P0D'

if __name__ == '__main__':
    kfp.compiler.Compiler().compile(resource_constraint_request_pipeline, __file__ + '.yaml')
