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

"""AutoML Forecasting Ensemble component spec."""

from typing import Optional

from google_cloud_pipeline_components import utils
from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Input
from kfp.dsl import Output


# pylint: disable=g-bare-generic,g-doc-args,unused-argument
@dsl.container_component
def automl_forecasting_ensemble(
    project: str,
    location: str,
    root_dir: str,
    transform_output: Input[Artifact],
    metadata: Input[Artifact],
    tuning_result_input: Input[Artifact],
    instance_baseline: Input[Artifact],
    instance_schema_path: Input[Artifact],
    prediction_image_uri: str,
    gcp_resources: dsl.OutputPath(str),
    model_architecture: Output[Artifact],
    example_instance: Output[Artifact],
    unmanaged_container_model: Output[UnmanagedContainerModel],
    explanation_metadata: dsl.OutputPath(dict),
    explanation_metadata_artifact: Output[Artifact],
    explanation_parameters: dsl.OutputPath(dict),
    encryption_spec_key_name: Optional[str] = '',
):
  # fmt: off
  """Ensembles AutoML Forecasting models.

  Args:
    project: Project to run the job in.
    location: Region to run the job in.
    root_dir: The Cloud Storage path to store the output.
    transform_output: The transform output artifact.
    metadata: The tabular example gen metadata.
    tuning_result_input: AutoML Tabular tuning result.
    instance_baseline: The instance baseline used to calculate explanations.
    instance_schema_path: The path to the instance schema, describing the input data for the tf_model at serving time.
    encryption_spec_key_name: Customer-managed encryption key.
    prediction_image_uri: URI of the Docker image to be used as the container for serving predictions. This URI must identify an image in Artifact Registry or Container Registry.

  Returns:
    gcp_resources: GCP resources created by this component. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
    model_architecture: The architecture of the output model.
    unmanaged_container_model: Model information needed to perform batch prediction.
    explanation_metadata: The explanation metadata used by Vertex online and batch explanations.
    explanation_metadata_artifact: The explanation metadata used by Vertex online and batch explanations in the format of a KFP Artifact.
    explanation_parameters: The explanation parameters used by Vertex online and batch explanations.
    example_instance: An example instance which may be used as an input for predictions.
  """
  # fmt: on
  job_id = dsl.PIPELINE_JOB_ID_PLACEHOLDER
  task_id = dsl.PIPELINE_TASK_ID_PLACEHOLDER
  image_uri = 'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/forecasting-training:20231002_0125'
  display_name = f'automl-forecasting-ensemble-{job_id}-{task_id}'

  error_file_path = f'{root_dir}/{job_id}/{task_id}/error.pb'
  model_relative_path = f'{job_id}/{task_id}/model'
  explanation_metadata_paths = (
      f'{explanation_metadata},{explanation_metadata_artifact.uri}'
  )

  job_args = [
      'forecasting_mp_ensemble',
      f'--transform_output_path={transform_output.uri}',
      f'--error_file_path={error_file_path}',
      f'--metadata_path={metadata.uri}',
      f'--tuning_result_input_path={tuning_result_input.uri}',
      f'--instance_baseline_path={instance_baseline.uri}',
      f'--instance_schema_path={instance_schema_path.uri}',
      f'--prediction_docker_uri={prediction_image_uri}',
      f'--model_relative_output_path={model_relative_path}',
      f'--explanation_metadata_path={explanation_metadata_paths}',
      f'--explanation_parameters_path={explanation_parameters}',
      f'--model_architecture_path={model_architecture.uri}',
      f'--example_instance_path={example_instance.uri}',
      '--use_json=true',
      '--executor_input={{$.json_escape[1]}}',
  ]

  payload = {
      'display_name': display_name,
      'encryption_spec': {
          'kms_key_name': encryption_spec_key_name,
      },
      'job_spec': {
          'worker_pool_specs': [{
              'replica_count': 1,
              'machine_spec': {
                  'machine_type': 'n1-highmem-8',
              },
              'container_spec': {
                  'image_uri': f'{image_uri}',
                  'args': job_args,
              },
          }]
      },
  }

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
          utils.container_component_dumps(payload),
      ],
  )
