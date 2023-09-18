# Copyright 2023 The Kubeflow Authors
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

from typing import NamedTuple

from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset


@dsl.component
def dataset_splitter(
    in_dataset: Dataset
) -> NamedTuple(
        'outputs',
        dataset1=Dataset,
        dataset2=Dataset,
):

    with open(in_dataset.path) as f:
        in_data = f.read()

    out_data1, out_data2 = in_data[:len(in_data) // 2], in_data[len(in_data) //
                                                                2:]

    dataset1 = Dataset(
        uri=dsl.get_uri(suffix='dataset1'),
        metadata={'original_data': in_dataset.name},
    )
    with open(dataset1.path, 'w') as f:
        f.write(out_data1)

    dataset2 = Dataset(
        uri=dsl.get_uri(suffix='dataset2'),
        metadata={'original_data': in_dataset.name},
    )
    with open(dataset2.path, 'w') as f:
        f.write(out_data2)

    outputs = NamedTuple(
        'outputs',
        dataset1=Dataset,
        dataset2=Dataset,
    )
    return outputs(dataset1=dataset1, dataset2=dataset2)


outputs = NamedTuple(
    'outputs',
    dataset1=Dataset,
    dataset2=Dataset,
)


@dsl.pipeline
def splitter_pipeline(in_dataset: Dataset) -> outputs:
    task = dataset_splitter(in_dataset=in_dataset)
    return outputs(
        task.outputs['dataset1'],
        task.outputs['dataset1'],
    )


@dsl.component
def make_dataset() -> Artifact:
    artifact = Artifact(uri=dsl.get_uri('dataset'))
    with open(artifact.path, 'w') as f:
        f.write('Hello, world')
    return artifact


@dsl.pipeline
def split_datasets_and_return_first() -> Dataset:
    t1 = make_dataset()
    return splitter_pipeline(in_dataset=t1.output).outputs['dataset1']


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=split_datasets_and_return_first,
        package_path=__file__.replace('.py', '.yaml'))
