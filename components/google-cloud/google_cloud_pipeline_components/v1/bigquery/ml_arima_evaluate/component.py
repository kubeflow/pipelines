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

from typing import Dict, List

from google_cloud_pipeline_components import _image
from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components.types.artifact_types import BQMLModel
from kfp.dsl import Artifact
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@container_component
def bigquery_ml_arima_evaluate_job(
    model: Input[BQMLModel],
    arima_evaluation_metrics: Output[Artifact],
    gcp_resources: OutputPath(str),
    location: str = 'us-central1',
    show_all_candidate_models: bool = False,
    query_parameters: List[str] = [],
    job_configuration_query: Dict[str, str] = {},
    labels: Dict[str, str] = {},
    encryption_spec_key_name: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
):
  # fmt: off
  """Launch a BigQuery ML.ARIMA_EVALUATE job and waits for it to finish.

  Args:
      location: Location to run the BigQuery model evaluation job. If not set, default to `US` multi-region. For more details, see https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model: BigQuery ML model for ML.ARIMA_EVALUATE. For more details, see https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-arima-evaluate#model_name
      show_all_candidate_models: You can use show_all_candidate_models to show evaluation metrics or an error message for either all candidate models or for only the best model with the lowest AIC. The value is type BOOL and is part of the settings STRUCT. For more details, see https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-arima-evaluate#show_all_candidate_models
      query_parameters: jobs.query parameters for standard SQL queries. If query_parameters are both specified in here and in job_configuration_query, the value in here will override the other one.
      job_configuration_query: A json formatted string describing the rest of the job configuration. For more details, see https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      labels: The labels associated with this job. You can use these to organize and group your jobs. Label keys and values can be no longer than 63 characters, can only containlowercase letters, numeric characters, underscores and dashes. International characters are allowed. Label values are optional. Label keys must start with a letter and each label in the list must have a different key.
        Example: { "name": "wrench", "mass": "1.3kg", "count": "3" }.
      encryption_spec_key_name: Describes the Cloud KMS encryption key that will be used to protect destination BigQuery table. The BigQuery Service Account associated with your project requires access to this encryption key. If encryption_spec_key_name are both specified in here and in job_configuration_query, the value in here will override the other one.
      project: Project to run BigQuery model evaluation job. Defaults to the project in which the PipelineJob is run.

  Returns:
      arima_evaluation_metrics: Describes arima metrics. For more details, see https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-arima-evaluate#mlarima_evaluate_output
      gcp_resources: Serialized gcp_resources proto tracking the BigQuery job. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.bigquery.ml_arima_evaluate.launcher',
      ],
      args=[
          '--type',
          'BigqueryMLArimaEvaluateJob',
          '--project',
          project,
          '--location',
          location,
          '--model_name',
          ConcatPlaceholder([
              model.metadata['projectId'],
              '.',
              model.metadata['datasetId'],
              '.',
              model.metadata['modelId'],
          ]),
          '--show_all_candidate_models',
          show_all_candidate_models,
          '--payload',
          ConcatPlaceholder([
              '{',
              '"configuration": {',
              '"query": ',
              job_configuration_query,
              ', "labels": ',
              labels,
              '}',
              '}',
          ]),
          '--job_configuration_query_override',
          ConcatPlaceholder([
              '{',
              '"query_parameters": ',
              query_parameters,
              ', "destination_encryption_configuration": {',
              '"kmsKeyName": "',
              encryption_spec_key_name,
              '"}',
              '}',
          ]),
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
