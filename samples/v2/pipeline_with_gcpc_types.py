# Copyright 2021-2022 The Kubeflow Authors
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
# import os
import subprocess
from typing import NamedTuple

# TODO: remove this subprocess when GCPC depends on KFP v2
GCPC_PACKAGE_PATH = 'git+https://github.com/kubeflow/pipelines@temp-gcpc-kfpv2-compatibility-branch#subdirectory=components/google-cloud'
PACKAGES_TO_INSTALL = [GCPC_PACKAGE_PATH]
try:
    print(
        subprocess.check_output(
            f'pip install --upgrade pip && pip install {GCPC_PACKAGE_PATH}',
            shell=True))
except subprocess.CalledProcessError as e:
    print(e.stderr)
    raise
#######

from google_cloud_pipeline_components.types import artifact_types
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import compiler
from kfp import dsl
from kfp.dsl import Input
from kfp.dsl import Output

print('next')


@dsl.component(packages_to_install=PACKAGES_TO_INSTALL)
def model_producer(model: Output[artifact_types.VertexModel]):
    with open(model.path, 'w') as f:
        f.write('my model')


@dsl.component(packages_to_install=PACKAGES_TO_INSTALL)
def model_consumer(model: Input[VertexModel]):
    print('artifact.type: ', type(model))
    print('artifact.name: ', model.name)
    print('artifact.uri: ', model.uri)
    print('artifact.metadata: ', model.metadata)


@dsl.component(packages_to_install=PACKAGES_TO_INSTALL)
def model_consumer_with_named_tuple(
    model: Input[VertexModel]
) -> NamedTuple('NamedTupleOutput', [('model1', artifact_types.VertexModel),
                                     ('model2', artifact_types.VertexModel)]):
    NamedTupleOutput = NamedTuple('NamedTupleOutput',
                                  [('model1', artifact_types.VertexModel),
                                   ('model2', artifact_types.VertexModel)])
    return NamedTupleOutput(model1=model, model2=model)


@dsl.pipeline(name='pipeline-with-gcpc-types')
def my_pipeline():
    producer_task = model_producer()
    model_consumer(model=producer_task.outputs['model'])
    model_consumer_with_named_tuple(model=producer_task.outputs['model'])


if __name__ == '__main__':
    ir_file = __file__.replace('.py', '.yaml')
    compiler.Compiler().compile(pipeline_func=my_pipeline, package_path=ir_file)
