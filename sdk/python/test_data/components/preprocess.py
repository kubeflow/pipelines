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

from typing import Dict, List

from kfp.dsl import component
from kfp.dsl import Dataset
from kfp.dsl import Output
from kfp.dsl import OutputPath


@component
def preprocess(
    # An input parameter of type string.
    message: str,
    # An input parameter of type dict.
    input_dict_parameter: Dict[str, int],
    # An input parameter of type list.
    input_list_parameter: List[str],
    # Use Output[T] to get a metadata-rich handle to the output artifact
    # of type `Dataset`.
    output_dataset_one: Output[Dataset],
    # A locally accessible filepath for another output artifact of type
    # `Dataset`.
    output_dataset_two_path: OutputPath('Dataset'),
    # A locally accessible filepath for an output parameter of type string.
    output_parameter_path: OutputPath(str),
    # A locally accessible filepath for an output parameter of type bool.
    output_bool_parameter_path: OutputPath(bool),
    # A locally accessible filepath for an output parameter of type dict.
    output_dict_parameter_path: OutputPath(Dict[str, int]),
    # A locally accessible filepath for an output parameter of type list.
    output_list_parameter_path: OutputPath(List[str]),
):
    """Dummy preprocessing step."""

    # Use Dataset.path to access a local file path for writing.
    # One can also use Dataset.uri to access the actual URI file path.
    with open(output_dataset_one.path, 'w') as f:
        f.write(message)

    # OutputPath is used to just pass the local file path of the output artifact
    # to the function.
    with open(output_dataset_two_path, 'w') as f:
        f.write(message)

    with open(output_parameter_path, 'w') as f:
        f.write(message)

    with open(output_bool_parameter_path, 'w') as f:
        f.write(
            str(True))  # use either `str()` or `json.dumps()` for bool values.

    import json
    with open(output_dict_parameter_path, 'w') as f:
        f.write(json.dumps(input_dict_parameter))

    with open(output_list_parameter_path, 'w') as f:
        f.write(json.dumps(input_list_parameter))


if __name__ == '__main__':
    from kfp import compiler
    compiler.Compiler().compile(
        pipeline_func=preprocess, package_path=__file__.replace('.py', '.yaml'))
