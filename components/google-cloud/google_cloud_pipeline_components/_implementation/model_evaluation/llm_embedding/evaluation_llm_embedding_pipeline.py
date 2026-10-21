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
"""LLM embedding evaluation pipeline based on information retrieval (IR) task."""

from typing import Dict, Optional, Union

from google_cloud_pipeline_components._implementation.model_evaluation.endpoint_batch_predict.component import evaluation_llm_endpoint_batch_predict_pipeline_graph_component as LLMEndpointBatchPredictOp
from google_cloud_pipeline_components._implementation.model_evaluation.llm_embedding_retrieval.component import llm_embedding_retrieval as LLMEmbeddingRetrievalOp
from google_cloud_pipeline_components._implementation.model_evaluation.llm_information_retrieval_preprocessor.component import llm_information_retrieval_preprocessor as LLMInformationRetrievalPreprocessorOp
from google_cloud_pipeline_components._implementation.model_evaluation.llm_retrieval_metrics.component import llm_retrieval_metrics as LLMRetrievalMetricsOp
from google_cloud_pipeline_components.preview.model_evaluation.model_evaluation_import_component import model_evaluation_import as ModelImportEvaluationOp
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
import kfp
from kfp.dsl import PIPELINE_ROOT_PLACEHOLDER


_PIPELINE_NAME = 'evaluation-llm-embedding-pipeline'


