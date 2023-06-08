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
from google_cloud_pipeline_components.types.artifact_types import BQMLModel
from google_cloud_pipeline_components.types.artifact_types import BQTable
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@container_component
def bigquery_detect_anomalies_job(
    project: str,
    model: Input[BQMLModel],
    destination_table: Output[BQTable],
    gcp_resources: OutputPath(str),
    location: str = 'us-central1',
    table_name: str = '',
    query_statement: str = '',
    contamination: float = -1.0,
    anomaly_prob_threshold: float = 0.95,
    query_parameters: List[str] = [],
    job_configuration_query: Dict[str, str] = {},
    labels: Dict[str, str] = {},
    encryption_spec_key_name: str = '',
):
  # fmt: off
  """Launch a BigQuery detect anomalies model job and waits for it to finish.

  Args:
      project: Project to run BigQuery model prediction job.
      location: Location to run the BigQuery model prediction job. If not set, default
        to `US` multi-region. For more details, see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model: BigQuery ML model for prediction.
        For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#model_name
      table_name: BigQuery table id of the input table that contains the data. For more
        details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#table_name
      query_statement: Query statement string used to generate
        the data. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#query_statement
      contamination: Contamination is the proportion of anomalies in the training dataset
        that are used to create the
        AUTOENCODER, KMEANS, or PCA input models. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#contamination
      anomaly_prob_threshold: The ARIMA_PLUS model supports the
          anomaly_prob_threshold custom threshold for anomaly detection. The
          value of the anomaly probability at each timestamp is calculated
          using the actual time-series data value and the values of the
          predicted time-series data and the variance from the model
          training. The actual time-series data value at a specific
          timestamp is identified as anomalous if the anomaly probability
          exceeds the anomaly_prob_threshold value. For more details, see
          https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#anomaly_prob_threshold
      query_parameters: Query parameters for standard SQL queries.
        If query_parameters are both specified in here and in
        job_configuration_query, the value in here will override the other one.
      job_configuration_query: A json formatted string describing the rest of the job configuration.
        For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      labels: The labels associated with this job. You can
        use these to organize and group your jobs. Label keys and values can
        be no longer than 63 characters, can only containlowercase letters,
        numeric characters, underscores and dashes. International characters
        are allowed. Label values are optional. Label keys must start with a
        letter and each label in the list must have a different key.
        Example: { "name": "wrench", "mass": "1.3kg", "count": "3" }.
      encryption_spec_key_name:
        Describes the Cloud
        KMS encryption key that will be used to protect destination
        BigQuery table. The BigQuery Service Account associated with your
        project requires access to this encryption key. If
        encryption_spec_key_name are both specified in here and in
        job_configuration_query, the value in here will override the other
        one.

  Returns:
      destination_table: Describes the table where the model prediction results should be
        stored.
        This property must be set for large results that exceed the maximum
        response size.
        For queries that produce anonymous (cached) results, this field will
        be populated by BigQuery.
      gcp_resources: Serialized gcp_resources proto tracking the BigQuery job. For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return ContainerSpec(
      image=_image.GCPC_IMAGE_TAG,
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.bigquery.detect_anomalies_model.launcher',
      ],
      args=[
          '--type',
          'BigqueryDetectAnomaliesModelJob',
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
          '--table_name',
          table_name,
          '--query_statement',
          query_statement,
          '--contamination',
          contamination,
          '--anomaly_prob_threshold',
          anomaly_prob_threshold,
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
