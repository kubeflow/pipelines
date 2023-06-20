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

from google_cloud_pipeline_components._implementation.model import GetVertexModelOp
from google_cloud_pipeline_components._implementation.model_evaluation import ErrorAnalysisAnnotationOp
from google_cloud_pipeline_components._implementation.model_evaluation import EvaluatedAnnotationOp
from google_cloud_pipeline_components._implementation.model_evaluation import EvaluationDatasetPreprocessorOp as DatasetPreprocessorOp
from google_cloud_pipeline_components._implementation.model_evaluation import FeatureExtractorOp
from google_cloud_pipeline_components._implementation.model_evaluation import ModelImportEvaluatedAnnotationOp
from google_cloud_pipeline_components._implementation.model_evaluation import ModelImportEvaluationOp
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
from google_cloud_pipeline_components.v1.dataset import GetVertexDatasetOp
from google_cloud_pipeline_components.v1.model_evaluation.classification_component import model_evaluation_classification as ModelEvaluationClassificationOp
import kfp
from kfp import dsl


@kfp.dsl.pipeline(name='vision-model-error-analysis-pipeline')
def vision_model_error_analysis_pipeline(  # pylint: disable=dangerous-default-value
    project: str,
    location: str,
    model_name: str,
    batch_predict_gcs_destination_output_uri: str,
    test_dataset_resource_name: str = '',
    test_dataset_annotation_set_name: str = '',
    training_dataset_resource_name: str = '',
    training_dataset_annotation_set_name: str = '',
    test_dataset_storage_source_uris: list = [],
    training_dataset_storage_source_uris: list = [],
    batch_predict_instances_format: str = 'jsonl',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_machine_type: str = 'n1-standard-32',
    batch_predict_starting_replica_count: int = 5,
    batch_predict_max_replica_count: int = 10,
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    dataflow_machine_type: str = 'n1-standard-8',
    dataflow_max_num_workers: int = 5,
    dataflow_disk_size_gb: int = 50,
    dataflow_service_account: str = '',
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_runner_mode: str = '',
):
  """The evaluation vision error analysis pipeline.

  This pipeline can help you to continuously discover dataset example errors
  with nearest neighbor distances and outlier flags, and provides you with
  actionable steps to improve the model performance. It uses GCP services
  including Dataflow and BatchPrediction.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
    model_name: The Vertex model resource name to be imported and used for batch
      prediction, in the format of
      projects/{project}/locations/{location}/models/{model} or
      projects/{project}/locations/{location}/models/{model}@{model_version_id
      or model_version_alias}
    batch_predict_gcs_destination_output_uri: The Google Cloud Storage location
      of the directory where the output is to be written to. In the given
      directory a new directory is created. Its name is
      ``prediction-<model-display-name>-<job-create-time>``, where timestamp is
      in YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format. Inside of it files
      ``predictions_0001.<extension>``, ``predictions_0002.<extension>``, ...,
      ``predictions_N.<extension>`` are created where ``<extension>`` depends on
      chosen ``predictions_format``, and N may equal 0001 and depends on the
      total number of successfully predicted instances. If the Model has both
      ``instance`` and ``prediction`` schemata defined then each such file
      contains predictions as per the ``predictions_format``. If prediction for
      any instance failed (partially or completely), then an additional
      ``errors_0001.<extension>``, ``errors_0002.<extension>``,...,
      ``errors_N.<extension>`` files are created (N depends on total number of
      failed predictions). These files contain the failed instances, as per
      their schema, followed by an additional ``error`` field which as value has
      ``google.rpc.Status`` containing only ``code`` and ``message`` fields. For
      more details about this output config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    test_dataset_resource_name: A Vertex dataset resource name of the test
      dataset. If ``test_dataset_storage_source_uris`` is also provided, this
      argument will override the GCS source.
    test_dataset_annotation_set_name: A string of the annotation_set resource
      name containing the ground truth of the test datset used for evaluation.
    training_dataset_resource_name: A Vertex dataset resource name of the
      training dataset. If ``training_dataset_storage_source_uris`` is also
      provided, this argument will override the GCS source.
    training_dataset_annotation_set_name: A string of the annotation_set
      resource name containing the ground truth of the test datset used for
      feature extraction.
    test_dataset_storage_source_uris: Google Cloud Storage URI(-s) to unmanaged
      test datasets.``jsonl`` is currently the only allowed format. If
      ``test_dataset`` is also provided, this field will be overriden by the
      provided Vertex Dataset.
    training_dataset_storage_source_uris: Google Cloud Storage URI(-s) to
      unmanaged test datasets.``jsonl`` is currently the only allowed format. If
      ``training_dataset`` is also provided, this field will be overriden by the
      provided Vertex Dataset.
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. For more details
      about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which Vertex AI gives the
      predictions. Must be one of the Model's supportedOutputStorageFormats. For
      more details about this output config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    batch_predict_machine_type: The type of machine for running batch prediction
      on dedicated resources. If the Model supports DEDICATED_RESOURCES this
      config may be provided (and the job will use these resources). If the
      Model doesn't support AUTOMATIC_RESOURCES, this config must be provided.
      For more details about the BatchDedicatedResources, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#BatchDedicatedResources.
        For more details about the machine spec, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_starting_replica_count: The number of machine replicas used at
      the start of the batch operation. If not set, Vertex AI decides starting
      number, not greater than ``max_replica_count``. Only used if
      ``machine_type`` is set.
    batch_predict_max_replica_count: The maximum number of machine replicas the
      batch operation may be scaled to. Only used if ``machine_type`` is set.
    batch_predict_accelerator_type: The type of accelerator(s) that may be
      attached to the machine as per ``batch_predict_accelerator_count``. Only
      used if ``batch_predict_machine_type`` is set. For more details about the
      machine spec, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_accelerator_count: The number of accelerators to attach to the
      ``batch_predict_machine_type``. Only used if
      ``batch_predict_machine_type`` is set.
    dataflow_machine_type: The Dataflow machine type for evaluation components.
    dataflow_max_num_workers: The max number of Dataflow workers for evaluation
      components.
    dataflow_disk_size_gb: Dataflow worker's disk size in GB for evaluation
      components.
    dataflow_service_account: Custom service account to run Dataflow jobs.
    dataflow_subnetwork: Dataflow's fully qualified subnetwork name, when empty
      the default subnetwork will be used. Example:
      https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    encryption_spec_key_name:  Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      ``projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key``.
      The key needs to be in the same region as where the compute resource is
      created.
    force_runner_mode: Indicate the runner mode to use forcely. Valid options
      are ``Dataflow`` and ``DirectRunner``.
  """
  evaluation_display_name = 'vision-model-error-analysis-pipeline'

  with kfp.dsl.Condition(
      (
          test_dataset_resource_name != ''
          and training_dataset_resource_name != ''
          and test_dataset_annotation_set_name != ''
          and training_dataset_annotation_set_name != ''
      ),
      name='VertexDataset',
  ):
    get_test_dataset_task = GetVertexDatasetOp(
        dataset_resource_name=test_dataset_resource_name
    )
    get_training_dataset_task = GetVertexDatasetOp(
        dataset_resource_name=training_dataset_resource_name
    )
    dataset_preprocessor_task = DatasetPreprocessorOp(
        project=project,
        location=location,
        test_dataset=get_test_dataset_task.outputs['dataset'],
        test_dataset_annotation_set_name=test_dataset_annotation_set_name,
        training_dataset=get_training_dataset_task.outputs['dataset'],
        training_dataset_annotation_set_name=training_dataset_annotation_set_name,
    )
    get_model_task = GetVertexModelOp(model_name=model_name)
    batch_predict_task = ModelBatchPredictOp(
        project=project,
        location=location,
        model=get_model_task.outputs['model'],
        job_display_name=(
            f'{evaluation_display_name}-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}'
        ),
        gcs_source_uris=dataset_preprocessor_task.outputs[
            'batch_prediction_storage_source'
        ],
        instances_format=batch_predict_instances_format,
        predictions_format=batch_predict_predictions_format,
        gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
        machine_type=batch_predict_machine_type,
        starting_replica_count=batch_predict_starting_replica_count,
        max_replica_count=batch_predict_max_replica_count,
        encryption_spec_key_name=encryption_spec_key_name,
        accelerator_type=batch_predict_accelerator_type,
        accelerator_count=batch_predict_accelerator_count,
    )
    eval_task = ModelEvaluationClassificationOp(
        project=project,
        location=location,
        target_field_name='ground_truth',
        ground_truth_format='jsonl',
        ground_truth_gcs_source=dataset_preprocessor_task.outputs[
            'model_evaluation_storage_source'
        ],
        predictions_format='jsonl',
        predictions_gcs_source=batch_predict_task.outputs[
            'gcs_output_directory'
        ],
        model=get_model_task.outputs['model'],
        dataflow_machine_type=dataflow_machine_type,
        dataflow_max_workers_num=dataflow_max_num_workers,
        dataflow_disk_size_gb=dataflow_disk_size_gb,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
        force_runner_mode=force_runner_mode,
        prediction_score_column='',
        prediction_label_column='',
    )
    evaluated_annotation_task = EvaluatedAnnotationOp(
        project=project,
        location=location,
        predictions_storage_source=batch_predict_task.outputs[
            'gcs_output_directory'
        ],
        ground_truth_storage_source=dataset_preprocessor_task.outputs[
            'test_data_items_storage_source'
        ],
        dataflow_machine_type=dataflow_machine_type,
        dataflow_max_workers_num=dataflow_max_num_workers,
        dataflow_disk_size_gb=dataflow_disk_size_gb,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
    )
    feature_extractor_task = FeatureExtractorOp(
        project=project,
        location=location,
        root_dir=batch_predict_gcs_destination_output_uri,
        test_dataset=get_test_dataset_task.outputs['dataset'],
        training_dataset=get_training_dataset_task.outputs['dataset'],
        preprocessed_test_dataset_storage_source=dataset_preprocessor_task.outputs[
            'test_data_items_storage_source'
        ],
        preprocessed_training_dataset_storage_source=dataset_preprocessor_task.outputs[
            'training_data_items_storage_source'
        ],
        feature_extractor_machine_type=batch_predict_machine_type,
        encryption_spec_key_name=encryption_spec_key_name,
    )
    error_analysis_task = ErrorAnalysisAnnotationOp(
        project=project,
        location=location,
        root_dir=batch_predict_gcs_destination_output_uri,
        embeddings_dir=feature_extractor_task.outputs['embeddings_dir'],
    )
    model_evaluation_importer_task = ModelImportEvaluationOp(
        classification_metrics=eval_task.outputs['evaluation_metrics'],
        model=get_model_task.outputs['model'],
        dataset_type=batch_predict_instances_format,
        dataset_paths=dataset_preprocessor_task.outputs[
            'batch_prediction_storage_source'
        ],
        display_name=(
            f'{evaluation_display_name}-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}'
        ),
    )
    ModelImportEvaluatedAnnotationOp(
        model=get_model_task.outputs['model'],
        evaluated_annotation_output_uri=evaluated_annotation_task.outputs[
            'evaluated_annotation_output_uri'
        ],
        evaluation_importer_gcp_resources=model_evaluation_importer_task.outputs[
            'gcp_resources'
        ],
        error_analysis_output_uri=error_analysis_task.outputs[
            'error_analysis_output_uri'
        ],
    )

  with kfp.dsl.Condition(
      (
          (
              test_dataset_resource_name == ''
              and training_dataset_resource_name == ''
              and test_dataset_annotation_set_name == ''
              and training_dataset_annotation_set_name == ''
          )
      ),
      name='CustomDataset',
  ):
    dataset_preprocessor_task = DatasetPreprocessorOp(
        project=project,
        location=location,
        test_dataset_storage_source_uris=test_dataset_storage_source_uris,
        training_dataset_storage_source_uris=training_dataset_storage_source_uris,
    )
    get_model_task = GetVertexModelOp(model_name=model_name)
    batch_predict_task = ModelBatchPredictOp(
        project=project,
        location=location,
        model=get_model_task.outputs['model'],
        job_display_name='sdk-batch-predict-evaluation',
        gcs_source_uris=dataset_preprocessor_task.outputs[
            'batch_prediction_storage_source'
        ],
        instances_format=batch_predict_instances_format,
        predictions_format=batch_predict_predictions_format,
        gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
        machine_type=batch_predict_machine_type,
        starting_replica_count=batch_predict_starting_replica_count,
        max_replica_count=batch_predict_max_replica_count,
        encryption_spec_key_name=encryption_spec_key_name,
        accelerator_type=batch_predict_accelerator_type,
        accelerator_count=batch_predict_accelerator_count,
    )
    eval_task = ModelEvaluationClassificationOp(
        project=project,
        location=location,
        target_field_name='ground_truth',
        ground_truth_format='jsonl',
        ground_truth_gcs_source=dataset_preprocessor_task.outputs[
            'model_evaluation_storage_source'
        ],
        predictions_format='jsonl',
        predictions_gcs_source=batch_predict_task.outputs[
            'gcs_output_directory'
        ],
        model=get_model_task.outputs['model'],
        dataflow_machine_type=dataflow_machine_type,
        dataflow_max_workers_num=dataflow_max_num_workers,
        dataflow_disk_size_gb=dataflow_disk_size_gb,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
        force_runner_mode=force_runner_mode,
        prediction_score_column='',
        prediction_label_column='',
    )
    evaluated_annotation_task = EvaluatedAnnotationOp(
        project=project,
        location=location,
        predictions_storage_source=batch_predict_task.outputs[
            'gcs_output_directory'
        ],
        ground_truth_storage_source=dataset_preprocessor_task.outputs[
            'test_data_items_storage_source'
        ],
        dataflow_machine_type=dataflow_machine_type,
        dataflow_max_workers_num=dataflow_max_num_workers,
        dataflow_disk_size_gb=dataflow_disk_size_gb,
        dataflow_service_account=dataflow_service_account,
        dataflow_subnetwork=dataflow_subnetwork,
        dataflow_use_public_ips=dataflow_use_public_ips,
        encryption_spec_key_name=encryption_spec_key_name,
    )
    feature_extractor_task = FeatureExtractorOp(
        project=project,
        location=location,
        root_dir=batch_predict_gcs_destination_output_uri,
        preprocessed_test_dataset_storage_source=dataset_preprocessor_task.outputs[
            'test_data_items_storage_source'
        ],
        preprocessed_training_dataset_storage_source=dataset_preprocessor_task.outputs[
            'training_data_items_storage_source'
        ],
        feature_extractor_machine_type=batch_predict_machine_type,
        encryption_spec_key_name=encryption_spec_key_name,
    )
    error_analysis_task = ErrorAnalysisAnnotationOp(
        project=project,
        location=location,
        root_dir=batch_predict_gcs_destination_output_uri,
        embeddings_dir=feature_extractor_task.outputs['embeddings_dir'],
    )
    model_evaluation_importer_task = ModelImportEvaluationOp(
        classification_metrics=eval_task.outputs['evaluation_metrics'],
        model=get_model_task.outputs['model'],
        dataset_type=batch_predict_instances_format,
        dataset_paths=dataset_preprocessor_task.outputs[
            'batch_prediction_storage_source'
        ],
        display_name=(
            f'{evaluation_display_name}-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}'
        ),
    )
    ModelImportEvaluatedAnnotationOp(
        model=get_model_task.outputs['model'],
        evaluated_annotation_output_uri=evaluated_annotation_task.outputs[
            'evaluated_annotation_output_uri'
        ],
        evaluation_importer_gcp_resources=model_evaluation_importer_task.outputs[
            'gcp_resources'
        ],
        error_analysis_output_uri=error_analysis_task.outputs[
            'error_analysis_output_uri'
        ],
    )