@kfp.dsl.pipeline(name=_PIPELINE_NAME)
def evaluation_llm_embedding_pipeline(
    project: str,
    location: str,
    corpus_gcs_source: str,
    query_gcs_source: str,
    golden_docs_gcs_source: str,
    model_name: str,
    qms_override: Optional[Dict[str, Union[int, float]]] = {},
    model_parameters: Optional[Dict[str, Union[int, float]]] = {},
    embedding_chunking_function: str = 'langchain-RecursiveCharacterTextSplitter',
    embedding_chunk_size: int = 0,
    embedding_chunk_overlap: int = 0,
    embedding_retrieval_combination_function: str = 'max',
    batch_predict_instances_format: str = 'jsonl',
    batch_predict_predictions_format: str = 'jsonl',
    embedding_retrieval_top_n: int = 10,
    retrieval_metrics_top_k_list: str = '10',
    machine_type: str = 'e2-highmem-16',
    service_account: str = '',
    network: str = '',
    runner: str = 'DirectRunner',
    dataflow_service_account: str = '',
    dataflow_disk_size_gb: int = 50,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
):
  """The LLM Embedding Evaluation Pipeline.

  Args:
    project: Required. The GCP project that runs the pipeline components.
    location: Required. The GCP region that runs the pipeline components.
    corpus_gcs_source: The gcs location for json file containing corpus
      documents.
    query_gcs_source: The gcs location for json file containing query documents.
    golden_docs_gcs_source: The gcs location for csv file containing mapping of
      each query to the golden docs.
    model_name: The path for model to generate embeddings, example,
      'publishers/google/models/textembedding-gecko-multilingual@latest'
    qms_override: Manual control of a large language model's qms. Write up when
      there's an approved quota increase for a LLM. Write down when limiting qms
      of a LLM for this pipeline. Should be provided as a dictionary, for
      example {'text-bison': 20}. For deployed model which doesn't have
      google-vertex-llm-tuning-base-model-id label, override the default here.
    model_parameters: The parameters that govern the prediction.
    embedding_chunking_function: function used to split a document into chunks.
      Supported values are `langchain-RecursiveCharacterTextSplitter` and
      `sentence-splitter`. langchain-RecursiveCharacterTextSplitter:
      langchain.text_splitter.RecursiveCharacterTextSplitter, with configurable
      chunk_size and chunk_overlap. sentence-splitter: splitter that will not
      break in the middle of a sentence. embedding_chunk_size and
      embedding_chunk_overlap are measured by the number of tokens.
    embedding_chunk_size: The length of each document chunk. If 0, chunking is
      not enabled.
    embedding_chunk_overlap: The length of the overlap part between adjacent
      chunks. Will only be used if embedding_chunk_size > 0.
    embedding_retrieval_combination_function: The function to combine
      query-chunk similarities to query-doc similarity. Supported functions are
      avg, max, and sum.
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. If not set,
      default to "jsonl".  For more details about this input config, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_instances_format: The format in which perdictions are made,
      must be one of the Model's supportedInputStorageFormats. If not set,
      default to "jsonl".  For more details about this input config, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    embedding_retrieval_top_n: Top N docs will be retrieved for each query,
      based on similarity.
    retrieval_metrics_top_k_list: k values for retrieval metrics, for example,
      precision@k, accuracy@k, etc. If more than one value, separated by comma.
      e.g., "1,5,10".
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
    runner: runner for the beam pipeline. DirectRunner and DataflowRunner are
      supported.
    dataflow_service_account: Service account to run the dataflow job. If not
      set, dataflow will use the default worker service account. For more
      details, see
      https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
    dataflow_disk_size_gb: The disk size (in GB) of the machine executing the
      evaluation run.
    dataflow_machine_type: The machine type executing the evaluation run.
    dataflow_workers_num: The number of workers executing the evaluation run.
    dataflow_max_workers_num: The max number of workers executing the evaluation
      run.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. More details:
      https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.
  """

  preprocessing_task = LLMInformationRetrievalPreprocessorOp(
      project=project,
      location=location,
      corpus_gcs_source=corpus_gcs_source,
      query_gcs_source=query_gcs_source,
      golden_docs_gcs_source=golden_docs_gcs_source,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      runner=runner,
      embedding_chunking_function=embedding_chunking_function,
      embedding_chunk_size=embedding_chunk_size,
      embedding_chunk_overlap=embedding_chunk_overlap,
      dataflow_service_account=dataflow_service_account,
      dataflow_disk_size_gb=dataflow_disk_size_gb,
      dataflow_machine_type=dataflow_machine_type,
      dataflow_workers_num=dataflow_workers_num,
      dataflow_max_workers_num=dataflow_max_workers_num,
      dataflow_subnetwork=dataflow_subnetwork,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  get_vertex_model_task = kfp.dsl.importer(
      artifact_uri=(
          f'https://{location}-aiplatform.googleapis.com/v1/{model_name}'
      ),
      artifact_class=VertexModel,
      metadata={'resourceName': model_name},
  )
  get_vertex_model_task.set_display_name('get-vertex-model')

  batch_predict_corpus = LLMEndpointBatchPredictOp(
      display_name=(
          'batch-prediction-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}'
      ),
      publisher_model=model_name,
      qms_override=qms_override,
      location=location,
      source_gcs_uris=preprocessing_task.outputs[
          'predictions_corpus_gcs_source'
      ],
      model_parameters=model_parameters,
      gcs_destination_output_uri_prefix=(
          f'{PIPELINE_ROOT_PLACEHOLDER}/batch_predict_output'
      ),
      service_account=service_account,
      encryption_spec_key_name=encryption_spec_key_name,
      project=project,
  )

  batch_predict_query = LLMEndpointBatchPredictOp(
      display_name=(
          'batch-prediction-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}'
      ),
      publisher_model=model_name,
      qms_override=qms_override,
      location=location,
      source_gcs_uris=preprocessing_task.outputs[
          'predictions_query_gcs_source'
      ],
      model_parameters=model_parameters,
      gcs_destination_output_uri_prefix=(
          f'{PIPELINE_ROOT_PLACEHOLDER}/batch_predict_output'
      ),
      service_account=service_account,
      encryption_spec_key_name=encryption_spec_key_name,
      project=project,
  )

  # batch_predict_corpus = ModelBatchPredictOp(
  #     project=project,
  #     location=location,
  #     model=get_vertex_model_task.outputs['artifact'],
  #     job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
  #     gcs_source_uris=preprocessing_task.outputs[
  #         'predictions_corpus_gcs_source'
  #     ],
  #     instances_format=batch_predict_instances_format,
  #     predictions_format=batch_predict_predictions_format,
  #     gcs_destination_output_uri_prefix=(
  #         f'{PIPELINE_ROOT_PLACEHOLDER}/batch_predict_output'
  #     ),
  #     encryption_spec_key_name=encryption_spec_key_name,
  # )

  # batch_predict_query = ModelBatchPredictOp(
  #     project=project,
  #     location=location,
  #     model=get_vertex_model_task.outputs['artifact'],
  #     job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
  #     gcs_source_uris=preprocessing_task.outputs[
  #         'predictions_query_gcs_source'
  #     ],
  #     instances_format=batch_predict_instances_format,
  #     predictions_format=batch_predict_predictions_format,
  #     gcs_destination_output_uri_prefix=(
  #         f'{PIPELINE_ROOT_PLACEHOLDER}/batch_predict_output'
  #     ),
  #     encryption_spec_key_name=encryption_spec_key_name,
  # )

  # TODO(b/290838262): Revisit if/when the concurrent jobs limit is increased/removed.
  batch_predict_query.after(batch_predict_corpus)

  embedding_retrieval_task = LLMEmbeddingRetrievalOp(
      project=project,
      location=location,
      query_embedding_source_directory=batch_predict_query.outputs[
          'gcs_output_directory'
      ],
      doc_embedding_source_directory=batch_predict_corpus.outputs[
          'gcs_output_directory'
      ],
      embedding_retrieval_top_n=embedding_retrieval_top_n,
      embedding_retrieval_combination_function=embedding_retrieval_combination_function,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      runner=runner,
      dataflow_service_account=dataflow_service_account,
      dataflow_disk_size_gb=dataflow_disk_size_gb,
      dataflow_machine_type=dataflow_machine_type,
      dataflow_workers_num=dataflow_workers_num,
      dataflow_max_workers_num=dataflow_max_workers_num,
      dataflow_subnetwork=dataflow_subnetwork,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  retrieval_metrics_task = LLMRetrievalMetricsOp(
      project=project,
      location=location,
      golden_docs_pattern=preprocessing_task.outputs[
          'embedding_retrieval_gcs_source'
      ],
      embedding_retrieval_results_pattern=embedding_retrieval_task.outputs[
          'embedding_retrieval_results_path'
      ],
      retrieval_metrics_top_k_list=retrieval_metrics_top_k_list,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      runner=runner,
      dataflow_service_account=dataflow_service_account,
      dataflow_disk_size_gb=dataflow_disk_size_gb,
      dataflow_machine_type=dataflow_machine_type,
      dataflow_workers_num=dataflow_workers_num,
      dataflow_max_workers_num=dataflow_max_workers_num,
      dataflow_subnetwork=dataflow_subnetwork,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  ModelImportEvaluationOp(
      embedding_metrics=retrieval_metrics_task.outputs['retrieval_metrics'],
      model=get_vertex_model_task.outputs['artifact'],
      display_name=_PIPELINE_NAME,
  )
