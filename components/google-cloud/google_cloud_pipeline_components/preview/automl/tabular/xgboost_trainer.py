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

"""AutoML XGBoost Trainer component spec."""

from typing import Optional

from kfp import dsl


@dsl.container_component
def xgboost_trainer(
    project: str,
    location: str,
    worker_pool_specs: list,
    gcp_resources: dsl.OutputPath(str),
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Trains an XGBoost model using Vertex CustomJob API.

  Args:
      project: The GCP project that runs the pipeline components.
      location: The GCP region that runs the pipeline components.
      worker_pool_specs: The worker pool specs.
      encryption_spec_key_name: The KMS key name.

  Returns:
      gcp_resources: Serialized gcp_resources proto tracking the custom training job.
  """
  # fmt: on

  return dsl.ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:1.0.44',
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.custom_job.launcher',
      ],
      args=[
          '--type',
          'CustomJob',
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
          '--payload',
          dsl.ConcatPlaceholder(
              items=[
                  (
                      '{"display_name":'
                      f' "xgboost-trainer-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  '"}, "job_spec": {"worker_pool_specs": ',
                  worker_pool_specs,
                  '}}',
              ]
          ),
      ],
  )
