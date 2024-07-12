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

"""AutoML Tabular Ensemble component spec."""

from typing import Optional

from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from kfp import dsl
from kfp.dsl import Artifact
from kfp.dsl import Dataset
from kfp.dsl import Input
from kfp.dsl import Output


@dsl.container_component
def automl_tabular_ensemble(
    project: str,
    location: str,
    root_dir: str,
    transform_output: Input[Artifact],
    metadata: Input[Artifact],
    dataset_schema: Input[Artifact],
    tuning_result_input: Input[Artifact],
    instance_baseline: Input[Artifact],
    gcp_resources: dsl.OutputPath(str),
    model_architecture: Output[Artifact],
    model: Output[Artifact],
    unmanaged_container_model: Output[UnmanagedContainerModel],
    model_without_custom_ops: Output[Artifact],
    explanation_metadata: dsl.OutputPath(dict),
    explanation_metadata_artifact: Output[Artifact],
    explanation_parameters: dsl.OutputPath(dict),
    warmup_data: Optional[Input[Dataset]] = None,
    encryption_spec_key_name: Optional[str] = '',
    export_additional_model_without_custom_ops: Optional[bool] = False,
):
  # fmt: off
  """Ensembles AutoML Tabular models.

  Args:
      project: Project to run Cross-validation trainer.
      location: Location for running the Cross-validation trainer.
      root_dir: The Cloud Storage location to store the output.
      transform_output: The transform output artifact.
      metadata: The tabular example gen metadata.
      dataset_schema: The schema of the dataset.
      tuning_result_input: AutoML Tabular tuning result.
      instance_baseline: The instance baseline used to calculate explanations.
      warmup_data: The warm up data. Ensemble component will save the warm up data together with the model artifact, used to warm up the model when prediction server starts.
      encryption_spec_key_name: Customer-managed encryption key.
      export_additional_model_without_custom_ops: True if export an additional model without custom TF operators to the `model_without_custom_ops` output.

  Returns:
      gcp_resources: GCP resources created by this component. For more details, see https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
      model_architecture: The architecture of the output model.
      model: The output model.
      model_without_custom_ops: The output model without custom TF operators, this output will be empty unless `export_additional_model_without_custom_ops` is set.
      model_uri: The URI of the output model.
      instance_schema_uri: The URI of the instance schema.
      prediction_schema_uri: The URI of the prediction schema.
      explanation_metadata: The explanation metadata used by Vertex online and batch explanations.
      explanation_metadata: The explanation parameters used by Vertex online and batch explanations.
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
                      f' "automl-tabular-ensemble-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}",'
                      ' "encryption_spec": {"kms_key_name":"'
                  ),
                  encryption_spec_key_name,
                  (
                      '"}, "job_spec": {"worker_pool_specs": [{"replica_count":'
                      ' 1, "machine_spec": {"machine_type": "n1-highmem-8"},'
                      ' "container_spec": {"image_uri":"'
                  ),
                  'us-docker.pkg.dev/vertex-ai-restricted/automl-tabular/training:20240710_0625',
                  '", "args": ["ensemble", "--transform_output_path=',
                  transform_output.uri,
                  '", "--model_output_path=',
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/model",'
                      ' "--custom_model_output_path='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/custom_model",'
                      ' "--error_file_path='
                  ),
                  root_dir,
                  (
                      f'/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}/{dsl.PIPELINE_TASK_ID_PLACEHOLDER}/error.pb",'
                      ' "--export_custom_model='
                  ),
                  export_additional_model_without_custom_ops,
                  '", "--metadata_path=',
                  metadata.uri,
                  '", "--dataset_schema_path=',
                  dataset_schema.uri,
                  '", "--tuning_result_input_path=',
                  tuning_result_input.uri,
                  '", "--instance_baseline_path=',
                  instance_baseline.uri,
                  '", "--warmup_data=',
                  warmup_data.uri,
                  '", "--prediction_docker_uri=',
                  'us-docker.pkg.dev/vertex-ai/automl-tabular/prediction-server:20240710_0625',
                  '", "--model_path=',
                  model.uri,
                  '", "--custom_model_path=',
                  model_without_custom_ops.uri,
                  '", "--explanation_metadata_path=',
                  explanation_metadata,
                  ',',
                  explanation_metadata_artifact.uri,
                  '", "--explanation_parameters_path=',
                  explanation_parameters,
                  '", "--model_architecture_path=',
                  model_architecture.uri,
                  (
                      '", "--use_json=true",'
                      ' "--executor_input={{$.json_escape[1]}}"]}}]}}'
                  ),
              ]
          ),
      ],
  )
