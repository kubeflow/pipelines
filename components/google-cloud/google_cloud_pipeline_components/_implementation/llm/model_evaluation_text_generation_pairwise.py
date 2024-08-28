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
    model_a_evaluation_path: dsl.OutputPath(str),  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    model_b_evaluation_path: dsl.OutputPath(str),  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    evaluation_count_path: dsl.OutputPath(int),  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    evaluation_dataset_path: dsl.OutputPath(str),  # pylint: disable=unused-argument # pytype: disable=unsupported-operands
    human_preference_column: str = '',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    encryption_spec_key_name: str = '',
    model_a: str = '',
    model_b: str = '',
    evaluation_dataset: str = '',
    evaluation_dataset_metadata: str = '',  # pylint: disable=unused-argument
    task: str = '',
) -> dsl.ContainerSpec:  # pylint: disable=g-doc-args
  """Compute AutoSXS metrics using judgments outputs from Arbiter.

  Args:
    judgments_dir: Path to store the Judgments.
    human_preference_column: The column containing ground truths. The default
      value is an empty string if not be provided by users.
    project: Project to upload evaluation metrics to.
    location: Location to upload evaluation metrics to.
    encryption_spec_key_name: Customer-managed encryption key options. If this
      is set, then all resources created by the component will be encrypted with
      the provided encryption key.
    model_a: Resource path for Model A.
    model_b: Resource path for Model B.
    evaluation_dataset: Path to the evaluation dataset.
    evaluation_dataset_metadata: AutoSxS metrics metadata json string.
    task: Task that was used for this AutoSxS run.

  Returns:
    autosxs_metrics: Autosxs win rate metrics and human alignment metrics.
    gcp_resources: Tracker for GCP resources created by this component.
    model_a_evaluation_path: Path to write the ModelEvaluation for Model A if it
    is a
      ModelRegistry model.
    model_b_evaluation: Path to write the ModelEvaluation for Model B if it is a
      ModelRegistry model.
    evaluation_count: Path to write the EvaluationCount number to.
    evaluation_dataset_path: Path to write the path to the evaluation dataset.
      This is needed because Pipeline outputs must be component outputs.
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
              f'--model_a={model_a}',
              f'--model_b={model_b}',
              f'--model_a_evaluation_path={model_a_evaluation_path}',
              f'--model_b_evaluation_path={model_b_evaluation_path}',
              f'--evaluation_count_path={evaluation_count_path}',
              f'--evaluation_dataset_path={evaluation_dataset_path}',
              f'--evaluation_dataset={evaluation_dataset}',
              "--evaluation_dataset_metadata={{$.inputs.parameters['evaluation_dataset_metadata'].json_escape[0]}}",
              f'--task={task}',
          ],
          encryption_spec_key_name=encryption_spec_key_name,
      ),
      gcp_resources=gcp_resources,
  )
