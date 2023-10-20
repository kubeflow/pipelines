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

"""KFP Container component for preprocessing predictions for the Arbiter."""

import os

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image() -> str:
  return (
      os.environ.get('AUTOSXS_IMAGE_OVERRIDE')
      or utils.get_default_image_uri('autosxs'))


@dsl.container_component
def arbiter_preprocess(
    prediction_uris: str,
    prompt_column: str,
    response_column: str,
    output_path: str,
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Preprocesses predictions tables for the AutoSxS Arbiter.

  Args:
    prediction_uris: A list of GCS or BigQuery URIs representing a dataset of
      prompts and responses.
    prompt_column: The column containing prompts.
    response_column: The column containing responses.
    output_path: Path to write the path where preprocessed predictions are
      stored.

  Returns:
    gcp_resources: Tracker for GCP resources created by this component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      location=_placeholders.LOCATION_PLACEHOLDER,
      custom_job_payload=utils.build_payload(
          display_name='arbiter_preprocess',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'arbiter_preprocess',
              f'--prediction_uris={prediction_uris}',
              f'--staging_dir={dsl.PIPELINE_ROOT_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
              f'--prompt_column={prompt_column}',
              f'--response_column={response_column}',
              f'--output_path={output_path}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
