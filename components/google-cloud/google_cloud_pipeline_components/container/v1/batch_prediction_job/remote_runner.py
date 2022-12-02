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
"""GCP remote runner for batch prediction jobs based on the AI Platform SDK."""

import json
import re

from google.api_core import retry
from google.cloud.aiplatform import explain
from google_cloud_pipeline_components.container.v1.gcp_launcher import job_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import artifact_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import gcp_labels_util
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import json_util
from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import VertexBatchPredictionJob
from kfp.v2 import dsl

UNMANAGED_CONTAINER_MODEL_ARTIFACT_NAME = 'unmanaged_container_model'
LABELS_PAYLOAD_KEY = 'labels'
_BATCH_PREDICTION_RETRY_DEADLINE_SECONDS = 10.0 * 60.0
_BQ_DATASET_TEMPLATE = r'(bq://(?P<project>.*)\.(?P<dataset>.*))'


def sanitize_job_spec(job_spec):
  """If the job_spec contains explanation metadata, convert to ExplanationMetadata for the job client to recognize."""
  if ('explanation_spec' in job_spec) and ('metadata'
                                           in job_spec['explanation_spec']):
    job_spec['explanation_spec'][
        'metadata'] = explain.ExplanationMetadata.from_json(
            json.dumps(job_spec['explanation_spec']['metadata']))
  return job_spec


def create_batch_prediction_job_with_client(job_client, parent, job_spec):
  job_spec = sanitize_job_spec(job_spec)
  create_batch_prediction_job_fn = None
  try:
    create_batch_prediction_job_fn = job_client.create_batch_prediction_job(
        parent=parent, batch_prediction_job=job_spec)
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return create_batch_prediction_job_fn


def get_batch_prediction_job_with_client(job_client, job_name):
  get_batch_prediction_job_fn = None
  try:
    get_batch_prediction_job_fn = job_client.get_batch_prediction_job(
        name=job_name,
        retry=retry.Retry(deadline=_BATCH_PREDICTION_RETRY_DEADLINE_SECONDS))
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return get_batch_prediction_job_fn


def insert_artifact_into_payload(executor_input, payload):
  job_spec = json.loads(payload)
  artifact = json.loads(executor_input).get('inputs', {}).get(
      'artifacts', {}).get(UNMANAGED_CONTAINER_MODEL_ARTIFACT_NAME,
                           {}).get('artifacts')
  if artifact:
    job_spec[
        UNMANAGED_CONTAINER_MODEL_ARTIFACT_NAME] = json_util.camel_case_to_snake_case_recursive(
            artifact[0].get('metadata', {}))
    job_spec[UNMANAGED_CONTAINER_MODEL_ARTIFACT_NAME][
        'artifact_uri'] = artifact[0].get('uri')
  return json.dumps(job_spec)

def insert_system_labels_into_payload(payload):
  job_spec = json.loads(payload)
  job_spec[LABELS_PAYLOAD_KEY] = gcp_labels_util.attach_system_labels(
      job_spec[LABELS_PAYLOAD_KEY] if LABELS_PAYLOAD_KEY in job_spec else {})
  return json.dumps(job_spec)

def create_batch_prediction_job(
    type,
    project,
    location,
    payload,
    gcp_resources,
    executor_input,
):
  """Create and poll batch prediction job status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the batch prediction job already exists in gcp_resources
     - If already exists, jump to step 3 and poll the job status. This happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the payload into the job spec and create the batch prediction
  job
  3. Poll the batch prediction job status every
  job_remote_runner._POLLING_INTERVAL_IN_SECONDS seconds
     - If the batch prediction job is succeeded, return succeeded
     - If the batch prediction job is cancelled/paused, it's an unexpected
     scenario so return failed
     - If the batch prediction job is running, continue polling the status

  Also retry on ConnectionError up to
  job_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.
  """
  remote_runner = job_remote_runner.JobRemoteRunner(type, project, location,
                                                    gcp_resources)

  try:
    # Create batch prediction job if it does not exist
    job_name = remote_runner.check_if_job_exists()
    if job_name is None:
      job_name = remote_runner.create_job(
          create_batch_prediction_job_with_client,
          insert_system_labels_into_payload(
              insert_artifact_into_payload(executor_input, payload)))

    # Poll batch prediction job status until "JobState.JOB_STATE_SUCCEEDED"
    get_job_response = remote_runner.poll_job(
        get_batch_prediction_job_with_client, job_name)

    vertex_uri_prefix = f'https://{location}-aiplatform.googleapis.com/v1/'
    vertex_batch_predict_job_artifact = VertexBatchPredictionJob(
        'batchpredictionjob', vertex_uri_prefix + get_job_response.name,
        get_job_response.name,
        get_job_response.output_info.bigquery_output_table,
        get_job_response.output_info.bigquery_output_dataset,
        get_job_response.output_info.gcs_output_directory)
    output_artifacts = [vertex_batch_predict_job_artifact]

    # Output the BQTable artifact
    if get_job_response.output_info.bigquery_output_dataset:
      bq_dataset_pattern = re.compile(_BQ_DATASET_TEMPLATE)
      match = bq_dataset_pattern.match(
          get_job_response.output_info.bigquery_output_dataset)
      try:
        project = match.group('project')
        dataset = match.group('dataset')
        bigquery_output_table_artifact = BQTable(
            'bigquery_output_table', project, dataset,
            get_job_response.output_info.bigquery_output_table)
        output_artifacts.append(bigquery_output_table_artifact)
      except AttributeError as err:
        error_util.exit_with_internal_error(
            'Invalid BQ dataset address from batch prediction output: {}. Expect: {}.'
            .format(get_job_response.output_info.bigquery_output_dataset,
                    'bq://[project_id].[dataset_id]'))

    # Output the GCS path via system.Artifact
    if get_job_response.output_info.gcs_output_directory:
      output_artifacts.append(
          dsl.Artifact('gcs_output_directory',
                       get_job_response.output_info.gcs_output_directory))

    artifact_util.update_output_artifacts(executor_input, output_artifacts)
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
