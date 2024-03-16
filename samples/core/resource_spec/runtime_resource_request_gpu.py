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

from kfp import dsl, compiler
from typing import NamedTuple


@dsl.component(base_image='pytorch/pytorch:1.7.1-cuda11.0-cudnn8-runtime')
def training_job():
    import torch
    use_cuda = torch.cuda.is_available()
    print(f'The gpus status is: {use_cuda}')
    if not use_cuda:
        raise ValueError('GPU not available')


@dsl.component
def generate_resource_constraints_request() -> NamedTuple('output', nbr_gpus=str, accelerator=str):
    """Returns the gpu resource and constraints settings"""
    output = NamedTuple('output', nbr_gpus=str, accelerator=str)

    return output('1', 'NVIDIA_TESLA_K80')


@dsl.pipeline(
    name='Runtime resource request pipeline',
    description='An example on how to make resource requests at runtime.'
)
def resource_constraint_request_pipeline():
    resource_constraints_task = generate_resource_constraints_request()

    # TODO: support PipelineParameterChannel for .set_accelerator_type
    # TypeError: expected string or bytes-like object, got 'PipelineParameterChannel'
    # traning_task = training_job()\
    #     .set_accelerator_limit(resource_constraints_task.outputs['nbr_gpus'])\
    #     .set_accelerator_type(resource_constraints_task.outputs['accelerator'])\
    traning_task = training_job()\
        .set_accelerator_limit(resource_constraints_task.outputs['nbr_gpus'])\
        .set_accelerator_type('NVIDIA_TESLA_K80')

if __name__ == '__main__':
    compiler.Compiler().compile(resource_constraint_request_pipeline, __file__ + '.yaml')
