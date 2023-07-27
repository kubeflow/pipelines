"""Vertex LLM safety metrics pipeline."""

import sys

import kfp
from vertexevaluation.llm.component import function_based
from vertexevaluation.llm.component.batch_predict import model_batch_predict
from google_cloud_pipeline_components._implementation.model_evaluation import SafetyMetricsOp
from vertexevaluation.llm.pipelines import utils


@kfp.dsl.pipeline(name='llm-safety-eval-pipeline')
def llm_safety_eval_pipeline(
    project: str,
    model_name: str,
    batch_predict_gcs_destination_output_uri: str,
    slice_spec_gcs_source: str = '',
    location: str = 'us-central1',
    batch_predict_gcs_source_uris: list = [],  # pylint: disable=g-bare-generic
    batch_predict_instances_format: str = 'jsonl',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    machine_type: str = 'n1-standard-4',
    service_account: str = '',
    enable_web_access: bool = True,
    network: str = '',
    reserved_ip_ranges: list = [],  # pylint: disable=g-bare-generic
    encryption_spec_key_name: str = '',
):
  """The LLM Data Slicing and Safety Metrics Evaluation pipeline with batch prediction.

  Args:
    project: Required. Project to run the component.
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
    slice_spec_gcs_source: The Google Cloud Storage location of the file where
      the slice spec definition is located.
    location: Location for running the component. If not set, defaulted to
      `us-central1`.
    batch_predict_gcs_source_uris: The Google Cloud Storage batch predict source
      locations.
    batch_predict_instances_format: The format in which instances are given,
      must be one of the Model's supportedInputStorageFormats. If not set,
      default to "jsonl".  For more details about this input config, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
    batch_predict_predictions_format: The format in which predictions are given,
      must be one of the Model's supportedInputStorageFormats. If not set,
      default to "jsonl".
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
    enable_web_access (Optional[bool]): Whether you want Vertex AI to enable
        [interactive shell access]
        https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell
        to training containers. If set to `true`, you can access interactive
        shells at the URIs given by [CustomJob.web_access_uris][].
    network: Dataflow's fully qualified subnetwork name, when empty the default
      subnetwork will be used. More details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
    reserved_ip_ranges: The reserved ip ranges.
    encryption_spec_key_name: Customer-managed encryption key options for the
      CustomJob. If this is set, then all resources created by the CustomJob
      will be encrypted with the provided encryption key.
  """

  batch_predict_task = model_batch_predict(
      project=project,
      location=location,
      model=model_name,
      job_display_name='evaluation-batch-predict-{{$.pipeline_job_uuid}}-{{$.pipeline_task_uuid}}',
      gcs_source_uris=batch_predict_gcs_source_uris,
      instances_format=batch_predict_instances_format,
      predictions_format=batch_predict_predictions_format,
      gcs_destination_output_uri_prefix=batch_predict_gcs_destination_output_uri,
      encryption_spec_key_name=encryption_spec_key_name,
      accelerator_type=batch_predict_accelerator_type,
      accelerator_count=batch_predict_accelerator_count,
  )

  converter_task = function_based.convert_artifact_to_string(
      input_artifact=batch_predict_task.outputs['gcs_output_directory']
  )

  SafetyMetricsOp(
      project=project,
      predictions_gcs_source=converter_task.output,
      slice_spec_gcs_source=slice_spec_gcs_source,
      location=location,
      machine_type=machine_type,
      service_account=service_account,
      enable_web_access=enable_web_access,
      network=network,
      reserved_ip_ranges=reserved_ip_ranges,
      encryption_spec_key_name=encryption_spec_key_name
  )


def main(argv: list[str]) -> None:
  parsed_args = utils.parse_args('llm_safety_eval_pipeline', argv)

  parameters = utils.get_parameters_from_input_args_for_pipeline(
      parsed_args, llm_safety_eval_pipeline
  )

  parameters.update(
      {
          'batch_predict_gcs_source_uris': [
              'gs://lakeyk-llm-test/golden_dataset/adversarial_with_gender_identity_1k_col_renamed.jsonl'
          ]
      }
  )

  job = utils.run_pipeline(
      llm_safety_eval_pipeline,
      parameters=parameters,
      project=parameters['project'],
      location=parameters['location'],
      pipeline_root=parameters['batch_predict_gcs_destination_output_uri'],
  )

  if parsed_args.wait:
    job.wait()


if __name__ == '__main__':
  main(sys.argv)
