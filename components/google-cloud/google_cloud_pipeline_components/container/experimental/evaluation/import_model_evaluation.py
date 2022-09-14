# Copyright 2022 The Kubeflow Authors. All Rights Reserved.
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
"""Module for importing a model evaluation to an existing Vertex model resource."""
import argparse
import base64
import json
import logging
import os
import six
import sys

from google.cloud import aiplatform
from google.api_core import gapic_v1
from google.protobuf.struct_pb2 import Value, Struct, NULL_VALUE, ListValue
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources

from google.protobuf import json_format
from typing import Any, Dict

PROBLEM_TYPE_TO_SCHEMA_URI = {
    'classification':
        'gs://google-cloud-aiplatform/schema/modelevaluation/classification_metrics_1.0.0.yaml',
    'regression':
        'gs://google-cloud-aiplatform/schema/modelevaluation/regression_metrics_1.0.0.yaml',
    'forecasting':
        'gs://google-cloud-aiplatform/schema/modelevaluation/forecasting_metrics_1.0.0.yaml',
}

RESOURCE_TYPE = 'ModelEvaluation'


def _make_parent_dirs_and_return_path(file_path: str):
  os.makedirs(os.path.dirname(file_path), exist_ok=True)
  return file_path


parser = argparse.ArgumentParser(
    prog='Vertex Model Service evaluation importer', description='')
parser.add_argument(
    '--metrics',
    dest='metrics',
    type=str,
    default=None)
parser.add_argument(
    '--classification_metrics',
    dest='classification_metrics',
    type=str,
    default=None)
parser.add_argument(
    '--forecasting_metrics',
    dest='forecasting_metrics',
    type=str,
    default=None)
parser.add_argument(
    '--regression_metrics',
    dest='regression_metrics',
    type=str,
    default=None)
parser.add_argument(
    '--feature_attributions',
    dest='feature_attributions',
    type=str,
    default=None)
parser.add_argument(
    '--metrics_explanation', dest='metrics_explanation', type=str, default=None)
parser.add_argument('--explanation', dest='explanation', type=str, default=None)
parser.add_argument(
    '--problem_type',
    dest='problem_type',
    type=str,
    default=None)
parser.add_argument(
    '--display_name', nargs='?', dest='display_name', type=str, default=None)
parser.add_argument(
    '--pipeline_job_id', dest='pipeline_job_id', type=str, default=None)
parser.add_argument(
    '--pipeline_job_resource_name',
    dest='pipeline_job_resource_name',
    type=str,
    default=None)
parser.add_argument(
    '--dataset_path', nargs='?', dest='dataset_path', type=str, default=None)
parser.add_argument(
    '--dataset_paths', nargs='?', dest='dataset_paths', type=str, default='[]')
parser.add_argument(
    '--dataset_type', nargs='?', dest='dataset_type', type=str, default=None)
parser.add_argument(
    '--model_name',
    dest='model_name',
    type=str,
    required=True,
    default=argparse.SUPPRESS)
parser.add_argument(
    '--gcp_resources',
    dest='gcp_resources',
    type=_make_parent_dirs_and_return_path,
    required=True,
    default=argparse.SUPPRESS)


def main(argv):
  """Calls ModelService.ImportModelEvaluation."""
  parsed_args, _ = parser.parse_known_args(argv)

  _, project_id, _, location, _, model_id = parsed_args.model_name.split('/')
  api_endpoint = location + '-aiplatform.googleapis.com'
  resource_uri_prefix = f'https://{api_endpoint}/v1/'

  if parsed_args.classification_metrics:
    metrics_file_path = parsed_args.classification_metrics
    problem_type = 'classification'
  elif parsed_args.forecasting_metrics:
    metrics_file_path = parsed_args.forecasting_metrics
    problem_type = 'forecasting'
  elif parsed_args.regression_metrics:
    metrics_file_path = parsed_args.regression_metrics
    problem_type = 'regression'
  else:
    metrics_file_path = parsed_args.metrics
    problem_type = parsed_args.problem_type

  metrics_file_path = metrics_file_path if not metrics_file_path.startswith(
      'gs://') else '/gcs' + metrics_file_path[4:]

  schema_uri = PROBLEM_TYPE_TO_SCHEMA_URI.get(problem_type)
  with open(metrics_file_path) as metrics_file:
    sliced_metrics = [{
        **one_slice, 'metrics':
            to_value(next(iter(one_slice['metrics'].values())))
    } for one_slice in json.loads(metrics_file.read())['slicedMetrics']]

  model_evaluation = {
      'metrics': sliced_metrics[0]['metrics'],
      'metrics_schema_uri': schema_uri
  }

  if parsed_args.explanation and parsed_args.explanation == "{{$.inputs.artifacts['explanation'].metadata['explanation_gcs_path']}}":
    # metrics_explanation must contain explanation_gcs_path when provided.
    logging.error(
        '"explanation" must contain explanations when provided.')
    sys.exit(13)
  elif parsed_args.feature_attributions:
    explanation_file_name = parsed_args.feature_attributions if not parsed_args.feature_attributions.startswith(
        'gs://') else '/gcs' + parsed_args.feature_attributions[4:]
  elif parsed_args.explanation:
    explanation_file_name = parsed_args.explanation if not parsed_args.explanation.startswith(
        'gs://') else '/gcs' + parsed_args.explanation[4:]
  elif parsed_args.metrics_explanation and parsed_args.metrics_explanation != "{{$.inputs.artifacts['metrics'].metadata['explanation_gcs_path']}}":
    explanation_file_name = parsed_args.metrics_explanation if not parsed_args.metrics_explanation.startswith(
        'gs://') else '/gcs' + parsed_args.metrics_explanation[4:]
  else:
    explanation_file_name = None
  if explanation_file_name:
    with open(explanation_file_name) as explanation_file:
      model_evaluation['model_explanation'] = {
          'mean_attributions': [{
              'feature_attributions':
                  to_value(
                      json.loads(explanation_file.read())['explanation']
                      ['attributions'][0]['featureAttributions'])
          }]
      }

  if parsed_args.display_name:
    model_evaluation['display_name'] = parsed_args.display_name

  try:
    dataset_paths = json.loads(parsed_args.dataset_paths)
    if not isinstance(dataset_paths, list) or not all(
        isinstance(el, str) for el in dataset_paths):
      dataset_paths = []
  except ValueError:
    dataset_paths = []

  if parsed_args.dataset_path:
    dataset_paths.append(parsed_args.dataset_path)

  metadata = {
      key: value for key, value in {
          'pipeline_job_id': parsed_args.pipeline_job_id,
          'pipeline_job_resource_name': parsed_args.pipeline_job_resource_name,
          'evaluation_dataset_type': parsed_args.dataset_type,
          'evaluation_dataset_path': dataset_paths or None,
      }.items() if value is not None
  }
  if metadata:
    model_evaluation['metadata'] = to_value(metadata)

  client = aiplatform.gapic.ModelServiceClient(
      client_info=gapic_v1.client_info.ClientInfo(
          user_agent='google-cloud-pipeline-components'),
      client_options={
          'api_endpoint': api_endpoint,
      })
  import_model_evaluation_response = client.import_model_evaluation(
      parent=parsed_args.model_name,
      model_evaluation=model_evaluation,
  )
  model_evaluation_name = import_model_evaluation_response.name

  resources = GcpResources()
  # Write the model evaluation resource to GcpResources output.
  model_eval_resource = resources.resources.add()
  model_eval_resource.resource_type = RESOURCE_TYPE
  model_eval_resource.resource_uri = f'{resource_uri_prefix}{model_evaluation_name}'

  if len(sliced_metrics) > 1:
    slice_resources = client.batch_import_model_evaluation_slices(
        parent=model_evaluation_name,
        model_evaluation_slices=[{
            'metrics': one_slice['metrics'],
            'metrics_schema_uri': schema_uri,
            'slice_': to_slice(one_slice['singleOutputSlicingSpec']),
        } for one_slice in sliced_metrics[1:]]).imported_model_evaluation_slices

    for slice_resource in slice_resources:
      slice_mlmd_resource = resources.resources.add()
      slice_mlmd_resource.resource_type = 'ModelEvaluationSlice'
      slice_mlmd_resource.resource_uri = f'{resource_uri_prefix}{slice_resource}'

  with open(parsed_args.gcp_resources, 'w') as f:
    f.write(json_format.MessageToJson(resources))


def to_slice(slicing_spec: Dict[str, Any]):
  value = ''
  if 'bytesValue' in slicing_spec:
    value = base64.b64decode(slicing_spec['bytesValue']).decode('utf-8')
  elif 'floatValue' in slicing_spec:
    value = str(slicing_spec['floatValue'])
  elif 'int64Value' in slicing_spec:
    value = str(slicing_spec['in64Value'])

  return {
      'dimension': 'annotationSpec',
      'value': value,
  }


def to_value(value):
  if value is None:
    return Value(null_value=NULL_VALUE)
  elif isinstance(value, bool):
    # This check needs to happen before isinstance(value, int),
    # isinstance(value, int) returns True when value is bool.
    return Value(bool_value=value)
  elif isinstance(value, six.integer_types) or isinstance(value, float):
    return Value(number_value=value)
  elif isinstance(value, six.string_types) or isinstance(value, six.text_type):
    return Value(string_value=value)
  elif isinstance(value, dict):
    return Value(
        struct_value=Struct(fields={k: to_value(v) for k, v in value.items()}))
  elif isinstance(value, list):
    return Value(
        list_value=ListValue(values=[to_value(item) for item in value]))
  else:
    raise ValueError('Unsupported data type: {}'.format(type(value)))


if __name__ == '__main__':
  print(sys.argv)
  main(sys.argv[1:])
