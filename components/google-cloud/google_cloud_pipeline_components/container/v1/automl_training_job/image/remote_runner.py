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
"""GCP remote runner for AutoML image training pipelines based on the AI Platform SDK."""

import json
import logging
from typing import Any, Dict, Optional, Sequence

from google.api_core import retry
from google.cloud.aiplatform import datasets
from google.cloud.aiplatform import gapic
from google.cloud.aiplatform import initializer
from google.cloud.aiplatform import models
from google.cloud.aiplatform import schema
from google.cloud.aiplatform import training_jobs
from google.cloud.aiplatform_v1.types import model
from google.cloud.aiplatform_v1.types import training_pipeline
from google_cloud_pipeline_components.container.v1.aiplatform import remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher import pipeline_remote_runner
from google_cloud_pipeline_components.container.v1.gcp_launcher.utils import error_util

from google.protobuf import struct_pb2
from google.protobuf import json_format


_GET_PIPELINE_RETRY_DEADLINE_SECONDS = 10.0 * 60.0
_CLASSIFICATION = 'classification'
_OBJECT_DETECTION = 'object_detection'


# pylint: disable=protected-access
def create_payload(
    project: str,
    location: str,
    display_name: Optional[str] = None,
    prediction_type: str = _CLASSIFICATION,
    multi_label: bool = False,
    model_type: str = 'CLOUD',
    labels: Optional[Dict[str, str]] = None,
    dataset: Optional[str] = None,
    disable_early_stopping: bool = False,
    training_encryption_spec_key_name: Optional[str] = None,
    model_encryption_spec_key_name: Optional[str] = None,
    model_display_name: Optional[str] = None,
    training_fraction_split: Optional[float] = None,
    validation_fraction_split: Optional[float] = None,
    test_fraction_split: Optional[float] = None,
    budget_milli_node_hours: Optional[int] = None,
    training_filter_split: Optional[str] = None,
    validation_filter_split: Optional[str] = None,
    test_filter_split: Optional[str] = None,
    base_model: Optional[str] = None,
    incremental_train_base_model: Optional[str] = None,
    parent_model: Optional[str] = None,
    is_default_version: Optional[bool] = True,
    model_version_aliases: Optional[Sequence[str]] = None,
    model_version_description: Optional[str] = None,
    model_labels: Optional[Dict[str, str]] = None,
) -> str:
  """Creates a AutoML Image Training Job payload."""
  # Override default model_type for object_detection
  if model_type == 'CLOUD' and prediction_type == _OBJECT_DETECTION:
    model_type = 'CLOUD_HIGH_ACCURACY_1'

  training_encryption_spec = initializer.global_config.get_encryption_spec(
      encryption_spec_key_name=training_encryption_spec_key_name
  )
  model_encryption_spec = initializer.global_config.get_encryption_spec(
      encryption_spec_key_name=model_encryption_spec_key_name
  )

  # Training task inputs.
  training_task_inputs = {
      # required inputs
      'modelType': model_type,
      'budgetMilliNodeHours': budget_milli_node_hours,
      # optional inputs
      'disableEarlyStopping': disable_early_stopping,
  }
  if prediction_type == _CLASSIFICATION:
    training_task_inputs['multiLabel'] = multi_label
  if incremental_train_base_model:
    training_task_inputs['uptrainBaseModelId'] = incremental_train_base_model

  training_task_definition = getattr(
      schema.training_job.definition, f'automl_image_{prediction_type}'
  )

  # Input data config.
  input_data_config = training_jobs._TrainingJob._create_input_data_config(
      dataset=dataset and datasets.ImageDataset(dataset_name=dataset),
      training_fraction_split=training_fraction_split,
      validation_fraction_split=validation_fraction_split,
      test_fraction_split=test_fraction_split,
      training_filter_split=training_filter_split,
      validation_filter_split=validation_filter_split,
      test_filter_split=test_filter_split,
  )

  # Model to upload.
  model_to_upload = model.Model(
      display_name=model_display_name or display_name,
      labels=model_labels or labels,
      encryption_spec=model_encryption_spec,
      version_aliases=models.ModelRegistry._get_true_alias_list(
          model_version_aliases, is_default_version
      ),
      version_description=model_version_description,
  )

  # Sets base_model.
  if base_model:
    training_task_inputs['baseModelId'] = base_model

  # Create training task inputs.
  training_task_inputs_struct = struct_pb2.Struct()
  training_task_inputs_struct.update(training_task_inputs)

  # Gets parent_model.
  parent_model = models.ModelRegistry._get_true_version_parent(
      parent_model=parent_model,
      project=project,
      location=location,
  )

  pipeline = training_pipeline.TrainingPipeline(
      display_name=display_name,
      training_task_definition=training_task_definition,
      training_task_inputs=struct_pb2.Value(
          struct_value=training_task_inputs_struct
      ),
      model_to_upload=model_to_upload,
      parent_model=parent_model,
      input_data_config=input_data_config,
      labels=labels,
      encryption_spec=training_encryption_spec,
  )

  return json_format.MessageToJson(
      pipeline._pb, preserving_proto_field_name=True
  )


# pylint: enable=protected-access


def create_pipeline_with_client(
    pipeline_client: gapic.PipelineServiceClient,
    parent,
    pipeline_spec: Any,
):
  """Creates a training pipeline with the client."""
  created_pipeline = None
  try:
    logging.info(
        'Creating AutoML Vision training pipeline with sanitized pipeline'
        ' spec: %s',
        pipeline_spec,
    )
    created_pipeline = pipeline_client.create_training_pipeline(
        parent=parent,
        training_pipeline=training_pipeline.TrainingPipeline(**pipeline_spec),
    )
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return created_pipeline


def get_pipeline_with_client(
    pipeline_client: gapic.PipelineServiceClient, pipeline_name: str
):
  """Gets training pipeline state with the client."""
  get_automl_vision_training_pipeline = None
  try:
    get_automl_vision_training_pipeline = pipeline_client.get_training_pipeline(
        name=pipeline_name,
        retry=retry.Retry(deadline=_GET_PIPELINE_RETRY_DEADLINE_SECONDS),
    )
  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
  return get_automl_vision_training_pipeline


def create_pipeline(
    type: str,  # pylint: disable=redefined-builtin
    project: str,
    location: str,
    gcp_resources: str,
    executor_input: str,
    **kwargs: Dict[str, Any],
):
  """Create and poll AutoML Vision training pipeline status till it reaches a final state.

  This follows the typical launching logic:
  1. Read if the training pipeline already exists in gcp_resources
     - If already exists, jump to step 3 and poll the pipeline status. This
     happens
     if the launcher container experienced unexpected termination, such as
     preemption
  2. Deserialize the payload into the pipeline spec and create the training
  pipeline
  3. Poll the training pipeline status every
  pipeline_remote_runner._POLLING_INTERVAL_IN_SECONDS seconds
     - If the training pipeline is succeeded, return succeeded
     - If the training pipeline is cancelled/paused, it's an unexpected
     scenario so return failed
     - If the training pipeline is running, continue polling the status

  Also retry on ConnectionError up to
  pipeline_remote_runner._CONNECTION_ERROR_RETRY_LIMIT times during the poll.

  Args:
    type: Job type.
    project: Project name.
    location: Location to start the training job.
    gcp_resources: URI for storing GCP resources.
    executor_input: Pipeline executor input.
    **kwargs: Extra args for creating the payload.
  """
  runner = pipeline_remote_runner.PipelineRemoteRunner(
      type, project, location, gcp_resources
  )

  try:
    # Create AutoML vision training pipeline if it does not exist
    pipeline_name = runner.check_if_pipeline_exists()
    if pipeline_name is None:
      payload = create_payload(project, location, **kwargs)
      logging.info(
          'AutoML Vision training payload formatted: %s',
          payload,
      )
      pipeline_name = runner.create_pipeline(
          create_pipeline_with_client,
          payload,
      )

    # Poll AutoML Vision training pipeline status until
    # "PipelineState.PIPELINE_STATE_SUCCEEDED"
    pipeline = runner.poll_pipeline(get_pipeline_with_client, pipeline_name)

  except (ConnectionError, RuntimeError) as err:
    error_util.exit_with_internal_error(err.args[0])
    return  # No-op, suppressing uninitialized `pipeline` variable lint error.

  # Writes artifact output on success.
  if not isinstance(pipeline, training_pipeline.TrainingPipeline):
    raise ValueError('Internal error: no training pipeline was created.')
  remote_runner.write_to_artifact(
      json.loads(executor_input),
      pipeline.model_to_upload.name,
  )
