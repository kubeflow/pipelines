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
"""Text2SQL evaluation pipeline."""

from google_cloud_pipeline_components import _placeholders
from google_cloud_pipeline_components._implementation.model_evaluation.text2sql_evaluation.component import text2sql_evaluation as Text2SQLEvaluationOp
from google_cloud_pipeline_components._implementation.model_evaluation.text2sql_preprocess.component import text2sql_evaluation_preprocess as Text2SQLEvaluationPreprocessOp
from google_cloud_pipeline_components._implementation.model_evaluation.text2sql_validate_and_process.component import text2sql_evaluation_validate_and_process as Text2SQLEvaluationValidateAndProcessOp
from google_cloud_pipeline_components.types import artifact_types
import kfp


_PIPELINE_NAME = 'evaluation_llm_text2sql_pipeline'


@kfp.dsl.pipeline(name=_PIPELINE_NAME)
def evaluation_llm_text2sql_pipeline(
    model_name: str,
    evaluation_data_source_path: str,
    tables_metadata_path: str,
    prompt_template_path: str = '',
    sql_dialect: str = 'bigquery',
    evaluation_method: str = 'parser',
    project: str = _placeholders.PROJECT_ID_PLACEHOLDER,
    location: str = _placeholders.LOCATION_PLACEHOLDER,
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
):
  """The LLM Evaluation Text2SQL Pipeline.

  Args:
    model_name: The Model used to run text2sql evaluation. Must be a publisher
      model or a managed Model sharing the same ancestor location. Starting this
      job has no impact on any existing deployments of the Model and their
      resources. Supported model is publishers/google/models/text-bison.
    evaluation_data_source_path: Required. The path for json file containing
      text2sql evaluation input dataset, including natural language question,
      ground truth SQL / SQL results.
    tables_metadata_path: Required. The path for json file containing database
      metadata, including table names, schema fields.
    prompt_template_path: Required. The path for json file containing prompt
      template. Will provide default value if users do not sepecify.
    sql_dialect: Optional. SQL dialect type, e.g. bigquery, mysql, etc. Default
      value is bigquery.
    evaluation_method: Optional. Text2SQL evaluation method, value can be
      'parser', 'execution', 'all'. Default value is 'parser'.
    project: Optional. The GCP project that runs the pipeline components.
      Default value is the same project used to run the pipeline.
    location: Optional. The GCP region that runs the pipeline components.
      Default value is the same location used to run the pipeline.
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-highmem-16`. More details:
      https://cloud.google.com/compute/docs/machine-resource
    service_account: Service account to run the dataflow job. If not set,
      dataflow will use the default worker service account. For more details,
      see
      https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
    network: Dataflow's fully qualified subnetwork name, when empty the default
      subnetwork will be used. More details:
      https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.
  """

  get_vertex_model_task = kfp.dsl.importer(
      artifact_uri=(
          f'https://{location}-aiplatform.googleapis.com/v1/{model_name}'
      ),
      artifact_class=artifact_types.VertexModel,
      metadata={'resourceName': model_name},
  )
  get_vertex_model_task.set_display_name('get-vertex-model')

  _ = Text2SQLEvaluationPreprocessOp(
      project=project,
      location=location,
      evaluation_data_source_path=evaluation_data_source_path,
      tables_metadata_path=tables_metadata_path,
      prompt_template_path=prompt_template_path,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  _ = Text2SQLEvaluationValidateAndProcessOp(
      project=project,
      location=location,
      # TODO(bozhengbz) Add value to model_inference_results_path
      # when model batch prediction component is added.
      model_inference_results_path='gs://test/model_inference_results.json',
      tables_metadata_path=tables_metadata_path,
      prompt_template_path=prompt_template_path,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  _ = Text2SQLEvaluationOp(
      project=project,
      location=location,
      sql_dialect=sql_dialect,
      evaluation_method=evaluation_method,
      # TODO(bozhengbz) Add value to model_inference_results_path
      # when model batch prediction component is added.
      model_inference_results_path='gs://test/model_inference_results.json',
      tables_metadata_path=tables_metadata_path,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )
