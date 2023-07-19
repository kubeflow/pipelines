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

"""AutoML Tabular Lanistr trainer component spec."""

from typing import Optional

from kfp import dsl


@dsl.container_component
def lanistr_trainer(
    project: str,
    location: str,
    root_dir: str,
    transform_output: dsl.Input[dsl.Artifact],
    materialized_data: dsl.Input[dsl.Artifact],
    gcp_resources: dsl.OutputPath(str),
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """AutoML Tabular Lanistr component.

  Args:
      project: Project to run Cross-validation trainer.
      location: Location for running the Cross-validation trainer.
      root_dir: The Cloud Storage location to store the output.
      transform_output: The transform output artifact.
      materialized_data: The materialized data output by FTE.
      encryption_spec_key_name: Customer-managed encryption key.

  Returns:
      gcp_resources: GCP resources created by this component. For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
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
                      f' "lanistr-trainer-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  (
                      '"}, "job_spec": {"worker_pool_specs": [{"replica_count":'
                      ' 1, "machine_spec": {"machine_type": "n1-standard-8"},'
                      ' "container_spec": {"image_uri":"'
                  ),
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/lanistr-training:dev',
                  '", "args": ["lanistr_trainer", "--transform_output_path=',
                  transform_output.uri,
                  '", "--materialized_data=',
                  materialized_data.uri,
                  '", "--training_docker_uri=',
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/lanistr-training:dev',
                  '", "--training_base_dir=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/train",'
                      ' "--gcp_resources_path='
                  ),
                  gcp_resources,
                  '"]}}]}}',
              ]
          ),
      ],
  )
