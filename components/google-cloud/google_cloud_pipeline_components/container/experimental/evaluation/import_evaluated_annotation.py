# Copyright 2023 The Kubeflow Authors. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
"""Module for importing model evaluated annotations to an existing Vertex model evaluation slice resource."""
import argparse
from collections import defaultdict
import json
import logging
import os
import sys
from typing import Any

from google.api_core import gapic_v1
from google.cloud import storage
from google.cloud import aiplatform_v1
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
import six

from google.protobuf.struct_pb2 import ListValue, NULL_VALUE, Struct, Value
from google.protobuf import json_format


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


def _parse_args(argv):
  """Parses KFP input arguments."""
  parser = argparse.ArgumentParser(
      prog='Vertex Model Service evaluated annotation importer', description=''
  )

  parser.add_argument(
      '--evaluated_annotation_output_uri',
      dest='evaluated_annotation_output_uri',
      type=str,
      default=None,
  )
  parser.add_argument(
      '--error_analysis_output_uri',
      dest='error_analysis_output_uri',
      type=str,
      default=None,
  )
  parser.add_argument(
      '--evaluation_importer_gcp_resources',
      dest='evaluation_importer_gcp_resources',
      type=str,
      default=None,
  )
  parser.add_argument(
      '--pipeline_job_id', dest='pipeline_job_id', type=str, default=None
  )
  parser.add_argument(
      '--pipeline_job_resource_name',
      dest='pipeline_job_resource_name',
      type=str,
      default=None,
  )
  parser.add_argument(
      '--model_name',
      dest='model_name',
      type=str,
      required=True,
      default=argparse.SUPPRESS,
  )
  parser.add_argument(
      '--gcp_resources',
      dest='gcp_resources',
      type=_make_parent_dirs_and_return_path,
      required=True,
      default=argparse.SUPPRESS,
  )
  return parser.parse_known_args(argv)[0]


def get_model_eval_resource_name(
    resource_uri_prefix: str, eval_importer_gcp_resources: str
) -> str:
  """Returns a Model Evaluation resource name from eval_importer_gcp_resource."""
  resources = json.loads(eval_importer_gcp_resources)['resources']
  for resource in resources:
    if resource['resourceType'] == 'ModelEvaluation':
      if resource['resourceUri'].startswith(resource_uri_prefix):
        return resource['resourceUri'][len(resource_uri_prefix) :]
  raise ValueError('Model Evaluation resource not found!')


def get_model_evaluation_slices_annotation_spec_map(
    client: aiplatform_v1.ModelServiceClient,
    model_evaluation_resource_name: str,
) -> dict[str, str]:
  """Builds a map for converting slice values to slice resource names."""
  annotation_spec_value_to_slice_name_map = {}
  # TODO(jsndai): add filter for only listing annotationSpec slice dimension.
  request = aiplatform_v1.ListModelEvaluationSlicesRequest(
      parent=model_evaluation_resource_name
  )
  evaluation_slices_page_result = client.list_model_evaluation_slices(
      request=request
  )
  for eval_slice in evaluation_slices_page_result:
    if eval_slice.slice_.dimension == 'annotationSpec':
      annotation_spec_value_to_slice_name_map.update(
          {eval_slice.slice_.value: eval_slice.name}
      )
  return annotation_spec_value_to_slice_name_map


def read_gcs_uri_as_text(gcs_uri: str) -> str:
  """Reads the contents of a file in Google Cloud Storage as text.

  Args:
    gcs_uri (str): The GCS URI to read, e.g.
      'gs://my-bucket/path/to/myfile.txt'.

  Returns:
      str: The contents of the file as a string.
  """
  if not gcs_uri.startswith('gs://'):
    raise ValueError('Invalid GCS URI: {}'.format(gcs_uri))
  bucket_name, file_path = gcs_uri.split('//')[1].split('/', 1)

  storage_client = storage.Client()
  bucket = storage_client.bucket(bucket_name)
  blob = bucket.blob(file_path)
  return blob.download_as_text()


def get_error_analysis_map(output_uri: str) -> dict[str, Any]:
  """Reads error analysis output and parses it into a dictionary.

  Args:
    output_uri (str): The GCS URI to error analysis output.

  Returns:
    A dictionary with annotation_resource_name as keys and
    `ErrorAnalysisAnnotation` objects as values.
  """
  error_analysis_map = defaultdict(list)
  error_analysis_file_contents = read_gcs_uri_as_text(output_uri)
  for line in error_analysis_file_contents.splitlines():
    try:
      json_object = json.loads(line.strip())
      error_analysis_map[json_object['annotationResourceName']] += [
          json_object['annotation']
      ]
    except json.JSONDecodeError as e:
      raise ValueError('Invalid JSONL file: {}'.format(output_uri)) from e
  return error_analysis_map


def main(argv):
  """Calls ModelService.BatchImportEvaluatedAnnotations."""
  parsed_args = _parse_args(argv)

  _, project_id, _, location, _, model_id = parsed_args.model_name.split('/')
  api_endpoint = location + '-aiplatform.googleapis.com'
  resource_uri_prefix = f'https://{api_endpoint}/v1/'
  client = aiplatform_v1.ModelServiceClient(
      client_info=gapic_v1.client_info.ClientInfo(
          user_agent='google-cloud-pipeline-components'
      ),
      client_options={
          'api_endpoint': api_endpoint,
      },
  )

  model_evaluation_resource_name = get_model_eval_resource_name(
      resource_uri_prefix, parsed_args.evaluation_importer_gcp_resources
  )
  slice_value_to_slice_resource_name_map = (
      get_model_evaluation_slices_annotation_spec_map(
          client, model_evaluation_resource_name
      )
  )
  logging.info(
      'target_column_slice_map: %s',
      slice_value_to_slice_resource_name_map,
  )


if __name__ == '__main__':
  print(sys.argv)
  main(sys.argv[1:])
