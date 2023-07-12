"""Vertex LLM standalone Evaluation for text generation task."""

from typing import NamedTuple

from google_cloud_pipeline_components._implementation.model_evaluation import function_based_components
from google_cloud_pipeline_components._implementation.model_evaluation import ModelEvaluationTextGenerationOp
from google_cloud_pipeline_components._implementation.model_evaluation import ModelImportEvaluationOp
from google_cloud_pipeline_components.v1.batch_predict_job import ModelBatchPredictOp
import kfp


@kfp.dsl.pipeline(name='evaluation-llm-text-generation-pipeline')
def llm_eval_text_generation_pipeline(  # pylint: disable=dangerous-default-value
    project: str,
    model_name: str,
    batch_predict_gcs_destination_output_uri: str,
    location: str = 'us-central1',
    evaluation_task: str = 'text-generation',
    batch_predict_gcs_source_uris: list = [],  # pylint: disable=g-bare-generic
    batch_predict_instances_format: str = 'jsonl',
    batch_predict_predictions_format: str = 'jsonl',
    batch_predict_accelerator_type: str = '',
    batch_predict_accelerator_count: int = 0,
    machine_type: str = 'e2-highmem-16',
    evaluation_display_name: str = 'evaluation-text-generation',
    service_account: str = '',
    enable_web_access: bool = True,
    network: str = '',
    reserved_ip_ranges: list = [],  # pylint: disable=g-bare-generic
    encryption_spec_key_name: str = '',
) -> NamedTuple('Outputs', evaluation_metrics=kfp.dsl.Metrics):
  """LLM Text Generation Evaluation pipeline.

  Supports evaluating large language models performing the following
  generative tasks: `summarization`,`question-answering`,`text-generation`.

  Args:
      project (str): Required. Project to run the component.
      location (Optional[str]): Location for running the component. If not set,
        defaulted to `us-central1`.
      model_name (Optional[str]): The Model name used to get predictions via
        this job. Must share the same ancestor location. Starting this job has
        no impact on any existing deployments of the Model and their resources.
      evaluation_task (Optional[str]): The task that the large language model
        will be evaluated on. The evaluation component computes a set of metrics
        relevant to that specific task. Currently supported tasks are:
        `summarization`,`question-answering`,`text-generation`.
      batch_predict_gcs_destination_output_uri (Optional[str]): The Google Cloud
        Storage location of the directory where the output is to be written to.
        In the given directory a new directory is created. Its name is
        ``prediction-<model-display-name>-<job-create-time>``, where timestamp
        is in YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format. Inside of it files
        ``predictions_0001.<extension>``, ``predictions_0002.<extension>``, ...,
        ``predictions_N.<extension>`` are created where ``<extension>`` depends
        on chosen ``predictions_format``, and N may equal 0001 and depends on
        the total number of successfully predicted instances. If the Model has
        both ``instance`` and ``prediction`` schemata defined then each such
        file contains predictions as per the ``predictions_format``. If
        prediction for any instance failed (partially or completely), then an
        additional ``errors_0001.<extension>``, ``errors_0002.<extension>``,...,
        ``errors_N.<extension>`` files are created (N depends on total number of
        failed predictions). These files contain the failed instances, as per
        their schema, followed by an additional ``error`` field which as value
        has ``google.rpc.Status`` containing only ``code`` and ``message``
        fields. For more details about this output config, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
      batch_predict_instances_format (Optional[str]): The format in which
        instances are given, must be one of the Model's
        supportedInputStorageFormats. If not set, default to "jsonl".  For more
        details about this input config, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
      batch_predict_accelerator_type (Optional[str]): The type of accelerator(s)
        that may be attached to the machine as per `accelerator_count`. Only
        used if `machine_type` is set.  For more details about the machine spec,
        see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
      batch_predict_accelerator_count (Optional[int]): The number of
        accelerators to attach to the `machine_type`. Only used if
        `machine_type` is set.  For more details about the machine spec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
      machine_type (Optional[str]): The machine type of this custom job. If not
        set, defaulted to `e2-highmem-16`. More details:
          https://cloud.google.com/compute/docs/machine-resource
      evaluation_display_name (Optional[str]): The name of the Evaluation job.
      service_account (Optional[str]): Sets the default service account for
        workload run-as account. The service account running the pipeline
        (https://cloud.google.com/vertex-ai/docs/pipelines/configure-project#service-account)
        submitting jobs must have act-as permission on this run-as account. If
        unspecified, the Vertex AI Custom Code Service
        Agent(https://cloud.google.com/vertex-ai/docs/general/access-control#service-agents)
        for the CustomJob's project.
      enable_web_access (Optional[bool]): Whether you want Vertex AI to enable
        [interactive shell
        access](https://cloud.google.com/vertex-ai/docs/training/monitor-debug-interactive-shell)
        to training containers. If set to `true`, you can access interactive
        shells at the URIs given by [CustomJob.web_access_uris][].
      network (Optional[str]): The full name of the Compute Engine network to
        which the job should be peered. For example,
        projects/12345/global/networks/myVPC. Format is of the form
        projects/{project}/global/networks/{network}. Where {project} is a
        project number, as in 12345, and {network} is a network name. Private
        services access must already be configured for the network. If left
        unspecified, the job is not peered with any network.
      reserved_ip_ranges (Optional[Sequence[str]]): A list of names for the
        reserved ip ranges under the VPC network that can be used for this job.
        If set, we will deploy the job within the provided ip ranges. Otherwise,
        the job will be deployed to any ip ranges under the provided VPC
        network.
      encryption_spec_key_name (Optional[str]): Customer-managed encryption key
        options for the CustomJob. If this is set, then all resources created by
        the CustomJob will be encrypted with the provided encryption key.
  """
  Outputs = NamedTuple('Outputs', evaluation_metrics=kfp.dsl.Metrics)

  batch_predict_task = ModelBatchPredictOp(
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

  converter_task = function_based_components.convert_artifact_to_string(
      input_artifact=batch_predict_task.outputs['gcs_output_directory']
  )

  eval_task = ModelEvaluationTextGenerationOp(
      project=project,
      location=location,
      evaluation_task=evaluation_task,
      target_field_name='instance.ground_truth',
      prediction_field_name='predictions.content',
      predictions_format=batch_predict_predictions_format,
      joined_predictions_gcs_source=converter_task.output,
      display_name=evaluation_display_name,
      machine_type=machine_type,
      service_account=service_account,
      enable_web_access=enable_web_access,
      network=network,
      reserved_ip_ranges=reserved_ip_ranges,
      encryption_spec_key_name=encryption_spec_key_name,
  )

  with kfp.dsl.Condition(
      evaluation_task == 'text-generation', name='Text-Generation'
  ):
    ModelImportEvaluationOp(
        text_generation_metrics=eval_task.outputs['evaluation_metrics'],
        model=model_name,
        dataset_type=batch_predict_predictions_format,
        dataset_paths=batch_predict_gcs_source_uris,
        display_name=evaluation_display_name,
    )
  with kfp.dsl.Condition(
      evaluation_task == 'summarization', name='Summarization'
  ):
    ModelImportEvaluationOp(
        summarization_metrics=eval_task.outputs['evaluation_metrics'],
        model=model_name,
        dataset_type=batch_predict_predictions_format,
        dataset_paths=batch_predict_gcs_source_uris,
        display_name=evaluation_display_name,
    )
  with kfp.dsl.Condition(
      evaluation_task == 'question-answering', name='Question-Answering'
  ):
    ModelImportEvaluationOp(
        question_answering_metrics=eval_task.outputs['evaluation_metrics'],
        model=model_name,
        dataset_type=batch_predict_predictions_format,
        dataset_paths=batch_predict_gcs_source_uris,
        display_name=evaluation_display_name,
    )

  return Outputs(evaluation_metrics=eval_task.outputs['evaluation_metrics'])
