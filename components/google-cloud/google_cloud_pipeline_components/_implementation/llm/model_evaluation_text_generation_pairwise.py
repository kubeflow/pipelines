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
"""KFP Container component for computing aggregate pairwise metrics."""

import os

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components import utils as gcpc_utils
from google_cloud_pipeline_components._implementation.llm import utils
from kfp import dsl


def _resolve_image() -> str:
  """Determines the image URI to create a container from."""
  return os.environ.get(
      'AUTOSXS_IMAGE_OVERRIDE'
  ) or utils.get_default_image_uri('autosxs')


@dsl.container_component
def model_evaluation_text_generation_pairwise(
    judgments_dir: str,
    autosxs_metrics: dsl.Output[dsl.Metrics],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
    human_preference_column: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Compute AutoSXS metrics using judgments outputs from Arbiter.

  Args:
    judgments_dir: Path where store the Judgments.
    human_preference_column: The column containing ground truths. The default
      value is an empty string if not be provided by users.
    project: Project to upload evaluation metrics to.
    location: Location to upload evaluation metrics to.

  Returns:
    autosxs_metrics: Autosxs win rate metrics and human alignment metrics.
    gcp_resources: Tracker for GCP resources created by this component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=project,
      location=location,
      custom_job_payload=utils.build_payload(
          display_name='model_evaluation_text_generation_pairwise',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'autosxs_metrics',
              f'--judgments_dir={judgments_dir}',
              f'--human_preference_column={human_preference_column}',
              f'--project={project}',
              f'--location={location}',
              '--executor_input={{$.json_escape[1]}}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
