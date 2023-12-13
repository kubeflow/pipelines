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
"""KFP Container component for computing AutoSXS metrics."""

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
def autosxs_metrics_computer(
    judgments_dir: str,
    has_human_preference: bool,
    autosxs_metrics: dsl.Output[dsl.Metrics],  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    gcp_resources: dsl.OutputPath(str),  # pytype: disable=invalid-annotation
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Compute AutoSXS metrics using judgments outputs from Arbiter.

  Args:
    judgments_dir: Path where store the Judgments.
    has_human_preference: Boolean value. True if users provided human preference
      data, otherwise false.

  Returns:
    autosxs_metrics: Autosxs win rate metrics and human alignment metrics.
    gcp_resources: Tracker for GCP resources created by this component.
  """
  return gcpc_utils.build_serverless_customjob_container_spec(
      project=_placeholders.PROJECT_ID_PLACEHOLDER,
      # Hardcode location to us-central1 for text-bison availability.
      location='us-central1',
      custom_job_payload=utils.build_payload(
          display_name='autosxs_metrics_computer',
          machine_type='n1-standard-4',
          image_uri=_resolve_image(),
          args=[
              '--',  # Used to mark the start of component flags.
              'autosxs_metrics',
              f'--judgments_dir={judgments_dir}',
              f'--has_human_preference={has_human_preference}',
              '--executor_input={{$.json_escape[1]}}',
          ],
      ),
      gcp_resources=gcp_resources,
  )
