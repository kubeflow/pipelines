# Copyright 2021 The Kubeflow Authors. All Rights Reserved.
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

import logging
import json
import re
import os
import requests
import time
import google.auth
import google.auth.transport.requests

from ...utils import execution_context
from .utils import json_util
from .utils import artifact_util
from google.cloud import bigquery
from google.protobuf import json_format
from google_cloud_pipeline_components.proto.gcp_resources_pb2 import GcpResources
from google_cloud_pipeline_components.types.artifact_types import BQTable, BQMLModel
from os import path
from typing import Optional

_POLLING_INTERVAL_IN_SECONDS = 20
_BQ_JOB_NAME_TEMPLATE = r'(https://www.googleapis.com/bigquery/v2/projects/(?P<project>.*)/jobs/(?P<job>.*)\?location=(?P<location>.*))'
_ARTIFACT_PROPERTY_KEY_SCHEMA = 'schema'
_ARTIFACT_PROPERTY_KEY_ROWS = 'rows'


def _back_quoted_if_needed(resource_name) -> str:
  """Enclose resource name with ` if it's not yet."""
  if not resource_name or resource_name.startswith('`'):
    return resource_name
  return '`{}`'.format(resource_name)


def _check_if_job_exists(gcp_resources) -> Optional[str]:
  """Check if the BigQuery job already created.

  Return the job url if created. Return None otherwise
  """
  if path.exists(gcp_resources) and os.stat(gcp_resources).st_size != 0:
    with open(gcp_resources) as f:
      serialized_gcp_resources = f.read()
      job_resources = json_format.Parse(serialized_gcp_resources,
                                        GcpResources())
      # Resources should only contain one item.
      if len(job_resources.resources) != 1:
        raise ValueError(
            f'gcp_resources should contain one resource, found {len(job_resources.resources)}'
        )
      # Validate the format of the resource uri.
      job_name_pattern = re.compile(_BQ_JOB_NAME_TEMPLATE)
      match = job_name_pattern.match(job_resources.resources[0].resource_uri)
      try:
        project = match.group('project')
        job = match.group('job')
      except AttributeError as err:
        raise ValueError('Invalid bigquery job uri: {}. Expect: {}.'.format(
            job_resources.resources[0].resource_uri,
            'https://www.googleapis.com/bigquery/v2/projects/[projectId]/jobs/[jobId]?location=[location]'
        ))

    return job_resources.resources[0].resource_uri
  else:
    return None


def _create_job(project, location, job_request_json, creds,
                gcp_resources) -> str:
  """Create a new BigQuery job


    Args:
        project: Project to launch the job.
        location: location to launch the job. For more details, see
          https://cloud.google.com/bigquery/docs/locations#specifying_your_location
        job_request_json: A json object of Job proto. For more details, see
          https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
        creds: Google auth credential.
        gcp_resources: File path for storing `gcp_resources` output parameter.

   Returns:
        The URI of the BigQuery Job.
  """
  # Overrides the location
  if location:
    if 'jobReference' not in job_request_json:
      job_request_json['jobReference'] = {}
    job_request_json['jobReference']['location'] = location

  creds.refresh(google.auth.transport.requests.Request())
  headers = {
      'Content-type': 'application/json',
      'Authorization': 'Bearer ' + creds.token,
      'User-Agent': 'google-cloud-pipeline-components'
  }
  insert_job_url = f'https://www.googleapis.com/bigquery/v2/projects/{project}/jobs'
  job = requests.post(
      url=insert_job_url, data=json.dumps(job_request_json),
      headers=headers).json()
  if 'selfLink' not in job:
    raise RuntimeError(
        'BigQquery Job failed. Cannot retrieve the job name. Response: {}.'
        .format(job))

  # Write the bigquey job uri to gcp resource.
  job_uri = job['selfLink']
  job_resources = GcpResources()
  job_resource = job_resources.resources.add()
  job_resource.resource_type = 'BigQueryJob'
  job_resource.resource_uri = job_uri
  with open(gcp_resources, 'w') as f:
    f.write(json_format.MessageToJson(job_resources))
  logging.info('Created the job ' + job_uri)
  return job_uri


def _create_query_job(project, location, payload,
                      job_configuration_query_override, creds,
                      gcp_resources) -> str:
  """Create a new BigQuery query job


    Args:
        project: Project to launch the query job.
        location: location to launch the query job. For more details, see
          https://cloud.google.com/bigquery/docs/locations#specifying_your_location
        payload: A json serialized Job proto. For more details, see
          https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
        job_configuration_query_override: A json serialized
          JobConfigurationQuery proto. For more details, see
          https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
        creds: Google auth credential.
        gcp_resources: File path for storing `gcp_resources` output parameter.

   Returns:
        The URI of the BigQuery Job.
  """
  job_request_json = json.loads(payload, strict=False)
  job_configuration_query_override_json = json_util.recursive_remove_empty(
      json.loads(job_configuration_query_override, strict=False))

  # Overrides json request with the value in job_configuration_query_override
  for key, value in job_configuration_query_override_json.items():
    job_request_json['configuration']['query'][key] = value

  # Remove empty fields before setting standard SQL.
  job_request_json = json_util.recursive_remove_empty(job_request_json)

  # Always uses standard SQL instead of legacy SQL.
  if 'useLegacySql' in job_request_json['configuration'][
      'query'] and job_request_json['configuration']['query'][
          'useLegacySql'] == True:
    raise ValueError('Legacy SQL is not supported. Use standard SQL instead.')
  job_request_json['configuration']['query']['useLegacySql'] = False

  return _create_job(project, location, job_request_json, creds, gcp_resources)


def _poll_job(job_uri, creds) -> dict:
  """Poll the bigquery job till it reaches a final state."""
  with execution_context.ExecutionContext(
      on_cancel=lambda: _send_cancel_request(job_uri, creds)):
    job = {}
    while ('status' not in job) or ('state' not in job['status']) or (
        job['status']['state'].lower() != 'done'):
      time.sleep(_POLLING_INTERVAL_IN_SECONDS)
      logging.info('The job is running...')
      if not creds.valid:
        creds.refresh(google.auth.transport.requests.Request())
      headers = {
          'Content-type': 'application/json',
          'Authorization': 'Bearer ' + creds.token
      }
      job = requests.get(job_uri, headers=headers).json()
      if 'status' in job and 'errorResult' in job['status']:
        raise RuntimeError('The BigQuery job failed. Error: {}'.format(
            job['status']))

  logging.info('BigQuery Job completed successfully. Job: %s.', job)
  return job


def _send_cancel_request(job_uri, creds):
  if not creds.valid:
    creds.refresh(google.auth.transport.requests.Request())
  headers = {
      'Content-type': 'application/json',
      'Authorization': 'Bearer ' + creds.token,
  }
  # Bigquery cancel API:
  # https://cloud.google.com/bigquery/docs/reference/rest/v2/jobs/cancel
  response = requests.post(
      url=f'{job_uri.split("?")[0]}/cancel',
      data='', headers=headers)
  logging.info('Cancel response: %s', response)


def bigquery_query_job(
    type,
    project,
    location,
    payload,
    job_configuration_query_override,
    gcp_resources,
    executor_input,
):
  """Create and poll bigquery job status till it reaches a final state.

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
      type: BigQuery job type.
      project: Project to launch the query job.
      location: location to launch the query job. For more details, see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      payload: A json serialized Job proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
      job_configuration_query_override: A json serialized JobConfigurationQuery
        proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      gcp_resources: File path for storing `gcp_resources` output parameter.
      executor_input: A json serialized pipeline executor input.
  """
  creds, _ = google.auth.default()
  job_uri = _check_if_job_exists(gcp_resources)
  if job_uri is None:
    job_uri = _create_query_job(project, location, payload,
                                job_configuration_query_override, creds,
                                gcp_resources)

  # Poll bigquery job status until finished.
  job = _poll_job(job_uri, creds)

  # write destination_table output artifact
  if 'destinationTable' in job['configuration']['query']:
    projectId = job['configuration']['query']['destinationTable']['projectId']
    datasetId = job['configuration']['query']['destinationTable']['datasetId']
    tableId = job['configuration']['query']['destinationTable']['tableId']
    bq_table_artifact = BQTable(
        'destination_table',
        projectId, datasetId, tableId)
    artifact_util.update_gcp_output_artifact(executor_input, bq_table_artifact)


