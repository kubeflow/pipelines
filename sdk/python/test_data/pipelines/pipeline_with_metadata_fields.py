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

from kfp import compiler
from kfp import dsl
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.component
def str_to_dataset(string: str, dataset: Output[Dataset]):
    """Convert string to dataset.

    Args:
        string: The string.

    Returns:
        dataset: The dataset.
    """
    with open(dataset.path, 'w') as f:
        f.write(string)


@dsl.component
def dataset_joiner(
    dataset_a: Input[Dataset],
    dataset_b: Input[Dataset],
    out_dataset: Output[Dataset],
) -> str:
    """Concatenate dataset_a and dataset_b.

    Also returns the concatenated string.

    Args:
        dataset_a: First dataset.
        dataset_b: Second dataset.

    Returns:
        out_dataset: The concatenated dataset.
        Output: The concatenated string.
    """
    with open(dataset_a.path) as f:
        content_a = f.read()

    with open(dataset_b.path) as f:
        content_b = f.read()

    concatenated_string = content_a + content_b
    with open(out_dataset.path, 'w') as f:
        f.write(concatenated_string)

    return concatenated_string


@dsl.pipeline(
    display_name='Concatenation pipeline',
    description='A pipeline that joins string to in_dataset.',
)
def dataset_concatenator(
    string: str,
    in_dataset: Input[Dataset],
) -> Dataset:
    """Pipeline to convert string to a Dataset, then concatenate with
    in_dataset.

    Args:
        string: String to concatenate to in_artifact.
        in_dataset: Dataset to which to concatenate string.

    Returns:
        Output: The final concatenated dataset.
    """
    first_dataset_task = str_to_dataset(string=string)
    t = dataset_joiner(
        dataset_a=first_dataset_task.outputs['dataset'],
        dataset_b=in_dataset,
    )
    return t.outputs['out_dataset']


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=dataset_concatenator,
        package_path=__file__.replace('.py', '.yaml'))
