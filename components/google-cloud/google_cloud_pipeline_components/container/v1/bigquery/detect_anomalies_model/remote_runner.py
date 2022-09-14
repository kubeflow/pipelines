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

import json
import logging

import google.auth
import google.auth.transport.requests
from google_cloud_pipeline_components.container.v1.bigquery.utils import bigquery_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import artifact_util
from google_cloud_pipeline_components.types.artifact_types import BQMLModel
import requests


def bigquery_detect_anomalies_model_job(
    type,
    project,
    location,
    model_name,
    table_name,
    query_statement,
    contamination,
    anomaly_prob_threshold,
    payload,
    job_configuration_query_override,
    gcp_resources,
    executor_input,
):
  """Create and poll bigquery detect anomalies model job till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the bigquery job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the payload into the job spec and create the bigquery job
  3. Poll the bigquery job status every
  job_remote_runner._POLLING_INTERVAL_IN_SECONDS seconds
     - If the bigquery job is succeeded, return succeeded
     - If the bigquery job is pending/running, continue polling the status

  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.


  Args:
      type: BigQuery model detect anomalies job type.
      project: Project to launch the query job.
      location: location to launch the query job. For more details, see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model_name: BigQuery ML model name for generating detected anomalies. For
        more details, see
      https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#model_name
      table_name: BigQuery table id of the input table that contains the
        prediction data. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#table_name
      query_statement: query statement string used to generate the prediction
        data. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#query_statement
      contamination: contamination is the proportion of anomalies in the
        training dataset that are used to create the AUTOENCODER, KMEANS, or PCA
        input models. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#contamination
      anomaly_prob_threshold: The ARIMA_PLUS model supports the
        anomaly_prob_threshold custom threshold for anomaly detection. The value
        of the anomaly probability at each timestamp is calculated using the
        actual time-series data value and the values of the predicted
        time-series data and the variance from the model training. The actual
        time-series data value at a specific timestamp is identified as
        anomalous if the anomaly probability exceeds the anomaly_prob_threshold
        value. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-detect-anomalies#anomaly_prob_threshold
      payload: A json serialized Job proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
      job_configuration_query_override: A json serialized JobConfigurationQuery
        proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      gcp_resources: File path for storing `gcp_resources` output parameter.
      executor_input: A json serialized pipeline executor input.
  """
  settings_field_sql_list = []
  if contamination is not None and contamination >= 0.0 and contamination <= 0.5:
    settings_field_sql_list.append('%s AS contamination' % contamination)

  if anomaly_prob_threshold is not None and anomaly_prob_threshold > 0.0 and anomaly_prob_threshold < 1.0:
    settings_field_sql_list.append('%s AS anomaly_prob_threshold' %
                                   anomaly_prob_threshold)

  settings_field_sql = ','.join(settings_field_sql_list)

  job_configuration_query_override_json = json.loads(
      job_configuration_query_override, strict=False)

  if query_statement or table_name:
    input_data_sql = ('TABLE %s' %
                      bigquery_util.back_quoted_if_needed(table_name)
                      if table_name else '(%s)' % query_statement)
    settings_sql = ' STRUCT(%s), ' % settings_field_sql
    job_configuration_query_override_json[
        'query'] = 'SELECT * FROM ML.DETECT_ANOMALIES(MODEL `%s`, %s%s)' % (
            model_name, settings_sql, input_data_sql)
  else:
    settings_sql = ' STRUCT(%s)' % settings_field_sql
    job_configuration_query_override_json[
        'query'] = 'SELECT * FROM ML.DETECT_ANOMALIES(MODEL `%s`, %s)' % (
            model_name, settings_sql)

  # TODO(mingge): check if model is a valid BigQuery model resource.
  return bigquery_util.bigquery_query_job(
      type, project, location, payload,
      json.dumps(job_configuration_query_override_json), gcp_resources,
      executor_input)