def bigquery_create_model_job(
    type,
    project,
    location,
    payload,
    job_configuration_query_override,
    gcp_resources,
    executor_input,
):
  """Create and poll bigquery job status till it reaches a final state.

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
      type: BigQuery job type.
      project: Project to launch the query job.
      location: location to launch the query job. For more details, see
        https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      payload: A json serialized Job proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
      job_configuration_query_override: A json serialized JobConfigurationQuery
        proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job#JobConfigurationQuery
      gcp_resources: File path for storing `gcp_resources` output parameter.
      executor_input: A json serialized pipeline executor input.
  """
  creds, _ = google.auth.default()
  job_uri = _check_if_job_exists(gcp_resources)
  if job_uri is None:
    job_uri = _create_query_job(project, location, payload,
                                job_configuration_query_override, creds,
                                gcp_resources)

  # Poll bigquery job status until finished.
  job = _poll_job(job_uri, creds)

  if 'statistics' not in job or 'query' not in job['statistics']:
    raise RuntimeError('Unexpected create model job: {}'.format(job))

  query_result = job['statistics']['query']

  if 'statementType' not in query_result or query_result[
      'statementType'] != 'CREATE_MODEL' or 'ddlTargetTable' not in query_result:
    raise RuntimeError(
        'Unexpected create model result: {}'.format(query_result))

  projectId = query_result['ddlTargetTable']['projectId']
  datasetId = query_result['ddlTargetTable']['datasetId']
  # tableId is the model ID
  modelId = query_result['ddlTargetTable']['tableId']
  bqml_model_artifact = BQMLModel(
      'model',
      projectId, datasetId, modelId)
  artifact_util.update_gcp_output_artifact(executor_input, bqml_model_artifact)


