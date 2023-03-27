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

from typing import Dict, List

from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import UnmanagedContainerModel
from google_cloud_pipeline_components.types.artifact_types import VertexBatchPredictionJob
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp.dsl import Artifact
from kfp.dsl import ConcatPlaceholder
from kfp.dsl import container_component
from kfp.dsl import ContainerSpec
from kfp.dsl import IfPresentPlaceholder
from kfp.dsl import Input
from kfp.dsl import Output
from kfp.dsl import OutputPath


@container_component
def model_batch_predict(
    project: str,
    job_display_name: str,
    gcp_resources: OutputPath(str),
    batchpredictionjob: Output[VertexBatchPredictionJob],
    bigquery_output_table: Output[BQTable],
    gcs_output_directory: Output[Artifact],
    model: Input[VertexModel] = None,
    unmanaged_container_model: Input[UnmanagedContainerModel] = None,
    location: str = 'us-central1',
    instances_format: str = 'jsonl',
    predictions_format: str = 'jsonl',
    gcs_source_uris: List[str] = [],
    bigquery_source_input_uri: str = '',
    model_parameters: Dict[str, str] = {},
    gcs_destination_output_uri_prefix: str = '',
    bigquery_destination_output_uri: str = '',
    machine_type: str = '',
    accelerator_type: str = '',
    accelerator_count: int = 0,
    starting_replica_count: int = 0,
    max_replica_count: int = 0,
    manual_batch_tuning_parameters_batch_size: int = 0,
    generate_explanation: bool = False,
    explanation_metadata: Dict[str, str] = {},
    explanation_parameters: Dict[str, str] = {},
    labels: Dict[str, str] = {},
    encryption_spec_key_name: str = '',
):
  """Creates a Google Cloud Vertex BatchPredictionJob and waits for it to complete.

  For more details, see
  https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs/create.

  Args:
      project (str):
        Required. Project to create the BatchPredictionJob.
      job_display_name (str):
        Required. The user-defined name of this BatchPredictionJob.
      location (Optional[str]):
        Location for creating the BatchPredictionJob.
        If not set, default to us-central1.
      instances_format (Optional[str]):
        The format in which instances are
        given, must be one of the Model's supportedInputStorageFormats. If not
        set, default to "jsonl".  For more details about this input config,
        see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
      predictions_format (Optional[str]):
        The format in which Vertex AI gives the predictions. Must be one of the
        Model's supportedOutputStorageFormats. If not set, default to "jsonl".
        For more details about this output config, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
      model (Optional[google.VertexModel]):
        The Model used to get predictions via this job. Must share the same
        ancestor Location. Starting this job has no impact on any existing
        deployments of the Model and their resources. Either this or
        unmanaged_container_model must be specified.
      unmanaged_container_model (Optional[google.UnmanagedContainerModel]):
        The unmanaged container model used to get predictions via this job.
        This should be used for models that are not uploaded to Vertex. Either
        this or model must be specified.
      gcs_source_uris (Optional[Sequence[str]]):
        Google Cloud Storage URI(-s) to your instances to run batch prediction
        on. They must match `instances_format`. May contain wildcards. For more
        information on wildcards, see
          https://cloud.google.com/storage/docs/gsutil/addlhelp/WildcardNames.
        For more details about this input config, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
      bigquery_source_input_uri (Optional[str]):
        BigQuery URI to a table, up to 2000 characters long. For example:
        `projectId.bqDatasetId.bqTableId`  For more details about this input
        config, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#InputConfig.
      model_parameters (Optional[dict]):
        The parameters that govern the predictions. The schema of the parameters
        may be specified via the Model's `parameters_schema_uri`.
      gcs_destination_output_uri_prefix (Optional[str]):
        The Google Cloud
        Storage location of the directory where the output is to be written
        to. In the given directory a new directory is created. Its name is
        ``prediction-<model-display-name>-<job-create-time>``, where timestamp
        is in YYYY-MM-DDThh:mm:ss.sssZ ISO-8601 format. Inside of it files
        ``predictions_0001.<extension>``, ``predictions_0002.<extension>``,
        ..., ``predictions_N.<extension>`` are created where ``<extension>``
        depends on chosen ``predictions_format``, and N may equal 0001 and
        depends on the total number of successfully predicted instances. If
        the Model has both ``instance`` and ``prediction`` schemata defined
        then each such file contains predictions as per the
        ``predictions_format``. If prediction for any instance failed
        (partially or completely), then an additional
        ``errors_0001.<extension>``, ``errors_0002.<extension>``,...,
        ``errors_N.<extension>`` files are created (N depends on total number
        of failed predictions). These files contain the failed instances, as
        per their schema, followed by an additional ``error`` field which as
        value has ``google.rpc.Status`` containing only ``code`` and
        ``message`` fields.  For more details about this output config, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
      bigquery_destination_output_uri (Optional[str]):
        The BigQuery project location where the output is to be written to. In
        the given project a new dataset is created with name
        ``prediction_<model-display-name>_<job-create-time>`` where is made
        BigQuery-dataset-name compatible (for example, most special characters
        become underscores), and timestamp is in YYYY_MM_DDThh_mm_ss_sssZ
        "based on ISO-8601" format. In the dataset two tables will be created,
        ``predictions``, and ``errors``. If the Model has both ``instance``
        and ``prediction`` schemata defined then the tables have columns as
        follows: The ``predictions`` table contains instances for which the
          prediction succeeded, it has columns as per a concatenation of the
          Model's instance and prediction schemata. The ``errors`` table
          contains rows for which the prediction has failed, it has instance
          columns, as per the instance schema, followed by a single "errors"
          column, which as values has ```google.rpc.Status`` <Status>`__
          represented as a STRUCT, and containing only ``code`` and
          ``message``. For more details about this output config, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#OutputConfig.
      machine_type (Optional[str]):
        The type of machine for running batch
        prediction on dedicated resources. If the Model supports
        DEDICATED_RESOURCES this config may be provided (and the job will use
        these resources). If the Model doesn't support AUTOMATIC_RESOURCES,
        this config must be provided.  For more details about the
        BatchDedicatedResources, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/projects.locations.batchPredictionJobs#BatchDedicatedResources.
        For more details about the machine spec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
      accelerator_type (Optional[str]):
        The type of accelerator(s) that may be
        attached to the machine as per `accelerator_count`. Only used if
        `machine_type` is set.  For more details about the machine spec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
      accelerator_count (Optional[int]):
        The number of accelerators to attach
        to the `machine_type`. Only used if `machine_type` is set.  For more
        details about the machine spec, see
          https://cloud.google.com/vertex-ai/docs/reference/rest/v1/MachineSpec
      starting_replica_count (Optional[int]):
        The number of machine replicas
        used at the start of the batch operation. If not set, Vertex AI
        decides starting number, not greater than `max_replica_count`. Only
        used if `machine_type` is set.
      max_replica_count (Optional[int]):
        The maximum number of machine replicas the batch operation may be scaled
        to. Only used if `machine_type` is set. Default is 10.
      manual_batch_tuning_parameters_batch_size (Optional[int]):
        The number of
        the records (e.g. instances) of the operation given in each batch to a
        machine replica. Machine type, and size of a single record should be
        considered when setting this parameter, higher value speeds up the
        batch operation's execution, but too high value will result in a whole
        batch not fitting in a machine's memory, and the whole operation will
        fail. The default value is 4.
      generate_explanation (Optional[bool]):
        Generate explanation along with
        the batch prediction results. This will cause the batch prediction
        output to include explanations based on the `prediction_format`: -
        `bigquery`: output includes a column named `explanation`. The value is
        a struct that conforms to the [aiplatform.gapic.Explanation] object. -
        `jsonl`: The JSON objects on each line include an additional entry
        keyed `explanation`. The value of the entry is a JSON object that
        conforms to the [aiplatform.gapic.Explanation] object. - `csv`:
        Generating explanations for CSV format is not supported.  If this
        field is set to true, either the Model.explanation_spec or
        explanation_metadata and explanation_parameters must be populated.
      explanation_metadata (Optional[dict]):
        Explanation metadata
        configuration for this BatchPredictionJob. Can be specified only if
        `generate_explanation` is set to `True`.  This value overrides the
        value of `Model.explanation_metadata`. All fields of
        `explanation_metadata` are optional in the request. If a field of the
        `explanation_metadata` object is not populated, the corresponding
        field of the `Model.explanation_metadata` object is inherited.  For
        more details, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#explanationmetadata.
      explanation_parameters (Optional[dict]):
        Parameters to configure
        explaining for Model's predictions. Can be specified only if
        `generate_explanation` is set to `True`.  This value overrides the
        value of `Model.explanation_parameters`. All fields of
        `explanation_parameters` are optional in the request. If a field of
        the `explanation_parameters` object is not populated, the
        corresponding field of the `Model.explanation_parameters` object is
        inherited.  For more details, see
        https://cloud.google.com/vertex-ai/docs/reference/rest/v1/ExplanationSpec#ExplanationParameters.
      labels (Optional[dict]):
        The labels with user-defined metadata to
        organize your BatchPredictionJobs.  Label keys and values can be no
        longer than 64 characters (Unicode codepoints), can only contain
        lowercase letters, numeric characters, underscores and dashes.
        International characters are allowed.  See https://goo.gl/xmQnxf for
        more information and examples of labels.
      encryption_spec_key_name (Optional[str]):
        Customer-managed encryption
        key options for a BatchPredictionJob. If this is set, then all
        resources created by the BatchPredictionJob will be encrypted with the
        provided encryption key.  Has the form:
        ``projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key``.
        The key needs to be in the same region as where the compute resource
        is created.

  Returns:
      batchpredictionjob (google.VertexBatchPredictionJob):
          [**Deprecated. Use gcs_output_directory and bigquery_output_table
          instead.**] Artifact
          representation of the created batch prediction job.
      gcs_output_directory (system.Artifact):
          Artifact tracking the batch prediction job output. This is only
          available if
          gcs_destination_output_uri_prefix is specified.
      bigquery_output_table (google.BQTable):
          Artifact tracking the batch prediction job output. This is only
          available if
          bigquery_output_table is specified.
      gcp_resources (str):
          Serialized gcp_resources proto tracking the batch prediction job.

          For more details, see
          https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  return ContainerSpec(
      image='gcr.io/ml-pipeline/google-cloud-pipeline-components:2.0.0b1',
      command=[
          'python3',
          '-u',
          '-m',
          'google_cloud_pipeline_components.container.v1.batch_prediction_job.launcher',
      ],
      args=[
          '--type',
          'BatchPredictionJob',
          '--payload',
          ConcatPlaceholder([
              '{',
              '"display_name": "',
              job_display_name,
              '", ',
              IfPresentPlaceholder(
                  input_name='model',
                  then=ConcatPlaceholder([
                      '"model": "',
                      "{{$.inputs.artifacts['model'].metadata['resourceName']}}",
                      '",',
                  ]),
              ),
              ' "input_config": {',
              '"instances_format": "',
              instances_format,
              '"',
              ', "gcs_source": {',
              '"uris":',
              gcs_source_uris,
              '}',
              ', "bigquery_source": {',
              '"input_uri": "',
              bigquery_source_input_uri,
              '"',
              '}',
              '}',
              ', "model_parameters": ',
              model_parameters,
              ', "output_config": {',
              '"predictions_format": "',
              predictions_format,
              '"',
              ', "gcs_destination": {',
              '"output_uri_prefix": "',
              gcs_destination_output_uri_prefix,
              '"',
              '}',
              ', "bigquery_destination": {',
              '"output_uri": "',
              bigquery_destination_output_uri,
              '"',
              '}',
              '}',
              ', "dedicated_resources": {',
              '"machine_spec": {',
              '"machine_type": "',
              machine_type,
              '"',
              ', "accelerator_type": "',
              accelerator_type,
              '"',
              ', "accelerator_count": ',
              accelerator_count,
              '}',
              ', "starting_replica_count": ',
              starting_replica_count,
              ', "max_replica_count": ',
              max_replica_count,
              '}',
              ', "manual_batch_tuning_parameters": {',
              '"batch_size": ',
              manual_batch_tuning_parameters_batch_size,
              '}',
              ', "generate_explanation": ',
              generate_explanation,
              ', "explanation_spec": {',
              '"parameters": ',
              explanation_parameters,
              ', "metadata": ',
              explanation_metadata,
              '}',
              ', "labels": ',
              labels,
              ', "encryption_spec": {"kms_key_name":"',
              encryption_spec_key_name,
              '"}',
              '}',
          ]),
          '--project',
          project,
          '--location',
          location,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
