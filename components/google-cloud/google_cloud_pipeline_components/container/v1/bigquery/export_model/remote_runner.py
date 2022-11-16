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
      type: BigQuery export model job type.
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
      exported_model_path: File path for the `exported_model_path` output
        parameter.
      gcp_resources: File path for storing `gcp_resources` output parameter.
      executor_input:A json serialized pipeline executor input.
  """
  creds, _ = google.auth.default()
  job_uri = bigquery_util.check_if_job_exists(gcp_resources)
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

    job_request_json['configuration']['query'][
        'query'] = f'EXPORT MODEL {bigquery_util.back_quoted_if_needed(model_name)} OPTIONS(URI="{model_destination_path}",add_serving_default_signature={True})'
    job_request_json['configuration']['query']['useLegacySql'] = False
    job_uri = bigquery_util.create_job(project, location, job_request_json,
                                       creds, gcp_resources)

  # Poll bigquery job status until finished.
  job = bigquery_util.poll_job(job_uri, creds)

  with open(exported_model_path, 'w') as f:
    f.write(model_destination_path)
