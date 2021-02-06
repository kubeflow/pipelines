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


def automl_prediction_service_batch_predict(
    model_path,
    gcs_input_uris: list = None,
    gcs_output_uri_prefix: str = None,
    bq_input_uri: str = None,
    bq_output_uri: str = None,
    params=None,
    retry=None, #google.api_core.gapic_v1.method.DEFAULT,
    timeout=None, #google.api_core.gapic_v1.method.DEFAULT,
    metadata: dict = None,
) -> NamedTuple('Outputs', [('gcs_output_directory', str), ('bigquery_output_dataset', str)]):
    import sys
    import subprocess
    subprocess.run([sys.executable, '-m', 'pip', 'install', 'google-cloud-automl==0.4.0', '--quiet', '--no-warn-script-location'], env={'PIP_DISABLE_PIP_VERSION_CHECK': '1'}, check=True)

    input_config = {}
    if gcs_input_uris:
        input_config['gcs_source'] = {'input_uris': gcs_input_uris}
    if bq_input_uri:
        input_config['bigquery_source'] = {'input_uri': bq_input_uri}

    output_config = {}
    if gcs_output_uri_prefix:
        output_config['gcs_destination'] = {'output_uri_prefix': gcs_output_uri_prefix}
    if bq_output_uri:
        output_config['bigquery_destination'] = {'output_uri': bq_output_uri}

    from google.cloud import automl
    client = automl.PredictionServiceClient()
    response = client.batch_predict(
        model_path,
        input_config,
        output_config,
        params,
        retry,
        timeout,
        metadata,
    )
    print('Operation started:')
    print(response.operation)
    result = response.result()
    metadata = response.metadata
    print('Operation finished:')
    print(metadata)
    output_info = metadata.batch_predict_details.output_info
    # Workaround for Argo issue - it fails when output is empty: https://github.com/argoproj/argo/pull/1277/files#r326028422
    return (output_info.gcs_output_directory or '-', output_info.bigquery_output_dataset or '-')


if __name__ == '__main__':
    import kfp
    kfp.components.func_to_container_op(automl_prediction_service_batch_predict, output_component_file='component.yaml', base_image='python:3.7')
