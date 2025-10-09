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

from typing import List

from kfp import dsl
from kfp.dsl import Dataset


@dsl.component
def make_dataset(text: str) -> Dataset:
    dataset = Dataset(uri=dsl.get_uri(), metadata={'length': len(text)})
    with open(dataset.path, 'w') as f:
        f.write(text)
    return dataset


@dsl.component
def join_datasets(datasets: List[Dataset]) -> Dataset:
    texts = []
    for dataset in datasets:
        with open(dataset.path, 'r') as f:
            texts.append(f.read())

    return ''.join(texts)


@dsl.pipeline
def make_and_join_datasets(
        texts: List[str] = ['Hello', ',', ' ', 'world!']) -> Dataset:
    with dsl.ParallelFor(texts) as text:
        t1 = make_dataset(text=text)

    return join_datasets(datasets=dsl.Collected(t1.output)).output


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=make_and_join_datasets,
        package_path=__file__.replace('.py', '.yaml'))
