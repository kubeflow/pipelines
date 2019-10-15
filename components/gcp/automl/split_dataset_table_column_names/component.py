# Copyright 2019 Google LLC
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


def automl_split_dataset_table_column_names(
    dataset_path: str,
    target_column_name: str,
    table_index: int = 0,
) -> NamedTuple('Outputs', [('target_column_path', str), ('feature_column_paths', list)]):
    import sys
    import subprocess
    subprocess.run([sys.executable, '-m', 'pip', 'install', 'google-cloud-automl==0.4.0', '--quiet', '--no-warn-script-location'], env={'PIP_DISABLE_PIP_VERSION_CHECK': '1'}, check=True)

    from google.cloud import automl
    client = automl.AutoMlClient()
    list_table_specs_response = client.list_table_specs(dataset_path)
    table_specs = [s for s in list_table_specs_response]
    print('table_specs=')
    print(table_specs)
    table_spec_name = table_specs[table_index].name

    list_column_specs_response = client.list_column_specs(table_spec_name)
    column_specs = [s for s in list_column_specs_response]
    print('column_specs=')
    print(column_specs)

    target_column_spec = [s for s in column_specs if s.display_name == target_column_name][0]
    feature_column_specs = [s for s in column_specs if s.display_name != target_column_name]
    feature_column_names = [s.name for s in feature_column_specs]

    import json
    return (target_column_spec.name, json.dumps(feature_column_names))


if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(automl_split_dataset_table_column_names, output_component_file='component.yaml', base_image='python:3.7')
