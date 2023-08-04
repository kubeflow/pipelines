"""Vertex LLM safety metrics pipeline."""

from typing import List, NamedTuple

from google_cloud_pipeline_components._implementation.model_evaluation import SafetyMetricsOp
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
from kfp import dsl
from kfp.components import importer_node
from kfp.dsl import Metrics


@dsl.pipeline(name='evaluation-llm-safety-pipeline')
def llm_safety_eval_pipeline(  # pylint: disable=dangerous-default-value
    project: str,
    location: str,
    batch_predict_gcs_destination_output_uri: str,
    model_name: str = 'publishers/google/models/text-bison@001',
    slice_spec_gcs_source: str = '',
    batch_predict_instances_format: str = 'jsonl',
    batch_predict_gcs_source_uris: List[str] = [],
    batch_predict_predictions_format: str = 'jsonl',
    machine_type: str = 'n1-standard-4',
    service_account: str = '',
    network: str = '',
    encryption_spec_key_name: str = '',
) -> NamedTuple('outputs', safety_metrics=Metrics):
  """The LLM Data Slicing and Safety Metrics Evaluation pipeline with batch prediction.

  Args:
    project: The GCP project that runs the pipeline components.
    location: The GCP region that runs the pipeline components.
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
    model_name: The Model name used to run evaluation. Must be a publisher Model
      or a managed Model sharing the same ancestor location. Starting this job
      has no impact on any existing deployments of the Model and their
      resources.
    slice_spec_gcs_source: The Google Cloud Storage location of the file where
      the slice spec definition is located.
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. Only "jsonl" is
      currently supported. For more details about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_gcs_source_uris: Google Cloud Storage URI(-s) to your
      instances data to run batch prediction on. The instances data should also
      contain the ground truth (target) data, used for evaluation. May contain
      wildcards. For more information on wildcards, see
      https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames. For
        more details about this input config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which Vertex AI gives the
      predictions. Must be one of the Model's supportedOutputStorageFormats.
      Only "jsonl" is currently supported. For more details about this output
      config, see
      https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
    machine_type: The machine type of this custom job. If not set, defaulted to
      ``n1-standard-4``. More details:
        https://cloud.google.com/compute/docs/machine-resource
    service_account: Sets the default service account for workload run-as
      account. The service account running the pipeline
      (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
      submitting jobs must have act-as permission on this run-as account. If
      unspecified, the Vertex AI Custom Code Service
      Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
      for the CustomJob's project.
    network: The full name of the Compute Engine network to which the job should
      be peered. For example, ``projects/12345/global/networks/myVPC``. Format
      is of the form ``projects/{project}/global/networks/{network}``. Where
      ``{project}`` is a project number, as in ``12345``, and ``{network}`` is a
      network name, as in ``myVPC``. To specify this field, you must have
      already configured VPC Network Peering for Vertex AI
      (https://cloud.google.com/vertex-ai/docs/general/vpc-peering). If left
      unspecified, the job is not peered with any network.
    encryption_spec_key_name:  Customer-managed encryption key options. If set,
      resources created by this pipeline will be encrypted with the provided
      encryption key. Has the form:
      ``projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key``.
      The key needs to be in the same region as where the compute resource is
      created.

  Returns:
    NamedTuple:
      safety_metrics: Metrics Artifact for Safety.
  """
  outputs = NamedTuple(
      'outputs',
      safety_metrics=Metrics,
  )

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
  )

  safety_task = SafetyMetricsOp(
      project=project,
      predictions_gcs_source=batch_predict_task.outputs['gcs_output_directory'],
      slice_spec_gcs_source=slice_spec_gcs_source,
      location=location,
      machine_type=machine_type,
      service_account=service_account,
      network=network,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  return outputs(safety_metrics=safety_task.outputs['bias_llm_metrics'])
