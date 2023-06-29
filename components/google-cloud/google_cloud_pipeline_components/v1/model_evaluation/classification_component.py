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

from google_cloud_pipeline_components._implementation.model_evaluation import version
from google_cloud_pipeline_components.types.artifact_types import BQTable
from google_cloud_pipeline_components.types.artifact_types import ClassificationMetrics
from google_cloud_pipeline_components.types.artifact_types import VertexModel
from kfp import dsl
from kfp.dsl import container_component


@container_component
def model_evaluation_classification(
    gcp_resources: dsl.OutputPath(str),
    evaluation_metrics: dsl.Output[ClassificationMetrics],
    project: str,
    target_field_name: str,
    model: dsl.Input[VertexModel] = None,
    location: str = 'us-central1',
    predictions_format: str = 'jsonl',
    predictions_gcs_source: dsl.Input[dsl.Artifact] = None,
    predictions_bigquery_source: dsl.Input[BQTable] = None,
    ground_truth_format: str = 'jsonl',
    ground_truth_gcs_source: list = [],
    ground_truth_bigquery_source: str = '',
    classification_type: str = 'multiclass',
    class_labels: list = [],
    prediction_score_column: str = 'prediction.scores',
    prediction_label_column: str = 'prediction.classes',
    slicing_specs: list = [],
    positive_classes: list = [],
    dataflow_service_account: str = '',
    dataflow_disk_size_gb: int = 50,
    dataflow_machine_type: str = 'n1-standard-4',
    dataflow_workers_num: int = 1,
    dataflow_max_workers_num: int = 5,
    dataflow_subnetwork: str = '',
    dataflow_use_public_ips: bool = True,
    encryption_spec_key_name: str = '',
    force_runner_mode: str = '',
):
  # fmt: off
  """Computes a ``google.ClassificationMetrics`` Artifact, containing evaluation
  metrics given a model's prediction results.

  Creates a Dataflow job with Apache Beam and TFMA to compute evaluation
  metrics.
  Supports multiclass classification evaluation for tabular, image, video, and
  text data.

  Args:
      project: Project to run evaluation container.
      location: Location for running the evaluation.
      predictions_format: The file format for the batch
        prediction results. ``jsonl``, ``csv``, and ``bigquery`` are the allowed
        formats, from Vertex Batch Prediction.
      predictions_gcs_source: An artifact with its
        URI pointing toward a GCS directory with prediction or explanation files
        to be used for this evaluation. For prediction results, the files should
        be named "prediction.results-*" or "predictions_". For explanation
        results, the files should be named "explanation.results-*".
      predictions_bigquery_source: BigQuery table
        with prediction or explanation data to be used for this evaluation. For
        prediction results, the table column should be named "predicted_*".
      ground_truth_format: Required for custom tabular and non
        tabular data. The file format for the ground truth files. ``jsonl``,
        ``csv``, and ``bigquery`` are the allowed formats.
      ground_truth_gcs_source: Required for custom
        tabular and non tabular data. The GCS URIs representing where the ground
        truth is located. Used to provide ground truth for each prediction
        instance when they are not part of the batch prediction jobs prediction
        instance.
      ground_truth_bigquery_source: Required for custom tabular.
        The BigQuery table URI representing where the ground truth is located.
        Used to provide ground truth for each prediction instance when they are
        not part of the batch prediction jobs prediction instance.
      classification_type: The type of classification problem,
        either ``multiclass`` or ``multilabel``.
      class_labels: The list of class names for the
        target_field_name, in the same order they appear in the batch
        predictions jobs predictions output file. For instance, if the values of
        target_field_name could be either ``1`` or ``0``, and the predictions output
        contains ["1", "0"] for the prediction_label_column, then the
        class_labels input will be ["1", "0"]. If not set, defaults to the
        classes found in the prediction_label_column in the batch prediction
        jobs predictions file.
      target_field_name: The full name path of the features target field
        in the predictions file. Formatted to be able to find nested columns,
        delimited by ``.``. Alternatively referred to as the ground truth (or
        ground_truth_column) field.
      model: The Vertex model used for evaluation. Must be located in the same
        region as the location argument. It is used to set the default
        configurations for AutoML and custom-trained models.
      prediction_score_column: The column name of the field
        containing batch prediction scores. Formatted to be able to find nested
        columns, delimited by ``.``.
      prediction_label_column: The column name of the field
        containing classes the model is scoring. Formatted to be able to find
        nested columns, delimited by ``.``.
      slicing_specs: List of
        ``google.cloud.aiplatform_v1.types.ModelEvaluationSlice.SlicingSpec``. When
        provided, compute metrics for each defined slice. See sample code in
        https://cloud.google.com/vertex-ai/docs/pipelines/model-evaluation-component
        Below is an example of how to format this input.

        1: First, create a SlicingSpec.
          ``from google.cloud.aiplatform_v1.types.ModelEvaluationSlice.Slice import SliceSpec``

          ``from google.cloud.aiplatform_v1.types.ModelEvaluationSlice.Slice.SliceSpec import SliceConfig``

          ``slicing_spec = SliceSpec(configs={ 'feature_a': SliceConfig(SliceSpec.Value(string_value='label_a'))})``
        2: Create a list to store the slicing specs into.
          ``slicing_specs = []``
        3: Format each SlicingSpec into a JSON or Dict.
          ``slicing_spec_json = json_format.MessageToJson(slicing_spec)``
          or
          ``slicing_spec_dict = json_format.MessageToDict(slicing_spec)``
        4: Combine each slicing_spec JSON into a list.
          ``slicing_specs.append(slicing_spec_json)``
        5: Finally, pass slicing_specs as an parameter for this component.
          ``ModelEvaluationClassificationOp(slicing_specs=slicing_specs)``
        For more details on configuring slices, see
        https://cloud.google.com/python/docs/reference/aiplatform/latest/google.cloud.aiplatform_v1.types.ModelEvaluationSlice
      positive_classes: The list of class
        names to create binary classification metrics based on one-vs-rest for
        each value of positive_classes provided.
      dataflow_service_account: Service account to run
        the Dataflow job. If not set, Dataflow will use the default worker
        service account. For more details, see
        https://cloud.google.com/dataflow/docs/concepts/security-and-permissions#default_worker_service_account
      dataflow_disk_size_gb: The disk size (in GB) of the machine
        executing the evaluation run.
      dataflow_machine_type: The machine type executing the
        evaluation run.
      dataflow_workers_num: The number of workers executing the
        evaluation run.
      dataflow_max_workers_num: The max number of workers
        executing the evaluation run.
      dataflow_subnetwork: Dataflow's fully qualified subnetwork
        name, when empty the default subnetwork will be used. More
        details:
        https://cloud.google.com/dataflow/docs/guides/specifying-networks#example_network_and_subnetwork_specifications
      dataflow_use_public_ips: Specifies whether Dataflow
        workers use public IP addresses.
      encryption_spec_key_name:  Customer-managed encryption key options.
        If set, resources created by this pipeline will be encrypted with the
        provided encryption key. Has the form:
        ``projects/my-project/locations/my-location/keyRings/my-kr/cryptoKeys/my-key``.
        The key needs to be in the same region as where the compute resource is
        created.
      force_runner_mode: Flag to choose Beam runner. Valid options are ``DirectRunner``
        and ``Dataflow``.

  Returns:
      evaluation_metrics:
        ``google.ClassificationMetrics`` representing the classification
        evaluation metrics in GCS.
      gcp_resources: Serialized gcp_resources proto tracking the Dataflow
        job. For more details, see
        https://github.com/kubeflow/pipelines/blob/master/components/google-cloud/google_cloud_pipeline_components/proto/README.md.
  """
  # fmt: on
  return dsl.ContainerSpec(
      image=version.EVAL_IMAGE_TAG,
      command=[
          'python3',
          '/main.py',
      ],
      args=[
          '--setup_file',
          '/setup.py',
          '--json_mode',
          'true',
          '--project_id',
          project,
          '--location',
          location,
          '--problem_type',
          'classification',
          '--target_field_name',
          dsl.ConcatPlaceholder(['instance.', target_field_name]),
          '--batch_prediction_format',
          predictions_format,
          dsl.IfPresentPlaceholder(
              input_name='predictions_gcs_source',
              then=[
                  '--batch_prediction_gcs_source',
                  predictions_gcs_source.uri,
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='predictions_bigquery_source',
              then=[
                  '--batch_prediction_bigquery_source',
                  dsl.ConcatPlaceholder([
                      'bq://',
                      predictions_bigquery_source.metadata['projectId'],
                      '.',
                      predictions_bigquery_source.metadata['datasetId'],
                      '.',
                      predictions_bigquery_source.metadata['tableId'],
                  ]),
              ],
          ),
          dsl.IfPresentPlaceholder(
              input_name='model',
              then=[
                  '--model_name',
                  model.metadata['resourceName'],
              ],
          ),
          '--ground_truth_format',
          ground_truth_format,
          '--ground_truth_gcs_source',
          ground_truth_gcs_source,
          '--ground_truth_bigquery_source',
          ground_truth_bigquery_source,
          '--root_dir',
          f'{dsl.PIPELINE_ROOT_PLACEHOLDER}/{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--classification_type',
          classification_type,
          '--class_labels',
          class_labels,
          '--prediction_score_column',
          prediction_score_column,
          '--prediction_label_column',
          prediction_label_column,
          dsl.IfPresentPlaceholder(
              input_name='slicing_specs',
              then=[
                  '--slicing_specs',
                  slicing_specs,
              ],
          ),
          '--positive_classes',
          positive_classes,
          '--dataflow_job_prefix',
          f'evaluation-classification-{dsl.PIPELINE_JOB_ID_PLACEHOLDER}-{dsl.PIPELINE_TASK_ID_PLACEHOLDER}',
          '--dataflow_service_account',
          dataflow_service_account,
          '--dataflow_disk_size',
          dataflow_disk_size_gb,
          '--dataflow_machine_type',
          dataflow_machine_type,
          '--dataflow_workers_num',
          dataflow_workers_num,
          '--dataflow_max_workers_num',
          dataflow_max_workers_num,
          '--dataflow_subnetwork',
          dataflow_subnetwork,
          '--dataflow_use_public_ips',
          dataflow_use_public_ips,
          '--kms_key_name',
          encryption_spec_key_name,
          '--force_runner_mode',
          force_runner_mode,
          '--output_metrics_gcs_path',
          evaluation_metrics.path,
          '--gcp_resources',
          gcp_resources,
          '--executor_input',
          '{{$}}',
      ],
  )
