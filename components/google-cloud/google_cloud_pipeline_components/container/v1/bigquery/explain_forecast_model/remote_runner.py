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


def bigquery_explain_forecast_model_job(
    type,
    project,
    location,
    model_name,
    horizon,
    confidence_level,
    payload,
    job_configuration_query_override,
    gcp_resources,
    executor_input,
):
  """Create and poll bigquery ML.EXPLAIN_FORECAST job till it reaches a final state.

  The launching logic is the same as bigquery_{predict|evaluate}_model_job.

  Args:
      type: BigQuery ML.EXPLAIN_FORECAST job type.
      project: Project to launch the query job.
      location: location to launch the query job. For more details, see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model_name: BigQuery ML model name for ML.EXPLAIN_FORECAST. For more
        details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-forecast
      horizon: Horizon is the number of time points to forecast. For more
        details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-forecast#horizon
      confidence_level: The percentage of the future values that fall in the
        prediction interval. For more details, see
        https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-forecast#confidence_level
      payload: A json serialized Job proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
      job_configuration_query_override: A json serialized JobConfigurationQuery
        proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      gcp_resources: File path for storing `gcp_resources` output parameter.
      executor_input:A json serialized pipeline executor input.
  """
  settings_field_sql_list = []
  if horizon is not None and horizon > 0:
    settings_field_sql_list.append('%s AS horizon' % horizon)

  if confidence_level is not None and confidence_level >= 0.0 and confidence_level < 1.0:
    settings_field_sql_list.append('%s AS confidence_level' % confidence_level)

  settings_field_sql = ','.join(settings_field_sql_list)
  settings_sql = ', STRUCT(%s)' % settings_field_sql

  job_configuration_query_override_json = json.loads(
      job_configuration_query_override, strict=False)
  job_configuration_query_override_json[
      'query'] = 'SELECT * FROM ML.EXPLAIN_FORECAST(MODEL %s %s)' % (
          bigquery_util.back_quoted_if_needed(model_name), settings_sql)

  return bigquery_util.bigquery_query_job(
      type, project, location, payload,
      json.dumps(job_configuration_query_override_json), gcp_resources,
      executor_input)
