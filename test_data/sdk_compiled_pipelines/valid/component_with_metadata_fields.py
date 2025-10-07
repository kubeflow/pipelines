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


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=dataset_joiner,
        package_path=__file__.replace('.py', '.yaml'))
