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
"""Feature Store grounding pipeline."""

from google_cloud_pipeline_components._implementation.model_evaluation.chunking.component import chunking as ChunkingOp
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from google_cloud_pipeline_components.v1 import bigquery
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
import kfp

_PIPELINE_NAME = 'feature-store-grounding-pipeline'


@kfp.dsl.pipeline(name=_PIPELINE_NAME)
def feature_store_grounding_pipeline(
    project: str,
    location: str,
    input_text_gcs_dir: str,
    bigquery_bp_input_uri: str,
    bigquery_bp_output_uri: str,
    output_text_gcs_dir: str,
    output_error_file_path: str,
    model_name: str,
    generation_threshold_microseconds: str = '0',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    encryption_spec_key_name: str = '',
):
  """The Feature Store grounding pipeline.

  Args:
    project: Required. The GCP project that runs the pipeline components.
    location: Required. The GCP region that runs the pipeline components.
    input_text_gcs_dir: the GCS directory containing the files to chunk.
    bigquery_bp_input_uri: The URI to a bigquery table as the input for the
      batch prediction component. The chunking component will populate data to
      this uri first before batch prediction.
    bigquery_bp_output_uri: The URI to a bigquery table as the output for the
      batch prediction component.
    output_text_gcs_dir: The GCS folder to hold intermediate data for chunking.
    output_error_file_path: The path to the file containing chunking error.
    model_name: The path for model to generate embeddings, example,
      'publishers/google/models/textembedding-gecko@latest'
    generation_threshold_microseconds: only files created on/after this
      generation threshold will be processed, in microseconds.
    machine_type: The machine type to run chunking component in the pipeline.
    service_account: Service account to run the pipeline.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.
  """

  get_vertex_model_task = kfp.dsl.importer(
      artifact_uri=(
          f'https://{location}-aiplatform.googleapis.com/v1/{model_name}'
      ),
      artifact_class=VertexModel,
      metadata={'resourceName': model_name},
  )
  get_vertex_model_task.set_display_name('get-vertex-model')

  chunking_task = ChunkingOp(
      project=project,
      location=location,
      input_text_gcs_dir=input_text_gcs_dir,
      output_bq_destination=bigquery_bp_input_uri,
      output_text_gcs_dir=output_text_gcs_dir,
      output_error_file_path=output_error_file_path,
      generation_threshold_microseconds=generation_threshold_microseconds,
      machine_type=machine_type,
      service_account=service_account,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  batch_predict_task = ModelBatchPredictOp(
      job_display_name='feature-store-grounding-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      project=project,
      location=location,
      model=get_vertex_model_task.outputs['artifact'],
      bigquery_source_input_uri=bigquery_bp_input_uri,
      bigquery_destination_output_uri=bigquery_bp_output_uri,
      service_account=service_account,
      encryption_spec_key_name=encryption_spec_key_name,
  )
  batch_predict_task.after(chunking_task)
