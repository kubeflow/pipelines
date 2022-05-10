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


def main(argv):
  """Calls ModelService.ImportModelEvaluation."""
  parser = argparse.ArgumentParser(
      prog='Vertex Model Service evaluation importer', description='')
  parser.add_argument(
      '--metrics',
      dest='metrics',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
  parser.add_argument(
      '--metrics_explanation',
      dest='metrics_explanation',
      type=str,
      default=None)
  parser.add_argument(
      '--explanation', dest='explanation', type=str, default=None)
  parser.add_argument(
      '--problem_type',
      dest='problem_type',
      type=str,
      required=True,
      default=argparse.SUPPRESS)
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
  parsed_args, _ = parser.parse_known_args(argv)

  _, project_id, _, location, _, model_id = parsed_args.model_name.split('/')
  api_endpoint = location + '-aiplatform.googleapis.com'
  resource_uri_prefix = f'https://{api_endpoint}/v1/'

  with open(parsed_args.metrics) as metrics_file:
    model_evaluation = {
        'metrics':
            to_value(
                next(
                    iter(
                        json.loads(metrics_file.read())['slicedMetrics'][0]
                        ['metrics'].values()))),
        'metrics_schema_uri':
            PROBLEM_TYPE_TO_SCHEMA_URI.get(parsed_args.problem_type),
    }

  if parsed_args.explanation and parsed_args.explanation == "{{$.inputs.artifacts['explanation'].metadata['explanation_gcs_path']}}":
    # metrics_explanation must contain explanation_gcs_path when provided.
    logging.error(
        '"explanation" must contain explanations when provided.')
    sys.exit(13)
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

  import_model_evaluation_response = aiplatform.gapic.ModelServiceClient(
      client_info=gapic_v1.client_info.ClientInfo(
          user_agent='google-cloud-pipeline-components',),
      client_options={
          'api_endpoint': api_endpoint,
      }).import_model_evaluation(
          parent=parsed_args.model_name,
          model_evaluation=model_evaluation,
      )
  model_evaluation_name = import_model_evaluation_response.name

  # Write the model evaluation resource to GcpResources output.
  model_eval_resources = GcpResources()
  model_eval_resource = model_eval_resources.resources.add()
  model_eval_resource.resource_type = RESOURCE_TYPE
  model_eval_resource.resource_uri = f'{resource_uri_prefix}{model_evaluation_name}'

  with open(parsed_args.gcp_resources, 'w') as f:
    f.write(json_format.MessageToJson(model_eval_resources))


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
