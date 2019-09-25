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


def automl_import_data_from_gcs(
    dataset_path: str,
    input_uris: list,
    retry=None, #=google.api_core.gapic_v1.method.DEFAULT,
    timeout=None, #=google.api_core.gapic_v1.method.DEFAULT,
    metadata: dict = None,
) -> NamedTuple('Outputs', [('dataset_path', str)]):
    import sys
    import subprocess
    subprocess.run([sys.executable, '-m', 'pip', 'install', 'google-cloud-automl==0.4.0', '--quiet', '--no-warn-script-location'], env={'PIP_DISABLE_PIP_VERSION_CHECK': '1'}, check=True)

    import google
    from google.cloud import automl
    client = automl.AutoMlClient()
    input_config = {
        'gcs_source': {
            'input_uris': input_uris,
        },
    }
    response = client.import_data(
        dataset_path,
        input_config,
        retry or google.api_core.gapic_v1.method.DEFAULT,
        timeout or google.api_core.gapic_v1.method.DEFAULT,
        metadata,
    )
    result = response.result()
    print(result)
    metadata = response.metadata
    print(metadata)
    return (dataset_path)


if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(automl_import_data_from_gcs, output_component_file='component.yaml', base_image='python:3.7')
