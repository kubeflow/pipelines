# Copyright 2024 The Kubeflow Authors
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

from typing import List

from kfp import compiler
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset


@dsl.component
def print_artifact_name(artifact: Artifact) -> str:
    print(artifact.name)
    return artifact.name


@dsl.component
def make_dataset(data: str) -> Dataset:
    dataset = Dataset(uri=dsl.get_uri(), metadata={'length': len(data)})
    with open(dataset.path, 'w') as f:
        f.write(data)
    return dataset


@dsl.pipeline
def make_datasets(
        texts: List[str] = ['Hello', ',', ' ', 'world!']) -> List[Dataset]:
    with dsl.ParallelFor(texts) as text:
        t1 = make_dataset(data=text)

    return dsl.Collected(t1.output)


@dsl.component
def make_artifact(data: str) -> Artifact:
    artifact = Artifact(uri=dsl.get_uri(), metadata={'length': len(data)})
    with open(artifact.path, 'w') as f:
        f.write(data)
    return artifact


@dsl.pipeline
def make_artifacts(
        texts: List[str] = ['Hello', ',', ' ', 'world!']) -> List[Artifact]:
    with dsl.ParallelFor(texts) as text:
        t1 = make_artifact(data=text)

    return dsl.Collected(t1.output)


@dsl.pipeline(name='pipeline-parallelfor-artifacts')
def my_pipeline():
    make_artifacts_task = make_artifacts()
    with dsl.ParallelFor(items=make_artifacts_task.output) as item:
        print_artifact_name(artifact=item)

    make_datasets_task = make_datasets()
    with dsl.ParallelFor(items=make_datasets_task.output) as item:
        print_artifact_name(artifact=item)


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=my_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
