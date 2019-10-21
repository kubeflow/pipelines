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


def automl_export_data_to_gcs(
    dataset_path: str,
    gcs_output_uri_prefix: str = None,
    #retry=None, #=google.api_core.gapic_v1.method.DEFAULT,
    timeout: float = None, #=google.api_core.gapic_v1.method.DEFAULT,
    metadata: dict = {},
) -> NamedTuple('Outputs', [('gcs_output_uri_prefix', str)]):
    """Exports dataset data to GCS."""
    import sys
    import subprocess
    subprocess.run([sys.executable, "-m", "pip", "install", "google-cloud-automl==0.4.0", "--quiet", "--no-warn-script-location"], env={"PIP_DISABLE_PIP_VERSION_CHECK": "1"}, check=True)

    import google
    from google.cloud import automl
    client = automl.AutoMlClient()

    output_config = {"gcs_destination": {"output_uri_prefix": gcs_output_uri_prefix}}

    response = client.export_data(
        name=dataset_path,
        output_config=output_config,
        #retry=retry or google.api_core.gapic_v1.method.DEFAULT
        timeout=timeout or google.api_core.gapic_v1.method.DEFAULT,
        metadata=metadata,
    )
    print('Operation started:')
    print(response.operation)
    result = response.result()
    metadata = response.metadata
    print('Operation finished:')
    print(metadata)
    return (gcs_output_uri_prefix, )

if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(automl_export_data_to_gcs, output_component_file='component.yaml', base_image='python:3.7')
