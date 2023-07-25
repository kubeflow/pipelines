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
from google_cloud_pipeline_components.container.utils import artifact_utils
from google_cloud_pipeline_components.container.v1.bigquery.utils import bigquery_util


def bigquery_evaluate_model_job(
    type,
    project,
    location,
    model_name,
    table_name,
    query_statement,
    threshold,
    payload,
    job_configuration_query_override,
    gcp_resources,
    executor_input,
):
  """Create and poll bigquery evaluation model job till it reaches a final state.

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
      type: BigQuery model prediction job type.
      project: Project to launch the query job.
      location: location to launch the query job. For more details, see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model_name: BigQuery ML model name for prediction. For more details, see
      https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-predict#predict_model_name
      table_name: BigQuery table id of the input table that contains the
        prediction data. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-predict#predict_table_name
      query_statement: query statement string used to generate the prediction
        data. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-predict#predict_query_statement
      threshold: A custom threshold for the binary logistic regression model
        used as the cutoff between two labels. Predictions above the threshold
        are treated as positive prediction. Predictions below the threshold are
        negative predictions. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-predict#threshold
      payload: A json serialized Job proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
      job_configuration_query_override: A json serialized JobConfigurationQuery
        proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      gcp_resources: File path for storing `gcp_resources` output parameter.
      executor_input:A json serialized pipeline executor input.
  """
  if query_statement and table_name:
    raise ValueError(
        'One and only one of query_statement and table_name should be '
        'populated for BigQuery evaluation model job.'
    )

  input_data_sql = ''
  if table_name:
    input_data_sql = ', TABLE %s' % bigquery_util.back_quoted_if_needed(
        table_name
    )
  if query_statement:
    input_data_sql = ', (%s)' % query_statement

  threshold_sql = ''
  if threshold is not None and threshold > 0.0 and threshold < 1.0:
    threshold_sql = ', STRUCT(%s AS threshold)' % threshold

  job_configuration_query_override_json = json.loads(
      job_configuration_query_override, strict=False
  )
  job_configuration_query_override_json['query'] = (
      'SELECT * FROM ML.EVALUATE(MODEL %s%s%s)'
      % (
          bigquery_util.back_quoted_if_needed(model_name),
          input_data_sql,
          threshold_sql,
      )
  )

  creds, _ = google.auth.default()
  job_uri = bigquery_util.check_if_job_exists(gcp_resources)
  if job_uri is None:
    job_uri = bigquery_util.create_query_job(
        project,
        location,
        payload,
        json.dumps(job_configuration_query_override_json),
        creds,
        gcp_resources,
    )

  # Poll bigquery job status until finished.
  job = bigquery_util.poll_job(job_uri, creds)
  logging.info('Getting query result for job ' + job['id'])
  _, job_id = job['id'].split('.')
  query_results = bigquery_util.get_query_results(
      project, job_id, location, creds
  )
  artifact_utils.update_output_artifact(
      executor_input,
      'evaluation_metrics',
      '',
      {
          bigquery_util.ARTIFACT_PROPERTY_KEY_SCHEMA: query_results[
              bigquery_util.ARTIFACT_PROPERTY_KEY_SCHEMA
          ],
          bigquery_util.ARTIFACT_PROPERTY_KEY_ROWS: query_results[
              bigquery_util.ARTIFACT_PROPERTY_KEY_ROWS
          ],
      },
  )