def bigquery_predict_model_job(
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
  """Create and poll bigquery predict model job till it reaches a final state.

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
      executor_input: A json serialized pipeline executor input.
  """
  if not (not query_statement) ^ (not table_name):
    raise ValueError(
        'Only and Only one of query_statment and table_name should be '
        'populated for BigQuery predict model job.')
  input_data_sql = ('TABLE %s' % _back_quoted_if_needed(table_name)
                    if table_name else '(%s)' % query_statement)

  threshold_sql = ''
  if threshold is not None and threshold > 0.0 and threshold < 1.0:
    threshold_sql = ', STRUCT(%s AS threshold)' % threshold

  job_configuration_query_override_json = json.loads(
      job_configuration_query_override, strict=False)
  job_configuration_query_override_json[
      'query'] = 'SELECT * FROM ML.PREDICT(MODEL `%s`, %s%s)' % (
          model_name, input_data_sql, threshold_sql)

  # TODO(mingge): check if model is a valid BigQuery model resource.
  return bigquery_query_job(type, project, location, payload,
                            json.dumps(job_configuration_query_override_json),
                            gcp_resources, executor_input)


def _get_model(model_reference, creds):
  if not creds.valid:
    creds.refresh(google.auth.transport.requests.Request())
  headers = {
      'Content-type': 'application/json',
      'Authorization': 'Bearer ' + creds.token
  }
  model_uri = 'https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/datasets/{datasetId}/models/{modelId}'.format(
      projectId=model_reference['projectId'],
      datasetId=model_reference['datasetId'],
      modelId=model_reference['modelId'],
  )
  return requests.get(model_uri, headers=headers).json()


def bigquery_export_model_job(
    type,
    project,
    location,
    model_name,
    model_destination_path,
    payload,
    exported_model_path,
    gcp_resources,
    executor_input,
):
  """Create and poll bigquery export model job till it reaches a final state.

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
      project: Project to run BigQuery model export job.
      location: Location of the job to export the BigQuery model. If not set,
        default to `US` multi-region. For more details, see
          https://cloud.google.com/bigquery/docs/locations#specifying_your_location
      model_name: BigQuery ML model name to export.
      model_destination_path: The gcs bucket to export the model to.
      trial_id: The trial id if it's a hp tuned model. For more details, see
          https://cloud.google.com/bigquery-ml/docs/reference/standard-sql/bigqueryml-syntax-export-model
      payload: A json serialized Job proto. For more details, see
        https://cloud.google.com/bigquery/docs/reference/rest/v2/Job
      exported_model_path: File path for the `exported_model_path`
        output parameter.
      gcp_resources: File path for storing `gcp_resources` output parameter.
      executor_input:A json serialized pipeline executor input.
  """
  creds, _ = google.auth.default()
  job_uri = _check_if_job_exists(gcp_resources)
  if job_uri is None:
    # Post a job only if no existing job
    model_name_split = model_name.split('.')
    if len(model_name_split) != 3:
      raise ValueError(
          'The model name must be in the format "projectId.datasetId.modelId"')
    model_reference = {
        'projectId': model_name_split[0],
        'datasetId': model_name_split[1],
        'modelId': model_name_split[2],
    }

    model = _get_model(model_reference, creds)
    if not model or 'modelType' not in model:
      raise ValueError(
          'Cannot get model resource. The model name must be in the format "projectId.datasetId.modelId" '
      )

    job_request_json = json.loads(payload, strict=False)

    job_request_json['configuration']['extract'][
        'sourceModel'] = model_reference

    if model['modelType'].startswith('BOOSTED_TREE'):
      # Default format is ML_TF_SAVED_MODEL.
      job_request_json['configuration']['extract'][
          'destinationFormat'] = 'ML_XGBOOST_BOOSTER'

    job_request_json['configuration']['extract']['destinationUris'] = [
        model_destination_path
    ]

    job_uri = _create_job(project, location, job_request_json, creds,
                          gcp_resources)

  # Poll bigquery job status until finished.
  job = _poll_job(job_uri, creds)

  # write destination_table output parameter
  with open(exported_model_path, 'w') as f:
    f.write(job['configuration']['extract']['destinationUris'][0])


def _get_query_results(project_id, job_id, location, creds):
  if not creds.valid:
    creds.refresh(google.auth.transport.requests.Request())
  headers = {
      'Content-type': 'application/json',
      'Authorization': 'Bearer ' + creds.token
  }
  query_results_uri = 'https://bigquery.googleapis.com/bigquery/v2/projects/{projectId}/queries/{jobId}'.format(
      projectId=project_id,
      jobId=job_id,
  )
  if location:
    query_results_uri += '?location=' + location
  response = requests.get(query_results_uri, headers=headers).json()
  logging.info(response)
  return response


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
    raise ValueError('At most one of query_statment and table_name should be '
                     'populated for BigQuery evaluation model job.')

  input_data_sql = ''
  if table_name:
    input_data_sql = ', TABLE %s' % _back_quoted_if_needed(table_name)
  if query_statement:
    input_data_sql = ', (%s)' % query_statement

  threshold_sql = ''
  if threshold is not None and threshold > 0.0 and threshold < 1.0:
    threshold_sql = ', STRUCT(%s AS threshold)' % threshold

  job_configuration_query_override_json = json.loads(
      job_configuration_query_override, strict=False)
  job_configuration_query_override_json[
      'query'] = 'SELECT * FROM ML.EVALUATE(MODEL %s%s%s)' % (
          _back_quoted_if_needed(model_name), input_data_sql, threshold_sql)

  creds, _ = google.auth.default()
  job_uri = _check_if_job_exists(gcp_resources)
  if job_uri is None:
    job_uri = _create_query_job(
        project, location, payload,
        json.dumps(job_configuration_query_override_json), creds, gcp_resources)

  # Poll bigquery job status until finished.
  job = _poll_job(job_uri, creds)
  logging.info('Getting query result for job ' + job['id'])
  _, job_id = job['id'].split('.')
  query_results = _get_query_results(project, job_id, location, creds)
  artifact_util.update_output_artifact(
      executor_input, 'evaluation_metrics', '', {
          _ARTIFACT_PROPERTY_KEY_SCHEMA:
              query_results[_ARTIFACT_PROPERTY_KEY_SCHEMA],
          _ARTIFACT_PROPERTY_KEY_ROWS:
              query_results[_ARTIFACT_PROPERTY_KEY_ROWS]
      })
