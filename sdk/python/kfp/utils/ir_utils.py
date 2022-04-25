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

from typing import Dict, Any
import warnings
import json
import yaml


def _write_ir_to_file(ir_dict: Dict[str, Any], output_file: str) -> None:
    """Writes the IR JSON to a file.

    Args:
        ir_dict (Dict[str, Any]): IR JSON.
        output_file (str): Output file path.

    Raises:
        ValueError: If output file path is not JSON or YAML.
    """

    if output_file.endswith(".json"):
        warnings.warn(
            ("Compiling to JSON is deprecated and will be "
             "removed in a future version. Please compile to a YAML file by "
             "providing a file path with a .yaml extension instead."),
            category=DeprecationWarning,
            stacklevel=2,
        )
        ir_json = json.dumps(ir_dict, sort_keys=True)
        with open(output_file, 'w') as json_file:
            json_file.write(ir_json)
    elif output_file.endswith((".yaml", ".yml")):
        with open(output_file, 'w') as yaml_file:
            yaml.dump(ir_dict, yaml_file, sort_keys=True)
    else:
        raise ValueError(
            f'The output path {output_file} should end with ".yaml".')