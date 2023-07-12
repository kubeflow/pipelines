"""Vertex LLM standalone Evaluation for text classification task."""

from typing import NamedTuple

from google_cloud_pipeline_components._implementation.model_evaluation import function_based_components
from google_cloud_pipeline_components._implementation.model_evaluation import ModelEvaluationLLMClassificationPostprocessorOp
from google_cloud_pipeline_components._implementation.model_evaluation import ModelImportEvaluationOp
from google_cloud_pipeline_components.types.artifact_types import ClassificationMetrics
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
from google_cloud_pipeline_components.v1.model_evaluation.classification_component import model_evaluation_classification as ModelEvaluationClassificationOp
import kfp
from kfp.components import importer_node


@kfp.dsl.pipeline(name='evaluation-llm-classification-pipeline')
def llm_eval_classification_pipeline(  # pylint: disable=dangerous-default-value
    project: str,
    model_name: str,
    batch_predict_gcs_destination_output_uri: str,
    location: str = 'us-central1',
    evaluation_class_labels: list = [],  # pylint: disable=g-bare-generic
    target_field_name: str = 'ground_truth',
    batch_predict_instances_format: str = 'jsonl',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_gcs_source_uris: list = [],  # pylint: disable=g-bare-generic
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    machine_type: str = 'n1-standard-4',
    service_account: str = '',
    network: str = '',
    dataflow_max_num_workers: int = 5,
    dataflow_disk_size_gb: int = 50,
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    evaluation_display_name: str = 'llm-evaluation-text-classification',
    evaluation_task: str = 'text-classification',
) -> NamedTuple('Outputs', evaluation_metrics=ClassificationMetrics):
  """The LLM Text Classification Evaluation pipeline.

  Args:
    project: Required. Project to run the component.
    location: Location for running the component. If not set, defaulted to
      `us-central1`.
    model_name: The Model name used to get predictions via this job. Must share
      the same ancestor location. Starting this job has no impact on any
      existing deployments of the Model and their resources.
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
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. If not set,
      default to "jsonl".  For more details about this input config, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_accelerator_type: The type of accelerator(s) that may be
      attached to the machine as per `accelerator_count`. Only used if
      `machine_type` is set.  For more details about the machine spec, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    batch_predict_accelerator_count: The number of accelerators to attach to the
      `machine_type`. Only used if `machine_type` is set.  For more details
      about the machine spec, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
    machine_type: The machine type of this custom job. If not set, defaulted to
      `e2-highmem-16`. More details:
        https://cloud.google.com/compute/docs/machine-resource
    service_account: Optional. Service account to run the dataflow job. If not
      set, dataflow will use the default worker service account. For more
      details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
    network: Dataflow's fully qualified subnetwork name, when empty the default
      subnetwork will be used. More
      details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    dataflow_disk_size: The disk size (in GB) of the machine executing the
      evaluation run. If not set, defaulted to `50`.
    dataflow_workers_num: The number of workers executing the evaluation run. If
      not set, defaulted to `10`.
    dataflow_max_workers_num: The max number of workers executing the evaluation
      run. If not set, defaulted to `25`.
    dataflow_use_public_ips: Specifies whether Dataflow workers use public IP
      addresses.
    evaluation_display_name: The name of the Evaluation job.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.

  Returns:
    ClassificationMetrics for LLM Text Classification.
  """
  Outputs = NamedTuple('Outputs', evaluation_metrics=ClassificationMetrics)

  get_vertex_model_task = importer_node.importer(
      artifact_uri=(
          f'https://{location}-aiplatform.googleapis.com/v1/{model_name}'
      ),
      artifact_class=VertexModel,
      metadata={'resourceName': model_name},
  )
  get_vertex_model_task.set_display_name('get-vertex-model')

  batch_predict_task = ModelBatchPredictOp(
      project=project,
      location=location,
      model=get_vertex_model_task.outputs['artifact'],
      job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      gcs_source_uris=batch_predict_gcs_source_uris,
      instances_format=batch_predict_instances_format,
      predictions_format=batch_predict_predictions_format,
      gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
      encryption_spec_key_name=encryption_spec_key_name,
      accelerator_type=batch_predict_accelerator_type,
      accelerator_count=batch_predict_accelerator_count,
  )

  postprocessor_task = ModelEvaluationLLMClassificationPostprocessorOp(
      project=project,
      location=location,
      class_labels=function_based_components.add_json_escape_class_labels(
          class_labels=evaluation_class_labels
      ).output,
      batch_prediction_results=batch_predict_task.outputs[
          'gcs_output_directory'
      ],
  )
  eval_task = ModelEvaluationClassificationOp(
      project=project,
      location=location,
      class_labels=postprocessor_task.outputs['postprocessed_class_labels'],
      target_field_name=target_field_name,
      predictions_gcs_source=postprocessor_task.outputs[
          'postprocessed_predictions_gcs_source'
      ],
      prediction_label_column='prediction.classes',
      prediction_score_column='prediction.scores',
      predictions_format=batch_predict_predictions_format,
      dataflow_machine_type=machine_type,
      dataflow_max_workers_num=dataflow_max_num_workers,
      dataflow_disk_size_gb=dataflow_disk_size_gb,
      dataflow_service_account=service_account,
      dataflow_subnetwork=network,
      dataflow_use_public_ips=dataflow_use_public_ips,
      encryption_spec_key_name=encryption_spec_key_name,
  )
  ModelImportEvaluationOp(
      classification_metrics=eval_task.outputs['evaluation_metrics'],
      model=get_vertex_model_task.outputs['artifact'],
      dataset_type=batch_predict_instances_format,
      dataset_paths=batch_predict_gcs_source_uris,
      display_name=evaluation_display_name,
  )

  return Outputs(evaluation_metrics=eval_task.outputs['evaluation_metrics'])
