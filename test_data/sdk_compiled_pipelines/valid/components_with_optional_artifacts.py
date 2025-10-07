# Copyright 2022 The Kubeflow Authors
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
from typing import Optional

from kfp import compiler
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Input


@dsl.component
def python_artifact_printer(artifact: Optional[Input[Artifact]] = None):
    if artifact is not None:
        print(artifact.name)
        print(artifact.uri)
        print(artifact.metadata)
    else:
        print('No artifact provided!')


@dsl.container_component
def custom_artifact_printer(artifact: Optional[Input[Artifact]] = None):
    return dsl.ContainerSpec(
        image='alpine',
        command=[
            dsl.IfPresentPlaceholder(
                input_name='artifact',
                then=['echo', artifact.uri],
                else_=['echo', 'No artifact provided!'])
        ])


@dsl.pipeline
def inner_pipeline(dataset: Optional[Input[Dataset]] = None):
    python_artifact_printer(artifact=dataset)


@dsl.pipeline(name='optional-artifact-pipeline')
def pipeline(dataset1: Optional[Input[Dataset]] = None):

    custom_artifact_printer(artifact=dataset1)
    custom_artifact_printer()

    dataset2 = dsl.importer(
        artifact_uri='gs://ml-pipeline-playground/shakespeare1.txt',
        artifact_class=Dataset,
    )
    inner_pipeline(dataset=dataset2.output)
    inner_pipeline()


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=pipeline, package_path=__file__.replace('.py', '.yaml'))
